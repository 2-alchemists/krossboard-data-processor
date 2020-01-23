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

// GetAllClustersUsageHistoryResp holds the message returned by the GetAllClustersUsageHistoryHandler API callback
type GetAllClustersUsageHistoryResp struct {
	Status               string                   `json:"status,omitempty"`
	Message              string                   `json:"message,omitempty"`
	ClustersUsageHistory map[string]*UsageHistory `json:"clustersUsageHistory,omitempty"`
}

func startAPI() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish")
	flag.Parse()

	router := mux.NewRouter()
	router.HandleFunc("/discovery", DiscoveryHandler)
	router.HandleFunc("/currentusage", GetAllClustersCurrentUsageHandler)
	router.HandleFunc("/usagehistory", GetAllClustersUsageHistoryHandler)

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

// GetAllClustersUsageHistoryHandler returns all clusters usage history
func GetAllClustersUsageHistoryHandler(outHandler http.ResponseWriter, inReq *http.Request) {
	outHandler.Header().Set("Content-Type", "application/json")

	systemStatus, err := systemstatus.LoadSystemStatus(viper.GetString("koamc_status_file"))
	if err != nil {
		log.WithError(err).Errorln("cannot load system status")
		outHandler.WriteHeader(http.StatusInternalServerError)
		outRaw, _ := json.Marshal(&GetAllClustersUsageHistoryResp{
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
		outRaw, _ := json.Marshal(&GetAllClustersUsageHistoryResp{
			Status:  "error",
			Message: "failed retrieving managed instances",
		})
		outHandler.Write(outRaw)
		return
	}

	queryParams := inReq.URL.Query()
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

	if invalidParam || actualStartDateUTC.After(actualEndDateUTC) {
		log.Errorln("invalid query parameters", queryStartDate, queryEndDate)
		outHandler.WriteHeader(http.StatusBadRequest)
		outRaw, _ := json.Marshal(&GetAllClustersUsageHistoryResp{
			Status:  "error",
			Message: "invalid query parameters",
		})
		outHandler.Write(outRaw)
		return
	}

	allClustersUsageHistory := &GetAllClustersUsageHistoryResp{
		Status:               "ok",
		ClustersUsageHistory: make(map[string]*UsageHistory, len(getInstancesResult.Instances)),
	}

	for _, instance := range getInstancesResult.Instances {
		rrdUsageHistory := fmt.Sprintf("%s/.usagehistory_%s", viper.GetString("koamc_root_data_dir"), instance.ClusterName)
		usageDb := NewUsageDb(rrdUsageHistory)
		usageHistory, err := usageDb.FetchUsage(actualStartDateUTC, actualEndDateUTC)
		if err != err {
			log.WithError(err).Errorln("failed retrieving data from rrd file")
		} else {
			if usageHistory != nil {
				allClustersUsageHistory.ClustersUsageHistory[instance.ClusterName] = usageHistory
			}
		}
	}

	outHandler.WriteHeader(http.StatusOK)
	outData, _ := json.Marshal(allClustersUsageHistory)
	outHandler.Write(outData)
}
