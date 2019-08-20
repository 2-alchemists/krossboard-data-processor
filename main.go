package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rchakode/kube-opex-analytics-mc/koainstance"
	"github.com/rchakode/kube-opex-analytics-mc/kubeconfig"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()

	// default config variables
	viper.SetDefault("docker_api_version", "1.39")
	viper.SetDefault("k8s_verify_ssl", "true")
	viper.SetDefault("gcp_gcloud_path", "gcloud")

	// fixed config variables
	viper.Set("koa_config_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", kubeconfig.UserHomeDir()))
	viper.Set("koa_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("koa_config_dir")))
	viper.Set("koa_credentials_dir", fmt.Sprintf("%s/cred", viper.GetString("koa_config_dir")))
	viper.Set("koa_k8s_token_file", fmt.Sprintf("%s/token", viper.GetString("koa_credentials_dir")))

	// create config folder of not exist
	err := createDirIfNotExists(viper.GetString("koa_config_dir"))
	if err != nil {
		log.Fatalln("failed initializing config directory", err.Error())
	}

	// create credentials folder of not exist
	err = createDirIfNotExists(viper.GetString("koa_credentials_dir"))
	if err != nil {
		log.Fatalln("failed initializing credential directory", err.Error())
	}

	const UpdatePeriod = 30 * time.Minute
	for {
		kubeConfig := kubeconfig.NewKubeConfig()

		koaClusters, err := kubeConfig.ListClusters()
		if err != nil {
			log.Fatalf("failed pulling container image: %v", err.Error())
		}

		instance := koainstance.NewInstance("rchakode/kube-opex-analytics")
		err = instance.PullImage()
		if err != nil {
			log.Errorln("failed pulling container image:", err.Error())
			time.Sleep(UpdatePeriod)
			continue
		}

		// get access token
		accessToken, err := kubeConfig.GetGKEAccessToken()
		if err != nil {
			log.Errorln("failed getting access token:", err.Error())
			time.Sleep(UpdatePeriod)
			continue
		}

		// udpate token file
		err = ioutil.WriteFile(viper.GetString("koa_k8s_token_file"), []byte(accessToken), 0600)
		if err != nil {
			log.Error("failed writing token file", err.Error())
			time.Sleep(UpdatePeriod)
			continue
		}

		// Create or update instance for each cluster
		hostPort := 15483
		for _, cluster := range koaClusters {
			log.Infoln(cluster.Context, cluster.APIEndpoint)

			dataVol := fmt.Sprintf("%s/%s", viper.GetString("koa_root_data_dir"), cluster.Context)
			err = createDirIfNotExists(dataVol)
			if err != nil {
				log.Errorln("failed creating data volume:", err)
				time.Sleep(UpdatePeriod)
				continue
			}

			instance.HostPort = int64(hostPort)
			instance.ContainerPort = int64(5483)
			instance.ClusterName = cluster.Context
			instance.ClusterEndpoint = cluster.APIEndpoint
			instance.TokenVol = viper.GetString("koa_credentials_dir")
			instance.DataVol = dataVol

			err = instance.CreateContainer()
			if err != nil {
				log.Errorln("failed creating container:", err)
				time.Sleep(UpdatePeriod)
				continue
			}
			log.Infoln("container created:", instance.ID)
			hostPort++
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
