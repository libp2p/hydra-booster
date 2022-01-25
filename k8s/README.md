# Example Kubenetes Configuration

These are the configuration files used to deploy the Hydra Booster nodes that operate on the IPFS network.

## Deploying to DigitalOcean

First create a cluster with some machines. A 50 Head Hydra requires about `12Gi` of RAM.

Next install [`doctl`](https://github.com/digitalocean/doctl) and [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and run the following commands to deploy Hydras:

```sh
# Get k8s config and set it as the current context
doctl kubernetes cluster kubeconfig save <your_cluster_name>
# Create the namespace that hydras run in
kubectl create -f k8s/namespace.yaml
kubectl apply -f k8s/alasybil.yaml
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

The config uses the latest `libp2p/hydra-booster:master` image, so if you've tagged and pushed a new version all you need to do is scale down and up each deployment:

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

## Deploying a branch

1. Publish a new tagged image to dockerhub. e.g. we use `libp2p/hydra-booster:next` for smoke testing upcoming releases. Add the tag to `docker commit` and `docker push` when [publishing](https://github.com/libp2p/hydra-booster#publish-a-new-image).
2. Update the `image:` property in the [deployment spec](https://github.com/libp2p/hydra-booster/blob/30b2924b519aeee8f3ff6c3e87e1215ea65e81ad/k8s/alasybil.yaml#L38) for the hydra(s) you want to use the image.
3. Apply the updated config to the hydra(s) using `kubectl apply -f k8s/HYDRA_NAME.yaml`
