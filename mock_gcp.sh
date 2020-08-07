#!/bin/bash

mkdir -p ./computeMetadata/v1/project
echo -n 10862655325 > computeMetadata/v1/project/numeric-project-id

python3 -m http.server

