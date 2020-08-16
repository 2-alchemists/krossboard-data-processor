package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	dkrTypes "github.com/docker/docker/api/types"
	dkrContainer "github.com/docker/docker/api/types/container"
	dkrFilters "github.com/docker/docker/api/types/filters"
	dkrMount "github.com/docker/docker/api/types/mount"
	dkrClient "github.com/docker/docker/client"
	dkrNat "github.com/docker/go-connections/nat"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Instance hold a Kubernetes Opex Analytics instance info
type Instance struct {
	ID              string    `json:"id,omitempty"`
	Name            string    `json:"name,omitempty"`
	Image           string    `json:"image,omitempty"`
	HostPort        int64     `json:"hostPort,omitempty"`
	ContainerPort   int64     `json:"containerPort,omitempty"`
	ClusterName     string    `json:"clusterNamet,omitempty"`
	ClusterEndpoint string    `json:"clusterEndpoint,omitempty"`
	DataVol         string    `json:"dataVol,omitempty"`
	TokenVol        string    `json:"tokenVol,omitempty"`
	CreationDate    time.Time `json:"creationDate,omitempty"`
}

// ContainerManager object to manipule containers
type ContainerManager struct {
	Image string `json:"image,omitempty"`
}

// NewContainerManager creates a new container manager object
func NewContainerManager(image string) *ContainerManager {
	return &ContainerManager{
		Image: image,
	}
}

// ImageExists returns true if the internal image exists locally
func (m *ContainerManager) ImageExists() bool {
	ctx := context.Background()
	cli, err := dkrClient.NewClientWithOpts(dkrClient.FromEnv, dkrClient.WithAPIVersionNegotiation())
	if err != nil {
		log.WithError(err).Debugln("failed instanciating a Docker client")
		return false
	}

	args := dkrFilters.NewArgs(dkrFilters.KeyValuePair{Key: "reference", Value: m.Image})
	images, err := cli.ImageList(ctx, dkrTypes.ImageListOptions{Filters: args})
	if err != nil {
		log.WithError(err).Debugf("failed looking up Docker image %v", m.Image)
		return false
	}

	return len(images) > 0
}

// PullImage pulls the referenced image
func (m *ContainerManager) PullImage() error {
	ctx := context.Background()
	cli, err := dkrClient.NewClientWithOpts(dkrClient.FromEnv, dkrClient.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "unable to create docker client")
	}

	reader, err := cli.ImagePull(ctx, m.Image, dkrTypes.ImagePullOptions{})
	if err != nil {
		return errors.Wrap(err, "failed pulling image")
	}

	io.Copy(os.Stdout, reader) //nolint:errcheck

	return nil
}

// CreateContainer creates a new container from given image
func (m *ContainerManager) CreateContainer(instance *Instance) error {
	cli, err := dkrClient.NewClientWithOpts(dkrClient.FromEnv, dkrClient.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "unable to create docker client")
	}

	hostBinding := dkrNat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: strconv.FormatInt(instance.HostPort, 10),
	}
	containerPort, err := dkrNat.NewPort("tcp", strconv.FormatInt(instance.ContainerPort, 10))
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
		fmt.Sprintf("KOA_DB_LOCATION=%s", instance.DataVol),
		fmt.Sprintf("KOA_K8S_API_ENDPOINT=%s", instance.ClusterEndpoint),
		fmt.Sprintf("KOA_K8S_API_VERIFY_SSL=%s", viper.GetString("krossboard_k8s_verify_ssl")),
		"KOA_K8S_CACERT=/var/run/secrets/kubernetes.io/serviceaccount/cacert.pem",
	}

	mounts := []dkrMount.Mount{
		{
			Type:   dkrMount.TypeBind,
			Source: instance.DataVol,
			Target: instance.DataVol,
		},
		{
			Type:   dkrMount.TypeBind,
			Source: instance.TokenVol,
			Target: "/var/run/secrets/kubernetes.io/serviceaccount",
		},
	}

	volumes := map[string]struct{}{
		fmt.Sprintf("%s:%s", instance.DataVol, instance.DataVol): {},
	}

	cont, err := cli.ContainerCreate(
		context.Background(),
		&dkrContainer.Config{
			Image:        instance.Image,
			Volumes:      volumes,
			Env:          envars,
			ExposedPorts: exposedPorts,
		},
		&dkrContainer.HostConfig{
			PortBindings: portBindings,
			Mounts:       mounts,
		},
		nil,
		nil,
		instance.Name)

	if err != nil {
		return errors.Wrap(err, "ContainerCreate failed")
	}

	err = cli.ContainerStart(context.Background(), cont.ID, dkrTypes.ContainerStartOptions{})
	if err != nil {
		return errors.Wrap(err, "ContainerStart failed")
	}
	instance.ID = cont.ID
	instance.CreationDate = time.Now().UTC()
	return nil
}

// PruneContainers clears all containers that are not running and returns the list of deleted items
func (m *ContainerManager) PruneContainers() ([]string, error) {

	cli, err := dkrClient.NewClientWithOpts(dkrClient.FromEnv, dkrClient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Wrap(err, "unable to create docker client")
	}

	var pruneReport dkrTypes.ContainersPruneReport
	pruneReport, err = cli.ContainersPrune(context.Background(), dkrFilters.Args{})
	if err != nil {
		return nil, errors.Wrap(err, "prune container failed")
	}
	return pruneReport.ContainersDeleted, nil
}

// GetAllContainersStates lists all the containers running on host machine
func (m *ContainerManager) GetAllContainersStates() (map[string]string, error) {
	cli, err := dkrClient.NewClientWithOpts(dkrClient.FromEnv, dkrClient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Wrap(err, "unable to get new docker client")
	}
	containers, err := cli.ContainerList(context.Background(), dkrTypes.ContainerListOptions{All: true})
	if err != nil {
		return nil, errors.Wrap(err, "unable to list containers")
	}
	cStates := make(map[string]string)
	for _, container := range containers {
		cStates[container.ID] = container.State
	}
	return cStates, nil
}
