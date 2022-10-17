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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
	"github.com/ziutek/rrd"
)

func getAllClustersCurrentUsage(clusterNames []string) ([]*K8sClusterUsage, error) {
	if len(clusterNames) == 0 {
		return nil, errors.New("no cluster provided")
	}
	var allUsage []*K8sClusterUsage
	for _, clusterName := range clusterNames {
		usage, err := getClusterCurrentUsage(clusterName)
		if err != nil {
			log.WithError(err).Warnln("error getting current cluster usage for entry:", clusterName)
		} else {
			allUsage = append(allUsage, usage)
		}
	}
	return allUsage, nil
}

func getClusterCurrentUsage(clusterName string) (*K8sClusterUsage, error) {
	const (
		RRDLastUsageFetchWindow = -2 * RRDStorageStep300Secs
	)
	baseDataDir := viper.GetString("krossboard_rawdb_dir")
	rrdDir := fmt.Sprintf("%s/%s", baseDataDir, clusterName)
	rrdEndEpoch := int64(int64(time.Now().Unix()/RRDStorageStep300Secs) * RRDStorageStep300Secs)
	rrdEnd := time.Unix(rrdEndEpoch, 0)
	rrdStart := rrdEnd.Add(RRDLastUsageFetchWindow * time.Second)

	foundFiles, err := ioutil.ReadDir(rrdDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading data folder")
	}

	usage := &K8sClusterUsage{
		ClusterName:       clusterName,
		CPUUsed:           0.0,
		MemUsed:           0.0,
		CPUNonAllocatable: 0.0,
		MemNonAllocatable: 0.0,
		OutToDate:         true,
	}
	for _, curFile := range foundFiles {
		if curFile.IsDir() {
			continue
		}
		rrdFile := fmt.Sprintf("%s/%s", rrdDir, curFile.Name())
		rrdFileInfo, err := rrd.Info(rrdFile)
		if err != nil {
			log.WithError(err).Warnln("seems to be not valid rrd file", rrdFile)
			continue
		}

		if rrdStart.Sub(time.Unix(int64(rrdFileInfo["last_update"].(uint)), 0)) > time.Duration(0) {
			log.Debugln("not recently updated rrd file", rrdFile)
			continue
		}

		fetchRes, err := rrd.Fetch(rrdFile, "LAST", rrdStart, rrdEnd, RRDStorageStep300Secs*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "unable to retrieve data from rrd file")
		}
		defer fetchRes.FreeValues()

		usage.OutToDate = true
		rrdRow := 0
		for ti := fetchRes.Start.Add(fetchRes.Step); ti.Before(rrdEnd) || ti.Equal(rrdEnd); ti = ti.Add(fetchRes.Step) {
			cpu := fetchRes.ValueAt(0, rrdRow)
			mem := fetchRes.ValueAt(1, rrdRow)
			if !math.IsNaN(cpu) && !math.IsNaN(mem) && cpu >= 0 && mem >= 0 {
				usage.OutToDate = false
				if curFile.Name() == "non-allocatable" {
					usage.CPUNonAllocatable = cpu
					usage.MemNonAllocatable = mem
				} else {
					usage.CPUUsed += cpu
					usage.MemUsed += mem
				}
			}
			rrdRow++
		}
	}

	return usage, nil
}

// getRecentNodesUsage returns nodes usage for a given cluster
func getRecentNodesUsage(clusterName string) (map[string]NodeUsage, error) {
	url := "http://127.0.0.1:1519/api/dataset/nodes.json"
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("http.NewRequest failed on URL %s", url))
	}

	httpReq.Header.Set("X-Krossboard-Cluster", clusterName)

	httpClient := http.Client{
		Timeout: time.Second * 5,
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("httpClient.Do failed on URL %s", url))
	}
	defer resp.Body.Close()

	respRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("ioutil.ReadAll failed on URL %s", url))
	}

	nodesUsage := &map[string]NodeUsage{}
	err = json.Unmarshal(respRaw, nodesUsage)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("ioutil.ReadAll failed on URL %s", url))
	}

	consolidatedUsage := make(map[string]NodeUsage)
	for nodeName, nodeUsage := range *nodesUsage {
		nodeUsage.CPUUsageByPods = 0.0
		nodeUsage.MEMUsageByPods = 0.0
		for _, podUsage := range nodeUsage.PodsUsage {
			nodeUsage.CPUUsageByPods += podUsage.CPUUsage
			nodeUsage.MEMUsageByPods += podUsage.MEMUsage
		}
		nodeUsage.PodsUsage = nil
		consolidatedUsage[nodeName] = nodeUsage
	}
	return consolidatedUsage, nil
}

