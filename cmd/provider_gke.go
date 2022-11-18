/*
    Copyright (C) 2020  2ALCHEMISTS SAS.

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as
    published by the Free Software Foundation, either version 3 of the
    License, or (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package cmd

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

func updateGKEClusters() {
	ctx := context.Background()
	clusterManagerClient, err := gcontainerv1.NewClusterManagerClient(ctx)
	if err != nil {
		log.WithError(err).Errorln("failed to instanciate GKE cluster manager")
		return
	}

	projectID, err := getGCPProjectID()
	if projectID <= int64(0) {
		log.WithError(err).Errorln("unable to retrieve GCP project ID")
		return
	}
	listReq := &gcontainerpbv1.ListClustersRequest{
		Parent: fmt.Sprintf("projects/%v/locations/-", projectID),
	}
	listResp, err := clusterManagerClient.ListClusters(ctx, listReq)
	if err != nil {
		log.WithError(err).Errorln("failed to list GKE clusters")
		return
	}

	for _, cluster := range listResp.Clusters {
		cmd := exec.Command(viper.GetString("krossboard_gcloud_command"),
			"container",
			"clusters",
			"get-credentials",
			cluster.Name,
			"--zone",
			cluster.Location)

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.WithField("cluster", cluster.Name).Errorln("failed getting GKE cluster credentials:", string(out))
		} else {
			log.WithField("cluster", cluster.Name).Debugln("added/updated GKE cluster credentials")
		}
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
	if err != nil {
		return -1, errors.Wrap(err, "failed calling GCP metadata service")
	}
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