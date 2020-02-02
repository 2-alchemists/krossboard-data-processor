#!/bin/bash

set -u
set -e
RED_COLOR='\033[0;31m'
NO_COLOR='\033[0m'

if [ $# -gt 1 ]; then
    echo -e "${RED_COLOR}usage:\n\t $0 [/PATH/TO/BINARY]\n${NO_COLOR}"
    exit 1
fi


if [ `id -u` -ne 0 ]; then
    echo -e "${RED_COLOR}the script must be run as root${NO_COLOR}"
    exit 1
fi

DISTRIB_DIR=$(dirname $0)
KOAMC_BACKEND_PROGRAM=koamc-cluster-manager
KOAMC_BACKEND_SERVICE=${KOAMC_BACKEND_PROGRAM}.service
KOAMC_FRONTEND_SERVICE=koamc-webui.service
KOAMC_BINARY_PATH=${1-$DISTRIB_DIR/$KOAMC_BACKEND_PROGRAM}
KOAMC_USER=koamc
KOAMC_ROOT_DIR=/opt/$KOAMC_USER
KOAMC_CONFIG_DIR=/opt/$KOAMC_USER/etc
KOAMC_CONFIG_FILE=$KOAMC_BACKEND_PROGRAM.env 


echo -e "${RED_COLOR}updaing apt source list and package versions...${NO_COLOR}"
apt update && apt -y upgrade

# dev requirements
# apt install -y make rrdtool librrd-dev upx-ucl pkg-config

echo -e "${RED_COLOR}Installing Docker, rrdtool and librrd-dev...${NO_COLOR}"
apt install -y docker.io rrdtool librrd-dev

echo -e "${RED_COLOR}installing ${KOAMC_BACKEND_PROGRAM} with binary $KOAMC_BINARY_PATH...${NO_COLOR}"
install -d $KOAMC_ROOT_DIR/{bin,data,etc}
install -m 755 $KOAMC_BINARY_PATH $KOAMC_ROOT_DIR/bin/
install -m 644 $DISTRIB_DIR/scripts/$KOAMC_CONFIG_FILE $KOAMC_CONFIG_DIR/
install -m 644 $DISTRIB_DIR/scripts/$KOAMC_BACKEND_SERVICE /lib/systemd/system/
install -m 644 $DISTRIB_DIR/scripts/$KOAMC_FRONTEND_SERVICE /lib/systemd/system/

echo -e "${RED_COLOR}setting up runtime user ${KOAMC_USER}...${NO_COLOR}"
id -u $KOAMC_USER &> /dev/null || useradd $KOAMC_USER
usermod -d $KOAMC_ROOT_DIR -G docker $KOAMC_USER

CLOUD_PROVIDER="UNSET"
echo -e "${RED_COLOR}checking cloud provider...${NO_COLOR}"

# checking for Azure cloud
if wget --header 'Metadata: true' -q "http://169.254.169.254/metadata/instance?api-version=2019-06-04" > /dev/null; then
    CLOUD_PROVIDER="Azure"
    echo -e "${RED_COLOR}cloud provider is ${CLOUD_PROVIDER}${NO_COLOR}"
    echo -e "${RED_COLOR}applying prerequisites for $CLOUD_PROVIDER cloud...${NO_COLOR}"
    curl -sL https://aka.ms/InstallAzureCLIDeb | bash
    KOAMC_AZ_COMMAND=$(which az || echo "")
    if [ "$KOAMC_AZ_COMMAND" != "" ]; then
        echo "KOAMC_AZ_COMMAND=$KOAMC_AZ_COMMAND" >> $KOAMC_CONFIG_DIR/$KOAMC_CONFIG_FILE
        echo -e "${RED_COLOR}command az found at $KOAMC_AZ_COMMAND${NO_COLOR}"
    fi    
fi

# checking for Google cloud
if wget --header 'Metadata-Flavor: Google' -q "http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id" > /dev/null; then
    CLOUD_PROVIDER="Google"
    echo -e "${RED_COLOR}cloud provider is ${CLOUD_PROVIDER}${NO_COLOR}"
    echo -e "${RED_COLOR}applying prerequisites for $CLOUD_PROVIDER cloud...${NO_COLOR}"
    snap remove google-cloud-sdk
    apt install -y google-cloud-sdk
    KOAMC_GCLOUD_COMMAND=$(which gcloud || echo "")
    if [ "$KOAMC_GCLOUD_COMMAND" != "" ]; then
        echo -e "${RED_COLOR}command gcloud found at $KOAMC_GCLOUD_COMMAND${NO_COLOR}"
        echo "KOAMC_GCLOUD_COMMAND=$KOAMC_GCLOUD_COMMAND" >> $KOAMC_CONFIG_DIR/$KOAMC_CONFIG_FILE 
    fi  
fi

# checking for AWS cloud
if wget -q "http://169.254.169.254/latest/meta-data/placement/availability-zone" > /dev/null; then
    CLOUD_PROVIDER="AWS"
    echo -e "${RED_COLOR}cloud provider is ${CLOUD_PROVIDER}${NO_COLOR}"
    echo -e "${RED_COLOR}applying prerequisites for $CLOUD_PROVIDER cloud...${NO_COLOR}"
    apt -y install python3-pip && pip3 install --upgrade awscli

    KOAMC_AWS_COMMAND=$(which aws || echo "")
    if [ "$KOAMC_AWS_COMMAND" != "" ]; then
        echo -e "${RED_COLOR}command aws found at $KOAMC_AWS_COMMAND${NO_COLOR}"
        echo "KOAMC_AWS_COMMAND=$KOAMC_AWS_COMMAND" >> $KOAMC_CONFIG_DIR/$KOAMC_CONFIG_FILE
    fi
fi


# signing the installation
echo -e "${RED_COLOR}signing the installation${NO_COLOR}"
stat -c %Z $KOAMC_ROOT_DIR/bin/$KOAMC_BINARY_PATH  | md5sum > /opt/$KOAMC_USER/data/.sign

echo -e "${RED_COLOR}setting permissions on files and updating systemd settings{NO_COLOR}"
chown -R $KOAMC_USER:$KOAMC_USER $KOAMC_ROOT_DIR/
systemctl enable $KOAMC_BACKEND_SERVICE
systemctl enable $KOAMC_FRONTEND_SERVICE
systemctl daemon-reload
echo "done"
