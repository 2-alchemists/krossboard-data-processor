package koainstance

import (
	"context"
	"fmt"
	"strconv"

	dkrTypes "github.com/docker/docker/api/types"
	dkrContainer "github.com/docker/docker/api/types/container"
	dkrMount "github.com/docker/docker/api/types/mount"
	dkrClient "github.com/docker/docker/client"
	dkrNat "github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
)

// KOAInstance hold a Kubernetes Opex Analytics instance info
type KOAInstance struct {
	ID              string `json:"id"`
	Image           string `json:"image"`
	HostPort        int64  `json:"hostPort"`
	ContainerPort   int64  `json:"containerPort"`
	ClusterName     string `json:"clusterName"`
	ClusterEndpoint string `json:"clusterEndpoint"`
	DataVol         string `json:"dataVol"`
	TokenVol        string `json:"tokenVol"`
}

// NewKOAInstance returns a pointer to KOAInstance object
func NewKOAInstance(image string) *KOAInstance {
	return &KOAInstance{
		Image: image,
	}
}

// CreateContainer creates a new container from given image
func (m *KOAInstance) CreateContainer() error {
	cli, err := dkrClient.NewClientWithOpts(dkrClient.FromEnv, dkrClient.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "unable to create docker client")
	}

	hostBinding := dkrNat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: strconv.FormatInt(m.HostPort, 10),
	}
	containerPort, err := dkrNat.NewPort("tcp", strconv.FormatInt(m.ContainerPort, 10))
	if err != nil {
		return errors.Wrap(err, "unable to get newPort")
	}

	portBindings := dkrNat.PortMap{
		containerPort: []dkrNat.PortBinding{hostBinding},
	}

	exposedPorts := dkrNat.PortSet{
		containerPort: struct{}{},
	}

	envars := []string{
		fmt.Sprintf("KOA_DB_LOCATION=%s", m.DataVol),
		fmt.Sprintf("KOA_K8S_API_ENDPOINT=%s", m.ClusterEndpoint),
	}

	mounts := []dkrMount.Mount{
		{
			Type:   dkrMount.TypeBind,
			Source: m.DataVol,
			Target: m.DataVol,
		},
		{
			Type:   dkrMount.TypeBind,
			Source: m.DataVol,
			Target: "/var/run/secrets/kubernetes.io/serviceaccount",
		},
	}
	
	volumes := map[string]struct{}{
		fmt.Sprintf("%s:%s", m.DataVol, m.DataVol): {},
	}

	cont, err := cli.ContainerCreate(
		context.Background(),
		&dkrContainer.Config{
			Image:        m.Image,
			Volumes:      volumes,
			Env:          envars,
			ExposedPorts: exposedPorts,
		},
		&dkrContainer.HostConfig{
			PortBindings: portBindings,
			Mounts:       mounts,
		},
		nil,
		"")
	if err != nil {
		return errors.Wrap(err, "ContainerCreate failed")
	}

	err = cli.ContainerStart(context.Background(), cont.ID, dkrTypes.ContainerStartOptions{})
	if err != nil {
		return errors.Wrap(err, "ContainerStart failed")
	}
	m.ID = cont.ID
	return nil
}

// StopContainer stops the container of given ID
func (m *KOAInstance) StopContainer() error {
	cli, err := dkrClient.NewClientWithOpts(dkrClient.FromEnv, dkrClient.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "unable to create docker client")
	}

	err = cli.ContainerStop(context.Background(), m.ID, nil)
	if err != nil {
		return errors.Wrap(err, "Stop container failed")
	}
	return nil
}
