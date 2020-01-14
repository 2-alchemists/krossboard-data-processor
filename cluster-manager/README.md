# Requirements

* Ubuntu Server 18.04 64 bits LTS

# Build source
Simple build without binary optimization

  ```
  $ make
  ```

# Build cloud images

## Google
Create a distrib archive and generate a cloud image

```
$ make dist
$ GOOGLE_APPLICATION_CREDENTIALS=/home/ubuntu/.gcp/serviceaccount/credentials-packer_image_builder.json \
    packer build -var-file=./deploy/packer/variables.json \
    ./deploy/packer/gcp.json
```

## AKS Dev integration
Connect to Azure Portal:

* From `Search -> Azure Active Directory -> App registrations`, select `New registration` and create a new app.
* Set a `Name` for the application and leave default settings for other options (e.g. *app-koamc-cluster-manager*).
* Click `Register`  to create the application.
* Note the `Application client ID` and `Directort Tenant ID`; you will need them later.
* Select `Certificates & secrets`.
* Under `Client secrets` section, click on `New client secret` to create a secret. Then note the value of the secret.
* From `Search -> Subcriptions`, select the Azure subscription you're using.
* Select `Access control (IAM)`.
* Select `Add -> Add role assignment`.
* In the field `Role`, select the role `Azure Kubernetes Service Cluster User Role`.
* In the field `Select`, type the name of the application created (e.g. *app-koamc-cluster-manager* as above).
* Select the application and click on `Save` to validate the assignement.
* In the field `Role`, select the role `Managed Applications Reader`.
* In the field `Select`, type the name of the application created (e.g. *app-koamc-cluster-manager* as above).
* Select the application and click on `Save` to validate the assignement.

Go the the source directory:
* Edit the file `run-koamc-azure.sh` and set the following variables according to the values generated in the previous steps: `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`.
* Run the script `./mock_azure.sh`; it allows to simulate an Azure metadata server
* Run the script `./run-koamc-azure.sh`.
* Your environment is now ready to take over all your AKS clusters.
* For each cluster, apply the file `./deploy/clusterrolebinding-aks.yml` to enable appropriated RBAC permissions to API needed by Kubernetes Opex Analytics.


# Installing koamc-cluster-manager
Run installation scripts

```
tar zxf koamc-cluster-manager-*-x86_64.tgz
cd koamc-cluster-manager-*-x86_64/
sudo ./install.sh
```

# Cleanup

```
sudo sudo apt autoremove -y
```