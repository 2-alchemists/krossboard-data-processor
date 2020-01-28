package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"bitbucket.org/koamc/kube-opex-analytics-mc/systemstatus"
)

// KOAAPI describes an instance Kubernetes Opex Analytics's API
type KOAAPI struct {
	ClusterName string `json:"clusterName,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
}

// DiscoveryResp holds the message returned by the discovery API
type DiscoveryResp struct {
	Status    string    `json:"status,omitempty"`
	Message   string    `json:"message,omitempty"`
	Instances []*KOAAPI `json:"instances,omitempty"`
}

// GetAllClustersCurrentUsageResp holds the message return edby the GetAllClustersCurrentUsageHandler API callback
type GetAllClustersCurrentUsageResp struct {
	Status       string             `json:"status,omitempty"`
	Message      string             `json:"message,omitempty"`
	ClusterUsage []*K8sClusterUsage `json:"clusterUsage,omitempty"`
}

// GetClusterUsageHistoryResp holds the message returned by the GetClusterUsageHistoryHandler API callback
type GetClusterUsageHistoryResp struct {
	Status             string                   `json:"status,omitempty"`
	Message            string                   `json:"message,omitempty"`
	ListOfUsageHistory map[string]*UsageHistory `json:"usageHistory,omitempty"`
}

func startAPI() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish")
	flag.Parse()

	router := mux.NewRouter()
	router.HandleFunc("/discovery", DiscoveryHandler)
	router.HandleFunc("/currentusage", GetAllClustersCurrentUsageHandler)
	router.HandleFunc("/usagehistory", GetClustersUsageHistoryHandler)

	srv := &http.Server{
		Addr:         viper.GetString("koamc_api_addr"),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handlers.CORS()(router),
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Infoln(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	srv.Shutdown(ctx)
	log.Infoln("shutting down")
	os.Exit(0)
}

// DiscoveryHandler returns current system status along with Kubernetes Opex Analytics instances' endpoints
func DiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	discoveryResp := &DiscoveryResp{}
	systemStatus, err := systemstatus.LoadSystemStatus(viper.GetString("koamc_status_file"))
	if err != nil {
		discoveryResp.Status = "error"
		discoveryResp.Message = "cannot load system status"
		log.WithField("message", err.Error()).Fatalln("Cannot load system status")
	}

	runningConfig, err := systemStatus.GetInstances()
	if err != nil {
		discoveryResp.Status = "error"
		discoveryResp.Message = "cannot load running configuration"
		log.WithField("message", err.Error()).Fatalln("Cannot load system status")
	} else {
		discoveryResp.Status = "ok"
		for _, instance := range runningConfig.Instances {
			discoveryResp.Instances = append(discoveryResp.Instances, &KOAAPI{
				ClusterName: instance.ClusterName,
				Endpoint:    fmt.Sprintf("http://127.0.0.1:%v", instance.HostPort),
			})
		}
	}

	w.WriteHeader(http.StatusOK)
	outRaw, _ := json.Marshal(discoveryResp)
	w.Write(outRaw)
}

// GetAllClustersCurrentUsageHandler returns current usage of all clusters
func GetAllClustersCurrentUsageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	currentUsageResp := &GetAllClustersCurrentUsageResp{}
	currentUsageFile := viper.GetString("koamc_current_usage_file")
	currentUsageData, err := ioutil.ReadFile(currentUsageFile)
	respHTTPStatus := http.StatusInternalServerError
	if err != nil {
		log.WithError(err).Errorln("failed reading current status file")
		currentUsageResp.Status = "error"
		currentUsageResp.Message = "failed reading current status file"
	} else {
		var currentUsage []*K8sClusterUsage
		err := json.Unmarshal(currentUsageData, &currentUsage)
		if err != nil {
			log.WithError(err).Errorln("failed decoding current usage data")
			currentUsageResp.Status = "error"
			currentUsageResp.Message = "invalid current usage data"
		} else {
			respHTTPStatus = http.StatusOK
			currentUsageResp.Status = "ok"
			currentUsageResp.ClusterUsage = currentUsage
		}
	}

	w.WriteHeader(respHTTPStatus)
	outRaw, _ := json.Marshal(currentUsageResp)
	w.Write(outRaw)
}

// GetClustersUsageHistoryHandler returns all clusters usage history
func GetClustersUsageHistoryHandler(outHandler http.ResponseWriter, inReq *http.Request) {
	outHandler.Header().Set("Content-Type", "application/json")

	systemStatus, err := systemstatus.LoadSystemStatus(viper.GetString("koamc_status_file"))
	if err != nil {
		log.WithError(err).Errorln("cannot load system status")
		outHandler.WriteHeader(http.StatusInternalServerError)
		outRaw, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "failed loading system status",
		})
		outHandler.Write(outRaw)
		return
	}

	getInstancesResult, err := systemStatus.GetInstances()
	if err != nil {
		log.WithError(err).Errorln("failed retrieving managed instances")
		outHandler.WriteHeader(http.StatusInternalServerError)
		outRaw, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "failed retrieving managed instances",
		})
		outHandler.Write(outRaw)
		return
	}

	queryParams := inReq.URL.Query()
	queryCluster := queryParams.Get("cluster")
	queryStartDate := queryParams.Get("startDateUTC")
	queryEndDate := queryParams.Get("endDateUTC")

	invalidParam := false

	const queryTimeLayout = "2006-01-02T15:04:05"
	actualEndDateUTC := time.Now().UTC()
	if queryEndDate != "" {
		queryParsedEndTime, err := time.Parse(queryTimeLayout, queryEndDate)
		if err != nil {
			invalidParam = true
			log.WithError(err).Errorln("failed parsing query end date", queryEndDate)
		} else {
			actualEndDateUTC = queryParsedEndTime
		}
	}

	const durationMinus24Hours = -1 * 24 * time.Hour
	actualStartDateUTC := actualEndDateUTC.Add(durationMinus24Hours)
	if queryStartDate != "" {
		queryParsedStartTime, err := time.Parse(queryTimeLayout, queryStartDate)
		if err != nil {
			invalidParam = true
			log.WithError(err).Errorln("failed parsing query end date ", queryStartDate)
		} else {
			actualStartDateUTC = queryParsedStartTime
		}
	}

	// check cluster parameters
	historyDbs := make(map[string]string)
	if queryCluster == "" || strings.ToLower(queryCluster) == "all" {
		for _, instance := range getInstancesResult.Instances {
			historyDbs[instance.ClusterName] = fmt.Sprintf("%s/.usagehistory_%s", viper.GetString("koamc_root_data_dir"), instance.ClusterName)
		}
	} else {
		dbdir := fmt.Sprintf("%s/%s", viper.GetString("koamc_root_data_dir"), queryCluster)
		err, dbfiles := listRegularDbFiles(dbdir)
		if err != nil {
			log.WithError(err).Errorln("failed listing dbs for cluster", queryCluster)
			invalidParam = true
		} else {
			for _, dbfile := range dbfiles {
				historyDbs[dbfile] = fmt.Sprintf("%s/%s", dbdir, dbfile)
			}
		}
	}

	// validate parameters
	if invalidParam || actualStartDateUTC.After(actualEndDateUTC) {
		log.Errorln("invalid query parameters", queryCluster, queryStartDate, queryEndDate)
		outHandler.WriteHeader(http.StatusBadRequest)
		outRaw, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "invalid query parameters",
		})
		outHandler.Write(outRaw)
		return
	}

	resultUsageHistory := &GetClusterUsageHistoryResp{
		Status:             "ok",
		ListOfUsageHistory: make(map[string]*UsageHistory, len(getInstancesResult.Instances)),
	}
	for dbname, dbfile := range historyDbs {
		usageDb := NewUsageDb(dbfile)
		usageHistory, err := usageDb.FetchUsage(actualStartDateUTC, actualEndDateUTC)
		if err != err {
			log.WithError(err).Errorln("failed retrieving data from rrd file")
		} else {
			if usageHistory != nil {
				resultUsageHistory.ListOfUsageHistory[dbname] = usageHistory
			}
		}
	}

	outHandler.WriteHeader(http.StatusOK)
	outData, _ := json.Marshal(resultUsageHistory)
	outHandler.Write(outData)
}

func listRegularDbFiles(folder string) (error, []string) {
	var files []string
	err := filepath.Walk(folder, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
			files = append(files, info.Name())
		}
		return nil
	})
	return err, files
}
