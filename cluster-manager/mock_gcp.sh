#!/bin/bash

mkdir -p ./computeMetadata/v1/project
echo -n 60213770375 > computeMetadata/v1/project/numeric-project-id

python3 -m http.server

