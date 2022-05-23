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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

const KrossboardVersion = "1.2.0"

var kubeconfig *KubeConfig

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
		processConsolidatedUsage(kubeconfig)
		log.Infoln(" analytics consolidation completed")
	},
}
var startCollectorServiceCmd = &cobra.Command{
	Use:   "collector",
	Short: "Run the clusters data collector",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infoln("starting clusters data collection")
		runClusterDataCollection()
		log.Infoln("clusters data collection completed")
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
	rootCmd.AddCommand(startCollectorServiceCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// handle default config variables
	viper.AutomaticEnv()
	// default parameters fixed parameters
	viper.SetDefault("krossboard_log_level", "info")
	viper.SetDefault("krossboard_cloud_provider", "AUTO")
	viper.SetDefault("krossboard_api_addr", "127.0.0.1:1519")
	viper.SetDefault("krossboard_root_dir", fmt.Sprintf("%s/.krossboard", UserHomeDir()))
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

	// fixed environment variables
	os.Setenv("DOCKER_API_VERSION", viper.GetString("docker_api_version"))

	// fixed parameters
	viper.Set("krossboard_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("krossboard_root_dir")))
	viper.Set("krossboard_credentials_dir", fmt.Sprintf("%s/.cred", viper.GetString("krossboard_root_dir")))
	viper.Set("krossboard_status_dir", fmt.Sprintf("%s/run", viper.GetString("krossboard_root_dir")))
	viper.Set("krossboard_status_file", fmt.Sprintf("%s/instances.json", viper.GetString("krossboard_status_dir")))
	viper.Set("krossboard_current_usage_file", fmt.Sprintf("%s/currentusage.json", viper.GetString("krossboard_status_dir")))
	viper.Set("krossboard_kubeconfig_dir", fmt.Sprintf("%s/kubeconfig.d", viper.GetString("krossboard_root_dir")))
	viper.Set("krossboard_kubeconfig_max_size_kb", 10)

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

	err = createDirIfNotExists(viper.GetString("krossboard_status_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("failed initializing status directory")
	}

	err = createDirIfNotExists(viper.GetString("krossboard_credentials_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("failed initializing credential directory")
	}

	kubeconfig = NewKubeConfig()

}
