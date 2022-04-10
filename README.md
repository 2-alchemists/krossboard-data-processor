![Test and build](https://github.com/2-alchemists/krossboard-data-processor/workflows/test-and-build/badge.svg)
![Test and build](https://github.com/2-alchemists/krossboard-data-processor/workflows/golangci-lint/badge.svg)
![Build cloud images](https://github.com/2-alchemists/krossboard-data-processor/workflows/build-cloud-imagess/badge.svg)


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


This requires to have [Packer](https://www.packer.io/) installed. The target `make build-deps` downloads and installs Packer under the `/usr/loca/bin` folder.
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
IMAGE_NAME=krossboard-beta-v20200726t1595786717
gcloud compute images add-iam-policy-binding $IMAGE_NAME \
    --member='allAuthenticatedUsers' \
    --role='roles/compute.imageUser' \
    --project=$PROJECT_ID
```

## Deploy from a public image

```
GCP_PROJECT=krossboard-demo
GCP_ZONE=us-central1-a
GCP_INSTANCE_TYPE=g1-small
KROSSBOARD_IMAGE=krossboard-v010-50e8488-1595589351
KROSSBOARD_SERVICE_ACCOUNT=krossboard@krossboard-demo.iam.gserviceaccount.com

gcloud compute instances create krossboard-demo-1 \
        --scopes=https://www.googleapis.com/auth/cloud-platform \
        --project=$GCP_PROJECT \
        --zone=$GCP_ZONE \
        --machine-type=$GCP_INSTANCE_TYPE \
        --service-account=$KROSSBOARD_SERVICE_ACCOUNT \
        --image=$KROSSBOARD_IMAGE \
        --image-project=krossboard-factory \
        --tags=krossboard-server
```

## Enable access to the Krossboard UI

```
gcloud compute firewall-rules create default-allow-http \
    --project=$PROJECT_ID \
    --direction=INGRESS \
    --priority=1000 --network=default \
    --action=ALLOW \
    --rules=tcp:80 \
    --source-ranges=0.0.0.0/0 \
    --target-tags=krossboard-server
```

## Microsoft Azure
For a development environement the following information are required to auhenticate to your Azure subscription:

* `AZURE_TENANT_ID`
* `AZURE_CLIENT_ID`
* `AZURE_CLIENT_SECRET`
* `AZURE_RESOURCE_GROUP`

### Runtime credentials for Krossboard (Azure registered app)
Here are steps to create/get this information from your Azure Portal:
* In the search field, type and select **App registrations**.
* Click **New registration** and create a new app.
* Under **Name**, type *krossboard-app* (or any other value).
* Leave the other settings as is, then select **Register**.
* On the overview page, copy and save the following information for use later.
  * **Application (client) ID**, it's defined as `AZURE_CLIENT_ID` variable.
  * **Directory (tenant) ID**,it's defined as `AZURE_TENANT_ID` variable.
* Select **Certificates & secrets**, and then select **New client secret**.
* Type a **Description** and leave the default value for **Expires** (i.e. *In 1 year*).
* Click on **Add**.
* Copy the value of the secret and save it. It'll be used as `AZURE_CLIENT_SECRET`) variable. It cannot be retrieved anymore after you leave the page.
* In the search field, type and select **Subcriptions**.
* In the list of subscriptions, select the target subscription.
* Click **Access control (IAM)**.
* Select **Add -> Add role assignment**.
* Under **Role**, select `Azure Kubernetes Service Cluster User Role`.
* Under **Assign access to:**, leave this as **Azure AD user, group, or service principal**.
* Under **Select**, type the name of the application created above and select it.
* When you are done, select Save. Click on **Save**.
* Repeat the last three steps above to assign the role of `Managed Applications Reader` to the application.


### Image builde (Packer)

 * Create credentials for Packer following the same steps as for a runtime application registration above, while making sure to assign the role of `Contributor` to the app.
 * Create an Azure resource group and export it via the environment variable `AZURE_RESOURCE_GROUP`.

### Run the development script for Azure
Go the the source directory:
* Edit the file `run-data-processor-azure.sh` and set the following variables according to the values generated in the previous steps: `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`.
* Run the script `./mock_azure.sh`; it allows to simulate an Azure metadata server
* Run the script `./run-data-processor-azure.sh`.
* Your environment is now ready to take over all your AKS clusters.
* For each cluster, apply the file `./deploy/k8s/clusterrolebinding-aks.yml` to enable appropriated RBAC permissions to API needed by Kubernetes Opex Analytics.
