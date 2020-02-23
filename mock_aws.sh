#!/bin/bash

MOCK_DIR=./latest/meta-data/placement
mkdir -p $MOCK_DIR
echo -n 'eu-central-1b' > $MOCK_DIR/availability-zone

python3 -m http.server

