package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const queryTimeLayout = "2006-01-02T15:04:05"

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func getCloudProvider() string {
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
		log.WithError(err).Debug("Azure cloud not detected")
	}
	return viper.GetString("KROSSBOARD_CLOUD_PROVIDER")
}

// RoundTime rounds the given time to the provided resolution.
func RoundTime(t time.Time, resolution time.Duration) time.Time {
	return time.Unix(0, (t.UnixNano()/resolution.Nanoseconds())*resolution.Nanoseconds())
}


func getNodeUsagePath(clusterName string) string {
	return fmt.Sprintf("%s/.nodeusage_%s", viper.GetString("krossboard_root_data_dir"), clusterName)
}

func getUsageHistoryPath(clusterName string) string {
	return fmt.Sprintf("%s/.usagehistory_%s", viper.GetString("krossboard_root_data_dir"), clusterName)
}

func listRegularFiles(folder string) (error, []string) {
	if _, err := os.Stat(folder); err != nil {
		return err, nil
	}
	var files []string
	err := filepath.Walk(folder, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
			files = append(files, fmt.Sprintf("%s/%s", folder, info.Name()))
		}
		return nil
	})
	return err, files
}