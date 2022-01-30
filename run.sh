#!/bin/bash
set -u
set -e
docker build . -t hydra-booster
docker run -it hydra-booster