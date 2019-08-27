#!/bin/bash

set -u
set -e

if [ $# -gt 1 ]; then
    echo -e "usage:\n\t $0 [/PATH/TO/BINARY]\n"
    exit 1
fi


if [ `id -u` -ne 0 ]; then
    echo "the script must be run as root"
    exit 1
fi

PROGRAM_NAME=kube-opex-analytics-mc
INSTALL_ASSET_DIR=$(dirname $0)
KOAMC_BINARY_PATH=${1-$INSTALL_ASSET_DIR/$PROGRAM_NAME}
KOAMC_USER=koamc
KOAMC_ROOT_DIR=/opt/$KOAMC_USER
DOCKER_GROUP=docker

echo "Binary to be installed: $KOAMC_BINARY_PATH"

id -u $KOAMC_USER &> /dev/null || \
     sudo useradd $KOAMC_USER -G $DOCKER_GROUP -m --home-dir $KOAMC_ROOT_DIR

install -d $KOAMC_ROOT_DIR/{bin,data,etc}
install -m 755 $KOAMC_BINARY_PATH $KOAMC_ROOT_DIR/bin/
install -m 644 $INSTALL_ASSET_DIR/$PROGRAM_NAME.service.env $KOAMC_ROOT_DIR/etc/
install -m 644 $INSTALL_ASSET_DIR/$PROGRAM_NAME.service /lib/systemd/system/

GOOGLE_GCLOUD_COMMAND_PATH=$(which gcloud || echo "")
if [ "$GOOGLE_GCLOUD_COMMAND_PATH" != "" ]; then
    echo -e "\nGOOGLE_GCLOUD_COMMAND_PATH=$GOOGLE_GCLOUD_COMMAND_PATH\n" >> $KOAMC_ROOT_DIR/etc/$PROGRAM_NAME.service.env 
fi

chown -R $KOAMC_USER:$KOAMC_USER $KOAMC_ROOT_DIR/
systemctl enable $PROGRAM_NAME
systemctl daemon-reload
echo "done"