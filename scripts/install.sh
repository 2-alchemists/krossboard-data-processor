#!/bin/bash
# ------------------------------------------------------------------------ #
# File: install.sh                                                          #
# Creation: August 22, 2020                                                #
# Copyright (c) 2020 2Alchemists SAS                                       #
#                                                                          #
# This file is part of Krossboard (https://krossboard.app/).               #
#                                                                          #
# The tool is distributed in the hope that it will be useful,              #
# but WITHOUT ANY WARRANTY; without even the implied warranty of           #
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the            #
# Krossboard terms of use: https://krossboard.app/legal/terms-of-use/      #
#--------------------------------------------------------------------------#
set -e
RED_COLOR='\033[0;31m'
NO_COLOR='\033[0m'

if [ $# -gt 1 ]; then
    echo -e "${RED_COLOR}usage:\n\t $0 [CLOUD_PROVIDER]\n${NO_COLOR}"
    exit 1
fi
CLOUD_PROVIDER=${1:-AUTO}
set -u

if [ `id -u` -ne 0 ]; then
    echo -e "${RED_COLOR}the script must be run as root${NO_COLOR}"
    exit 1
fi

DISTRIB_DIR=$(dirname $0)
PRODUCT_NAME=krossboard
PRODUCT_BACKEND=krossboard-data-processor
DISTRIB_BINARY_PATH=$DISTRIB_DIR/$PRODUCT_BACKEND
PRODUCT_USER=krossboard
PRODUCT_HOME_DIR=/opt/$PRODUCT_USER
CONFIG_DIR=/opt/$PRODUCT_USER/etc
CONFIG_FILENAME=$PRODUCT_NAME.env
CONFIG_PATH=$CONFIG_DIR/$CONFIG_FILENAME

echo -e "${RED_COLOR}➥ Building cloud image => $CLOUD_PROVIDER ${NO_COLOR}"

echo -e "${RED_COLOR}➥ Updating apt-get source list and package versions... ${NO_COLOR}"
apt-get update && apt-get -y upgrade

echo -e "${RED_COLOR}➥ Installing Docker, rrdtool and librrd-dev... ${NO_COLOR}"
apt-get install -y docker.io rrdtool librrd-dev vim curl

echo -e "${RED_COLOR}➥ Installing ${PRODUCT_BACKEND} binary from $DISTRIB_BINARY_PATH... ${NO_COLOR}"
install -d $PRODUCT_HOME_DIR/{bin,data,etc}
install -m 755 $DISTRIB_BINARY_PATH $PRODUCT_HOME_DIR/bin/
install -m 644 $DISTRIB_DIR/scripts/$CONFIG_FILENAME $CONFIG_DIR/
install -m 644 $DISTRIB_DIR/scripts/${PRODUCT_NAME}-*.{service,timer} /lib/systemd/system/

echo -e "${RED_COLOR}➥ Setting up runtime user ${PRODUCT_USER}...${NO_COLOR}"
id -u $PRODUCT_USER &> /dev/null || useradd $PRODUCT_USER
usermod -d $PRODUCT_HOME_DIR -G docker $PRODUCT_USER

##############################################################
# INSTALLING AZ CLI                                          #
##############################################################
if wget --timeout=2 --tries=1 --header 'Metadata: true' -q "http://169.254.169.254/metadata/instance?api-version=2019-06-04" > /dev/null ; then
    echo -e "${RED_COLOR}➥ We're running on Azure cloud ${NO_COLOR}"
fi

echo -e "${RED_COLOR}➥ Installing az CLI.. ${NO_COLOR}"
curl -sL https://aka.ms/InstallAzureCLIDeb | bash
DIST_AZ_COMMAND=$(command -v az || echo "")
if [ "$DIST_AZ_COMMAND" != "" ]; then
    echo "DIST_AZ_COMMAND=$DIST_AZ_COMMAND" >> $CONFIG_PATH
    echo -e "${RED_COLOR}➥ ✓ az CLI found at $DIST_AZ_COMMAND${NO_COLOR}"
fi

##############################################################
# INSTALLING GCLOUD CLI                                      #
##############################################################
echo -e "${RED_COLOR}➥ Installing gcloud CLI... ${NO_COLOR}"
if wget --timeout=2 --tries=1 --header 'Metadata-Flavor: Google' -q "http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id" > /dev/null ; then
  echo -e "${RED_COLOR}➥ We're running on Google cloud ${NO_COLOR}"

  echo -e "${RED_COLOR}➥ Removing gcloud installed with snap that causes permission problems ${NO_COLOR}"
  snap remove google-cloud-sdk
fi

echo -e "${RED_COLOR}➥ Installing gcloud using Ubuntu packages ${NO_COLOR}"
echo "deb http://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
apt-get update
apt-get install -y google-cloud-sdk

DIST_GCLOUD_COMMAND=$(command -v gcloud || echo "")
if [ "$DIST_GCLOUD_COMMAND" != "" ]; then
    echo -e "${RED_COLOR}➥ ✓ gcloud CLI found at $DIST_GCLOUD_COMMAND${NO_COLOR}➥ "
    echo "DIST_GCLOUD_COMMAND=$DIST_GCLOUD_COMMAND" >> $CONFIG_PATH
fi

##############################################################
# INSTALLING AWS CLI                                         #
##############################################################
echo -e "${RED_COLOR}➥ Installing aws CLI... ${NO_COLOR}"
if wget --timeout=2 --tries=1 -q "http://169.254.169.254/latest/meta-data/placement/availability-zone" > /dev/null ; then
  echo -e "${RED_COLOR}➥ We're running on AWS cloud ${NO_COLOR}"
fi

apt-get -y install python3-pip && pip3 install --upgrade awscli
DIST_AWS_COMMAND=$(command -v aws || echo "")
if [ "$DIST_AWS_COMMAND" != "" ]; then
    echo -e "${RED_COLOR}➥ ✓ aws CLI found at $DIST_AWS_COMMAND ${NO_COLOR}"
    echo "DIST_AWS_COMMAND=$DIST_AWS_COMMAND" >> $CONFIG_PATH
fi

echo -e "${RED_COLOR}➥ Dumping configuration file ${NO_COLOR}"
cat $CONFIG_PATH

echo -e "${RED_COLOR}➥ Creating Caddy configuration file ${CONFIG_DIR}/Caddyfile ${NO_COLOR}"
cat <<EOF > ${CONFIG_DIR}/Caddyfile
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

echo -e "${RED_COLOR}➥ Installing script to change basicauth password ${NO_COLOR}"
install -m 755 $DISTRIB_DIR/scripts/krossboard-change-passwd.sh $PRODUCT_HOME_DIR/bin/krossboard-change-passwd

echo -e "${RED_COLOR}➥ Setting up permissions on files and updating systemd settings ${NO_COLOR}"
chown -R $PRODUCT_USER:$PRODUCT_USER $PRODUCT_HOME_DIR/


echo -e "${RED_COLOR}➥ Enabling systemd units ${NO_COLOR}"
service_units=$(find /lib/systemd/system/ -name "${PRODUCT_NAME}-*.service"  -printf "%f\n")
for u in $service_units; do
    systemctl enable $u
done

timer_units=$(find /lib/systemd/system/ -name "${PRODUCT_NAME}-*.timer"  -printf "%f\n")
for u in $timer_units; do
    systemctl enable $u
done

echo -e "${RED_COLOR}➥ Reloading systemd units ${NO_COLOR}"
systemctl daemon-reload

echo -e "${RED_COLOR}➥ Unpacking Docker images${NO_COLOR}"
source $CONFIG_PATH
docker load -i ./dimages.tgz

echo -e "${RED_COLOR}➥ Timestamping creation date ${NO_COLOR}"
mkdir /etc/krossboard/
date > /etc/krossboard/build_date

echo -e "${RED_COLOR}➥ ==> DONE <== ${NO_COLOR}"
