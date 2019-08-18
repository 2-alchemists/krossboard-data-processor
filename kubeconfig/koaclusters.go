package kubeconfig

import (
	"encoding/base64"
	"flag"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KoaCluster holds an object describing a K8s Cluster
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
	if home := userHomeDir(); home != "" {
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

// GetBearerTokenForCluster retrieves the Bearer token for the connected user
func (m *KubeConfig) GetBearerTokenForCluster(contextName string) (string, error) {
	config, err := m.buildConfigFromFlags(contextName)
	if err != nil {
		return "", errors.Wrap(err, "failed to load config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", errors.Wrap(err, "failed to create clientset")
	}

	secretList, err := clientset.CoreV1().Secrets("").List(metav1.ListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to listing secrets")
	}

	tokenFound := false
	tokenRaw := ""
	for _, secret := range secretList.Items {
		if anno, found := secret.Annotations["kubernetes.io/service-account.name"]; found {
			if anno == "default" {
				tokenRaw = base64.StdEncoding.EncodeToString(secret.Data[corev1.ServiceAccountTokenKey])
				tokenFound = true
				break
			}
		}
	}

	if !tokenFound {
		return "", errors.New("no token found for the default serviceaccount")
	}

	tokenBase64 := make([]byte, base64.StdEncoding.DecodedLen(len(tokenRaw)))
	n, err := base64.StdEncoding.Decode(tokenBase64, []byte(tokenRaw))
	if err != nil {
		return "", errors.Wrap(err, "failed decoding token")
	}

	return string(tokenBase64[:n]), nil
}

func (m *KubeConfig) buildConfigFromFlags(contextName string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: m.Path},
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		}).ClientConfig()
}

func userHomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
