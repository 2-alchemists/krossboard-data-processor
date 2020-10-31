#!/bin/bash

set -u
set -e

GIT_LATEST_TAG=$(git describe --tags `git rev-list --tags --max-count=1` || echo "")
GIT_LATEST_SHA=$(git rev-parse --short HEAD)
if [ -z "$GIT_LATEST_TAG" ]; then
    KB_RELEASE_VERSION=$GIT_LATEST_SHA
else
    MATCHED_TAG=$(git describe --exact-match $GIT_LATEST_SHA || echo "TAG_NOT_MATCHED")
    KB_RELEASE_VERSION=$(echo $GIT_LATEST_TAG | sed 's/v//')
    if [ "$MATCHED_TAG" != "$GIT_LATEST_TAG" ]; then
    KB_RELEASE_VERSION="${KB_RELEASE_VERSION}-${GIT_LATEST_SHA}"
    fi
fi

echo $KB_RELEASE_VERSION