package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

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

// DiscoveryResp message returned by the discovery API
type DiscoveryResp struct {
	Status    string    `json:"status,omitempty"`
	Message   string    `json:"message,omitempty"`
	Instances []*KOAAPI `json:"instances,omitempty"`
}

func startAPI() {

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish")
	flag.Parse()

	router := mux.NewRouter()
	router.HandleFunc("/discovery", DiscoveryHandler)

	srv := &http.Server{
		Addr:         viper.GetString("koamc_api_addr"),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
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
