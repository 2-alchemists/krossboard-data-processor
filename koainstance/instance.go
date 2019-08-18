package koainstance

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	dkrTypes "github.com/docker/docker/api/types"
	dkrContainer "github.com/docker/docker/api/types/container"
	dkrMount "github.com/docker/docker/api/types/mount"
	dkrClient "github.com/docker/docker/client"
	dkrNat "github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
)

// Instance hold a Kubernetes Opex Analytics instance info
type Instance struct {
	ID              string `json:"id,omitempty"`
	Image           string `json:"image,omitempty"`
	HostPort        int64  `json:"hostPort,omitempty"`
	ContainerPort   int64  `json:"containerPort,omitempty"`
	ClusterName     string `json:"clusterName,omitempty"`
	ClusterEndpoint string `json:"clusterEndpoint,omitempty"`
	DataVol         string `json:"dataVol,omitempty"`
	TokenVol        string `json:"tokenVol,omitempty"`
}

// NewInstance returns a pointer to an Instance object
func NewInstance(image string) *Instance {
	return &Instance{
		Image: image,
	}
}

// PullImage pulls image
func (m *Instance) PullImage() error {
	ctx := context.Background()
	cli, err := dkrClient.NewClientWithOpts(dkrClient.FromEnv, dkrClient.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "unable to create docker client")
	}

	reader, err := cli.ImagePull(ctx, m.Image, dkrTypes.ImagePullOptions{})
	if err != nil {
		return errors.Wrap(err, "failed pulling image")
	}
	io.Copy(os.Stdout, reader)

	return nil
}

// CreateContainer creates a new container from given image
func (m *Instance) CreateContainer() error {
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
			Source: m.TokenVol,
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
func (m *Instance) StopContainer() error {
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