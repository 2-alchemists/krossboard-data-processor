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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const KrossboardVersion = "1.3.0"
const queryTimeLayout = "2006-01-02T15:04:05"

var rootCmd = &cobra.Command{
	Use:     "krossboard-data-processor",
	Short:   "Multi-cluster Kubernetes usage analytics tool",
	Long:    `Krossboard tracks the usage of your Kubernetes clusters at an one single place`,
	Version: KrossboardVersion,
	//	Run: func(cmd *cobra.Command, args []string) { },
}

var startAPIServiceCmd = &cobra.Command{
	Use:   "api",
	Short: "Start the REST API service",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infoln("API service started")
		startAPI()
	},
}

var startConsolidatorServiceCmd = &cobra.Command{
	Use:   "consolidator",
	Short: "Start the resource usage consolidator",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infoln("starting analytics consolidation")
		processConsolidatedUsage()
		log.Infoln("analytics consolidation completed")
	},
}

var startClusterCredentialsHandlerCmd = &cobra.Command{
	Use:   "cluster-credentials-handler",
	Short: "Start cluster credentials handler",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infoln("starting cluster credentials handler")
		processClusterCredentials()
		log.Infoln("cluster credentials handler completed")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(startAPIServiceCmd)
	rootCmd.AddCommand(startConsolidatorServiceCmd)
	rootCmd.AddCommand(startClusterCredentialsHandlerCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// handle default config variables
	viper.AutomaticEnv()
	// default parameters fixed parameters
	viper.SetDefault("krossboard_log_level", "info")
	viper.SetDefault("krossboard_cloud_provider", "AUTO")
	viper.SetDefault("krossboard_api_addr", "127.0.0.1:1519")
	viper.SetDefault("krossboard_k8s_verify_ssl", "true")
	viper.SetDefault("krossboard_koainstance_image", "rchakode/kube-opex-analytics:latest")
	viper.SetDefault("krossboard_koainstance_token_dir", "/var/run/secrets/kubernetes.io/serviceaccount")
	viper.SetDefault("krossboard_cost_model", "CUMULATIVE_RATIO")
	viper.SetDefault("krossboard_cors_origins", "*")
	viper.SetDefault("docker_api_version", "1.39")
	viper.SetDefault("krossboard_awscli_command", "aws")
	viper.SetDefault("krossboard_aws_metadata_service", "http://169.254.169.254")
	viper.SetDefault("krossboard_gcloud_command", "gcloud")
	viper.SetDefault("krossboard_gcp_metadata_service", "http://metadata.google.internal")
	viper.SetDefault("krossboard_az_command", "az")
	viper.SetDefault("krossboard_azure_metadata_service", "http://169.254.169.254")
	viper.SetDefault("krossboard_azure_keyvault_aks_password_secret", "krossboard-aks-password")
	viper.SetDefault("krossboard_root_dir", fmt.Sprintf("%s/.krossboard", UserHomeDir()))
	viper.SetDefault("krossboard_rawdb_dir", fmt.Sprintf("%s/db-raw", viper.GetString("krossboard_root_dir")))
	viper.SetDefault("krossboard_historydb_dir", fmt.Sprintf("%s/db-history", viper.GetString("krossboard_root_dir")))
	viper.SetDefault("krossboard_run_dir", fmt.Sprintf("%s/run", viper.GetString("krossboard_root_dir")))
	viper.SetDefault("krossboard_credentials_dir", fmt.Sprintf("%s/.cred", viper.GetString("krossboard_root_dir")))
	viper.SetDefault("krossboard_kubeconfig_dir", fmt.Sprintf("%s/kubeconfig.d", viper.GetString("krossboard_root_dir")))
	viper.SetDefault("krossboard_kubeconfig_max_size_kb", 10)
	viper.SetDefault("krossboard_k8s_api_endpoint", "https://kubernetes.default.svc")
	viper.SetDefault("krossboard_operator_api_version", "v1alpha1")

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	logLevel, err := log.ParseLevel(viper.GetString("krossboard_log_level"))
	if err != nil {
		log.WithError(err).Error("failed parsing log level")
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	// initialize folder tree
	err = createDirIfNotExists(viper.GetString("krossboard_root_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("failed initializing config directory")
	}
}

func createDirIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// RoundTime rounds the given time to the provided resolution.
func RoundTime(t time.Time, resolution time.Duration) time.Time {
	return time.Unix(0, (t.UnixNano()/resolution.Nanoseconds())*resolution.Nanoseconds())
}

func getHistoryDbPath(clusterName string) string {
	return fmt.Sprintf("%s/historydb-%s", viper.GetString("krossboard_historydb_dir"), clusterName)
}

func getCurrentClusterUsagePath() string {
	return fmt.Sprintf("%s/currentusage.json", viper.GetString("krossboard_run_dir"))
}

func listRegularFiles(folder string) ([]string, error) {
	if _, err := os.Stat(folder); err != nil {
		return nil, err
	}
	var files []string
	err := filepath.Walk(folder, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
			files = append(files, fmt.Sprintf("%s/%s", folder, info.Name()))
		}
		return nil
	})
	return files, err
}
