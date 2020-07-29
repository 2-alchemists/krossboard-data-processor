package cmd

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestSpec(t *testing.T) {
	Convey("Given valid base config settings", t, func() {
		viper.Set("krossboard_root_dir", fmt.Sprintf("%s/.krossboard", UserHomeDir()))
		viper.Set("krossboard_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("krossboard_root_dir")))

		Convey("Empty tests", func() {
			So(nil, ShouldBeNil)
		})
	})
}

// func TestSpec(t *testing.T) {
// 	Convey("Given valid base config settings", t, func() {
// 		viper.Set("krossboard_root_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", UserHomeDir()))
// 		viper.Set("krossboard_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("krossboard_root_dir")))

// 		Convey("If a given cluster name is valid", func() {
// 			clusterName := "toto"

// 			Convey("The call getAllClustersCurrentUsage  should succeed", func() {
// 				_, err := getAllClustersCurrentUsage()
// 				So(err, ShouldBeNil)
// 			})
// 		})
// 	})
// }
