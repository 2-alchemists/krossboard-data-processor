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

## Base OS

* Ubuntu Server 18.04 64 bits LTS

## Common requirements
librrd

```
sudo ln -s /usr/lib/x86_64-linux-gnu/librrd.so /usr/lib/librrd.so
sudo ln -s /usr/lib/x86_64-linux-gnu/librrd.so /usr/lib/librrd.so.4
```

> Warning: If you already have a version of Docker installed through snap, please remove it first `snap remove docker`.

Disconnect on the host and reconnect again so that group assignation takes effect.

## EKS additional requirements
AWS CLI

```
sudo apt-get update
sudo apt-get -y install python3-pip
pip3 install --upgrade --user awscli
sudo ln -s $HOME/.local/bin/aws /usr/local/bin/
```

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