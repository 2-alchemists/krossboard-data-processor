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
PRODUCT_NAME=krossboard
PRODUCT_BACKEND=krossboard-data-processor
DISTRIB_BINARY_PATH=${1-$DISTRIB_DIR/$PRODUCT_BACKEND}
PRODUCT_USER=krossboard
PRODUCT_HOME_DIR=/opt/$PRODUCT_USER
PRODUCT_CONFIG_DIR=/opt/$PRODUCT_USER/etc
PRODUCT_CONFIG_FILE=$PRODUCT_NAME.env

echo -e "${RED_COLOR} Updating apt-get source list and package versions... ${NO_COLOR}"
apt-get update && apt-get -y upgrade

# dev requirements
# apt-get install -y make rrdtool librrd-dev upx-ucl pkg-config

echo -e "${RED_COLOR} Installing Docker, rrdtool and librrd-dev... ${NO_COLOR}"
apt-get install -y docker.io rrdtool librrd-dev vim curl

echo -e "${RED_COLOR} Installing ${PRODUCT_BACKEND} binary from $DISTRIB_BINARY_PATH... ${NO_COLOR}"
install -d $PRODUCT_HOME_DIR/{bin,data,etc}
install -m 755 $DISTRIB_BINARY_PATH $PRODUCT_HOME_DIR/bin/
install -m 644 $DISTRIB_DIR/scripts/$PRODUCT_CONFIG_FILE $PRODUCT_CONFIG_DIR/
install -m 644 $DISTRIB_DIR/scripts/${PRODUCT_NAME}-*.{service,timer} /lib/systemd/system/

echo -e "${RED_COLOR} Setting up runtime user ${PRODUCT_USER}...${NO_COLOR}"
id -u $PRODUCT_USER &> /dev/null || useradd $PRODUCT_USER
usermod -d $PRODUCT_HOME_DIR -G docker $PRODUCT_USER

CLOUD_PROVIDER="AUTO"
echo -e "${RED_COLOR} Checking cloud provider... ${NO_COLOR}"

# Checking for Azure cloud builder
if [ -z "$KB_VBOX_BUILDER" ] && wget --header 'Metadata: true' -q "http://169.254.169.254/metadata/instance?api-version=2019-06-04" > /dev/null; then
    CLOUD_PROVIDER="Azure"
    echo -e "${RED_COLOR} Cloud provider is ${CLOUD_PROVIDER} ${NO_COLOR}"
    echo -e "${RED_COLOR} Applying prerequisites for $CLOUD_PROVIDER cloud... ${NO_COLOR}"
    curl -sL https://aka.ms/InstallAzureCLIDeb | bash
    DIST_AZ_COMMAND=$(which az || echo "")
    if [ "$DIST_AZ_COMMAND" != "" ]; then
        echo "DIST_AZ_COMMAND=$DIST_AZ_COMMAND" >> $PRODUCT_CONFIG_DIR/$PRODUCT_CONFIG_FILE
        echo -e "${RED_COLOR} Command az found at $DIST_AZ_COMMAND${NO_COLOR}"
    fi    
fi

# Checking for Google cloud builder
if [ -z "$KB_VBOX_BUILDER" ] && wget --header 'Metadata-Flavor: Google' -q "http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id" > /dev/null; then
    CLOUD_PROVIDER="Google"
    echo -e "${RED_COLOR} Cloud provider is ${CLOUD_PROVIDER} ${NO_COLOR}"
    echo -e "${RED_COLOR} Applying prerequisites for $CLOUD_PROVIDER cloud... ${NO_COLOR}"
    snap remove google-cloud-sdk
    apt-get install -y google-cloud-sdk
    DIST_GCLOUD_COMMAND=$(which gcloud || echo "")
    if [ "$DIST_GCLOUD_COMMAND" != "" ]; then
        echo -e "${RED_COLOR} Command gcloud found at $DIST_GCLOUD_COMMAND${NO_COLOR} "
        echo "DIST_GCLOUD_COMMAND=$DIST_GCLOUD_COMMAND" >> $PRODUCT_CONFIG_DIR/$PRODUCT_CONFIG_FILE 
    fi  
fi

# checking for AWS cloud
if [ -z "$KB_VBOX_BUILDER" ] &&  wget -q "http://169.254.169.254/latest/meta-data/placement/availability-zone" > /dev/null; then
    CLOUD_PROVIDER="AWS"
    echo -e "${RED_COLOR} Cloud provider is ${CLOUD_PROVIDER} ${NO_COLOR}"
    echo -e "${RED_COLOR} Applying prerequisites for $CLOUD_PROVIDER cloud... ${NO_COLOR}"
    apt-get -y install python3-pip && pip3 install --upgrade awscli
    DIST_AWS_COMMAND=$(which aws || echo "")
    if [ "$DIST_AWS_COMMAND" != "" ]; then
        echo -e "${RED_COLOR} Command aws found at $DIST_AWS_COMMAND ${NO_COLOR}"
        echo "DIST_AWS_COMMAND=$DIST_AWS_COMMAND" >> $PRODUCT_CONFIG_DIR/$PRODUCT_CONFIG_FILE
    fi
fi

echo -e "${RED_COLOR} Dumping configuration file ${NO_COLOR}"
cat $PRODUCT_CONFIG_DIR/$PRODUCT_CONFIG_FILE

echo -e "${RED_COLOR} Creating Caddy configuration file ${PRODUCT_CONFIG_DIR}/Caddyfile ${NO_COLOR}"
cat <<EOF >> ${PRODUCT_CONFIG_DIR}/Caddyfile
# domain name.
:80

# Set this path to your site's directory.
root * /var/www/html

# Enable the static file server.
file_server

# Add reverse proxy for the API
route /api/* {
  reverse_proxy 127.0.0.1:1519
}

# Rewrites other URI to index.html
route /* {
  try_files {path} {path}/ /index.html
}

# Enable basic auth
basicauth /* {
    krossboard JDJhJDEwJGxGMmN2ZDJ4NjgycjVTbi5pRThSNGVnaWViSGpiNWpKVVpPLjRkRGNCVmV4VGtOUnBiSjRL
}
EOF

echo -e "${RED_COLOR} Installing script to change basicauth password ${NO_COLOR}"
install -m 755 $DISTRIB_DIR/scripts/krossboard-change-passwd.sh $PRODUCT_HOME_DIR/bin/krossboard-change-passwd

echo -e "${RED_COLOR} Setting up permissions on files and updating systemd settings ${NO_COLOR}"
chown -R $PRODUCT_USER:$PRODUCT_USER $PRODUCT_HOME_DIR/


echo -e "${RED_COLOR} Enabling systemd units ${NO_COLOR}"
service_units=$(find /lib/systemd/system/ -name "${PRODUCT_NAME}-*.service"  -printf "%f\n")
for u in $service_units; do
    systemctl enable $u
done
timer_units=$(find /lib/systemd/system/ -name "${PRODUCT_NAME}-*.timer"  -printf "%f\n")
for u in $timer_units; do
    systemctl enable $u
done

systemctl daemon-reload

echo "${RED_COLOR} Timestamping creation date ${NO_COLOR}"
mkdir /etc/krossboard/
date > /etc/krossboard/build_date

echo "${RED_COLOR} ==> DONE <== ${NO_COLOR}"
