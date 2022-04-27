/*
Copyright (c) 2020 2Alchemists SAS.

This file is part of Krossboard.

Krossboard is free software: you can redistribute it and/or modify it under the terms of the
GNU General Public License as published by the Free Software Foundation, either version 3
of the License, or (at your option) any later version.

Krossboard is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with Krossboard.
If not, see <https://www.gnu.org/licenses/>.
*/

package cmd

import (
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"io/ioutil"
	"testing"
)

func TestNewKubeConfig(t *testing.T) {
	Convey("Test KUBECONFIG settings", t, func() {
		kubeconfigPath := "/tmp/config"
		_ = ioutil.WriteFile(kubeconfigPath,
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
		viper.Set("KUBECONFIG", kubeconfigPath)
		Convey("Test with no KUBECONFIG is set", func() {
			cfg := NewKubeConfig()
			So(len(cfg.Paths), ShouldEqual, 1)
			So(cfg.Paths[0], ShouldEqual, kubeconfigPath)
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