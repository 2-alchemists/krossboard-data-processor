![build](https://github.com/2-alchemists/krossboard-data-processor/workflows/Go/badge.svg)

# Requirements

* Ubuntu Server 18.04 64 bits LTS

```
sudo apt update && apt install -y rrdtool librrd-dev upx-ucl
```

# Build

## Install build dependencies
```
make build-deps
```

## Basic build

```
make
```

## Create distribution

```
make dist
```

## Build cloud images
First review and set the following environement variables accordingly.

```
export KROSSBOARD_UI_IMAGE="krossboard/krossboard-ui:latest"
export KROSSBOARD_KOAINSTANCE_IMAGE="rchakode/kube-opex-analytics:latest"
export GOOGLE_PROJECT_ID="krossboard-factory"
export GOOGLE_APPLICATION_CREDENTIALS="google_credentials_TBD"
export AWS_ACCESS_KEY="aws_access_key_TBD"
export AWS_SECRET_ACCESS_KEY="aws_secret_access_key_TBD"
export AZURE_SUBSCRIPTION_ID="azure_subscription_TBD"
export AZURE_TENANT_ID="azure_tenant_TBD"
export AZURE_CLIENT_ID="azure_client_id_TBD"
export AZURE_CLIENT_SECRET="azure_client_secret_TBD"
export AZURE_RESOURCE_GROUP="azure_resource_group_TBD"
```


This requires to have [Packer](https://www.packer.io/) installed. The `make build-deps` target downloads and installs Packer under the `/usr/loca/bin` folder.
```
make dist-cloud-image
```

The following flavors are also available for cloud-specific build: `dist-cloud-image-gcp`,  `dist-cloud-image-aws`,  `dist-cloud-image-azure`.

# Development integration

## Enable GCP API
 * Compute Engine API
 
  
## Credentials for Packer

* Log into the Google Developers Console and select a project.
* Under the "API Manager" section, click "Credentials."
* Click the "Create credentials" button, select "Service account key"
* Create a new service account that at least has `Compute Engine Instance Admin (v1)` and `Service Account User` roles.
* Choose JSON as the Key type and click "Create". A JSON file will be downloaded automatically. This is your account file.

## Make image public

```
PROJECT_ID=krossboard-factory
IMAGE_NAME=krossboard-release-20200726t1595766485-ubuntux8664	us
gcloud compute images add-iam-policy-binding $IMAGE_NAME \
    --member='allAuthenticatedUsers' \
    --role='roles/compute.imageUser' \
    --project=$PROJECT_ID
```

## Deploy from a public image

```
GCP_PROJECT=krossboard-test
GCP_ZONE=us-central1-a
GCP_INSTANCE_TYPE=g1-small
KROSSBOARD_IMAGE=krossboard-v010-50e8488-1595589351
KROSSBOARD_SERVICE_ACCOUNT=krossboard@krossboard-test.iam.gserviceaccount.com

gcloud compute instances create krossboard-demo-1 \
    --project=$GCP_PROJECT \
    --zone=$GCP_ZONE \
    --machine-type=$GCP_INSTANCE_TYPE \
    --service-account=$KROSSBOARD_SERVICE_ACCOUNT \
    --image=$KROSSBOARD_IMAGE \
    --image-project=krossboard-factory
```


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
