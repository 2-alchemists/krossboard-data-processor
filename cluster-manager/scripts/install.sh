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

PROGRAM_NAME=koamc-cluster-manager
INSTALL_DIR=$(dirname $0)
KOAMC_BINARY_PATH=${1-$INSTALL_DIR/$PROGRAM_NAME}
KOAMC_USER=koamc
KOAMC_ROOT_DIR=/opt/$KOAMC_USER
DOCKER_GROUP=docker

echo "Binary to be installed: $KOAMC_BINARY_PATH"

id -u $KOAMC_USER &> /dev/null || \
     sudo useradd $KOAMC_USER -G $DOCKER_GROUP -m --home-dir $KOAMC_ROOT_DIR

install -d $KOAMC_ROOT_DIR/{bin,data,etc}
install -m 755 $KOAMC_BINARY_PATH $KOAMC_ROOT_DIR/bin/
install -m 644 $INSTALL_DIR/scripts/$PROGRAM_NAME.service.env $KOAMC_ROOT_DIR/etc/
install -m 644 $INSTALL_DIR/scripts/$PROGRAM_NAME.service /lib/systemd/system/

KOAMC_GCLOUD_COMMAND=$(which gcloud || echo "")
if [ "$KOAMC_GCLOUD_COMMAND" != "" ]; then
    echo "gcloud found at $KOAMC_GCLOUD_COMMAND"
    echo "KOAMC_GCLOUD_COMMAND=$KOAMC_GCLOUD_COMMAND" >> $KOAMC_ROOT_DIR/etc/$PROGRAM_NAME.service.env 
fi

KOAMC_AWS_COMMAND=$(which aws || echo "")
if [ "$KOAMC_AWS_COMMAND" != "" ]; then
    echo "aws found at $KOAMC_AWS_COMMAND"
    echo "KOAMC_AWS_COMMAND=$KOAMC_AWS_COMMAND" >> $KOAMC_ROOT_DIR/etc/$PROGRAM_NAME.service.env 
fi

KOAMC_AZURE_COMMAND=$(which az || echo "")
if [ "$KOAMC_AZURE_COMMAND" != "" ]; then
    echo "az found at $KOAMC_AZURE_COMMAND"
    echo "KOAMC_AZURE_COMMAND=$KOAMC_AZURE_COMMAND" >> $KOAMC_ROOT_DIR/etc/$PROGRAM_NAME.service.env 
fi

chown -R $KOAMC_USER:$KOAMC_USER $KOAMC_ROOT_DIR/
systemctl enable $PROGRAM_NAME
systemctl daemon-reload
echo "done"