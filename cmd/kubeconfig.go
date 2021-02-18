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

	"k8s.io/client-go/tools/clientcmd"
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
	} else {
		var pathPtr *string
		if home := UserHomeDir(); home != "" {
			pathPtr = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			pathPtr = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		config.Paths = append(config.Paths, *pathPtr)
	}
	return config
}

// ListClusters lists Kubernetes clusters available in KUBECONFIG
func (m *KubeConfig) ListClusters() map[string]*ManagedCluster {
	discoveredClusters := make(map[string]*ManagedCluster)
	for _, path := range m.Paths {
		config, err := clientcmd.LoadFromFile(path)
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
