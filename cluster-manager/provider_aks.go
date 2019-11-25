package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	azcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	azautorest "github.com/Azure/go-autorest/autorest"
	azauth "github.com/Azure/go-autorest/autorest/azure/auth"
)

func updateAKSClusters() {
	defer workers.Done()

	updatePeriod := time.Duration(viper.GetInt64("koacm_update_interval")) * time.Minute
	for {
		azsession, err := newAzureSession()
		if err != nil {
			log.WithError(err).Errorln("failed to create Azure session")
			time.Sleep(updatePeriod)
			continue
		}

		aksClient, err := newAKSClient(azsession)
		if err != nil {
			log.WithError(err).Errorln("failed to configure AKS clients")
			time.Sleep(updatePeriod)
			continue
		}
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Hour*1))
		defer cancel()

		groupClient := azresources.NewGroupsClient(azsession.SubscriptionID)
		groupClient.Authorizer = azsession.Authorizer
		var groups []string
		for groupIter, err := groupClient.ListComplete(context.Background(), "", nil); groupIter.NotDone(); err = groupIter.Next() {
			if err != nil {
				log.WithError(err).Errorln("error traversing resource group list")
				continue
			}
			groups = append(groups, *groupIter.Value().Name)
		}

		err = azLogin()
		if err != nil {
			log.WithError(err).Errorln("az login failed")
			time.Sleep(updatePeriod)
			continue
		}

		for _, group := range groups {

			for clusterIter, err := aksClient.ListByResourceGroupComplete(context.Background(), group); clusterIter.NotDone(); err = clusterIter.NextWithContext(ctx) {
				if err != nil {
					log.WithError(err).Errorln("error traversing AKS list")
					continue
				}

				cluster := clusterIter.Value()
				cmdout, err := exec.Command(
					viper.GetString("koamc_az_command"),
					"aks",
					"get-credentials",
					"--resource-group",
					group,
					"--name",
					*cluster.Name,
					"--overwrite-existing").CombinedOutput()

				if err != nil {
					log.WithError(err).Errorln("failed getting AKS cluster credentials " + *cluster.Name + ": " + string(cmdout))
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
	subscriptionID, err := jsonparser.GetString(body, "compute", "subscriptionId")
	if err != nil {
		return "", errors.Wrap(err, "failed getting resource group from instance metadata")
	}

	return subscriptionID, nil
}

func newAKSClient(azsession *AzureSession) (*azcontainerservice.ManagedClustersClient, error) {
	aksClient := azcontainerservice.NewManagedClustersClient(azsession.SubscriptionID)
	aksClient.Authorizer = azsession.Authorizer
	aksClient.PollingDuration = time.Hour * 1
	return &aksClient, nil
}

// AzureSession is an object representing session for subscription
type AzureSession struct {
	SubscriptionID string
	Authorizer     azautorest.Authorizer
}

func newAzureSession() (*AzureSession, error) {
	subscriptionID, err := getAzureSubscriptionID()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get Azure subscription")
	}
	authorizer, err := azauth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, errors.Wrap(err, "cannot initialize Azure authorizer")
	}
	return &AzureSession{
		SubscriptionID: subscriptionID,
		Authorizer:     authorizer,
	}, nil
}

func listGroups(azsession *AzureSession) ([]string, error) {
	groupsClient := azresources.NewGroupsClient(azsession.SubscriptionID)
	groupsClient.Authorizer = azsession.Authorizer
	groups := make([]string, 0)
	var err error
	for list, err := groupsClient.ListComplete(context.Background(), "", nil); list.NotDone(); err = list.Next() {
		if err != nil {
			return nil, errors.Wrap(err, "error traverising Azure groups list")
		}
		rgName := *list.Value().Name
		groups = append(groups, rgName)
	}
	return groups, err
}

func azLogin() error {
	cmdout, err := exec.Command(
		viper.GetString("koamc_az_command"),
		"login",
		"--identity").CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "failed login to Azure: "+string(cmdout))
	}
	return nil
}
