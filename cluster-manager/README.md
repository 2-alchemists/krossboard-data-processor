# Build

  ```
  $ make
  ```


# Requirements

## AWS

### Install aws client

```
$ sudo apt-get update
$ sudo apt-get -y install python3-pip
$ sudo pip3 install --upgrade --user awscli
$ sudo ln -s $HOME/.local/bin/aws /usr/local/bin/
```

### Install Kubernetes Metrics Server 

Refer to official docs: https://docs.aws.amazon.com/eks/latest/userguide/metrics-server.html

### IAM role
Create a role with the following policies

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


# Installation


Use official Docker distribution, which can be installed as follows

```
$ sudo apt install docker.io
```

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
