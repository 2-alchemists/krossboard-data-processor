#!/bin/bash

set -u
set -e
RED_COLOR='\033[0;31m'
NO_COLOR='\033[0m' # No Color

if [ $# -gt 1 ]; then
    echo -e "${RED_COLOR}usage:\n\t $0 [/PATH/TO/BINARY]\n${NO_COLOR}"
    exit 1
fi


if [ `id -u` -ne 0 ]; then
    echo -e "${RED_COLOR}the script must be run as root${NO_COLOR}"
    exit 1
fi

PROGRAM_NAME=koamc-cluster-manager
INSTALL_DIR=$(dirname $0)
KOAMC_BINARY_PATH=${1-$INSTALL_DIR/$PROGRAM_NAME}
KOAMC_USER=koamc
KOAMC_ROOT_DIR=/opt/$KOAMC_USER
DOCKER_GROUP=docker

# dev requirements
# sudo apt install pkg-config

echo -e "${RED_COLOR}Installing Docker, rrdtool and librrd-dev...${NO_COLOR}"
sudo apt update && \
    apt install -y docker.io rrdtool librrd-dev

echo -e "${RED_COLOR}installing ${PROGRAM_NAME} with binary $KOAMC_BINARY_PATH...${NO_COLOR}"
install -d $KOAMC_ROOT_DIR/{bin,data,etc}
install -m 755 $KOAMC_BINARY_PATH $KOAMC_ROOT_DIR/bin/
install -m 644 $INSTALL_DIR/scripts/$PROGRAM_NAME.service.env $KOAMC_ROOT_DIR/etc/
install -m 644 $INSTALL_DIR/scripts/$PROGRAM_NAME.service /lib/systemd/system/

echo -e "${RED_COLOR}setting up runtime user ${KOAMC_USER}...${NO_COLOR}"
id -u $KOAMC_USER &> /dev/null || sudo useradd $KOAMC_USER
sudo usermod -d $KOAMC_ROOT_DIR -G $DOCKER_GROUP $KOAMC_USER

CLOUD_PROVIDER="UNSET"
echo -e "${RED_COLOR}checking cloud provider...${NO_COLOR}"

# checking for Azure cloud
if wget --header 'Metadata: true' -q "http://169.254.169.254/metadata/instance?api-version=2019-06-04" > /dev/null; then
    CLOUD_PROVIDER="Azure"
    echo -e "${RED_COLOR}cloud provider is ${CLOUD_PROVIDER}${NO_COLOR}"
    echo -e "${RED_COLOR}applying prerequisites for $CLOUD_PROVIDER cloud...${NO_COLOR}"
    sudo curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
    KOAMC_AZ_COMMAND=$(which az || echo "")
    if [ "$KOAMC_AZ_COMMAND" != "" ]; then
        echo "KOAMC_AZ_COMMAND=$KOAMC_AZ_COMMAND" >> $KOAMC_ROOT_DIR/etc/$PROGRAM_NAME.service.env 
        echo -e "${RED_COLOR}command az found at $KOAMC_AZ_COMMAND${NO_COLOR}"
    fi    
fi

# checking for Google cloud
if wget --header 'Metadata-Flavor: Google' -q "http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id" > /dev/null; then
    CLOUD_PROVIDER="Google"
    echo -e "${RED_COLOR}cloud provider is ${CLOUD_PROVIDER}${NO_COLOR}"
    echo -e "${RED_COLOR}applying prerequisites for $CLOUD_PROVIDER cloud...${NO_COLOR}"
    sudo snap remove google-cloud-sdk
    sudo apt install -y google-cloud-sdk
    KOAMC_GCLOUD_COMMAND=$(which gcloud || echo "")
    if [ "$KOAMC_GCLOUD_COMMAND" != "" ]; then
        echo -e "${RED_COLOR}command gcloud found at $KOAMC_GCLOUD_COMMAND${NO_COLOR}"
        echo "KOAMC_GCLOUD_COMMAND=$KOAMC_GCLOUD_COMMAND" >> $KOAMC_ROOT_DIR/etc/$PROGRAM_NAME.service.env 
    fi  
fi

# checking for AWS cloud
if wget -q "http://169.254.169.254/latest/meta-data/placement/availability-zone" > /dev/null; then
    CLOUD_PROVIDER="AWS"
    echo -e "${RED_COLOR}cloud provider is ${CLOUD_PROVIDER}${NO_COLOR}"
    echo -e "${RED_COLOR}applying prerequisites for $CLOUD_PROVIDER cloud...${NO_COLOR}"
    sudo apt-get update && \
        sudo apt-get -y install python3-pip && \
        sudo pip3 install --upgrade awscli

    KOAMC_AWS_COMMAND=$(which aws || echo "")
    if [ "$KOAMC_AWS_COMMAND" != "" ]; then
        echo -e "${RED_COLOR}command aws found at $KOAMC_AWS_COMMAND${NO_COLOR}"
        echo "KOAMC_AWS_COMMAND=$KOAMC_AWS_COMMAND" >> $KOAMC_ROOT_DIR/etc/$PROGRAM_NAME.service.env 
    fi
fi

chown -R $KOAMC_USER:$KOAMC_USER $KOAMC_ROOT_DIR/
systemctl enable $PROGRAM_NAME
systemctl daemon-reload
echo "done"
