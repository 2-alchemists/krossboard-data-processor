package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	gcontainerv1 "cloud.google.com/go/container/apiv1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	gcontainerpbv1 "google.golang.org/genproto/googleapis/container/v1"
)

func updateGKEClusters(updateIntervalMin time.Duration) {
	workers.Add(1)
	defer workers.Done()

	ctx := context.Background()
	clusterManagerClient, err := gcontainerv1.NewClusterManagerClient(ctx)
	if err != nil {
		log.WithError(err).Errorln("failed to instanciate GKE cluster manager")
		return
	}

	for {
		projectID, err := getGCPProjectID()
		if projectID <= int64(0) {
			log.WithError(err).Errorln("unable to retrieve GCP project ID")
			time.Sleep(updateIntervalMin)
			continue
		}
		listReq := &gcontainerpbv1.ListClustersRequest{
			Parent: fmt.Sprintf("projects/%v/locations/-", projectID),
		}
		listResp, err := clusterManagerClient.ListClusters(ctx, listReq)
		if err != nil {
			log.WithError(err).Errorln("Failed to list GKE clusters")
			time.Sleep(updateIntervalMin)
			continue
		}

		for _, cluster := range listResp.Clusters {
			cmdout, err := exec.Command(viper.GetString("krossboard_gcloud_command"),
				"container",
				"clusters",
				"get-credentials",
				cluster.Name,
				"--zone",
				cluster.Zone).CombinedOutput()

			if err != nil {
				log.WithField("cluster", cluster.Name).Errorln("failed getting GKE cluster credentials:", string(cmdout))
			} else {
				log.WithField("cluster", cluster.Name).Debugln("added/updated GKE cluster credentials")
			}
		}
		time.Sleep(updateIntervalMin)
	}
}

func getGCPProjectID() (int64, error) {
	timeout := time.Duration(time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET",
		viper.GetString("krossboard_gcp_metadata_service")+"/computeMetadata/v1/project/numeric-project-id",
		nil)
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return -1, errors.Wrap(err, "failed calling GCP metadata service")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, errors.Wrap(err, "failed ready response from GCP metadata service")
	}

	if resp.StatusCode != http.StatusOK {
		return -1, errors.New("GCP metadata server returned error: " + string(body))
	}
	projectID, err := strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		return -1, errors.Wrap(err, "unexpected non integer for project id")
	}

	return projectID, nil
}
