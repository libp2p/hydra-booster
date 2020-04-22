# Hydra Boosters Kubenetes Configuration

These are the configuration files used to deploy the Hydra Booster nodes that operate on the IPFS network.

## Deploying to DigitalOcean

First create a cluster with some machines. A 250 Head Hydra requires about `12Gi` of RAM.

Next install [`doctl`](https://github.com/digitalocean/doctl) and [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and run the following commands to deploy Hydras:

```sh
# Get k8s config and set it as the current context
doctl kubernetes cluster kubeconfig save <your_cluster_name>
# Create the namespace that hydras run in
kubectl create -f k8s/namespace.yaml
# Create Alasybil first.
# Alasybil is the idgen server, you'll need to get the cluster IP address for
# alasybil-nodeport-service and use it in HYDRA_IDGEN_ADDR env var for the
# other Hydras in the cluster.
kubectl apply -f k8s/alasybil.yaml
# Create the other Hydra nodes
kubectl apply -f k8s/bubbles.yaml
kubectl apply -f k8s/chumpy.yaml
kubectl apply -f k8s/domino.yaml
kubectl apply -f k8s/euclid.yaml
kubectl apply -f k8s/flake.yaml
kubectl apply -f k8s/grendel.yaml
kubectl apply -f k8s/hojo.yaml
kubectl apply -f k8s/ibycus.yaml
kubectl apply -f k8s/jetta.yaml
```

## Updating a deployment

The config uses the latest `libp2p/hydra-booster` image, so if you've tagged an pushed a new version all you need to do is scale down and up each deployment:

```sh
# Scale down all deployments
kubectl scale deployment/alasybil-deployment --replicas=0 -n hydra-boosters
kubectl scale deployment/bubbles-deployment --replicas=0 -n hydra-boosters
kubectl scale deployment/chumpy-deployment --replicas=0 -n hydra-boosters
kubectl scale deployment/domino-deployment --replicas=0 -n hydra-boosters
kubectl scale deployment/euclid-deployment --replicas=0 -n hydra-boosters
kubectl scale deployment/flake-deployment --replicas=0 -n hydra-boosters
kubectl scale deployment/grendel-deployment --replicas=0 -n hydra-boosters
kubectl scale deployment/hojo-deployment --replicas=0 -n hydra-boosters
kubectl scale deployment/ibycus-deployment --replicas=0 -n hydra-boosters
kubectl scale deployment/jetta-deployment --replicas=0 -n hydra-boosters

# Scale up all deployments
kubectl scale deployment/alasybil-deployment --replicas=1 -n hydra-boosters
# Pause for Alasybil to scale up (it's the idgen server)
kubectl scale deployment/bubbles-deployment --replicas=1 -n hydra-boosters
kubectl scale deployment/chumpy-deployment --replicas=1 -n hydra-boosters
kubectl scale deployment/domino-deployment --replicas=1 -n hydra-boosters
kubectl scale deployment/euclid-deployment --replicas=1 -n hydra-boosters
kubectl scale deployment/flake-deployment --replicas=1 -n hydra-boosters
kubectl scale deployment/grendel-deployment --replicas=1 -n hydra-boosters
kubectl scale deployment/hojo-deployment --replicas=1 -n hydra-boosters
kubectl scale deployment/ibycus-deployment --replicas=1 -n hydra-boosters
kubectl scale deployment/jetta-deployment --replicas=1 -n hydra-boosters
```