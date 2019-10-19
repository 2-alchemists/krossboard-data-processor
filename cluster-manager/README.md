# Build
Simple build without binary optimization

  ```
  $ make
  ```

Build and make release (include a binany compression step with upx)

  ```
  $ make dist
  ```

# Installation

## Requirements

* Ubuntu Server 18.04 64 bits LTS 

## EKS Requirements

### AWS CLI

```
sudo apt-get update
sudo apt-get -y install python3-pip
pip3 install --upgrade --user awscli
sudo ln -s $HOME/.local/bin/aws /usr/local/bin/
```

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
$ sudo install -m 644 ./scripts/koamc-cluster-manager.service.env $KOAMC_ROOT_DIR/etc/
$ sudo install -m 644 ./scripts/koamc-cluster-manager.service /lib/systemd/system/
```

Fix permissions on directories

```
$ sudo chown -R $KOAMC_USER:$KOAMC_USER $KOAMC_ROOT_DIR/{data,run}
```
