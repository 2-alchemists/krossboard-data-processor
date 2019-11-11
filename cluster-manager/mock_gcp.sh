#!/bin/bash

mkddir -p ./computeMetadata/v1/project
echo -n 10213780375 > computeMetadata/v1/project/numeric-project-id

python3 -m http.server

