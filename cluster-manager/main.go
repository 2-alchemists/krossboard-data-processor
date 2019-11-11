package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	gcontainerpbv1 "google.golang.org/genproto/googleapis/container/v1"
)

var workers sync.WaitGroup

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("koamc_api_addr", "127.0.0.1:1519")
	viper.SetDefault("koamc_log_level", "http://metadata.google.internal")
	viper.SetDefault("docker_api_version", "1.39")
	viper.SetDefault("koacm_k8s_verify_ssl", "true")
	viper.SetDefault("koacm_update_interval", 30)
	viper.SetDefault("koacm_default_image", "rchakode/kube-opex-analytics")
	viper.SetDefault("koamc_gcloud_command", "gcloud")
	viper.SetDefault("koamc_awscli_command", "aws")
	viper.SetDefault("koacm_eksctl_command", "eksctl")
	viper.SetDefault("koamc_root_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", kubeconfig.UserHomeDir()))
	viper.SetDefault("koamc_cloud_provider", "")
	viper.SetDefault("koamc_aws_metadata_service", "http://169.254.169.254")
	viper.SetDefault("koamc_gcp_metadata_service", "http://metadata.google.internal")
	viper.SetDefault("koamc_log_level", "info")

	// fixed config variables
	viper.Set("koamc_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("koamc_root_dir")))
	viper.Set("koamc_credentials_dir", fmt.Sprintf("%s/cred", viper.GetString("koamc_root_dir")))
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
		logLevel = log.InfoLevel
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

	containerManager := koainstance.NewContainerManager("")
	cStates, err := containerManager.GetAllContainersStates()
	if err != nil {
		log.WithError(err).Fatalln("cannot list containers")
	}
	containerNotRunningStatus := map[string]bool{
		"exited": true,
		"dead":   true,
	}
	for cID, cState := range cStates {
		if _, statusFound := containerNotRunningStatus[cState]; statusFound {
			err := systemStatus.RemoveInstanceByContainerID(cID)
			if err != nil {
				log.WithError(err).Errorln("failed cleaning from status database:", cID)
			} else {
				log.Infoln("instance cleaned from status database:", cID)
			}
		}
	}

	containersDeleted, err := containerManager.PruneContainers()
	if err != nil {
		log.WithError(err).Fatalln("cannot delete failed containers")
	} else {
		log.Infoln(len(containersDeleted), "not running container(s) cleaned")
	}

	workers.Add(2)
	cloudProvider := getCloudProvider()
	switch cloudProvider {
	case "GCP":
		go updateGKEClusters()
	case "AWS":
		go updateEKSClusters()
	default:
		log.Fatalln("invalid or unauthorized execution environment:", cloudProvider)
	}

	go orchestrateInstances(systemStatus)

	log.Infoln("service started")

	startAPI()

	workers.Wait()
}

