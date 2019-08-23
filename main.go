package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"time"

	gcontainerv1 "cloud.google.com/go/container/apiv1"
	"github.com/rchakode/kube-opex-analytics-mc/koainstance"
	"github.com/rchakode/kube-opex-analytics-mc/kubeconfig"
	"github.com/rchakode/kube-opex-analytics-mc/systemstatus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	gcontainerpbv1 "google.golang.org/genproto/googleapis/container/v1"
)

var workers sync.WaitGroup

func main() {
	viper.AutomaticEnv()
	// default config variables
	viper.SetDefault("docker_api_version", "1.39")
	viper.SetDefault("k8s_verify_ssl", "false")
	viper.SetDefault("koacm_update_interval", 30)
	viper.SetDefault("koacm_default_image", "rchakode/kube-opex-analytics")
	viper.SetDefault("google_gcloud_command_path", "gcloud")

	// fixed config variables
	viper.Set("koamc_config_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", kubeconfig.UserHomeDir()))
	viper.Set("koamc_status_file", fmt.Sprintf("%s/status.json", viper.GetString("koamc_config_dir")))
	viper.Set("koamc_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("koamc_config_dir")))
	viper.Set("koamc_credentials_dir", fmt.Sprintf("%s/cred", viper.GetString("koamc_config_dir")))
	viper.Set("koamc_k8s_token_file", fmt.Sprintf("%s/token", viper.GetString("koamc_credentials_dir")))

	// create config folder of not exist
	err := createDirIfNotExists(viper.GetString("koamc_config_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing config directory")
	}

	// create credentials folder if not exist
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
		log.WithField("message", err.Error()).Fatalln("Cannot load instance set")
	}

	workers.Add(2)

	go orchestrateInstances(systemStatus, instanceSet)
	go updateGKEClusters()

	workers.Wait()
}

func orchestrateInstances(systemStatus *systemstatus.SystemStatus, instanceSet *systemstatus.InstanceSet) {
	defer workers.Done()

	updatePeriod := time.Duration(viper.GetInt64("koacm_update_interval")) * time.Minute
	kubeConfig := kubeconfig.NewKubeConfig()

	log.WithFields(log.Fields{
		"configDir":  viper.Get("koamc_config_dir"),
		"kubeconfig": kubeConfig.Path,
	}).Infoln("Service started successully")

	for {
		managedClusters, err := kubeConfig.ListClusters()
		if err != nil {
			log.WithError(err).Errorln("Failed to get clusters from KUBECONFIG")
			time.Sleep(updatePeriod)
			continue
		}

		// Manage an instance for each cluster
		for _, cluster := range managedClusters {
			log.WithFields(log.Fields{
				"cluster":  cluster.Name,
				"endpoint": cluster.APIEndpoint,
			}).Debugln("Start processing new cluster")

			if cluster.AuthInfo == nil || cluster.AuthInfo.AuthProvider == nil {
				log.WithField("cluster", cluster.Name).Warn("Ignoring cluster with either no auth info or no auth provider")
				continue
			}

			// get credentials plugin and write in a file
			accessToken, err := kubeConfig.GetAccessToken(cluster.AuthInfo.AuthProvider)
			if err != nil {
				log.Errorln("failed getting access token:", err.Error())
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
	// loop periodically after each updatePeriod
	for {
		listReq := &gcontainerpbv1.ListClustersRequest{
			// TODO :: Idea of extension: automatically discover all projects associated to the users and list all clusters included
			Parent: fmt.Sprintf("projects/%v/locations/-", viper.GetString("google_project_id")),
		}

		listResp, err := clusterManagerClient.ListClusters(ctx, listReq)
		if err != nil {
			log.WithError(err).Errorln("Failed to list GKE clusters")
			time.Sleep(updatePeriod)
			continue
		}

		for _, cluster := range listResp.Clusters {
			_, err := exec.Command(viper.GetString("google_gcloud_command_path"),
				"container",
				"clusters",
				"get-credentials",
				cluster.Name).Output()
			if err != nil {
				log.WithError(err).Errorf("Failed getting credentials for GKE cluster %v", cluster.Name)
			} else {
				log.WithField("clusterName", cluster.Name).Debugln("Added/updated credentials for GKE cluster in KUBECONFIG")
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
