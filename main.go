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

	// fixed config variables
	viper.Set("koamc_config_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", kubeconfig.UserHomeDir()))
	viper.Set("koamc_status_file", fmt.Sprintf("%s/status.json", viper.GetString("koamc_config_dir")))
	viper.Set("koamc_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("koamc_config_dir")))
	viper.Set("koamc_credentials_dir", fmt.Sprintf("%s/cred", viper.GetString("koamc_config_dir")))
	viper.Set("koamc_k8s_token_file", fmt.Sprintf("%s/token", viper.GetString("koamc_credentials_dir")))

	// create config folder of not exist
	err := createDirIfNotExists(viper.GetString("koamc_config_dir"))
	if err != nil {
		log.Fatalln("failed initializing config directory", err.Error())
	}

	// create credentials folder of not exist
	err = createDirIfNotExists(viper.GetString("koamc_credentials_dir"))
	if err != nil {
		log.Fatalln("failed initializing credential directory", err.Error())
	}

	systemStatus := systemstatus.NewSystemStatus(viper.GetString("koamc_status_file"))
	err = systemStatus.InitializeStatusIfEmpty()
	if err != nil {
		log.Fatalln("cannot initialize status file", err.Error())
	}

	instanceSet, err := systemStatus.LoadInstanceSet()
	if err != nil {
		log.Fatalln("cannot load status file", err.Error())
	}

	const UpdatePeriod = 30 * time.Minute
	for {
		kubeConfig := kubeconfig.NewKubeConfig()

		koaClusters, err := kubeConfig.ListClusters()
		if err != nil {
			log.Fatalf("failed pulling container image: %v", err.Error())
		}

		// get access token
		accessToken, err := kubeConfig.GetGKEAccessToken()
		if err != nil {
			log.Errorln("failed getting access token:", err.Error())
			time.Sleep(UpdatePeriod)
			continue
		}

		// udpate token file
		err = ioutil.WriteFile(viper.GetString("koamc_k8s_token_file"), []byte(accessToken), 0600)
		if err != nil {
			log.Error("failed writing token file", err.Error())
			time.Sleep(UpdatePeriod)
			continue
		}

		// Create or update instance for each cluster
		for _, cluster := range koaClusters {
			log.Infoln(cluster.Context, cluster.APIEndpoint)

			if index, err := systemStatus.FindInstance(cluster.Context); err != nil || index >= 0 {
				if err != nil {
					log.Error("failed finding instance", err.Error())
				} else {
					log.Infoln("instance already exists")
				}
				continue
			}

			dataVol := fmt.Sprintf("%s/%s", viper.GetString("koamc_root_data_dir"), cluster.Context)
			err = createDirIfNotExists(dataVol)
			if err != nil {
				log.Errorln("failed creating data volume:", err)
				time.Sleep(UpdatePeriod)
				break
			}

			instance := koainstance.NewInstance("rchakode/kube-opex-analytics")
			if err := instance.PullImage(); err != nil {
				log.Errorln("failed pulling container image:", err.Error())
				time.Sleep(UpdatePeriod)
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
				log.Errorln("failed creating container:", err)
				time.Sleep(UpdatePeriod)
				break
			}
			log.Infoln("instance created:", instance.ID)
			instanceSet.Instances = append(instanceSet.Instances, instance)
			instanceSet.NextHostPort++
			err := systemStatus.UpdateInstanceSet(instanceSet)
			if err != nil {
				log.Errorln("failed to update system status:", err)
				time.Sleep(UpdatePeriod)
				break // or exist ?
			}
		}

		time.Sleep(UpdatePeriod)
	}
}

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}
