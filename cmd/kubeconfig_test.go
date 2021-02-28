package cmd

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"io/ioutil"
	"testing"
)

func TestNewKubeConfig(t *testing.T) {
	Convey("Test KUBECONFIG settings", t, func() {

		kconfigDir := UserHomeDir()+"/.kube"
		_ = createDirIfNotExists(kconfigDir)

		_ = ioutil.WriteFile(kconfigDir+"/config",
			[]byte(`
apiVersion: v1
clusters:
- cluster:
    certificate-authority: fake-ca-file
    server: https://1.2.3.4
  name: development
- cluster:
    insecure-skip-tls-verify: true
    server: https://5.6.7.8
  name: scratch
contexts:
- context:
    cluster: development
    namespace: frontend
    user: developer
  name: dev-frontend
- context:
    cluster: development
    namespace: storage
    user: developer
  name: dev-storage
- context:
    cluster: scratch
    namespace: default
    user: experimenter
  name: exp-scratch
current-context: ""
kind: Config
preferences: {}
users:
- name: developer
  user:
    client-certificate: fake-cert-file
    client-key: fake-key-file
- name: experimenter
  user:
    password: some-password
    username: exp`), 0600)
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