+++
title = "Installation for AWS EKS clusters"
description = ""
weight = 20
draft = false
bref = ""
toc = true 
+++



## Install Kubernetes Metrics Server 
The following steps is based on [EKS's official documentation](https://docs.aws.amazon.com/eks/latest/userguide/metrics-server.html) and require install [jq](https://stedolan.github.io/jq/).

```
DOWNLOAD_URL=$(curl --silent "https://api.github.com/repos/kubernetes-incubator/metrics-server/releases/latest" | jq -r .tarball_url)
DOWNLOAD_VERSION=$(grep -o '[^/v]*$' <<< $DOWNLOAD_URL)
curl -Ls $DOWNLOAD_URL -o metrics-server-$DOWNLOAD_VERSION.tar.gz
mkdir metrics-server-$DOWNLOAD_VERSION
tar -xzf metrics-server-$DOWNLOAD_VERSION.tar.gz --directory metrics-server-$DOWNLOAD_VERSION --strip-components 1
kubectl apply -f metrics-server-$DOWNLOAD_VERSION/deploy/1.8+/
```

## Create Required IAM Role
We describe below two approaches, one based on manual creation through AWS Management Console and the other one thorough Terraform.

### Create Role Using AWS Management Console
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

### Create Role Using Terraform
Use the following Terraform content (see file `create-koamc-iam-role.tf`).

Known issue

```
Value (koamc-cluster-manager) for parameter iamInstanceProfile.name is invalid. Invalid IAM Instance Profile name
```

## Enable RBAC Permissions
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
