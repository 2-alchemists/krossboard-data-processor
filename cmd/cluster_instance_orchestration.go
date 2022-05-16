/*
Copyright (c) 2020 2Alchemists SAS.

This file is part of Krossboard.

Krossboard is free software: you can redistribute it and/or modify it under the terms of the
GNU General Public License as published by the Free Software Foundation, either version 3
of the License, or (at your option) any later version.

Krossboard is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with Krossboard.
If not, see <https://www.gnu.org/licenses/>.
*/

package cmd

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func orchestrateInstances(systemStatus *SystemStatus, kubeconfig *KubeConfig) {
	containerManager := NewContainerManager(viper.GetString("krossboard_koainstance_image"))

	if !containerManager.ImageExists() {
		log.Infoln("image does not exists, try to pulling it", containerManager.Image)
		if err := containerManager.PullImage(); err != nil {
			log.WithFields(log.Fields{
				"image":   containerManager.Image,
				"message": err.Error(),
			}).Fatalln("failed pulling base container image")
		}
	}

	discoveredClusters := kubeconfig.ListClusters()
	if len(discoveredClusters) == 0 {
		log.Errorln("no cluster to process")
		return
	}

	runningConfig, err := systemStatus.GetInstances()
	if err != nil {
		log.WithField("message", err.Error()).Errorln("cannot load running configuration")
		return
	}

	// Manage an instance for each cluster
	orchestrationRoundErrors := int64(0)
	for _, cluster := range discoveredClusters {
		log.WithFields(log.Fields{"cluster": cluster.Name, "endpoint": cluster.APIEndpoint}).Debugln("processing new cluster")

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
			continue
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

		tokenFile := fmt.Sprintf("%s/token", tokenVol)
		x509ClientCert := fmt.Sprintf("%s/cert.pem", tokenVol)
		x509ClientCertKey := fmt.Sprintf("%s/cert_key.pem", tokenVol)

		cluster.AuthType = AuthTypeUnknown
		bearerToken, err := kubeconfig.GetAccessToken(cluster.AuthInfo)
		if err == nil {
			err = ioutil.WriteFile(tokenFile, []byte(bearerToken), 0600)
			if err != nil {
				log.WithError(err).Errorln("failed writing bearer token file")
				continue
			}
			cluster.AuthType = AuthTypeBearerToken
		} else if cluster.AuthInfo.ClientCertificateData != nil && len(cluster.AuthInfo.ClientCertificateData) != 0 {
			if cluster.AuthInfo.ClientKeyData != nil && len(cluster.AuthInfo.ClientKeyData) != 0 {
				err = ioutil.WriteFile(x509ClientCert, cluster.AuthInfo.ClientCertificateData, 0644)
				if err != nil {
					log.WithError(err).Errorln("failed writing client certificate file")
					continue
				}
				err = ioutil.WriteFile(x509ClientCertKey, cluster.AuthInfo.ClientKeyData, 0600)
				if err != nil {
					log.WithError(err).Errorln("failed writing client certificate key file")
					continue
				}
				cluster.AuthType = AuthTypeX509Cert
			}
		} else if cluster.AuthInfo.Username != "" && cluster.AuthInfo.Password != "" {
			basicToken := base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cluster.AuthInfo.Username, cluster.AuthInfo.Password)))
			err = ioutil.WriteFile(tokenFile, []byte(basicToken), 0600)
			if err != nil {
				log.WithError(err).Errorln("failed writing basic token file")
				continue
			}
			cluster.AuthType = AuthTypeBasicToken
		} else {
			log.WithField("cluster", cluster.Name).Error("failed getting cluster credentials: ", err.Error())
			continue
		}

		instanceID, err := systemStatus.FindInstance(cluster.Name)
		if instanceID != "" {
			if err != nil {
				log.WithError(err).WithFields(log.Fields{"cluster": cluster.Name}).Debugln("failed querying instance info")
			}

			currentContainers, err := containerManager.GetAllContainersStates()
			if err != nil {
				log.WithError(err).WithFields(log.Fields{"cluster": cluster.Name, "containerId": instanceID}).Errorln("failed to get all containers states")
			}
			if _, cfound := currentContainers[instanceID]; cfound {
				log.WithFields(log.Fields{"cluster": cluster.Name, "containerId": instanceID}).Debugln("instance found")
				continue
			}

			// cleanup container in unexpected state and move forward
			err = systemStatus.RemoveInstanceByContainerID(instanceID)
			if err != nil {
				log.WithError(err).Errorln("failed cleaning from status database:", instanceID)
			} else {
				log.WithFields(log.Fields{"cluster": cluster.Name, "containerId": instanceID}).Infoln("instance cleaned")
			}
		}

		rawName := fmt.Sprintf("%s-%v", cluster.Name, time.Now().Format("20060102T1504050700"))
		instance := &Instance{
			Image:           containerManager.Image,
			Name:            strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(rawName, "@", "_"), ":", "_"), "/", "_"),
			HostPort:        int64(runningConfig.NextHostPort),
			ContainerPort:   int64(5483),
			ClusterName:     cluster.Name,
			ClusterEndpoint: cluster.APIEndpoint,
			TokenVol:        tokenVol,
			DataVol:         dataVol,
			AuthType:        cluster.AuthType,
		}

		err = containerManager.CreateContainer(instance)
		if err != nil {
			log.WithFields(log.Fields{"image": instance.Image, "message": err.Error()}).Errorln("failed creating container")
			orchestrationRoundErrors += 1
			time.Sleep(time.Duration(fibonacci(orchestrationRoundErrors)) * time.Second)
			continue
		}
		log.WithFields(log.Fields{"cluster": cluster.Name, "containerId": instance.ID}).Infoln("new instance created")

		runningConfig.Instances = append(runningConfig.Instances, instance)
		runningConfig.NextHostPort++
		err = systemStatus.UpdateRunningConfig(runningConfig)
		if err != nil {
			log.WithFields(log.Fields{"cluster": cluster.Name, "message": err.Error()}).Errorln("failed to update system status")
			orchestrationRoundErrors += 1
			time.Sleep(time.Duration(fibonacci(orchestrationRoundErrors)) * time.Second)
			continue
		}

		log.Infoln("system status updated with cluster", cluster.Name)
	}
}

func fibonacci(n int64) int64 {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}
