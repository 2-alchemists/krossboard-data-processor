package main

import (
	"os"

	"github.com/rchakode/kube-opex-analytics-mc/koainstance"
	"github.com/rchakode/kube-opex-analytics-mc/koakubeconfig"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("docker_api_version", "1.39")
	os.Setenv("DOCKER_API_VERSION", viper.GetString("docker_api_version"))

	koaClusters, err := koakubeconfig.FindClustersFromConfig()
	if err != nil {
		log.Fatalf("failed pulling container image: %v", err.Error())
	}

	for _, c := range koaClusters {
		log.Infoln(c.Name, c.APIEndpoint)
	}

	instance := koainstance.NewInstance("rchakode/kube-opex-analytics")

	err = instance.PullImage()
	if err != nil {
		log.Fatalf("failed pulling container image: %v", err.Error())
	}

	instance.HostPort = int64(15483)
	instance.ContainerPort = int64(5483)
	instance.ClusterName = "gke-1"
	instance.ClusterEndpoint = "http://127.0.0.1:8001"
	instance.DataVol = "/tmp/koa"
	instance.TokenVol = "/tmp/serviceaccount"
	err = instance.CreateContainer()
	if err != nil {
		log.Errorln("failed creating container:", err)
	} else {
		log.Infoln("container created:", instance.ID)
	}

}
