package kubeconfig

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kapi "k8s.io/client-go/tools/clientcmd/api"
)

// KubeConfig holds an object describing a K8s Cluster
type KubeConfig struct {
	Path string `json:"path,omitempty"`
}

// ManagedCluster holds an object describing managed clusters
type ManagedCluster struct {
	Name        string         `json:"name,omitempty"`
	APIEndpoint string         `json:"apiEndpoint,omitempty"`
	AuthInfo    *kapi.AuthInfo `json:"authInfo,omitempty"`
}

// NewKubeConfig create a new KubeConfig object
func NewKubeConfig() *KubeConfig {
	var kubeconfig *string
	if home := UserHomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	return &KubeConfig{
		Path: *kubeconfig,
	}
}

// ListClusters list Kubernetes clusters available in KUBECONFIG
func (m *KubeConfig) ListClusters() (map[string]*ManagedCluster, error) {
	config, err := clientcmd.LoadFromFile(m.Path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}

	// TODO test to extract access token from config.AuthInfos
	// b, _ := json.Marshal(config.AuthInfos)
	// fmt.Println(string(b))

	authInfos := make(map[string]string)
	for user, authInfo := range config.AuthInfos {
		authInfos[user] = authInfo.Token
	}

	managedClusters := map[string]*ManagedCluster{}
	for clusterName, clusterInfo := range config.Clusters {
		managedClusters[clusterName] = &ManagedCluster{
			Name:        clusterName,
			APIEndpoint: clusterInfo.Server,
		}
	}
	for _, context := range config.Contexts {
		if cluster, found := managedClusters[context.Cluster]; found {
			cluster.AuthInfo = config.AuthInfos[context.AuthInfo]
		}
	}
	return managedClusters, nil
}

// GetAccessToken retrieves access token from credentials plugin
func (m *KubeConfig) GetAccessToken(authProvider *kapi.AuthProviderConfig) (string, error) {
	if authProvider == nil {
		return "", errors.New("nil exec object provided")

	}
	fmt.Println(authProvider.Config["cmd-path"])
	out, err := exec.Command(authProvider.Config["cmd-path"], strings.Split(authProvider.Config["cmd-args"], " ")...).Output()
	if err != nil {
		return "", errors.Wrap(err, "credentials plugin failed")
	}

	accessToken := ""
	if authProvider.Name == "gcp" {
		gcloudConfig := struct {
			Credential struct {
				AccessToken string `json:"access_token"`
			} `json:"credential,omitempty"`
		}{}
		err = json.Unmarshal(out, &gcloudConfig)
		if err != nil {
			return "", errors.Wrap(err, "failed decodng gcoud output")
		}
		accessToken = gcloudConfig.Credential.AccessToken
	}

	return accessToken, nil
}

func (m *KubeConfig) buildConfigFromFlags(contextName string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: m.Path},
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		}).ClientConfig()
}

// UserHomeDir returns the current use home directory
func UserHomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
