#!/bin/bash

set -u
set -e

GITHUB_TAG=$1
release_package=$2

GITHUB_USER="2-alchemists"
GITHUB_REPOSITORY_NAME="krossboard"
gh_release_cmd_upstream=linux-amd64-github-release
gh_release_cmd=github-release

if ! command -v ${gh_release_cmd} &> /dev/null; then
  echo "\e[31m${gh_release_cmd} not found, installing it...\e[0m"
  gh_release_cmd_upstream_bz2=${gh_release_cmd_upstream}.bz2
  gh_release_version=v0.9.0
  wget -O ${gh_release_cmd_upstream_bz2} \
        https://github.com/github-release/github-release/releases/download/${gh_release_version}/${gh_release_cmd_upstream_bz2}
  bunzip2 ${gh_release_cmd_upstream_bz2}
  chmod +x ./${gh_release_cmd_upstream}
  sudo mv ./${gh_release_cmd_upstream} /usr/local/bin/${gh_release_cmd}
fi

echo "==> Creating a release => $GITHUB_USER/${GITHUB_REPOSITORY_NAME}:${GITHUB_TAG}"
github-release release \
    --user $GITHUB_USER \
    --repo "${GITHUB_REPOSITORY_NAME}" \
    --tag ${GITHUB_TAG} \
    --name "Release ${GITHUB_TAG}" \
    --description "Release notes: https://krossboard.app/docs/releases/" \
    --pre-release

echo "==> Uploading artifacts..."

binary_artifact="${release_package}.tgz"
echo "    artifact => ${binary_artifact}"
${gh_release_cmd} upload \
    --user $GITHUB_USER \
    --repo "${GITHUB_REPOSITORY_NAME}" \
    --tag ${GITHUB_TAG} \
    --name "${binary_artifact}" \
    --file ./${binary_artifact}

ovf_dir="${release_package}-ovf-vmdk"
cp EULA INSTALLATION_NOTICE ${ovf_dir}/
ls -l ${ovf_dir}/
ovf_artifact=${ovf_dir}.zip
zip -r ${ovf_artifact} ${ovf_dir}
echo "    artifact => ${ovf_artifact}"
${gh_release_cmd} upload \
    --user $GITHUB_USER \
    --repo "${GITHUB_REPOSITORY_NAME}" \
    --tag ${GITHUB_TAG} \
    --name "${ovf_dir}" \
    --file ./${ovf_artifact}