package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func updateAKSClusters() {
	defer workers.Done()

	updatePeriod := time.Duration(viper.GetInt64("krossboard_update_interval")) * time.Minute
	for {
		err := azLogin()
		if err != nil {
			log.WithError(err).Errorln("Azure login failed")
			time.Sleep(updatePeriod)
			continue
		}

		groups, err := listGroups()
		if err != nil {
			log.WithError(err).Errorln("failed to list resource groups")
			time.Sleep(updatePeriod)
			continue
		}

		for _, group := range groups {
			cmdout, err := exec.Command(viper.GetString("krossboard_az_command"),
				"aks",
				"list",
				"--resource-group",
				group,
				"-o",
				"json").CombinedOutput()
			if err != nil {
				log.WithError(err).Errorln("failed listing AKS clusters for resource group" + group + ": " + string(cmdout))
				continue
			}
			var clusters []string
			_, err = jsonparser.ArrayEach(cmdout, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				cn, _ := jsonparser.GetString(value, "name")
				clusters = append(clusters, cn)
			})

			for _, cluster := range clusters {
				cmdout, err := exec.Command(viper.GetString("krossboard_az_command"),
					"aks",
					"get-credentials",
					"--resource-group",
					group,
					"--name",
					cluster,
					"--overwrite-existing").CombinedOutput()
				if err != nil {
					log.WithError(err).Errorln("failed getting AKS cluster credentials" + cluster + ": " + string(cmdout))
					continue
				}
			}
		}
		time.Sleep(updatePeriod)
	}
}

func getAzureSubscriptionID() (string, error) {
	timeout := time.Duration(time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET",
		viper.GetString("krossboard_azure_metadata_service")+"/metadata/instance?api-version=2019-06-04",
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
	subscriptionID, err := jsonparser.GetString(body, "compute", "subscriptionId")
	if err != nil {
		return "", errors.Wrap(err, "failed getting resource group from instance metadata")
	}

	return subscriptionID, nil
}

func listGroups() ([]string, error) {
	cmdout, err := exec.Command(viper.GetString("krossboard_az_command"),
		"group",
		"list",
		"-o",
		"json").CombinedOutput()

	if err != nil {
		return nil, errors.Wrap(err, "failed listing Azure resource groups:"+string(cmdout))
	}

	var groups []string
	_, err = jsonparser.ArrayEach(cmdout, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		rg, _ := jsonparser.GetString(value, "name")
		groups = append(groups, rg)
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed decoding output of Azure list resource groups: "+string(cmdout))
	}

	return groups, nil
}

func azLogin() error {
	tenant := os.Getenv("AZURE_TENANT_ID")
	appID := os.Getenv("AZURE_CLIENT_ID")
	appSecret := os.Getenv("AZURE_CLIENT_SECRET")

	errOut := error(nil)
	var cmdOut []byte
	if tenant != "" && appID != "" && appSecret != "" {
		cmdOut, errOut = exec.Command(
			viper.GetString("krossboard_az_command"),
			"login",
			"--service-principal",
			"--username",
			appID,
			"--password",
			appSecret,
			"--tenant",
			tenant,
		).CombinedOutput()
	} else {
		cmdOut, errOut = exec.Command(
			viper.GetString("krossboard_az_command"),
			"login",
			"--identity",
		).CombinedOutput()
	}

	if errOut != nil {
		return errors.Wrap(errOut, string(cmdOut))
	}

	return nil
}
