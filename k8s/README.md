# Hydra Boosters Kubenetes Configuration

These are the configuration files used to deploy the Hydra Booster nodes that operate on the IPFS network.

## Deploying to DigitalOcean

First create a cluster with some machines. A 200 Head Hydra requires about `15Gi` of RAM.

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
```
