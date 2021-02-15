package cmd

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"testing"
)

func TestNewKubeConfig(t *testing.T) {
	Convey("Test KUBECONFIG settings", t, func() {

		Convey("Test with no KUBECONFIG is set", func() {
			cfg := NewKubeConfig()
			So(len(cfg.Paths), ShouldEqual, 1)
			So(cfg.Paths[0], ShouldEqual, fmt.Sprintf("%s/.kube/config", UserHomeDir()))
		})


		Convey("Empty tests", func() {
			viper.Set(KubeConfigKey, "/opt/krossboard/.kube/config-cluster-1;/opt/krossboard/.kube/config-cluster-2")
			cfg := NewKubeConfig()
			So(len(cfg.Paths), ShouldEqual, 2)
			So(cfg.Paths[0], ShouldEqual, "/opt/krossboard/.kube/config-cluster-1")
			So(cfg.Paths[0], ShouldNotEqual, "/opt/krossboard/.kube/config-cluster-4")
			So(cfg.Paths[1], ShouldEqual, "/opt/krossboard/.kube/config-cluster-2")
			So(cfg.Paths[1], ShouldNotBeEmpty)
		})
	})
}