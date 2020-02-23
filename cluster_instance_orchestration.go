package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func orchestrateInstances(systemStatus *SystemStatus) {
	workers.Add(1)
	defer workers.Done()

	containerManager := NewContainerManager(viper.GetString("krossboard_koainstance_image"))
	if err := containerManager.PullImage(); err != nil {
		log.WithFields(log.Fields{
			"image":   containerManager.Image,
			"message": err.Error(),
		}).Fatalln("failed pulling base container image")
	}

	kubeconfig := NewKubeConfig()
	log.WithFields(log.Fields{
		"kubeconfig": kubeconfig.Path,
	}).Debugln("KUBECONFIG selected")

	updatePeriodMin := time.Duration(viper.GetInt64("krossboard_update_interval_min")) * time.Minute
	orchestrationRoundErrors := int64(0)
	for {
		discoveredClusters, err := kubeconfig.ListClusters()
		if err != nil {
			log.WithError(err).Errorln("Failed reading discovered clusters")
			orchestrationRoundErrors += 1
			time.Sleep(time.Duration(fibonacci(orchestrationRoundErrors)) * time.Second)
			continue
		}

		runningConfig, err := systemStatus.GetInstances()
		if err != nil {
			log.WithField("message", err.Error()).Errorln("cannot load running configuration")
			orchestrationRoundErrors += 1
			time.Sleep(time.Duration(fibonacci(orchestrationRoundErrors)) * time.Second)
			continue
		}

		// Manage an instance for each cluster
		for _, cluster := range discoveredClusters {
			log.WithFields(log.Fields{
				"cluster":  cluster.Name,
				"endpoint": cluster.APIEndpoint,
			}).Debugln("processing new cluster")

			if cluster.AuthInfo == nil {
				log.WithField("cluster", cluster.Name).Warn("ignoring cluster with no AuthInfo")
				continue
			}

			dataVol := fmt.Sprintf("%s/%s", viper.GetString("krossboard_root_data_dir"), cluster.Name)
			err = createDirIfNotExists(dataVol)
			if err != nil {
				log.WithFields(log.Fields{"path": dataVol, "message": err.Error()}).Errorln("failed creating data volume")
				orchestrationRoundErrors += 1
				time.Sleep(time.Duration(fibonacci(orchestrationRoundErrors)) * time.Second)
				break
			}

			tokenVol := fmt.Sprintf("%s/%s", viper.GetString("krossboard_credentials_dir"), cluster.Name)
			err = createDirIfNotExists(tokenVol)
			if err != nil {
				log.WithFields(log.Fields{"path": tokenVol, "message": err.Error()}).Errorln("failed creating token volume")
				orchestrationRoundErrors += 1
				time.Sleep(time.Duration(fibonacci(orchestrationRoundErrors)) * time.Second)
				continue
			}

			caFile := fmt.Sprintf("%s/cacert.pem", tokenVol)
			err = ioutil.WriteFile(caFile, cluster.CaData, 0600)
			if err != nil {
				log.WithError(err).Errorln("failed writing CA file")
				continue
			}

			accessToken, err := kubeconfig.GetAccessToken(cluster.AuthInfo)
			if err != nil {
				log.WithField("cluster", cluster.Name).Error("failed getting access token from credentials plugin: ", err.Error())
				continue
			}
			tokenFile := fmt.Sprintf("%s/token", tokenVol)
			err = ioutil.WriteFile(tokenFile, []byte(accessToken), 0600)
			if err != nil {
				log.WithError(err).Errorln("failed writing token file")
				continue
			}

			if ii, err := systemStatus.FindInstance(cluster.Name); err != nil || ii >= 0 {
				if err != nil {
					log.WithFields(log.Fields{"cluster": cluster.Name, "message": err.Error()}).Errorln("failed finding instance")
				} else {
					log.WithFields(log.Fields{"cluster": cluster.Name, "containerId": runningConfig.Instances[ii].ID}).Debugln("instance found")
				}
				continue
			}

			rawName := fmt.Sprintf("%s-%v", cluster.Name, time.Now().Format("20060102T1504050700"))
			instance := &Instance{
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
				log.WithFields(log.Fields{"image": instance.Image, "message": err.Error()}).Errorln("Failed creating container")
				orchestrationRoundErrors += 1
				time.Sleep(time.Duration(fibonacci(orchestrationRoundErrors)) * time.Second)
				break
			}
			log.WithFields(log.Fields{"cluster": cluster.Name, "containerId": instance.ID}).Infoln("new instance created")

			runningConfig.Instances = append(runningConfig.Instances, instance)
			runningConfig.NextHostPort++
			err = systemStatus.UpdateRunningConfig(runningConfig)
			if err != nil {
				log.WithFields(log.Fields{"cluster": cluster.Name, "message": err.Error()}).Errorln("failed to update system status")
				orchestrationRoundErrors += 1
				time.Sleep(time.Duration(fibonacci(orchestrationRoundErrors)) * time.Second)
				break // or exit ?
			}

			log.Infoln("system status updated with cluster", cluster.Name)
		}
		orchestrationRoundErrors = 0
		time.Sleep(updatePeriodMin)
	}
}

func fibonacci(n int64) int64 {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}