func orchestrateInstances(systemStatus *systemstatus.SystemStatus) {
	defer workers.Done()

	const WaitSecondBeforePulling = 1
	log.Infof("wait %v seconds before starting instance orchestration\n", WaitSecondBeforePulling)
	time.Sleep(WaitSecondBeforePulling * time.Second)
	log.Infoln("starting instance orchestration")

	containerManager := koainstance.NewContainerManager(viper.GetString("koacm_default_image"))
	if err := containerManager.PullImage(); err != nil {
		log.WithFields(log.Fields{
			"image":   containerManager.Image,
			"message": err.Error(),
		}).Fatalln("failed pulling base container image")
	}

	kubeConfig := kubeconfig.NewKubeConfig()
	log.WithFields(log.Fields{
		"kubeconfig": kubeConfig.Path,
	}).Infoln("KUBECONFIG loaded")

	updatePeriod := time.Duration(viper.GetInt64("koacm_update_interval")) * time.Minute
	for {
		managedClusters, err := kubeConfig.ListClusters()
		if err != nil {
			log.WithError(err).Errorln("Failed reading KUBECONFIG")
			time.Sleep(updatePeriod)
			continue
		}

		runningConfig, err := systemStatus.GetInstances()
		if err != nil {
			log.WithField("message", err.Error()).Errorln("cannot load running configuration")
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

			dataVol := fmt.Sprintf("%s/%s", viper.GetString("koamc_root_data_dir"), cluster.Name)
			err = createDirIfNotExists(dataVol)
			if err != nil {
				log.WithFields(log.Fields{
					"path":    dataVol,
					"message": err.Error(),
				}).Errorln("failed creating data volume")
				time.Sleep(updatePeriod)
				break
			}

			tokenVol := fmt.Sprintf("%s/%s", viper.GetString("koamc_credentials_dir"), cluster.Name)
			err = createDirIfNotExists(tokenVol)
			if err != nil {
				log.WithFields(log.Fields{
					"path":    tokenVol,
					"message": err.Error(),
				}).Errorln("failed creating token volume")
				time.Sleep(updatePeriod)
				continue
			}

			caFile := fmt.Sprintf("%s/cacert.pem", tokenVol)
			err = ioutil.WriteFile(caFile, cluster.CaData, 0600)
			if err != nil {
				log.Errorln("failed writing CA file", err.Error())
				continue
			}

			accessToken, err := kubeConfig.GetAccessToken(cluster.AuthInfo)
			if err != nil {
				log.WithField("cluster", cluster.Name).Warn("failed getting access token from credentials plugin: ", err.Error())
				continue
			}
			tokenFile := fmt.Sprintf("%s/token", tokenVol)
			err = ioutil.WriteFile(tokenFile, []byte(accessToken), 0600)
			if err != nil {
				log.Errorln("failed writing token file", err.Error())
				continue
			}

			if ii, err := systemStatus.FindInstance(cluster.Name); err != nil || ii >= 0 {
				if err != nil {
					log.WithFields(log.Fields{
						"cluster": cluster.Name,
						"message": err.Error(),
					}).Errorln("failed finding instance")
				} else {
					log.WithFields(log.Fields{
						"cluster":     cluster.Name,
						"containerId": runningConfig.Instances[ii].ID,
					}).Debugln("instance found")
				}
				continue
			}

			rawName := fmt.Sprintf("%s-%v", cluster.Name, time.Now().Format("20060102T1504050700"))
			instance := &koainstance.Instance{
				Image:           containerManager.Image,
				Name:            strings.Replace(strings.Replace(rawName, ":", "_", -1), "/", "_", -1),
				HostPort:        int64(runningConfig.NextHostPort),
				ContainerPort:   int64(5483),
				ClusterName:     cluster.Name,
				ClusterEndpoint: cluster.APIEndpoint,
				TokenVol:        tokenVol,
				DataVol:         dataVol,
			}

			err = containerManager.CreateContainer(instance)
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
			}).Infoln("new instance created")

			runningConfig.Instances = append(runningConfig.Instances, instance)
			runningConfig.NextHostPort++
			err = systemStatus.UpdateRunningConfig(runningConfig)
			if err != nil {
				log.WithFields(log.Fields{
					"cluster": cluster.Name,
					"message": err.Error(),
				}).Errorln("failed to update system status")
				time.Sleep(updatePeriod)
				break // or exit ?
			}

			log.Infoln("system status updated")
		}

		time.Sleep(updatePeriod)
	}
}

func updateGKEClusters() {
	defer workers.Done()

	ctx := context.Background()
	clusterManagerClient, err := gcontainerv1.NewClusterManagerClient(ctx)
	if err != nil {
		log.WithError(err).Errorln("failed to instanciate GKE cluster manager")
		return
	}

	updatePeriod := time.Duration(viper.GetInt64("koacm_update_interval")) * time.Minute
	for {
		projectID, err := getGCPProjectID()
		if projectID <= int64(0) {
			log.WithError(err).Errorln("unable to retrieve GCP project ID")
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
			cmdout, err := exec.Command(viper.GetString("koamc_gcloud_command"),
				"container",
				"clusters",
				"get-credentials",
				cluster.Name,
				"--zone",
				cluster.Zone).CombinedOutput()

			if err != nil {
				log.WithField("cluster", cluster.Name).Errorln("failed getting credentials from GKE cluster:", string(cmdout))
			} else {
				log.WithField("cluster", cluster.Name).Debugln("added/updated credentials for GKE cluster in KUBECONFIG")
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
		svc := eks.New(session.New(), aws.NewConfig().WithRegion(awsRegion))
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
			cmdout, err := exec.Command(viper.GetString("koamc_awscli_command"),
				"eks",
				"update-kubeconfig",
				"--name",
				*clusterName,
				"--region",
				awsRegion).CombinedOutput()

			if err != nil {
				log.WithField("cluster", clusterName).Errorf("failed getting credentials for GKE cluster %v", string(cmdout))
			} else {
				log.WithField("cluster", clusterName).Debugln("added/updated credentials for GKE cluster in KUBECONFIG")
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
		viper.GetString("koamc_aws_metadata_service")+"/latest/meta-data/placement/availability-zone",
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

	bodyText := string(body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("AWS metadata service returned error: " + bodyText)
	}

	return bodyText[:len(bodyText)-1], nil
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
		return viper.GetInt64("google_project_id"), errors.Wrap(err, "failed calling GCP metadata service")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, errors.Wrap(err, "failed ready response from GCP metadata service")
	}

	if resp.StatusCode != http.StatusOK {
		return -1, errors.New("GCP metadata server returned error: " + string(body))
	}
	projectID, err := strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		return -1, errors.Wrap(err, "unexpected non integer for project id")
	}

	return projectID, nil
}

func getCloudProvider() string {
	provider := viper.GetString("KOAMC_CLOUD_PROVIDER")
	if provider == "" {
		_, err := getGCPProjectID()
		if err != nil {
			log.WithError(err).Debug("not GCP cloud")
			_, err = getAWSRegion()
			if err != nil {
				log.WithError(err).Debug("not AWS cloud")
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
