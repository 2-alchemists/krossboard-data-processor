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
