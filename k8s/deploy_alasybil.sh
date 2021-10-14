#!/bin/bash
set -u
set -e
TAG=$(date +"%Y-%m-%d-%H%M%S")
docker build -t libp2p/hydra-booster:${TAG} ../
docker tag libp2p/hydra-booster:${TAG} registry.digitalocean.com/protocol/libp2p/hydra-booster:${TAG}
docker push registry.digitalocean.com/protocol/libp2p/hydra-booster:${TAG}
yq eval -i "select(di == 0).spec.template.spec.containers[0].image = \"registry.digitalocean.com/protocol/libp2p/hydra-booster:${TAG}\"" alasybil.yaml
kubectl apply -f alasybil.yaml
