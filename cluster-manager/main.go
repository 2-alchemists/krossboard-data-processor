package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"

	"bitbucket.org/koamc/kube-opex-analytics-mc/koainstance"
	"bitbucket.org/koamc/kube-opex-analytics-mc/kubeconfig"
	"bitbucket.org/koamc/kube-opex-analytics-mc/systemstatus"
	gcontainerv1 "cloud.google.com/go/container/apiv1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/buger/jsonparser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	gcontainerpbv1 "google.golang.org/genproto/googleapis/container/v1"
)

var workers sync.WaitGroup

func main() {
	viper.AutomaticEnv()
	// default config variables
	viper.SetDefault("koamc_log_level", "http://metadata.google.internal")
	viper.SetDefault("docker_api_version", "1.39")
	viper.SetDefault("koacm_k8s_verify_ssl", "false")
	viper.SetDefault("koacm_update_interval", 30)
	viper.SetDefault("koacm_default_image", "rchakode/kube-opex-analytics")
	viper.SetDefault("koamc_gcloud_command", "gcloud")
	viper.SetDefault("koamc_awscli_command", "aws")
	viper.SetDefault("koacm_eksctl_command", "eksctl")
	viper.SetDefault("koamc_root_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", kubeconfig.UserHomeDir()))
	viper.SetDefault("koamc_cloud_provider", "")
	viper.SetDefault("koamc_aws_metadata_service", "http://169.254.169.254")
	viper.SetDefault("koamc_gcp_metadata_service", "http://metadata.google.internal")
	viper.SetDefault("koamc_log_level", "warn")

	// fixed config variables
	viper.Set("koamc_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("koamc_root_dir")))
	viper.Set("koamc_credentials_dir", fmt.Sprintf("%s/cred", viper.GetString("koamc_root_dir")))
	viper.Set("koamc_k8s_token_file", fmt.Sprintf("%s/token", viper.GetString("koamc_credentials_dir")))
	viper.Set("koamc_status_dir", fmt.Sprintf("%s/run", viper.GetString("koamc_root_dir")))
	viper.Set("koamc_status_file", fmt.Sprintf("%s/status.json", viper.GetString("koamc_status_dir")))

	// configure logger
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	logLevel, err := log.ParseLevel(viper.GetString("koamc_log_level"))
	if err != nil {
		log.WithError(err).Error("failed parsing log level")
		logLevel = log.WarnLevel
	}
	log.SetLevel(logLevel)

	// actual processing
	err = createDirIfNotExists(viper.GetString("koamc_root_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing config directory")
	}

	err = createDirIfNotExists(viper.GetString("koamc_status_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing status directory")
	}

	err = createDirIfNotExists(viper.GetString("koamc_credentials_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing credential directory")
	}

	systemStatus, err := systemstatus.LoadSystemStatus(viper.GetString("koamc_status_file"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Cannot load system status")
	}

	instanceSet, err := systemStatus.LoadInstanceSet()
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("cannot load instance set")
	}

	workers.Add(2)

	cloudProvider := getCloudProvider()
	switch cloudProvider {
	case "GCP":
		go updateGKEClusters()
	case "AWS":
		go updateEKSClusters()
	default:
		log.Fatalln("not supported cloud provider:", cloudProvider)
	}

	go orchestrateInstances(systemStatus, instanceSet)

	workers.Wait()
}

