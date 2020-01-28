# Requirements

* Ubuntu Server 18.04 64 bits LTS

```
sudo apt update && apt install -y rrdtool librrd-dev upx-ucl
```

# Build source
Simple build without binary optimization

  ```
  $ make
  ```

# Build cloud images

Build images for GCP, AWS and Azure.

```
$ export AWS_ACCESS_KEY=...
$ export AWS_SECRET_ACCESS_KEY=...
$ export AZURE_SUBSCRIPTION_ID=...
$ export AZURE_TENANT_ID=...
$ export AZURE_CLIENT_ID=...
$ export AZURE_CLIENT_SECRET=...
$ export AZURE_RESOURCE_GROUP=...
$ export GOOGLE_PROJECT_ID=...
$ export GOOGLE_APPLICATION_CREDENTIALS=...

$ make dist-cloud-image
```

## AKS Dev integration
The development integration requires to:
* to have/create an Azure service principal to authenticate against Azure
* to expose the service principal credentials to use with the application

### Create Azure service principal
Connect to Azure Portal:
* Select `Home -> Azure Active Directory -> App registrations`, select `New registration` and create a new app.
* Set a `Name` for the application and leave default settings for other options (e.g. *app-koamc-cluster-manager*).
* Click `Register` to create the application, then **note** the `Application (client) ID` and `Directory (tenant) ID`; they will be needed later.
* Select `Certificates & secrets -> New client secret` 
* Type a description if applicable and select a validity period.
* Click on `Add` to create a secret, then **note** the value of the secret, it will also be needed later.
* Select `Home -> Subcriptions` and then select the subscription you're using.
* Select `Access control (IAM) -> Add -> Add role assignment`.
* In the field `Role`, select the role `Azure Kubernetes Service Cluster User Role`.
* In the field `Select`, type the name of the application created (e.g. *app-koamc-cluster-manager* as above) to select the application your application.
* Click on `Save` to validate the assignement.
* Again in the field `Role`, select the role `Managed Applications Reader`.
* In the field `Select`, type the name of the application created (e.g. *app-koamc-cluster-manager* as above).
* Select the application and click on `Save` to validate the assignement.

More details, please refer to [Azure documentation related to the service principal subject](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest).

### Expose the service principal credentials to the application
Go the the source directory:
* Edit the file `run-koamc-azure.sh` and set the following variables according to the values generated in the previous steps: `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`.
* Run the script `./mock_azure.sh`; it allows to simulate an Azure metadata server
* Run the script `./run-koamc-azure.sh`.
* Your environment is now ready to take over all your AKS clusters.
* For each cluster, apply the file `./deploy/k8s/clusterrolebinding-aks.yml` to enable appropriated RBAC permissions to API needed by Kubernetes Opex Analytics.