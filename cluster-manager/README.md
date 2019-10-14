# Build

  ```
  $ make
  ```

# Requirements

Ubuntu Server 18.04 64 bits LTS 

## EKS Requirements

### Install AWS CLI tool

```
sudo apt-get update
sudo apt-get -y install python3-pip
pip3 install --upgrade --user awscli
sudo ln -s $HOME/.local/bin/aws /usr/local/bin/
```

### Install Kubernetes Metrics Server 
The following steps is based on [official EKS docs](https://docs.aws.amazon.com/eks/latest/userguide/metrics-server.html) and require install [jq](https://stedolan.github.io/jq/).

```
DOWNLOAD_URL=$(curl --silent "https://api.github.com/repos/kubernetes-incubator/metrics-server/releases/latest" | jq -r .tarball_url)
DOWNLOAD_VERSION=$(grep -o '[^/v]*$' <<< $DOWNLOAD_URL)
curl -Ls $DOWNLOAD_URL -o metrics-server-$DOWNLOAD_VERSION.tar.gz
mkdir metrics-server-$DOWNLOAD_VERSION
tar -xzf metrics-server-$DOWNLOAD_VERSION.tar.gz --directory metrics-server-$DOWNLOAD_VERSION --strip-components 1
kubectl apply -f metrics-server-$DOWNLOAD_VERSION/deploy/1.8+/
```

### Create IAM Role for KOAMC Cluster Manager
We describe below two approaches, one based on manual creation through AWS Management Console and the other one thorough Terraform.

#### Role creation through AWS Management Console
Log into the AWS Management Console:

* Select IAM service
* Go to `Role` section
* Create a role name `koamc-cluster-manager`
* Create a policy and set it with below JSON content
    ```
    {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "eks:ListClusters",
                    "eks:DescribeCluster"
                ],
                "Resource": "*"
            }
        ]
    }
    ```

#### Role creation though Terraform
Use the following Terraform content (see file `create-koamc-iam-role.tf`).

Known issue

```
Value (koamc-cluster-manager) for parameter iamInstanceProfile.name is invalid. Invalid IAM Instance Profile name
```

### Set RBAC permissions for the EKS cluster
First edit the EKS's aws-auth ConfigMap

```
kubectl -n kube-system edit configmap aws-auth
```

Add the following entry under `mapRoles` section.
```
    - groups:
      - koamc-cluster-manager
      rolearn: arn:aws:iam::{{AccountID}}:role/koamc-cluster-manager
      username: arn:aws:iam::{{AccountID}}:role/koamc-cluster-manager
```

> Take care to not remove existing `mapRoles` entries. And also replace `{{AccountID}}` with the id of your AWS account.

That above entry is meant to enable instances holding the given role to act as members of the group `koamc-cluster-manager` within Kubernetes cluster. For additional details you can refer to [EKS documentation](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html).

# Installation
Connect to a terminal on the host and perform the following steps.

```
sudo apt install -y docker.io
sudo usermod -G docker $USER
```

Disconnect on the host and reconnect again so that group assignation takes effect.

> Warning: If you already have a version of Docker installed by snap, pealease remove it first `snap remove docker`.

Set user and root installation directory

```
$ export KOAMC_USER=koamc
$ export KOAMC_ROOT_DIR=/opt/$KOAMC_USER
$ export DOCKER_GROUP=docker
$ export KOAMC_BINARY=koamc-cluster-manager
```

Create koamc user

```
$ sudo useradd $KOAMC_USER -G $DOCKER_GROUP -m --home-dir $KOAMC_ROOT_DIR
```

Create installation tree

```
$ sudo install -d $KOAMC_ROOT_DIR/{bin,data,etc,run}
```

Copy binaries

```
$ sudo install -m 755 $KOAMC_BINARY $KOAMC_ROOT_DIR/bin/
```

Copy systemd scripts

```
$ sudo install -m 644 ./scripts/kube-opex-analytics-mc.service.env $KOAMC_ROOT_DIR/etc/
$ sudo install -m 644 ./scripts/kube-opex-analytics-mc.service /lib/systemd/system/
```

Fix permissions on directories

```
$ sudo chown -R $KOAMC_USER:$KOAMC_USER $KOAMC_ROOT_DIR/{data,run}
```
