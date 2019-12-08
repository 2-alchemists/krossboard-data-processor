package main

import (
	"fmt"
	"os"
	"sync"

	"bitbucket.org/koamc/kube-opex-analytics-mc/koainstance"
	"bitbucket.org/koamc/kube-opex-analytics-mc/kubeconfig"
	"bitbucket.org/koamc/kube-opex-analytics-mc/systemstatus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var workers sync.WaitGroup

func main() {
	viper.AutomaticEnv()
	// KOAMC default settings
	viper.SetDefault("koamc_cloud_provider", "")
	viper.SetDefault("koamc_api_addr", "127.0.0.1:1519")
	viper.SetDefault("koamc_log_level", "http://metadata.google.internal")
	viper.SetDefault("koamc_root_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", kubeconfig.UserHomeDir()))
	viper.SetDefault("koacm_k8s_verify_ssl", "true")
	viper.SetDefault("koacm_update_interval", 30)
	viper.SetDefault("koacm_default_image", "rchakode/kube-opex-analytics")
	viper.SetDefault("koamc_log_level", "info")
	// Docker default settings
	viper.SetDefault("docker_api_version", "1.39")
	// AWS default settings
	viper.SetDefault("koamc_awscli_command", "aws")
	viper.SetDefault("koamc_aws_metadata_service", "http://169.254.169.254")
	// GCP default settings
	viper.SetDefault("koamc_gcloud_command", "gcloud")
	viper.SetDefault("koamc_gcp_metadata_service", "http://metadata.google.internal")
	// AZURE default settings
	viper.SetDefault("koamc_az_command", "az")
	viper.SetDefault("koamc_azure_metadata_service", "http://169.254.169.254")
	viper.SetDefault("koamc_azure_keyvault_aks_password_secret", "koamc-aks-password")

	// fixed config variables
	viper.Set("koamc_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("koamc_root_dir")))
	viper.Set("koamc_credentials_dir", fmt.Sprintf("%s/cred", viper.GetString("koamc_root_dir")))
	viper.Set("koamc_status_dir", fmt.Sprintf("%s/run", viper.GetString("koamc_root_dir")))
	viper.Set("koamc_status_file", fmt.Sprintf("%s/instances.json", viper.GetString("koamc_status_dir")))
	viper.Set("koamc_current_usage_file", fmt.Sprintf("%s/currentusage.json", viper.GetString("koamc_status_dir")))

	// configure logger
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	logLevel, err := log.ParseLevel(viper.GetString("koamc_log_level"))
	if err != nil {
		log.WithError(err).Error("failed parsing log level")
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	// actual processing
	err = createDirIfNotExists(viper.GetString("koamc_root_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing config directory")
	}

	err = createDirIfNotExists(viper.GetString("koamc_status_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing status directory")
	}

	err = createDirIfNotExists(viper.GetString("koamc_credentials_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing credential directory")
	}

	systemStatus, err := systemstatus.LoadSystemStatus(viper.GetString("koamc_status_file"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Cannot load system status")
	}

	containerManager := koainstance.NewContainerManager("")
	cStates, err := containerManager.GetAllContainersStates()
	if err != nil {
		log.WithError(err).Fatalln("cannot list containers")
	}
	containerNotRunningStatus := map[string]bool{
		"exited": true,
		"dead":   true,
	}
	for cID, cState := range cStates {
		if _, statusFound := containerNotRunningStatus[cState]; statusFound {
			err := systemStatus.RemoveInstanceByContainerID(cID)
			if err != nil {
				log.WithError(err).Errorln("failed cleaning from status database:", cID)
			} else {
				log.Infoln("instance cleaned from status database:", cID)
			}
		}
	}

	containersDeleted, err := containerManager.PruneContainers()
	if err != nil {
		log.WithError(err).Fatalln("cannot delete failed containers")
	} else {
		log.Infoln(len(containersDeleted), "not running container(s) cleaned")
	}

	workers.Add(2)
	cloudProvider := getCloudProvider()
	log.Infoln("cloud provider =>", cloudProvider)
	switch cloudProvider {
	case "AWS":
		go updateEKSClusters()
	case "AZURE":
		go updateAKSClusters()
	case "GCP":
		go updateGKEClusters()
	default:
		log.Fatalln("unauthorized execution environment:", cloudProvider)
	}

	go orchestrateInstances(systemStatus)

	log.Infoln("service started")

	startAPI()

	workers.Wait()
}

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func getCloudProvider() string {
	provider := viper.GetString("KOAMC_CLOUD_PROVIDER")
	if provider != "" {
		return provider
	}
	_, err := getGCPProjectID()
	if err == nil {
		return "GCP"
	}
	_, err = getAWSRegion()
	if err == nil {
		return "AWS"
	}
	_, err = getAzureSubscriptionID()
	if err == nil {
		return "AZURE"
	}
	return "UNDEFINED"
}
