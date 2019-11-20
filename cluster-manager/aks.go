package main

import (
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func updateAKSClusters() {
	defer workers.Done()

	updatePeriod := time.Duration(viper.GetInt64("koacm_update_interval")) * time.Minute
	for {
		resourceGroups, err := listAzureResourceGroups()
		if err != nil {
			log.WithError(err).Errorln("failed list Azure resource groups")
			time.Sleep(updatePeriod)
			continue
		}
		for _, rg := range resourceGroups {
			cmdout, err := exec.Command(viper.GetString("koamc_az_command"),
				"aks",
				"list",
				"--resource-group",
				rg,
				"-o",
				"json").CombinedOutput()
			if err != nil {
				log.WithError(err).Errorln("failed listing AKS clusters for resource group" + rg + ": " + string(cmdout))
				continue
			}
			var clusterList []string
			_, err = jsonparser.ArrayEach(cmdout, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				cn, _ := jsonparser.GetString(value, "name")
				clusterList = append(clusterList, cn)
			})

			for _, clusterName := range clusterList {
				cmdout, err := exec.Command(viper.GetString("koamc_az_command"),
					"aks",
					"get-credentials",
					"--resource-group",
					rg,
					"--name",
					clusterName,
					"--overwrite-existing").CombinedOutput()
				if err != nil {
					log.WithError(err).Errorln("failed getting AKS cluster credentials" + clusterName + ": " + string(cmdout))
					continue
				}
			}
		}

		time.Sleep(updatePeriod)
	}
}

func getAzureResourceGroup() (string, error) {
	timeout := time.Duration(time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET",
		viper.GetString("koamc_azure_metadata_service")+"/metadata/instance?api-version=2019-06-04",
		nil)
	req.Header.Add("Metadata", "true")

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed calling Azure metadata service")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed ready response from Azure metadata service")
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("Azure metadata server returned error: " + string(body))
	}
	resourceGroup, err := jsonparser.GetString(body, "compute", "resourceGroupName")
	if err != nil {
		return "", errors.Wrap(err, "failed getting resource group from instance metadata")
	}

	return resourceGroup, nil
}

func listAzureResourceGroups() ([]string, error) {
	cmdout, err := exec.Command(viper.GetString("koamc_az_command"),
		"group",
		"list",
		"-o",
		"json").CombinedOutput()

	if err != nil {
		return nil, errors.Wrap(err, "failed listing Azure resource groups:"+string(cmdout))
	}

	var resourceGroups []string
	_, err = jsonparser.ArrayEach(cmdout, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		rg, _ := jsonparser.GetString(value, "name")
		resourceGroups = append(resourceGroups, rg)
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed decoding output of Azure list resource groups: "+string(cmdout))
	}

	return resourceGroups, nil
}
