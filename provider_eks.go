package main

import (
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func updateEKSClusters() {
	awsRegion, err := getAWSRegion()
	if err != nil {
		log.WithError(err).Error("cannot retrieve AWS region")
		return
	}
	svc := eks.New(
		session.New(), // nolint: staticcheck // as New() is deprecated we should use NewSession() but behaviour seems different...
		aws.NewConfig().WithRegion(awsRegion))
	listInput := &eks.ListClustersInput{}
	listResult, err := svc.ListClusters(listInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			log.WithError(err).Errorf("failed listing EKS clusters (%v)", aerr.Code())
		} else {
			log.WithError(err).Error("failed listing EKS clusters")
		}
		return
	}
	for _, clusterName := range listResult.Clusters {
		cmd := exec.Command(viper.GetString("krossboard_awscli_command"),
			"eks",
			"update-kubeconfig",
			"--name",
			*clusterName,
			"--region",
			awsRegion)

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.WithField("cluster", clusterName).Errorf("failed getting EKS cluster credentials: %v", string(out))
		} else {
			log.WithField("cluster", clusterName).Debugln("added/updated  EKS cluster credentials")
		}
	}

}

func getAWSRegion() (string, error) {
	timeout := time.Duration(time.Second)
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET",
		viper.GetString("krossboard_aws_metadata_service")+"/latest/meta-data/placement/availability-zone",
		nil)
	if err != nil {
		return "", errors.Wrap(err, "failed calling AWS metadata service")
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed calling AWS metadata service")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed ready response from GCP metadata server")
	}

	bodyText := string(body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("AWS metadata service returned error: " + bodyText)
	}

	return bodyText[:len(bodyText)-1], nil
}
