package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rchakode/kube-opex-analytics-mc/koainstance"
	"github.com/rchakode/kube-opex-analytics-mc/kubeconfig"
	"github.com/rchakode/kube-opex-analytics-mc/systemstatus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()

	// default config variables
	viper.SetDefault("docker_api_version", "1.39")
	viper.SetDefault("k8s_verify_ssl", "false")
	viper.SetDefault("gcp_gcloud_path", "gcloud")
	viper.SetDefault("koacm_update_interval", 30)

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

	updatePeriod := time.Duration(viper.GetInt64("koacm_update_interval")) * time.Minute
	kubeConfig := kubeconfig.NewKubeConfig()

	log.WithFields(log.Fields{
		"configDir":  viper.Get("koamc_config_dir"),
		"kubeconfig": kubeConfig.Path,
	}).Info("Service started successully")

	for {
		koaClusters, err := kubeConfig.ListClusters()
		if err != nil {
			log.Fatalf("failed pulling container image: %v", err.Error())
		}

		// get access token
		accessToken, err := kubeConfig.GetGKEAccessToken()
		if err != nil {
			log.Errorln("failed getting access token:", err.Error())
			time.Sleep(updatePeriod)
			continue
		}

		// udpate token file
		err = ioutil.WriteFile(viper.GetString("koamc_k8s_token_file"), []byte(accessToken), 0600)
		if err != nil {
			log.Error("failed writing token file", err.Error())
			time.Sleep(updatePeriod)
			continue
		}

		// Manage an instance for each context
		for _, cluster := range koaClusters {
			log.WithFields(log.Fields{
				"context":  cluster.Context,
				"endpoint": cluster.APIEndpoint,
			}).Debug("Start processing new context")

			if index, err := systemStatus.FindInstance(cluster.Context); err != nil || index >= 0 {
				if err != nil {
					log.WithFields(log.Fields{
						"context": cluster.Context,
						"message": err.Error(),
					}).Error("Failed finding instance")
				} else {
					log.WithFields(log.Fields{
						"context":     cluster.Context,
						"containerId": instanceSet.Instances[index].ID,
					}).Debug("Instance already exists")
				}
				continue
			}

			dataVol := fmt.Sprintf("%s/%s", viper.GetString("koamc_root_data_dir"), cluster.Context)
			err = createDirIfNotExists(dataVol)
			if err != nil {
				log.WithFields(log.Fields{
					"path":    dataVol,
					"message": err.Error(),
				}).Errorln("Failed creating data volume")
				time.Sleep(updatePeriod)
				break
			}

			instance := koainstance.NewInstance("rchakode/kube-opex-analytics")
			if err := instance.PullImage(); err != nil {
				log.WithFields(log.Fields{
					"image":   instance.Image,
					"message": err.Error(),
				}).Errorln("Failed pulling container image")
				time.Sleep(updatePeriod)
				break
			}
			instance.HostPort = int64(instanceSet.NextHostPort)
			instance.ContainerPort = int64(5483)
			instance.ClusterContext = cluster.Context
			instance.ClusterEndpoint = cluster.APIEndpoint
			instance.TokenVol = viper.GetString("koamc_credentials_dir")
			instance.DataVol = dataVol

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
				"context":     cluster.Context,
				"containerId": instance.ID,
			}).Info("New instance created")

			instanceSet.Instances = append(instanceSet.Instances, instance)
			instanceSet.NextHostPort++
			err := systemStatus.UpdateInstanceSet(instanceSet)
			if err != nil {
				log.WithFields(log.Fields{
					"context": cluster.Context,
					"message": err.Error(),
				}).Errorln("Failed to update system status")
				time.Sleep(updatePeriod)
				break // or exist ?
			}

			log.WithFields(log.Fields{
				"context":     cluster.Context,
				"containerId": instance.ID,
			}).Info("System status updated")
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
