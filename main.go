package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const ProgramVersion = "0.1.0"

var workers sync.WaitGroup

func main() {
	// parse options
	versionFlag := flag.Bool("version", false, "print the program version")
	flag.Parse()
	if *versionFlag {
		fmt.Printf("%v %v\n", path.Base(os.Args[0]), ProgramVersion)
		return
	}

	// handle startup settings
	viper.AutomaticEnv()
	viper.SetDefault("krossboard_log_level", "info")
	viper.SetDefault("krossboard_cloud_provider", "")
	viper.SetDefault("krossboard_api_addr", "127.0.0.1:1519")
	viper.SetDefault("krossboard_root_dir", fmt.Sprintf("%s/.krossboard", UserHomeDir()))
	viper.SetDefault("krossboard_k8s_verify_ssl", "true")
	viper.SetDefault("krossboard_update_interval_min", 30)
	viper.SetDefault("krossboard_koainstance_image", "rchakode/kube-opex-analytics:latest")
	viper.SetDefault("krossboard_cost_model", "CUMULATIVE_RATIO")
	viper.SetDefault("krossboard_cors_origins", "*")

	// Docker default settings
	viper.SetDefault("docker_api_version", "1.39")
	// AWS default settings
	viper.SetDefault("krossboard_awscli_command", "aws")
	viper.SetDefault("krossboard_aws_metadata_service", "http://169.254.169.254")
	// GCP default settings
	viper.SetDefault("krossboard_gcloud_command", "gcloud")
	viper.SetDefault("krossboard_gcp_metadata_service", "http://metadata.google.internal")
	// AZURE default settings
	viper.SetDefault("krossboard_az_command", "az")
	viper.SetDefault("krossboard_azure_metadata_service", "http://169.254.169.254")
	viper.SetDefault("krossboard_azure_keyvault_aks_password_secret", "krossboard-aks-password")

	// fixed config variables
	viper.Set("krossboard_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("krossboard_root_dir")))
	viper.Set("krossboard_credentials_dir", fmt.Sprintf("%s/cred", viper.GetString("krossboard_root_dir")))
	viper.Set("krossboard_status_dir", fmt.Sprintf("%s/run", viper.GetString("krossboard_root_dir")))
	viper.Set("krossboard_status_file", fmt.Sprintf("%s/instances.json", viper.GetString("krossboard_status_dir")))
	viper.Set("krossboard_current_usage_file", fmt.Sprintf("%s/currentusage.json", viper.GetString("krossboard_status_dir")))

	// configure logger
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	// FIXME: handle unauthorized copies
	// err := checkLicense()
	// if err != nil {
	// 	log.WithError(err).Fatalln("initilization failed")
	// 	return
	// }

	logLevel, err := log.ParseLevel(viper.GetString("krossboard_log_level"))
	if err != nil {
		log.WithError(err).Error("failed parsing log level")
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	// actual processing
	err = createDirIfNotExists(viper.GetString("krossboard_root_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing config directory")
	}

	err = createDirIfNotExists(viper.GetString("krossboard_status_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing status directory")
	}

	err = createDirIfNotExists(viper.GetString("krossboard_credentials_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Failed initializing credential directory")
	}

	systemStatus, err := LoadSystemStatus(viper.GetString("krossboard_status_file"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("Cannot load system status")
	}

	containerManager := NewContainerManager("")
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
	log.Infoln("discover started")

	go orchestrateInstances(systemStatus)
	log.Infoln("orchestrator started")

	go processConsolidatedUsage()
	log.Infoln("consolidator started")

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
	provider := viper.GetString("KROSSBOARD_CLOUD_PROVIDER")
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
