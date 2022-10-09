/*
   Copyright (C) 2022  2ALCHEMISTS SAS.
   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as
   published by the Free Software Foundation, either version 3 of the
   License, or (at your option) any later version.
   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.
   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package cmd

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func processClusterCredentials() {
	kubeconfig := NewKubeConfig()

	managedClusters := kubeconfig.ListClusters()
	if len(managedClusters) == 0 {
		log.Errorln("no cluster found in KUBECONFIG")
		return
	}

	for _, managedCluster := range managedClusters {
		log.WithFields(log.Fields{"cluster": managedCluster.Name, "endpoint": managedCluster.APIEndpoint}).Debugln("processing new cluster")

		if managedCluster.AuthInfo == nil {
			log.WithField("cluster", managedCluster.Name).Warn("ignoring cluster with no AuthInfo")
			continue
		}

		dataVol := fmt.Sprintf("%s/%s", viper.GetString("krossboard_data_dir"), managedCluster.Name)
		err := createDirIfNotExists(dataVol)
		if err != nil {
			log.WithFields(log.Fields{"path": dataVol, "message": err.Error()}).Errorln("failed creating data volume")
			continue
		}

		tokenVol := fmt.Sprintf("%s/%s", viper.GetString("krossboard_credentials_dir"), managedCluster.Name)
		err = createDirIfNotExists(tokenVol)
		if err != nil {
			log.WithFields(log.Fields{"path": tokenVol, "message": err.Error()}).Errorln("failed creating token volume")
			continue
		}

		caFile := fmt.Sprintf("%s/cacert.pem", tokenVol)
		err = ioutil.WriteFile(caFile, managedCluster.CaData, 0600)
		if err != nil {
			log.WithError(err).Errorln("failed writing CA file")
			continue
		}

		tokenFile := fmt.Sprintf("%s/token", tokenVol)
		x509ClientCert := fmt.Sprintf("%s/cert.pem", tokenVol)
		x509ClientCertKey := fmt.Sprintf("%s/cert_key.pem", tokenVol)

		managedCluster.AuthType = AuthTypeUnknown
		bearerToken, err := kubeconfig.GetAccessToken(managedCluster.AuthInfo)
		if err == nil {
			err = ioutil.WriteFile(tokenFile, []byte(bearerToken), 0600)
			if err != nil {
				log.WithError(err).Errorln("failed writing bearer token file")
				continue
			}
			managedCluster.AuthType = AuthTypeBearerToken
		} else if managedCluster.AuthInfo.ClientCertificateData != nil && len(managedCluster.AuthInfo.ClientCertificateData) != 0 {
			if managedCluster.AuthInfo.ClientKeyData != nil && len(managedCluster.AuthInfo.ClientKeyData) != 0 {
				err = ioutil.WriteFile(x509ClientCert, managedCluster.AuthInfo.ClientCertificateData, 0644)
				if err != nil {
					log.WithError(err).Errorln("failed writing client certificate file")
					continue
				}
				err = ioutil.WriteFile(x509ClientCertKey, managedCluster.AuthInfo.ClientKeyData, 0600)
				if err != nil {
					log.WithError(err).Errorln("failed writing client certificate key file")
					continue
				}
				managedCluster.AuthType = AuthTypeX509Cert
			}
		} else if managedCluster.AuthInfo.Username != "" && managedCluster.AuthInfo.Password != "" {
			basicToken := base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", managedCluster.AuthInfo.Username, managedCluster.AuthInfo.Password)))
			err = ioutil.WriteFile(tokenFile, []byte(basicToken), 0600)
			if err != nil {
				log.WithError(err).Errorln("failed writing basic token file")
				continue
			}
			managedCluster.AuthType = AuthTypeBasicToken
		} else {
			log.WithField("cluster", managedCluster.Name).Error("failed getting cluster credentials: ", err.Error())
			continue
		}

		log.Infoln("cluster credentials updated =>", managedCluster.Name)
	}
}
