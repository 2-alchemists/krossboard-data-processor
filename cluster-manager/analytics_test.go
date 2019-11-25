package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"bitbucket.org/koamc/kube-opex-analytics-mc/kubeconfig"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestSpec(t *testing.T) {
	Convey("Given valid base config settings", t, func() {
		viper.Set("koamc_root_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", kubeconfig.UserHomeDir()))
		viper.Set("koamc_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("koamc_root_dir")))

		Convey("The call getAllClustersCurrentUsage should succeed", func() {
			allUsage, err := getAllClustersCurrentUsage()
			So(err, ShouldBeNil)
			b, _ := json.Marshal(allUsage)
			fmt.Println("\ntest output", string(b))
		})
	})
}

// func TestSpec(t *testing.T) {
// 	Convey("Given valid base config settings", t, func() {
// 		viper.Set("koamc_root_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", kubeconfig.UserHomeDir()))
// 		viper.Set("koamc_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("koamc_root_dir")))

// 		Convey("If a given cluster name is valid", func() {
// 			clusterName := "toto"

// 			Convey("The call getAllClustersCurrentUsage  should succeed", func() {
// 				_, err := getAllClustersCurrentUsage()
// 				So(err, ShouldBeNil)
// 			})
// 		})
// 	})
// }
