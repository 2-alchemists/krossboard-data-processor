package kubeconfig

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeConfig holds an object describing a K8s Cluster
type KubeConfig struct {
	Path string `json:"path,omitempty"`
}

// KoaCluster holds an object describing a K8s Cluster
type KoaCluster struct {
	Context     string `json:"context,omitempty"`
	APIEndpoint string `json:"apiEndpoint,omitempty"`
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
func (m *KubeConfig) ListClusters() (map[string]*KoaCluster, error) {
	config, err := clientcmd.LoadFromFile(m.Path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}

	koaClusters := map[string]*KoaCluster{}
	for name, item := range config.Clusters {
		koaClusters[name] = &KoaCluster{
			APIEndpoint: item.Server,
		}
	}
	for name, item := range config.Contexts {
		if kc, found := koaClusters[item.Cluster]; found {
			kc.Context = name
		}
	}
	return koaClusters, nil
}

// GetGKEAccessToken retrieves access token from GKE credentials plugin
func (m *KubeConfig) GetGKEAccessToken() (string, error) {
	out, err := exec.Command(viper.GetString("gcp_gcloud_path"), "config", "config-helper", "--format=json").Output()
	if err != nil {
		return "", errors.Wrap(err, "'gcoud config config-helper' failed")
	}

	gcloudConfig := struct {
		Credential struct {
			AccessToken string `json:"access_token"`
		} `json:"credential,omitempty"`
	}{}
	err = json.Unmarshal(out, &gcloudConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed decodng gcoud output")
	}

	return gcloudConfig.Credential.AccessToken, nil
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
