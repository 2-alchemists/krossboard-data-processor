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
PROGRAM_NAME=krossboard
PROGRAM_BACKEND=krossboard-data-processor
PROGRAM_BACKEND_SERVICE=${PROGRAM_BACKEND}.service
PROGRAM_FRONTEND_SERVICE=krossboard-ui.service
DISTRIB_BINARY_PATH=${1-$DISTRIB_DIR/$PROGRAM_BACKEND}
PROGRAM_USER=krossboard
PROGRAM_HOME_DIR=/opt/$PROGRAM_USER
PROGRAM_CONFIG_DIR=/opt/$PROGRAM_USER/etc
PROGRAM_CONFIG_FILE=$PROGRAM_NAME.env 


echo -e "${RED_COLOR}updaing apt source list and package versions...${NO_COLOR}"
apt update && apt -y upgrade

# dev requirements
# apt install -y make rrdtool librrd-dev upx-ucl pkg-config

echo -e "${RED_COLOR}Installing Docker, rrdtool and librrd-dev...${NO_COLOR}"
apt install -y docker.io rrdtool librrd-dev

echo -e "${RED_COLOR}installing ${PROGRAM_BACKEND} with binary $DISTRIB_BINARY_PATH...${NO_COLOR}"
install -d $PROGRAM_HOME_DIR/{bin,data,etc}
install -m 755 $DISTRIB_BINARY_PATH $PROGRAM_HOME_DIR/bin/
install -m 644 $DISTRIB_DIR/scripts/$PROGRAM_CONFIG_FILE $PROGRAM_CONFIG_DIR/
install -m 644 $DISTRIB_DIR/scripts/$PROGRAM_BACKEND_SERVICE /lib/systemd/system/
install -m 644 $DISTRIB_DIR/scripts/$PROGRAM_FRONTEND_SERVICE /lib/systemd/system/

echo -e "${RED_COLOR}setting up runtime user ${PROGRAM_USER}...${NO_COLOR}"
id -u $PROGRAM_USER &> /dev/null || useradd $PROGRAM_USER
usermod -d $PROGRAM_HOME_DIR -G docker $PROGRAM_USER

CLOUD_PROVIDER="UNSET"
echo -e "${RED_COLOR}checking cloud provider...${NO_COLOR}"

# checking for Azure cloud
if wget --header 'Metadata: true' -q "http://169.254.169.254/metadata/instance?api-version=2019-06-04" > /dev/null; then
    CLOUD_PROVIDER="Azure"
    echo -e "${RED_COLOR}cloud provider is ${CLOUD_PROVIDER}${NO_COLOR}"
    echo -e "${RED_COLOR}applying prerequisites for $CLOUD_PROVIDER cloud...${NO_COLOR}"
    curl -sL https://aka.ms/InstallAzureCLIDeb | bash
    DIST_AZ_COMMAND=$(which az || echo "")
    if [ "$DIST_AZ_COMMAND" != "" ]; then
        echo "DIST_AZ_COMMAND=$DIST_AZ_COMMAND" >> $PROGRAM_CONFIG_DIR/$PROGRAM_CONFIG_FILE
        echo -e "${RED_COLOR}command az found at $DIST_AZ_COMMAND${NO_COLOR}"
    fi    
fi

# checking for Google cloud
if wget --header 'Metadata-Flavor: Google' -q "http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id" > /dev/null; then
    CLOUD_PROVIDER="Google"
    echo -e "${RED_COLOR}cloud provider is ${CLOUD_PROVIDER}${NO_COLOR}"
    echo -e "${RED_COLOR}applying prerequisites for $CLOUD_PROVIDER cloud...${NO_COLOR}"
    snap remove google-cloud-sdk
    apt install -y google-cloud-sdk
    DIST_GCLOUD_COMMAND=$(which gcloud || echo "")
    if [ "$DIST_GCLOUD_COMMAND" != "" ]; then
        echo -e "${RED_COLOR}command gcloud found at $DIST_GCLOUD_COMMAND${NO_COLOR}"
        echo "DIST_GCLOUD_COMMAND=$DIST_GCLOUD_COMMAND" >> $PROGRAM_CONFIG_DIR/$PROGRAM_CONFIG_FILE 
    fi  
fi

# checking for AWS cloud
if wget -q "http://169.254.169.254/latest/meta-data/placement/availability-zone" > /dev/null; then
    CLOUD_PROVIDER="AWS"
    echo -e "${RED_COLOR}cloud provider is ${CLOUD_PROVIDER}${NO_COLOR}"
    echo -e "${RED_COLOR}applying prerequisites for $CLOUD_PROVIDER cloud...${NO_COLOR}"
    apt -y install python3-pip && pip3 install --upgrade awscli

    DIST_AWS_COMMAND=$(which aws || echo "")
    if [ "$DIST_AWS_COMMAND" != "" ]; then
        echo -e "${RED_COLOR}command aws found at $DIST_AWS_COMMAND${NO_COLOR}"
        echo "DIST_AWS_COMMAND=$DIST_AWS_COMMAND" >> $PROGRAM_CONFIG_DIR/$PROGRAM_CONFIG_FILE
    fi
fi


# signing the installation
echo -e "${RED_COLOR}signing the installation${NO_COLOR}"
stat -c %Z $PROGRAM_HOME_DIR/bin/$DISTRIB_BINARY_PATH  | md5sum > /opt/$PROGRAM_USER/data/.sign

# set default update interval
echo "KROSSBOARD_UPDATE_INTERVAL=30" >> $PROGRAM_CONFIG_DIR/$PROGRAM_CONFIG_FILE

echo -e "${RED_COLOR}setting permissions on files and updating systemd settings{NO_COLOR}"
chown -R $PROGRAM_USER:$PROGRAM_USER $PROGRAM_HOME_DIR/
systemctl enable $PROGRAM_BACKEND_SERVICE
systemctl enable $PROGRAM_FRONTEND_SERVICE
systemctl daemon-reload
echo "done"
