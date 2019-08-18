package kubeconfig

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"k8s.io/client-go/tools/clientcmd"
)

// KoaCluster holds an object describing a K8s Cluster
type KoaCluster struct {
	Name        string `json:"name,omitempty"`
	APIEndpoint string `json:"apiEndpoint,omitempty"`
}

// FindKoaClusters list Kubernetes clusters available in KUBECONFIG
func FindKoaClusters() ([]*KoaCluster, error) {
	var kubeconfig *string
	if home := koaHomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.LoadFromFile(*kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}

	koaClusters := []*KoaCluster{}
	for name, item := range config.Clusters {
		koaClusters = append(koaClusters, &KoaCluster{
			Name:        name,
			APIEndpoint: item.Server,
		})
	}

	return koaClusters, nil
}

func koaHomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
