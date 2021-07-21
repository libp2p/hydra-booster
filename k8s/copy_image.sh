#!/bin/bash
docker pull docker.io/libp2p/hydra-booster
docker tag docker.io/libp2p/hydra-booster registry.digitalocean.com/protocol/libp2p/hydra-booster
docker push registry.digitalocean.com/protocol/libp2p/hydra-booster
