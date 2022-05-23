/*
    Copyright (C) 2020  2ALCHEMISTS SAS.

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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func runClusterDataCollection() {

	cloudProvider := getCloudProvider()
	log.Infoln("detected cloud provider =>", cloudProvider)

	switch cloudProvider {
	case "AWS":
		updateEKSClusters()
	case "AZURE":
		updateAKSClusters()
	case "GCP":
		updateGKEClusters()
	default:
	}

	// load system statut
	systemStatus, err := LoadSystemStatus(viper.GetString("krossboard_status_file"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Cannot load system status")
	}

	// first cleanup orphaned instances

	containerManager := NewContainerManager("")
	currentContainers, err := containerManager.GetAllContainersStates()
	if err != nil {
		log.WithError(err).Fatalln("cannot get current containers")
	}

	containerNotRunningStates := map[string]bool{
		"exited": true,
		"dead":   true,
	}

	for cid, cstate := range currentContainers {
		if _, sfound := containerNotRunningStates[cstate]; sfound {
			err := systemStatus.RemoveInstanceByContainerID(cid)
			if err != nil {
				log.WithError(err).Errorln("failed cleaning from status database:", cid)
			} else {
				log.Infoln("instance cleaned", cid)
			}
		}
	}

	containersDeleted, err := containerManager.PruneContainers()
	if err != nil {
		log.WithError(err).Fatalln("cannot delete failed containers")
	} else {
		log.Infoln(len(containersDeleted), "not-running container(s) cleaned")
	}

	// now refresh instances
	orchestrateInstances(systemStatus, kubeconfig)
}
