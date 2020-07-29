package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

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
	} else {
		log.WithError(err).Debug("GCP cloud not detected")
	}
	_, err = getAWSRegion()
	if err == nil {
		return "AWS"
	} else {
		log.WithError(err).Debug("AWS cloud not detected")
	}
	_, err = getAzureSubscriptionID()
	if err == nil {
		return "AZURE"
	} else {
		log.WithError(err).Debug("Azure cloud not  detected")
	}

	return "UNDEFINED"
}
