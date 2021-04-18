package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/patrickmn/go-cache"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
	"github.com/ziutek/rrd"
)

func getAllClustersCurrentUsage(kconfig *KubeConfig) ([]*K8sClusterUsage, error) {
	clusters := kconfig.ListClusters()
	if len(clusters) == 0 {
		return nil, errors.New("no cluster found")
	}

	var allUsage []*K8sClusterUsage
	baseDataDir := viper.GetString("krossboard_root_data_dir")
	for clusterName := range clusters {
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


// getClusterNodesUsage returns nodes usage for a given cluster
func getClusterNodesUsage(clusterName string) (map[string]NodeUsage, error){
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
	for nname, nodeUsage := range *nodesUsage {
		nodeUsage.CPUUsageByPods =  0.0
		nodeUsage.MEMUsageByPods =  0.0
		for _, podUsage := range nodeUsage.PodsUsage {
			nodeUsage.CPUUsageByPods += podUsage.CPUUsage
			nodeUsage.MEMUsageByPods += podUsage.MEMUsage
		}
		nodeUsage.PodsUsage = nil
		consolidatedUsage[nname] = nodeUsage
	}
	return consolidatedUsage, nil
}


func processConsolidatedUsage(kubeconfig *KubeConfig) {
	allClustersUsage, err := getAllClustersCurrentUsage(kubeconfig)
	if err != nil {
		log.WithError(err).Errorln("failed getting all clusters usage")
	} else {
		currentUsageFile := viper.GetString("krossboard_current_usage_file")
		serializedData, _ := json.Marshal(allClustersUsage)
		err = ioutil.WriteFile(currentUsageFile, serializedData, 0644)
		if err != nil {
			log.WithError(err).Errorln("failed writing current usage file")
			return
		}
	}

	sampleTimeUTC := time.Now().Format(queryTimeLayout)
	for _, clusterUsage := range allClustersUsage {
		if ! clusterUsage.OutToDate {
			processClusterNamespaceUsage(clusterUsage)
		}
		processClusterNodeUsage(clusterUsage, sampleTimeUTC)

	}
}

func processClusterNamespaceUsage(clusterUsage *K8sClusterUsage) {
	rrdFile := getUsageHistoryPath(clusterUsage.ClusterName)
	usageDb := NewUsageDb(rrdFile)
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



func processClusterNodeUsage(clusterUsage *K8sClusterUsage, sampleTimeUTC string) {
	nodeUsageDbPath := getNodeUsagePath(clusterUsage.ClusterName)
	nodeUsageDb, err := NewNodeUsageDB(nodeUsageDbPath, true)
	if err != nil {
		log.WithError(err).Errorln("NewNodeUsageDB failed")
		return
	}

	err = nodeUsageDb.Load()
	if err != nil {
		log.WithError(err).Errorln("Failed loading nodes usage file", nodeUsageDb.Path)
		return
	}

	nodeUsage, err := getClusterNodesUsage(clusterUsage.ClusterName)
	if err != nil {
		log.WithError(err).Errorln("getClusterNodesUsage failed")
		return
	}

	nodeUsageDb.Data.Set(fmt.Sprint(sampleTimeUTC), nodeUsage, cache.DefaultExpiration)
	err = nodeUsageDb.Save()
	if err != nil {
		log.WithError(err).Errorln("Failed saving node usage cache")
		return
	}
}