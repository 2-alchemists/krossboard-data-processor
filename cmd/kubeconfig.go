package cmd

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/buger/jsonparser"
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
	CaData      []byte         `json:"cacert,omitempty"`
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
		return nil, errors.Wrap(err, "failed loading KUBECONFIG")
	}

	authInfos := make(map[string]string)
	for user, authInfo := range config.AuthInfos {
		authInfos[user] = authInfo.Token
	}

	discoveredClusters := map[string]*ManagedCluster{}
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
	return discoveredClusters, nil
}

// GetAccessToken retrieves access token from AuthInfo
func (m *KubeConfig) GetAccessToken(authInfo *kapi.AuthInfo) (string, error) {
	if authInfo == nil {
		return "", errors.New("no AuthInfo provided")
	}

	if authInfo.Token != "" {
		return authInfo.Token, nil // AKS and alike
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

// nolint:unused // TODO: to remove?
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