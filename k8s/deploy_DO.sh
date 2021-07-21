#!/bin/bash
set -u
set -e
docker build -t libp2p/hydra-booster:latest ../
docker tag libp2p/hydra-booster:latest registry.digitalocean.com/protocol/libp2p/hydra-booster:latest
docker push registry.digitalocean.com/protocol/libp2p/hydra-booster:latest
