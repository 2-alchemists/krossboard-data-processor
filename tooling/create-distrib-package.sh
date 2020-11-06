#!/bin/bash

set -e
set -u

PACKAGE_BUILD_ARTIFACT=$1
RELEASE_DIST_PACKAGE=$2
KROSSBOARD_KOAINSTANCE_IMAGE=$3
KROSSBOARD_UI_IMAGE=$4

mkdir -p ${RELEASE_DIST_PACKAGE}/scripts/
cp ${PACKAGE_BUILD_ARTIFACT} ${RELEASE_DIST_PACKAGE}/
cp ./scripts/krossboard* ${RELEASE_DIST_PACKAGE}/scripts/
			"sed -i 's|krossboard_koainstance_image|${KROSSBOARD_KOAINSTANCE_IMAGE}|' ${RELEASE_DIST_PACKAGE}/scripts/krossboard.env",
			"sed -i 's|krossboard_ui_image|${KROSSBOARD_UI_IMAGE}|' ${RELEASE_DIST_PACKAGE}/scripts/krossboard.env",	
cp EULA INSTALLATION_NOTICE ${RELEASE_DIST_PACKAGE}/
install -m 755 ./scripts/install.sh ${RELEASE_DIST_PACKAGE}/
tar zcf ${RELEASE_DIST_PACKAGE}.tgz ${RELEASE_DIST_PACKAGE}
rm -rf ${RELEASE_DIST_PACKAGE}/
