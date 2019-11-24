+++
title = "Installation for Azure AKS"
description = ""
weight = 20
draft = false
bref = ""
toc = true 
+++

## Before you begin
You will need an Azure having permissions
* admin role on AKS clusters
* An

## Steps:
 * From Dashboard -> Virtual machines
   * Create a Marketplace instance based on the latest version Kubernetes Opex Analytics Multi Cluster (it's based on Ubuntu Server 18.04 LTS)
 * From Dashboard -> Azure Active Directory
   * Create a group named `koamc-cluster-manager`. Note the created group ID, it will be next referred to as $GROUP_ID)
 * From Dashboard -> Azure Active Directory
   * Create a user that KOAMC will use to access AKS clusters
     * You can use an existing user according to your IT security policies.
     * Assign the user to the group `koamc-cluster-manager` created previously.
   * Keep the user password in mind or store it in a safe place
 * From Dashboard -> Azure Active Directory
   * Register an application for KOAMC or 
   * Copy the Principal information of the application (tenant id, client id, secret)
 * From Dashboard -> Security -> Key vaults
   * Create a new Key Vault or select an existing one
   * Create a secret to store the password of the created AKS user
   * Create a secret name `koamc-aks-password` and set it value with the password of AKS user defined in the previous step
   * Create an access policy to allows the registered application to the secret
 * Set koamc-cluster-manager configuration variables


```
az role assignment create --assignee $GROUP_ID \
 --role "Azure Kubernetes Service Cluster User Role"
``̀


## Edit/update `koamc-cluster-manager` config file

Edit `/opt/koamc/etc/koamc-cluster-manager.service.env` and add the following lines with appropriate values:

```
KOAMC_AZURE_KEYVAULT_NAME=kv-koamc-cluster-manager
KOAMC_AZURE_KEYVAULT_AKS_PASSWORD_VERSION=5ae6a32f47ca44c5a6f6bb9cb02ebbbc
```