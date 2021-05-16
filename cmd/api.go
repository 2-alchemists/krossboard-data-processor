package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	kclient "k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// KOAAPI describes an instance Kubernetes Opex Analytics's API
type KOAAPI struct {
	ClusterName string `json:"clusterName,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
}

// ErrorResp holds an error response
type ErrorResp struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
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

var routes = map[string]map[string]interface{}{
	"/api/dataset/{filename}": {
		"method": "GET",
		"handler": GetDatasetHandler,
	},
	"/api/discovery": {
		"method": "GET",
		"handler": DiscoveryHandler,
	},
	"/api/currentusage": {
		"method": "GET",
		"handler" :GetAllClustersCurrentUsageHandler,
	},
	"/api/usagehistory": {
		"method": "GET",
		"handler": GetClustersUsageHistoryHandler,
	},
	"/api/nodesusage/{clustername}": {
		"method":  "GET",
		"handler": GetNodesUsageHandler,
	},
	"/api/kubeconfig": {
		"method": "POST",
		"handler": KubeConfigHandler,
	},
}


func startAPI() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish")
	flag.Parse()

	router := mux.NewRouter()
	for r, h := range routes {
		router.HandleFunc(r, h["handler"].(func(http.ResponseWriter, *http.Request))).Methods(h["method"].(string), "OPTIONS")
	}

	appCors := cors.New(cors.Options{
		AllowedOrigins:   []string{viper.GetString("krossboard_cors_origins")},
		AllowedHeaders:   []string{"Authorization", "X-Krossboard-Cluster"},
		AllowCredentials: true,
	})
	srv := &http.Server{
		Addr:         viper.GetString("krossboard_api_addr"),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      appCors.Handler(router),
	}

	// Run the server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Errorln(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	srv.Shutdown(ctx) //nolint:errcheck
	log.Infoln("shutting down")
	os.Exit(0)
}

// GetDatasetHandler provides reverse proxy to download dataset from KOA instances
func GetDatasetHandler(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	datafile := params["filename"]

	if datafile == "nodes.json" {
		if _, err := validateLicenseFromEnvConfig(KrossboardVersion); err != nil {
			log.WithError(err).Errorln("invalid license getting nodes occupation")
			w.WriteHeader(http.StatusPaymentRequired)
			apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
				Status:  "error",
				Message: "nodes usage analytics is not supported by your current license",
			})
			_, _ = w.Write(apiResp)
			return
		}
	}

	clusterName := req.Header.Get("X-Krossboard-Cluster")

	systemStatus, err := LoadSystemStatus(viper.GetString("krossboard_status_file"))
	if err != nil {
		log.WithError(err).Errorln("cannot load system status")
		b, _ := json.Marshal(&ErrorResp{
			Status: "error",
			Message: "cannot load system status",
		})
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}

	instance, err := systemStatus.GetInstance(clusterName)
	if err != nil {
		log.WithError(err).Errorln("requested resource not found", clusterName)
		b, _ := json.Marshal(&ErrorResp{
			Status: "error",
			Message: "requested resource not found",
		})
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}
	apiUrl := fmt.Sprintf("http://127.0.0.1:%v/dataset/%v", instance.HostPort, datafile)

	if req.RequestURI == "/" {
		log.Errorln("no API context")
		b, _ := json.Marshal(&ErrorResp{
			Status: "error",
			Message: "no API context",
		})
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}

	proxyReq, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		log.WithError(err).Errorln("http.NewRequest failed on URL", apiUrl)
		b, _ := json.Marshal(&ErrorResp{
			Status: "error",
			Message: "failed calling target API",
		})
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}

	httpClient := http.Client{}
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		log.WithError(err).Errorln("httpClient.Do failed on URL", apiUrl)
		b, _ := json.Marshal(&ErrorResp{
			Status: "error",
			Message: "failed calling target API",
		})
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	apiResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Errorln("failed reading response body")
		b, _ := json.Marshal(&ErrorResp{Status: "error", Message: "failed reading response body"})
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}

	for hk, hvalues := range req.Header {
		for _, hval := range hvalues {
			w.Header().Add(hk, hval)
		}
	}
	_, _ = w.Write(apiResp)
}

// DiscoveryHandler returns current system status along with Kubernetes Opex Analytics instances' endpoints
func DiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	discoveryResp := &DiscoveryResp{}
	systemStatus, err := LoadSystemStatus(viper.GetString("krossboard_status_file"))
	if err != nil {
		discoveryResp.Status = "error"
		discoveryResp.Message = "cannot load system status"
		log.WithField("message", err.Error()).Errorln("Cannot load system status")
	}

	runningConfig, err := systemStatus.GetInstances()
	if err != nil {
		discoveryResp.Status = "error"
		discoveryResp.Message = "cannot load running configuration"
		log.WithField("message", err.Error()).Errorln("Cannot load system status")
	} else {
		isValidLicense := false
		licenseDoc, licenseCheckErr := validateLicenseFromEnvConfig(KrossboardVersion)
		if licenseCheckErr == nil {
			isValidLicense = true
			discoveryResp.Message = fmt.Sprintf("Active Krossboard License: v%v (+bug/security changes)", licenseDoc.MajorVersion)
		}
		discoveryResp.Status = "ok"
		for instanceItemIndex, instance := range runningConfig.Instances {
			if ! isValidLicense && instanceItemIndex == 3 {
				discoveryResp.Message = fmt.Sprintf("Free License. Limited to 3 Kubernetes clusters (out of %v discovered)", len(runningConfig.Instances))
				break
			}
			discoveryResp.Instances = append(discoveryResp.Instances, &KOAAPI{
				ClusterName: instance.ClusterName,
				Endpoint:    fmt.Sprintf("http://127.0.0.1:%v", instance.HostPort),
			})
		}
	}

	w.WriteHeader(http.StatusOK)
	apiResp, _ := json.Marshal(discoveryResp)
	_, _ = w.Write(apiResp)
}

// GetAllClustersCurrentUsageHandler returns current usage of all clusters
func GetAllClustersCurrentUsageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	currentUsageResp := &GetAllClustersCurrentUsageResp{}
	currentUsageFile := viper.GetString("krossboard_current_usage_file")
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
	apiResp, _ := json.Marshal(currentUsageResp)
	_, _ = w.Write(apiResp)
}

// GetClustersUsageHistoryHandler returns all clusters usage history
func GetClustersUsageHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	systemStatus, err := LoadSystemStatus(viper.GetString("krossboard_status_file"))
	if err != nil {
		log.WithError(err).Errorln("cannot load system status")
		w.WriteHeader(http.StatusInternalServerError)
		apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "failed loading system status",
		})
		_, _ = w.Write(apiResp)
		return
	}

	getInstancesResult, err := systemStatus.GetInstances()
	if err != nil {
		log.WithError(err).Errorln("failed retrieving managed instances")
		w.WriteHeader(http.StatusInternalServerError)
		apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "failed retrieving managed instances",
		})
		_, _ = w.Write(apiResp)
		return
	}

	queryParams := r.URL.Query()
	queryCluster := queryParams.Get("cluster")
	queryStartDate := queryParams.Get("startDateUTC")
	queryEndDate := queryParams.Get("endDateUTC")
	queryFormat := strings.ToLower(queryParams.Get("format"))
	queryPeriod := strings.ToLower(queryParams.Get("period"))

	// process format
	if queryFormat != "" && queryFormat != "json" && queryFormat != "csv" {
		err := fmt.Errorf("invalid value '%s' for query parameter 'format'. Valid values are: 'json', 'csv'", queryFormat)
		log.WithError(err).WithField("param", "format").Warnln("Bad request")
		w.WriteHeader(http.StatusBadRequest)
		apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: err.Error(),
		})
		_, _ = w.Write(apiResp)
		return
	}

	// process period
	if queryPeriod != "" && queryPeriod != "hourly" && queryPeriod != "monthly" {
		err := fmt.Errorf("invalid value '%s' for query parameter 'period'. Valid values are: 'hourly', 'monthly'", queryPeriod)
		log.WithError(err).WithField("param", "period").Warnln("Bad request")
		apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: err.Error(),
		})
		_, _ = w.Write(apiResp)
		return
	}
	if queryPeriod == "" {
		queryPeriod = "hourly"
	}

	// process  end date parameter
	parametersAreInvalid := false
	now := time.Now().UTC()
	actualEndDateUTC := now
	if queryEndDate != "" {
		queryParsedEndTime, err := time.Parse(queryTimeLayout, queryEndDate)
		if err != nil {
			parametersAreInvalid = true
			log.WithError(err).Errorln("failed parsing query end date", queryEndDate)
		} else {
			actualEndDateUTC = queryParsedEndTime
		}
	}

	// process start date parameter
	const durationMinus24Hours = -1 * 24 * time.Hour
	actualStartDateUTC := actualEndDateUTC.Add(durationMinus24Hours)
	if queryStartDate != "" {
		queryParsedStartTime, err := time.Parse(queryTimeLayout, queryStartDate)
		if err != nil {
			parametersAreInvalid = true
			log.WithError(err).Errorln("failed parsing query end date ", queryStartDate)
		} else {
			actualStartDateUTC = queryParsedStartTime
		}
	}

	// check license and review the start and end date parameters
	// according to the license capability
	date90DaysBefore := now.Add(-1 * 90 * 24 * time.Hour)
	_, errLicense := validateLicenseFromEnvConfig(KrossboardVersion)
	if errLicense != nil &&
		(actualStartDateUTC.Before(date90DaysBefore) || actualEndDateUTC.Before(date90DaysBefore)) {
		log.Errorln("the selected time frame is out of 90 days (not supported by your current license)", actualStartDateUTC, actualEndDateUTC)
		w.WriteHeader(http.StatusPaymentRequired)
		apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "the selected time frame is out of 90 days (not supported by your current license)",
		})
		_, _ = w.Write(apiResp)
		return
	}

	// process cluster parameter
	historyDbs := make(map[string]string)
	if queryCluster == "" || strings.ToLower(queryCluster) == "all" {
		for _, instance := range getInstancesResult.Instances {
			historyDbs[instance.ClusterName] = getUsageHistoryPath(instance.ClusterName)
		}
	} else {
		dbdir := fmt.Sprintf("%s/%s", viper.GetString("krossboard_root_data_dir"), queryCluster)
		err, dbfiles := listRegularFiles(dbdir)
		if err != nil {
			log.WithError(err).Errorln("failed listing dbs for cluster", queryCluster)
			parametersAreInvalid = true
		} else {
			for _, dbfile := range dbfiles {
				historyDbs[dbfile] = dbfile
			}
		}
	}

	// finalize parameters validation before actually processing the request
	if parametersAreInvalid || actualStartDateUTC.After(actualEndDateUTC) {
		log.Errorln("invalid query parameters", queryCluster, queryStartDate, queryEndDate)
		w.WriteHeader(http.StatusBadRequest)
		apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "invalid query parameters",
		})
		_, _ = w.Write(apiResp)
		return
	}

	usageHistoryResult := &GetClusterUsageHistoryResp{
		Status:             "ok",
		ListOfUsageHistory: make(map[string]*UsageHistory, len(getInstancesResult.Instances)),
	}
	for dbname, dbfile := range historyDbs {
		usageDb := NewUsageDb(dbfile, 100)
		usageHistory, err := func() (*UsageHistory, error) {
			if queryPeriod == "monthly" {
				return usageDb.FetchUsageMonthly(actualStartDateUTC, actualEndDateUTC)
			} else {
				return usageDb.FetchUsageHourly(actualStartDateUTC, actualEndDateUTC)
			}
		}()
		if err != nil {
			log.WithError(err).Errorln("failed retrieving data from rrd file")
		} else {
			if usageHistory != nil {
				usageHistoryResult.ListOfUsageHistory[dbname] = usageHistory
			}
		}
	}

	var respPayload []byte
	if queryFormat != "csv" {
		respPayload, _ = json.Marshal(usageHistoryResult)
	} else {
		var csvBuf strings.Builder
		for itemName, itemUsage := range usageHistoryResult.ListOfUsageHistory {
			countUsageEntries := len(itemUsage.CPUUsage)
			if countUsageEntries != len(itemUsage.MEMUsage) {
				log.Errorf("usage entries for CPU and memory for entry %v don't match (%v != %v)\n",
					itemUsage, countUsageEntries, len(itemUsage.MEMUsage))
				continue
			}
			fmt.Fprintf(&csvBuf, "Name,Date UTC,CPU Usage,Memory usage\n")
			for i := 0; i < countUsageEntries; i++ {
				fmt.Fprintf(&csvBuf, "%v,%v,%v,%v\n",
					itemName,
					itemUsage.CPUUsage[i].DateUTC,
					itemUsage.CPUUsage[i].Value,
					itemUsage.MEMUsage[i].Value)
			}
		}
		respPayload = []byte(csvBuf.String())
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment; filename=\"usagehistory_%v_FROM_%v_TO_%v.csv\"",
				queryCluster,
				actualStartDateUTC.Format(queryTimeLayout),
				actualEndDateUTC.Format(queryTimeLayout),
			),
		)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respPayload)
}

// GetNodesUsageHandler returns the node usage for a cluster set in the "X-Krossboard-Cluster header
func GetNodesUsageHandler(w http.ResponseWriter, req *http.Request) {

	if _, err := validateLicenseFromEnvConfig(KrossboardVersion); err != nil {
		log.WithError(err).Errorln("invalid license getting nodes usage history")
		w.WriteHeader(http.StatusPaymentRequired)
		apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "nodes usage analytics is not supported by your current license",
		})
		_, _ = w.Write(apiResp)
		return
	}

	params := mux.Vars(req)
	clusterName := params["clustername"]
	queryParams := req.URL.Query()
	queryStartDate := queryParams.Get("startDateUTC")
	queryEndDate := queryParams.Get("endDateUTC")

	// process  end date parameter
	parametersAreInvalid := false
	actualEndDateUTC := time.Now().UTC()
	if queryEndDate != "" {
		queryParsedEndTime, err := time.Parse(queryTimeLayout, queryEndDate)
		if err != nil {
			parametersAreInvalid = true
			log.WithError(err).Errorln("failed parsing query end date", queryEndDate)
		} else {
			actualEndDateUTC = queryParsedEndTime
		}
	}

	// process start date parameter
	const durationMinus24Hours = -1 * 24 * time.Hour
	actualStartDateUTC := actualEndDateUTC.Add(durationMinus24Hours)
	if queryStartDate != "" {
		queryParsedStartTime, err := time.Parse(queryTimeLayout, queryStartDate)
		if err != nil {
			parametersAreInvalid = true
			log.WithError(err).Errorln("failed parsing query end date ", queryStartDate)
		} else {
			actualStartDateUTC = queryParsedStartTime
		}
	}

	if parametersAreInvalid || actualEndDateUTC.Before(actualStartDateUTC){
		log.Errorln("invalid query parameters", queryStartDate, queryEndDate)
		w.WriteHeader(http.StatusBadRequest)
		apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "invalid query parameters",
		})
		_, _ = w.Write(apiResp)
		return
	}

	recentNodesUsage, err := getRecentNodesUsage(clusterName)
	if err != nil {
		log.WithError(err).Errorln("failed getting recent cluster nodes")
		w.WriteHeader(http.StatusInternalServerError)
		apiResp, _ := json.Marshal(&GetClusterUsageHistoryResp{
			Status:  "error",
			Message: "failed getting recent cluster nodes",
		})
		_, _ = w.Write(apiResp)
		return
	}

	step := time.Duration(RRDStorageStep3600Secs)*time.Second
	if actualEndDateUTC.Sub(actualStartDateUTC) <= time.Duration(24)*time.Hour {
		step = time.Duration(RRDStorageStep300Secs)*time.Second
	}

	nodeUsageMap := make(map[string]map[string]UsageHistory)
	for nodeName := range recentNodesUsage {
		nodeUsageDb := NewNodeUsageDB(nodeName)
		capacityHistory, err := nodeUsageDb.CapacityDb.FetchUsage(actualStartDateUTC, actualEndDateUTC, step)
		if err != nil {
			capacityHistory = &UsageHistory{}
			log.WithError(err).Errorln("failed retrieving node capacity history", nodeUsageDb.CapacityDb.RRDFile)
		}
		allocatableHistory, err := nodeUsageDb.AllocatableDb.FetchUsage(actualStartDateUTC, actualEndDateUTC, step)
		if err != nil {
			allocatableHistory = &UsageHistory{}
			log.WithError(err).Errorln("failed retrieving node allocatable history", nodeUsageDb.CapacityDb.RRDFile)
		}
		usageByPodsHistory, err := nodeUsageDb.UsageByPodsDb.FetchUsage(actualStartDateUTC, actualEndDateUTC, step)
		if err != nil {
			usageByPodsHistory = &UsageHistory{}
			log.WithError(err).Errorln("failed retrieving usage by pods for node", nodeUsageDb.CapacityDb.RRDFile)
		}

		nodeUsageMap[nodeName] = map[string]UsageHistory {
			"capacityItems" : *capacityHistory,
			"allocatableItems" : *allocatableHistory,
			"usageByPodItems" : *usageByPodsHistory,
		}
	}

	var result []NodeUsage
	encodedResult, err := json.Marshal(nodeUsageMap)
	if err != nil {
		log.WithError(err).Errorln("Failed encoding result in JSON", result)
		b, _ := json.Marshal(&ErrorResp{Status: "error", Message: "server internal error"})
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encodedResult)
}

// KubeConfigHandler handles API calls to manage KUBECONFIG
func KubeConfigHandler(w http.ResponseWriter, req *http.Request) {
	maxUploadKb := viper.GetInt64("krossboard_kubeconfig_max_size_kb")
	err := req.ParseMultipartForm(maxUploadKb *  (1 << 10))
	if err != nil {
		log.WithError(err).Errorln("failed parsing multi-part form")
		b, _ := json.Marshal(&ErrorResp{Status: "error", Message: "failed parsing input"})
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}

	uploadedFile, uploadHandler, err := req.FormFile("kubeconfig")
	if err != nil {
		log.WithError(err).Errorln("error reading upload parameters")
		b, _ := json.Marshal(&ErrorResp{Status: "error", Message: "error reading file content"})
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}
	defer uploadedFile.Close()

	log.Infoln("File received:", uploadHandler.Filename)
	log.Infoln("  Size:", uploadHandler.Size)
	log.Infoln("  MIME Header:", uploadHandler.Header)

	uploadBytes, err := ioutil.ReadAll(uploadedFile)
	if err != nil {
		log.WithError(err).Errorln("Failed reading the uploaded file content")
		b, _ := json.Marshal(&ErrorResp{Status: "error", Message: "failed reading file content"})
		http.Error(w, string(b), http.StatusBadRequest)
		return
	}

	kconfigDir := viper.GetString("krossboard_kubeconfig_dir")
	err = createDirIfNotExists(kconfigDir)
	if err != nil {
		log.WithError(err).Errorln("Failed creating target directory")
		b, _ := json.Marshal(&ErrorResp{Status: "error", Message: "internal server error"})
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}

	destFilename := fmt.Sprintf("%s/kubeconfig-uploaded-%s", kconfigDir, time.Now().Format("20060102T150405"))
	err = ioutil.WriteFile(destFilename, uploadBytes, 0600)
	if err != nil {
		log.WithError(err).Errorln("Failed saving the uploaded file", destFilename)
		b, _ := json.Marshal(&ErrorResp{Status: "error", Message: "internal server error"})
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}

	_, err = kclient.LoadFromFile(destFilename)
	if err != nil {
		log.WithError(err).Errorln("failed parsing KUBECONFIG", destFilename)
		b, _ := json.Marshal(&ErrorResp{Status: "error", Message: "invalid KUBECONFIG content"})
		http.Error(w, string(b), http.StatusBadRequest)
		_ = os.Remove(destFilename)
		return
	}

	w.WriteHeader(http.StatusOK)
	b, _ := json.Marshal(&ErrorResp{Status: "success", Message: "upload completed successfully "+destFilename})
	http.Error(w, string(b), http.StatusBadRequest)
	_, _ = w.Write(b)
}