# Requirements

* Ubuntu Server 18.04 64 bits LTS

```
sudo apt update && apt install -y rrdtool librrd-dev upx-ucl
```

# Build

## Install build dependencies
Simple build without binary optimization

```
make build-deps
```

## Build without binary compression

```
make
```

## Build with binary compression
This makes a build followed by an [UPX](https://upx.github.io/) compression

```
make build-compress
```

## Make a distribution archive
This makes a compressed binary and generates an archive with installation script.
```
make dist
```

## Build cloud images
This make a  distribution followed by a [Packer](https://www.packer.io/) build to generate cloud image for GCP, AWS and Azure.

```
make dist-cloud-image
```

> The following environement variables shall be set to enable the authentication on the different cloud environments needed for Packer builds (Amazon AWS, Microsoft Azure, and Google GCP).  
  ```
  export AWS_ACCESS_KEY=...
  export AWS_SECRET_ACCESS_KEY=...
  export AZURE_SUBSCRIPTION_ID=...
  export AZURE_TENANT_ID=...
  export AZURE_CLIENT_ID=...
  export AZURE_CLIENT_SECRET=...
  export AZURE_RESOURCE_GROUP=...
  export GOOGLE_PROJECT_ID=...
  export GOOGLE_APPLICATION_CREDENTIALS=...
  ```

# Development integration

## Microsoft Azure
To integrate a development environement with Microsoft Azure, you need to the following information to auhenticate to your Azure subscription with appropriate permissions: `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` and `AZURE_RESOURCE_GROUP`. 

### Create/get authentication information
Here are steps to create/get this information from your Azure Portal:
* Select `Home -> Azure Active Directory -> App registrations`, select `New registration` and create a new app.
* Set a `Name` for the application and leave default settings for other options (e.g. *app-krossboard-data-processor*).
* Click `Register` to create the application, then **note** the `Application (client) ID` and `Directory (tenant) ID`; they will be needed later.
* Select `Certificates & secrets -> New client secret` 
* Type a description if applicable and select a validity period.
* Click on `Add` to create a secret, then **note** the value of the secret, it will also be needed later.
* Select `Home -> Subcriptions` and then select the subscription you're using.
* Select `Access control (IAM) -> Add -> Add role assignment`.
* In the field `Role`, select the role `Azure Kubernetes Service Cluster User Role`.
* In the field `Select`, type the name of the application created (e.g. *app-krossboard-data-processor* as above) to select the application your application.
* Click on `Save` to validate the assignement.
* Again in the field `Role`, select the role `Managed Applications Reader`.
* In the field `Select`, type the name of the application created (e.g. *app-krossboard-data-processor* as above).
* Select the application and click on `Save` to validate the assignement.

More details, please refer to [Azure documentation related to the service principal subject](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest).

### Run the development script for Azure
Go the the source directory:
* Edit the file `run-data-processor-azure.sh` and set the following variables according to the values generated in the previous steps: `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`.
* Run the script `./mock_azure.sh`; it allows to simulate an Azure metadata server
* Run the script `./run-data-processor-azure.sh`.
* Your environment is now ready to take over all your AKS clusters.
* For each cluster, apply the file `./deploy/k8s/clusterrolebinding-aks.yml` to enable appropriated RBAC permissions to API needed by Kubernetes Opex Analytics.