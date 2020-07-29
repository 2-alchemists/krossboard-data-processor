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
	err := azLogin()
	if err != nil {
		log.WithError(err).Errorln("Azure login failed")
		return
	}

	groups, err := listGroups()
	if err != nil {
		log.WithError(err).Errorln("failed to list resource groups")
		return
	}

	for _, group := range groups {
		cmd := exec.Command(viper.GetString("krossboard_az_command"),
			"aks",
			"list",
			"--resource-group",
			group,
			"-o",
			"json")

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.WithError(err).Errorln("failed listing AKS clusters for resource group" + group + ": " + string(out))
			continue
		}

		var clusters []string
		_, err = jsonparser.ArrayEach(out, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			cn, _ := jsonparser.GetString(value, "name")
			clusters = append(clusters, cn)
		})
		if err != nil {
			log.WithError(err).WithFields(log.Fields{"group": group, "output": out}).Errorln("failed extracting cluster names from output")
			continue
		}

		for _, cluster := range clusters {
			cmd := exec.Command(viper.GetString("krossboard_az_command"),
				"aks",
				"get-credentials",
				"--resource-group",
				group,
				"--name",
				cluster,
				"--overwrite-existing")

			out, err := cmd.CombinedOutput()
			if err != nil {
				log.WithError(err).Errorln("failed getting AKS cluster credentials" + cluster + ": " + string(out))
				continue
			}
		}
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
	if err != nil {
		return "", errors.Wrap(err, "failed calling Azure metadata service")
	}
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
	cmd := exec.Command(viper.GetString("krossboard_az_command"),
		"group",
		"list",
		"-o",
		"json")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "failed listing Azure resource groups:"+string(out))
	}

	var groups []string
	_, err = jsonparser.ArrayEach(out, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		rg, _ := jsonparser.GetString(value, "name")
		groups = append(groups, rg)
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed decoding output of Azure list resource groups: "+string(out))
	}

	return groups, nil
}

func azLogin() error {
	tenant := os.Getenv("AZURE_TENANT_ID")
	appID := os.Getenv("AZURE_CLIENT_ID")
	appSecret := os.Getenv("AZURE_CLIENT_SECRET")

	var errOut error
	var out []byte
	if tenant != "" && appID != "" && appSecret != "" {
		cmd := exec.Command(
			viper.GetString("krossboard_az_command"),
			"login",
			"--service-principal",
			"--username",
			appID,
			"--password",
			appSecret,
			"--tenant",
			tenant,
		)
		out, errOut = cmd.CombinedOutput()
	} else {
		cmd := exec.Command(
			viper.GetString("krossboard_az_command"),
			"login",
			"--identity",
		)
		out, errOut = cmd.CombinedOutput()
	}

	if errOut != nil {
		return errors.Wrap(errOut, string(out))
	}

	return nil
}
