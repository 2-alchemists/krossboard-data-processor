package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
	"github.com/ziutek/rrd"
)

// ClusterUsage holds used and non-allocatable memory and CPU resource of a cluster
type ClusterUsage struct {
	ClusterName       string  `json:"clusterName,omitempty"`
	CPUUsed           float64 `json:"cpuUsed,omitempty"`
	MemUsed           float64 `json:"memUsed,omitempty"`
	CPUNonAllocatable float64 `json:"cpuNonAllocatable,omitempty"`
	MemNonAllocatable float64 `json:"memNonAllocatable,omitempty"`
}

func getAllClustersCurrentUsage() ([]*ClusterUsage, error) {
	baseDataDir := viper.GetString("koamc_root_data_dir")
	entries, err := ioutil.ReadDir(baseDataDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed listing data dir")
	}

	var allUsage []*ClusterUsage
	for _, entry := range entries {
		if entry.IsDir() {
			usage, err := getClusterCurrentUsage(baseDataDir, entry.Name())
			if err != nil {
				log.WithError(err).Warnln("error getting current cluster usage for entry:", entry.Name())
			} else {
				b, _ := json.Marshal(usage)
				log.Infoln(string(b))
				allUsage = append(allUsage, usage)
			}
		}
	}

	return allUsage, nil
}

func getClusterCurrentUsage(baseDataDir string, clusterName string) (*ClusterUsage, error) {
	const RRDStepFiveMins = 300
	rrdEndEpoch := int64(int64(time.Now().Unix()/RRDStepFiveMins) * RRDStepFiveMins)
	rrdEnd := time.Unix(rrdEndEpoch, 0)
	rrdStart := rrdEnd.Add(-1 * RRDStepFiveMins * time.Second)

	rrdDir := fmt.Sprintf("%s/%s", baseDataDir, clusterName)
	dbfiles, err := ioutil.ReadDir(rrdDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading data folder")
	}

	usage := &ClusterUsage{
		ClusterName:       clusterName,
		CPUUsed:           0.0,
		MemUsed:           0.0,
		CPUNonAllocatable: 0.0,
		MemNonAllocatable: 0.0,
	}
	for _, dbfile := range dbfiles {
		if dbfile.IsDir() {
			continue
		}
		rrdfile := fmt.Sprintf("%s/%s", rrdDir, dbfile.Name())
		fetchRes, err := rrd.Fetch(rrdfile, "AVERAGE", rrdStart, rrdEnd, RRDStepFiveMins*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "unable to fetch data from rrd database")
		}
		defer fetchRes.FreeValues()

		rrdRow := 0
		for ti := fetchRes.Start.Add(fetchRes.Step); ti.Before(rrdEnd) || ti.Equal(rrdEnd); ti = ti.Add(fetchRes.Step) {
			if dbfile.Name() == "non-allocatable" {
				usage.CPUNonAllocatable = fetchRes.ValueAt(0, rrdRow)
				usage.MemNonAllocatable = fetchRes.ValueAt(1, rrdRow)
			} else {
				usage.CPUUsed += fetchRes.ValueAt(0, rrdRow)
				usage.MemUsed += fetchRes.ValueAt(1, rrdRow)
			}
			rrdRow++
		}
	}
	return usage, nil
}
