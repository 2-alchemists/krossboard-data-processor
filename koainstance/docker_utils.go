package koainstance

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// PruneContainers clears all containers that are not running
func PruneContainers() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "unable to create docker client")
	}
	_, err = cli.ContainersPrune(context.Background(), filters.Args{})
	if err != nil {
		return errors.Wrap(err, "Prune container failed")
	}
	return nil
}

// ListContainer lists all the containers running on host machine
func ListContainer() error {
	cli, err := client.NewClientWithOpts(client.WithVersion(viper.GetString("docker_api_version")))
	if err != nil {
		return errors.Wrap(err, "unable to get new docker client")
	}
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return errors.Wrap(err, "unable to list containers: %v")
	}
	if len(containers) > 0 {
		for _, container := range containers {
			//TODO handle returned containers
			log.Printf("container ID: %s", container.ID)
		}
	} else {
		log.Println("there are no containers running")
	}
	return nil
}
