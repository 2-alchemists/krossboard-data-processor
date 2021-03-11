#!/bin/bash
# ------------------------------------------------------------------------ #
# File: update.sh                                                          #
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
set -u

RED_COLOR='\033[0;31m'
NO_COLOR='\033[0m'

DISTRIB_DIR=$(dirname $0)
PRODUCT_NAME=krossboard
PRODUCT_USER=krossboard
PRODUCT_BACKEND=krossboard-data-processor
DISTRIB_BINARY_PATH=${1-$DISTRIB_DIR/$PRODUCT_BACKEND}
PRODUCT_HOME_DIR=/opt/$PRODUCT_USER
CONFIG_DIR=/opt/$PRODUCT_USER/etc
CONFIG_FILENAME=$PRODUCT_NAME.env
CONFIG_PATH=$CONFIG_DIR/$CONFIG_FILENAME

echo -e "${RED_COLOR}➥ Installing ${PRODUCT_BACKEND} binary from $DISTRIB_BINARY_PATH...${NO_COLOR}"
install -m 755 $DISTRIB_BINARY_PATH $PRODUCT_HOME_DIR/bin/

echo -e "${RED_COLOR}➥ Updating configuration file ...${NO_COLOR}"
install -m 644 $DISTRIB_DIR/scripts/$CONFIG_FILENAME $CONFIG_DIR/

echo -e "${RED_COLOR}➥ Enabling systemd units${NO_COLOR}"
service_units=$(find /lib/systemd/system/ -name "${PRODUCT_NAME}-*.service"  -printf "%f\n")
for u in $service_units; do
    systemctl enable $u
done
timer_units=$(find /lib/systemd/system/ -name "${PRODUCT_NAME}-*.timer"  -printf "%f\n")
for u in $timer_units; do
    systemctl enable $u
done
systemctl daemon-reload

echo -e "${RED_COLOR}➥ Unpacking Docker images${NO_COLOR}"
source $CONFIG_PATH
docker load -i ./dimages.tgz

systemctl restart krossboard-data-processor-api
systemctl restart krossboard-ui

echo "done"