func processConsolidatedUsage() {

	err := createDirIfNotExists(viper.GetString("krossboard_run_dir"))
	if err != nil {
		log.WithField("message", err.Error()).Fatalln("failed initializing status directory")
	}

	var clusterNames []string
	clusterNamesFromConfigVar := viper.GetString("krossboard_selected_cluster_names")
	if clusterNamesFromConfigVar != "" {
		clusterNames = strings.Split(clusterNamesFromConfigVar, " ")
	} else {
		kubeconfig := NewKubeConfig()
		managedClusters := kubeconfig.ListClusters()
		clusterNames = make([]string, len(managedClusters))
		for cname := range managedClusters {
			clusterNames = append(clusterNames, cname)
		}
	}

	allClustersUsage, err := getAllClustersCurrentUsage(clusterNames)
	if err != nil {
		log.WithError(err).Errorln("failed getting all clusters usage")
	} else {
		currentUsageFile := getCurrentClusterUsagePath()
		serializedData, _ := json.Marshal(allClustersUsage)
		err = ioutil.WriteFile(currentUsageFile, serializedData, 0644)
		if err != nil {
			log.WithError(err).Errorln("failed writing current usage file")
			return
		}
	}

	sampleTimeUTC := time.Now().UTC()
	for _, clusterUsage := range allClustersUsage {
		if !clusterUsage.OutToDate {
			processClusterNamespaceUsage(clusterUsage)
		}
		processClusterNodesUsage(clusterUsage, sampleTimeUTC)
	}
}

func processClusterNamespaceUsage(clusterUsage *K8sClusterUsage) {
	rrdFile := getHistoryDbPath(clusterUsage.ClusterName)
	usageDb := NewUsageDb(rrdFile, 100)
	_, err := os.Stat(rrdFile)
	if os.IsNotExist(err) {
		err := usageDb.CreateRRD()
		if err != nil {
			log.WithError(err).Errorln("failed creating RRD file", rrdFile)
			return
		}
		time.Sleep(time.Second) // otherwise update will fail with 'illegal attempt to update' error
	}
	cpuUsage := clusterUsage.CPUUsed + clusterUsage.CPUNonAllocatable
	memUsage := clusterUsage.MemUsed + clusterUsage.MemNonAllocatable
	err = usageDb.UpdateRRD(time.Now(), cpuUsage, memUsage)
	if err != nil {
		log.WithError(err).Errorln("failed to update RRD file", rrdFile)
	}

}

func processClusterNodesUsage(clusterUsage *K8sClusterUsage, sampleTimeUTC time.Time) {
	recentNodesUsage, err := getRecentNodesUsage(clusterUsage.ClusterName)
	if err != nil {
		log.WithError(err).Errorln("failed getting cluster nodes usage")
		return
	}
	for nodeName, nodeUsage := range recentNodesUsage {
		nodeUsageDb := NewNodeUsageDB(nodeName)

		err = nodeUsageDb.CapacityDb.UpdateRRD(sampleTimeUTC, nodeUsage.CPUCapacity, nodeUsage.MEMCapacity)
		if err != nil {
			log.WithError(err).Errorln("failed saving node capacity", nodeName)
		}
		err = nodeUsageDb.AllocatableDb.UpdateRRD(sampleTimeUTC, nodeUsage.CPUAllocatable, nodeUsage.MEMAllocatable)
		if err != nil {
			log.WithError(err).Errorln("failed saving allocatable capacity", nodeName)
		}
		err = nodeUsageDb.UsageByPodsDb.UpdateRRD(sampleTimeUTC, nodeUsage.CPUUsageByPods, nodeUsage.MEMUsageByPods)
		if err != nil {
			log.WithError(err).Errorln("failed saving capacity used by pods", nodeName)
		}
	}
}
