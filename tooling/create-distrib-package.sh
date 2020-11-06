#!/bin/bash

set -e
set -u

RED_COLOR='\033[0;31m'
NO_COLOR='\033[0m'

PROGRAM_ARTIFACT=$1
RELEASE_DIST_PACKAGE=$2
KROSSBOARD_KOAINSTANCE_IMAGE=$3
KROSSBOARD_UI_IMAGE=$4

mkdir -p ${RELEASE_DIST_PACKAGE}/scripts/
cp ${PROGRAM_ARTIFACT} ${RELEASE_DIST_PACKAGE}/
cp ./scripts/krossboard* ${RELEASE_DIST_PACKAGE}/scripts/
CONFIG_FILE="${RELEASE_DIST_PACKAGE}/scripts/krossboard.env"
sed -i 's|krossboard_koainstance_image|'${KROSSBOARD_KOAINSTANCE_IMAGE}'|' ${CONFIG_FILE}
sed -i 's|krossboard_ui_image|'${KROSSBOARD_UI_IMAGE}'|' ${CONFIG_FILE}

echo -e "${RED_COLOR} === Generated config file === "
cat ${CONFIG_FILE}
echo -e "===============================${NO_COLOR}"

install -m 755 ./scripts/install.sh ${RELEASE_DIST_PACKAGE}/
install -m 755 ./scripts/update.sh ${RELEASE_DIST_PACKAGE}/
cp EULA INSTALLATION_NOTICE ${RELEASE_DIST_PACKAGE}/
tar zcf ${RELEASE_DIST_PACKAGE}.tgz ${RELEASE_DIST_PACKAGE}
rm -rf ${RELEASE_DIST_PACKAGE}/


echo " Release completed => ${RELEASE_DIST_PACKAGE}.tgz"
