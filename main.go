package main

import (
	"os"

	koa "github.com/rchakode/kube-opex-analytics-mc/koainstance"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("docker_api_version", "1.39")
	os.Setenv("DOCKER_API_VERSION", viper.GetString("docker_api_version"))

	instance := koa.NewKOAInstance("rchakode/kube-opex-analytics")
	instance.HostPort = int64(15483)
	instance.ContainerPort = int64(5483)
	instance.ClusterName = "gke-1"
	instance.ClusterEndpoint = "http://127.0.0.1:8001"
	instance.DataVol = "/tmp/koa"
	instance.TokenVol = "/tmp/serviceaccount"
	err := instance.CreateContainer()
	if err != nil {
		log.Println(err.Error())
	} else {
		log.Infoln("container created:", instance.ID)
	}

}