func orchestrateInstances(systemStatus *systemstatus.SystemStatus, instanceSet *systemstatus.InstanceSet) {
	defer workers.Done()

	updatePeriod := time.Duration(viper.GetInt64("koacm_update_interval")) * time.Minute
	kubeConfig := kubeconfig.NewKubeConfig()

	log.WithFields(log.Fields{
		"configDir":  viper.Get("koamc_root_dir"),
		"kubeconfig": kubeConfig.Path,
	}).Infoln("service started successully")

	for {
		managedClusters, err := kubeConfig.ListClusters()
		if err != nil {
			log.WithError(err).Errorln("Failed getting clusters from KUBECONFIG")
			time.Sleep(updatePeriod)
			continue
		}

		// Manage an instance for each cluster
		for _, cluster := range managedClusters {
			log.WithFields(log.Fields{
				"cluster":  cluster.Name,
				"endpoint": cluster.APIEndpoint,
			}).Debugln("processing new cluster")

			if cluster.AuthInfo == nil {
				log.WithField("cluster", cluster.Name).Warn("ignoring cluster with no AuthInfo")
				continue
			}

			accessToken, err := kubeConfig.GetAccessToken(cluster.AuthInfo)
			if err != nil {
				log.WithField("cluster", cluster.Name).Warn("failed authenticating to cluster: ", err.Error())
				continue
			}

			// TODO use a separated credentials volume per container ?
			err = ioutil.WriteFile(viper.GetString("koamc_k8s_token_file"), []byte(accessToken), 0600)
			if err != nil {
				log.Errorln("failed writing token file", err.Error())
				continue
			}

			if index, err := systemStatus.FindInstance(cluster.Name); err != nil || index >= 0 {
				if err != nil {
					log.WithFields(log.Fields{
						"cluster": cluster.Name,
						"message": err.Error(),
					}).Errorln("Failed finding instance")
				} else {
					// TODO check if instance is running or not
					log.WithFields(log.Fields{
						"cluster":     cluster.Name,
						"containerId": instanceSet.Instances[index].ID,
					}).Debugln("Instance already exists")
				}
				continue
			}

			dataVol := fmt.Sprintf("%s/%s", viper.GetString("koamc_root_data_dir"), cluster.Name)
			err = createDirIfNotExists(dataVol)
			if err != nil {
				log.WithFields(log.Fields{
					"path":    dataVol,
					"message": err.Error(),
				}).Errorln("Failed creating data volume")
				time.Sleep(updatePeriod)
				break
			}

			instance := koainstance.NewInstance(viper.GetString("koacm_default_image"))
			instance.HostPort = int64(instanceSet.NextHostPort)
			instance.ContainerPort = int64(5483)
			instance.ClusterName = cluster.Name
			instance.ClusterEndpoint = cluster.APIEndpoint
			instance.TokenVol = viper.GetString("koamc_credentials_dir")
			instance.DataVol = dataVol

			if err := instance.PullImage(); err != nil {
				log.WithFields(log.Fields{
					"image":   instance.Image,
					"message": err.Error(),
				}).Errorln("Failed pulling container image")
				time.Sleep(updatePeriod)
				break
			}

			err = instance.CreateContainer()
			if err != nil {
				log.WithFields(log.Fields{
					"image":   instance.Image,
					"message": err.Error(),
				}).Errorln("Failed creating container")
				time.Sleep(updatePeriod)
				break
			}
			log.WithFields(log.Fields{
				"cluster":     cluster.Name,
				"containerId": instance.ID,
			}).Infoln("New instance created")

			instanceSet.Instances = append(instanceSet.Instances, instance)
			instanceSet.NextHostPort++
			err = systemStatus.UpdateInstanceSet(instanceSet)
			if err != nil {
				log.WithFields(log.Fields{
					"cluster": cluster.Name,
					"message": err.Error(),
				}).Errorln("Failed to update system status")
				time.Sleep(updatePeriod)
				break // or exist ?
			}

			log.Infoln("System status updated")
		}

		time.Sleep(updatePeriod)
	}
}

