+++
title = "Installation for Azure AKS"
description = ""
weight = 20
draft = false
bref = ""
toc = true 
+++

## Before you begin
You will need:
* An active Azure subscription.
* An admin access to the Azure subscription

## Summary of steps:
The steps are the following:
* Create a VM instance
* Assign a managed identity to the instance
* Assign the following built-in Azure roles to the instance: `Managed Application Reader` and `Azure Kubernetes Service Cluster User Role`


# Step by step installation

* From Dashboard -> Virtual machines
   * Create a Marketplace instance based on the latest version Kubernetes Opex Analytics Multi Cluster (it's based on Ubuntu Server 18.04 LTS)

```
KOAMC_CLUSTER_MANAGER_IID=<objectId of the KOAMC instance>
az role assignment create --assignee ${KOAMC_CLUSTER_MANAGER_IID} --role "Azure Kubernetes Service Cluster User Role"
az role assignment create --assignee ${KOAMC_CLUSTER_MANAGER_IID} --role "Managed Applications Reader"
kubectl apply -f ./deploy/k8s/clusterrolebinding-aks.yml
``̀