package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
	"github.com/ziutek/rrd"
)

// K8sClusterUsage holds used and non-allocatable memory and CPU resource of a K8s cluster
type K8sClusterUsage struct {
	ClusterName       string  `json:"clusterName"`
	CPUUsed           float64 `json:"cpuUsed"`
	MemUsed           float64 `json:"memUsed"`
	CPUNonAllocatable float64 `json:"cpuNonAllocatable"`
	MemNonAllocatable float64 `json:"memNonAllocatable"`
	OutToDate         bool    `json:"outToDate"`
}

func getAllClustersCurrentUsage(kubeconfig *KubeConfig) ([]*K8sClusterUsage, error) {
	clusters, err := kubeconfig.ListClusters()
	if err != nil {
		return nil, errors.Wrap(err, "failed reading clusters")
	}

	var allUsage []*K8sClusterUsage
	baseDataDir := viper.GetString("krossboard_root_data_dir")
	for clusterName, _ := range clusters {
		usage, err := getClusterCurrentUsage(baseDataDir, clusterName)
		if err != nil {
			log.WithError(err).Warnln("error getting current cluster usage for entry:", clusterName)
		} else {
			allUsage = append(allUsage, usage)
		}
	}
	return allUsage, nil
}

func getClusterCurrentUsage(baseDataDir string, clusterName string) (*K8sClusterUsage, error) {
	const (
		RRDLastUsageFetchWindow = -2 * RRDStorageStep300Secs
	)
	rrdEndEpoch := int64(int64(time.Now().Unix()/RRDStorageStep300Secs) * RRDStorageStep300Secs)
	rrdEnd := time.Unix(rrdEndEpoch, 0)
	rrdStart := rrdEnd.Add(RRDLastUsageFetchWindow * time.Second)
	rrdDir := fmt.Sprintf("%s/%s", baseDataDir, clusterName)
	dbfiles, err := ioutil.ReadDir(rrdDir)
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
	for _, dbfile := range dbfiles {
		if dbfile.IsDir() {
			continue
		}
		rrdfile := fmt.Sprintf("%s/%s", rrdDir, dbfile.Name())
		rrdFileInfo, err := rrd.Info(rrdfile)
		if err != nil {
			log.WithError(err).Warnln("seems to be not valid rrd file", rrdfile)
			continue
		}

		if rrdStart.Sub(time.Unix(int64(rrdFileInfo["last_update"].(uint)), 0)) > time.Duration(0) {
			log.Debugln("not recently updated rrd file", rrdfile)
			continue
		}

		fetchRes, err := rrd.Fetch(rrdfile, "LAST", rrdStart, rrdEnd, RRDStorageStep300Secs*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "unable to retrieve data from rrd file")
		}
		defer fetchRes.FreeValues()

		usage.OutToDate = false
		rrdRow := 0
		for ti := fetchRes.Start.Add(fetchRes.Step); ti.Before(rrdEnd) || ti.Equal(rrdEnd); ti = ti.Add(fetchRes.Step) {
			cpu := fetchRes.ValueAt(0, rrdRow)
			mem := fetchRes.ValueAt(1, rrdRow)
			if !math.IsNaN(cpu) && !math.IsNaN(mem) {
				if dbfile.Name() == "non-allocatable" {
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

	if usage.MemNonAllocatable <= 0 || usage.CPUNonAllocatable <= 0 {
		usage.OutToDate = true
	}
	return usage, nil
}

func processConsolidatedUsage(kubeconfig *KubeConfig) {
	workers.Add(1)
	defer workers.Done()

	for {
		allClustersUsage, err := getAllClustersCurrentUsage(kubeconfig)
		if err != nil {
			log.WithError(err).Errorln("get processing all clusters usage")
		} else {
			currentUsageFile := viper.GetString("krossboard_current_usage_file")
			serializedData, _ := json.Marshal(allClustersUsage)
			err = ioutil.WriteFile(currentUsageFile, serializedData, 0644)
			if err != nil {
				log.WithError(err).Errorln("failed writing current usage file")
				continue
			}
		}

		for _, clusterUsage := range allClustersUsage {
			if clusterUsage.OutToDate {
				continue
			}
			rrdFile := fmt.Sprintf("%s/.usagehistory_%s", viper.GetString("krossboard_root_data_dir"), clusterUsage.ClusterName)
			usageDb := NewUsageDb(rrdFile)
			_, err := os.Stat(rrdFile)
			if os.IsNotExist(err) {
				err := usageDb.CreateRRD()
				if err != nil {
					log.WithError(err).Errorln("failed creating RRD file", rrdFile)
					continue
				}
				time.Sleep(time.Second) // otherwise update will fail with 'illegal attempt to update' error
			}
			err = usageDb.UpdateRRD(time.Now(),
				clusterUsage.CPUUsed+clusterUsage.CPUNonAllocatable,
				clusterUsage.MemUsed+clusterUsage.MemNonAllocatable)
			if err != nil {
				log.WithError(err).Errorln("failed to udpate RRD file", rrdFile)
			}
		}
		time.Sleep(RRDStorageStep300Secs * time.Second)
	}
}
