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
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"

	kclient "k8s.io/client-go/tools/clientcmd"
	kapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	AuthTypeUnknown     = 0
	AuthTypeBearerToken = 1
	AuthTypeX509Cert    = 2
	AuthTypeBasicToken  = 3
	KubeConfigKey       = "kubeconfig"
)

// KubeConfig holds an object describing a K8s Cluster
type KubeConfig struct {
	Paths []string `json:"path,omitempty"`
}

// ManagedCluster holds an object describing managed clusters
type ManagedCluster struct {
	Name        string         `json:"name,omitempty"`
	APIEndpoint string         `json:"apiEndpoint,omitempty"`
	AuthInfo    *kapi.AuthInfo `json:"authInfo,omitempty"`
	CaData      []byte         `json:"cacert,omitempty"`
	AuthType    int            `json:"authType,omitempty"`
}

// NewKubeConfig creates a new KubeConfig object
func NewKubeConfig() *KubeConfig {
	config := &KubeConfig{
		Paths: []string{},
	}

	kubeConfigEnv := viper.GetString(KubeConfigKey)
	if kubeConfigEnv != "" {
		config.Paths = append(config.Paths, strings.Split(kubeConfigEnv, ";")...)
		return config
	}

	var defaultKubeConfig *string
	if home := UserHomeDir(); home != "" {
		defaultKubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		defaultKubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	if _, err := os.Stat(*defaultKubeConfig); err != nil {
		log.WithError(err).Debugln("ignoring the default KUBECONFIG path", *defaultKubeConfig)
	} else {
		config.Paths = append(config.Paths, *defaultKubeConfig)
	}

	kconfigDir := viper.GetString("krossboard_kubeconfig_dir")
	err, kconfigFiles := listRegularFiles(kconfigDir)
	if err != nil {
		log.WithError(err).Debugln("ignoring KUBECONFIG directory", kconfigDir)
	} else {
		config.Paths = append(config.Paths, kconfigFiles...)
	}
	return config
}

// NewKubeConfig creates a new KubeConfig from a given file path
func NewKubeConfigFrom(path string) *KubeConfig {
	return &KubeConfig{
		Paths: []string{path},
	}
}

// ListClusters lists Kubernetes clusters available in KUBECONFIG
func (m *KubeConfig) ListClusters() map[string]*ManagedCluster {
	discoveredClusters := make(map[string]*ManagedCluster)
	for _, path := range m.Paths {
		config, err := kclient.LoadFromFile(path)
		if err != nil {
			log.WithError(err).Errorln("failed reading KUBECONFIG", path)
			continue
		}

		authInfos := make(map[string]string)
		for user, authInfo := range config.AuthInfos {
			authInfos[user] = authInfo.Token
		}

		for clusterName, clusterInfo := range config.Clusters {
			clusterNameEscaped := strings.ReplaceAll(clusterName, "/", "@")
			discoveredClusters[clusterNameEscaped] = &ManagedCluster{
				Name:        clusterNameEscaped,
				APIEndpoint: clusterInfo.Server,
				CaData:      clusterInfo.CertificateAuthorityData,
			}
		}
		for _, context := range config.Contexts {
			clusterNameEscaped := strings.ReplaceAll(context.Cluster, "/", "@")
			if cluster, found := discoveredClusters[clusterNameEscaped]; found {
				cluster.AuthInfo = config.AuthInfos[context.AuthInfo]
			}
		}
	}
	return discoveredClusters
}

// GetAccessToken retrieves access token from AuthInfo
func (m *KubeConfig) GetAccessToken(authInfo *kapi.AuthInfo) (string, error) {
	if authInfo == nil {
		return "", errors.New("no AuthInfo provided")
	}

	if authInfo.Token != "" {
		return authInfo.Token, nil // auth with Bearer token
	}

	authHookCmd := ""
	var args []string
	if authInfo.AuthProvider != nil {
		authHookCmd = authInfo.AuthProvider.Config["cmd-path"]
		args = strings.Split(authInfo.AuthProvider.Config["cmd-args"], " ")
	} else if authInfo.Exec != nil {
		authHookCmd = authInfo.Exec.Command
		args = authInfo.Exec.Args
	} else {
		return "", errors.New("no AuthInfo command provided")
	}

	cmd := exec.Command(authHookCmd, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, string(out))
	}

	token, err := jsonparser.GetString(out, "credential", "access_token") // GKE and alike
	if err != nil {
		errOut := errors.Wrap(err, "credentials string not compliant with GKE")
		token, err = jsonparser.GetString(out, "status", "token") // EKS and alike
		if err != nil {
			return "", errors.Wrap(errOut, "credentials string not compliant with EKS")
		}
	}

	return token, nil
}

// UserHomeDir returns the current use home directory
func UserHomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