func updateGKEClusters() {
	defer workers.Done()

	ctx := context.Background()
	clusterManagerClient, err := gcontainerv1.NewClusterManagerClient(ctx)
	if err != nil {
		log.WithError(err).Errorln("Failed to instanciate GKE cluster manager")
		return
	}

	updatePeriod := time.Duration(viper.GetInt64("koacm_update_interval")) * time.Minute
	for {
		// TODO An Idea of extension would be to automatically discover all projects associated
		// to the authentocated users and list all GKE clusters included
		// Doc: https://godoc.org/google.golang.org/api/cloudresourcemanager/v1beta1
		projectID, err := getGCPProjectID()
		if projectID <= int64(0) {
			log.WithError(err).Errorln("Unable to retrieve GCP project ID")
			time.Sleep(updatePeriod)
			continue
		}
		listReq := &gcontainerpbv1.ListClustersRequest{
			Parent: fmt.Sprintf("projects/%v/locations/-", projectID),
		}
		listResp, err := clusterManagerClient.ListClusters(ctx, listReq)
		if err != nil {
			log.WithError(err).Errorln("Failed to list GKE clusters")
			time.Sleep(updatePeriod)
			continue
		}

		for _, cluster := range listResp.Clusters {
			_, err := exec.Command(viper.GetString("koamc_gcloud_command"),
				"container",
				"clusters",
				"get-credentials",
				cluster.Name,
				"--zone",
				cluster.Zone).Output()

			if err != nil {
				log.WithError(err).Errorf("Failed getting credentials for GKE cluster %v", cluster.Name)
			} else {
				log.WithField("clusterName", cluster.Name).Debugln("Added/updated credentials for GKE cluster in KUBECONFIG")
			}
		}

		time.Sleep(updatePeriod)
	}
}

func updateEKSClusters() {
	defer workers.Done()

	updatePeriod := time.Duration(viper.GetInt64("koacm_update_interval")) * time.Minute
	for {
		awsRegion, err := getAWSRegion()
		if err != nil {
			log.WithError(err).Error("cannot retrieve AWS region")
			time.Sleep(updatePeriod)
			continue
		}
		svc := eks.New(session.New(), aws.NewConfig().WithRegion("us-west-2"))
		listInput := &eks.ListClustersInput{}
		listResult, err := svc.ListClusters(listInput)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				log.WithError(err).Errorf("failed listing EKS clusters (%v)", aerr.Code())
			} else {
				log.WithError(err).Error("failed listing EKS clusters")
			}
			time.Sleep(updatePeriod)
			continue
		}
		for _, clusterName := range listResult.Clusters {
			_, err := exec.Command(viper.GetString("koamc_awscli_command"),
				"eks",
				"update-kubeconfig",
				"--name",
				*clusterName,
				"--region",
				awsRegion).Output()

			if err != nil {
				log.WithError(err).Errorf("failed getting credentials for GKE cluster %v", clusterName)
			} else {
				log.WithField("clusterName", clusterName).Debugln("added/updated credentials for GKE cluster in KUBECONFIG")
			}
		}

		time.Sleep(updatePeriod)
	}
}

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func getAWSRegion() (string, error) {
	timeout := time.Duration(time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET",
		viper.GetString("koamc_aws_metadata_service")+"/latest/meta-data/instance-identity/document",
		nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed calling AWS metadata service")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed ready response from GCP metadata server")
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("AWS metadata service returned error: " + string(body))
	}

	out, err := jsonparser.GetString(body, "region")
	if err != nil {
		return "", errors.Wrap(err, "unexpected metatdata content")
	}

	return out, nil
}

func getGCPProjectID() (int64, error) {
	timeout := time.Duration(time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET",
		viper.GetString("koamc_gcp_metadata_service")+"/computeMetadata/v1/project/numeric-project-id",
		nil)
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return viper.GetInt64("google_project_id"), errors.Wrap(err, "failed calling GCP metadata server")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, errors.Wrap(err, "failed ready response from GCP metadata server")
	}

	if resp.StatusCode != http.StatusOK {
		return -1, errors.New("GCP metadata server returned error: " + string(body))
	}
	projectID, err := strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		return -1, errors.Wrap(err, "unexpected non integer value as project id")

	}

	return projectID, nil
}

func getCloudProvider() string {
	provider := viper.GetString("KOAMC_CLOUD_PROVIDER")
	if provider == "" {
		_, err := getGCPProjectID()
		if err != nil {
			log.WithError(err).Debug("not AWS cloud")
			_, err = getAWSRegion()
			if err != nil {
				log.WithError(err).Debug("not GCP cloud")
				provider = "UNDEFINED"
			} else {
				provider = "AWS"
			}
		} else {
			provider = "GCP"
		}
	}
	return provider
}
