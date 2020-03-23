# Hydra Booster Deployment Notes

The Hydra nodes are deployed with Google Cloud Platform to the Kubernetes Engine (GKE).

It was setup using the web UI so reproducing or changing it will be _manual_ for the time being.

## Continuous deployment

We're using Cloud Build to build the container that will be deployed. It is setup to build one automatically (Github trigger) when a semver tag of the form `v[0-9]+\.[0-9]+\.[0-9]+` is created. Check "Cloud Build" => "History" and then "Build Artifcats" for a given build. See "Cloud Build" => "Triggers" to find the trigger.

Built containers are currently _not_ auto-deployed to the hydra cluster.

### Deploying a built image

1. Head to "Cloud Build" => "History" and find the build you want to deploy. Under "Build Artifcats", copy the image name.
1. Now go to "Kubernetes Engine" => "Workloads" and click on "hydra-booster-node-0"
1. Click "Actions" => "Rolling update" and paste the new build image name
1. Repeat for "hydra-booster-node-1" etc.

## Cluster setup

We have one _cluster_  in `us-central1-c` with a _deployment_ for each Hydra. Each deployment has one _pod_ and a `NodePort` service forwards external ports to internal ports on the pod.

We're using a `NodePort` not `LoadBalancer` service to expose sybils externally to avoid associated costs with forwarding many many ports. This [blog post](https://medium.com/google-cloud/kubernetes-nodeport-vs-loadbalancer-vs-ingress-when-should-i-use-what-922f010849e0) has some good info and diagrams on the differences between the different types of "services" that Kubernetes has.

TLDR; `NodePort` restricts you to exposing public services on ports in the range `30000-32767`. For people wanting to expose HTTP services on port `80` this is problematic but we don't care. We also do not need any actual load balancing to happen, we just need ports exposed publically.

Sybil swarm listening ports are allocated as such:

|         | Port range    |
| ------- | ------------- |
| Hydra 1 | `30000-30199` |
| Hydra 2 | `30200-30399` |
| Hydra 3 | `30400-30599` |
| ...     | ...           |

This gives us **up to 13 hydras and 2,600 sybils per cluster**. It assumes we can run up to 200 sybils on a single node. We may want to revist these allocations if the hardware is not capable.

Ports `32600-32767` are reserved for misc other services. We currently have 2 per hydra (httpapi and metrics).

Misc service ports are allocated as such:

|         | HTTP API port | Metrics port | ...     |
| ------- | ------------- | ------------ | ------- |
| Hydra 1 | `32600`       | `32601`      | `32602` |
| Hydra 2 | `32610`       | `32611`      | `32612` |
| Hydra 3 | `32620`       | `32621`      | `32621` |
| ...     | ...           | ...          | ...     |

This gives us **up to 10 misc service ports per hydra**.

There is one firewall rule ("VPC Network" => "Firewall rules") that opens up ports `30000-32767` (the ports that `NodePort` allows us to bind to).

We're currently running **2 Hydras** with **2 sybils** per Hydra.

## Metrics and reporting

Prometheus/Grafana server was deployed using a pre-bundled "application" from the "Applications" menu in GKE.

The service `prometheus-grafana-0-grafana` ("Kubernetes Engine" => "Services & Ingress") was switched from "ClusterIP" to "LoadBalancer" so that it can be accessed publically.

Grafana can be accessed at: http://35.184.172.3

### Prometheus config

The configuration `prometheus-grafana-0-prometheus-config` ("Kubernetes Engine" => "Configuration") has been updated to add the following hydra metrics endpoints:

```yaml
- job_name: 'hydrabooster'
  scrape_interval: 10s
  static_configs:
    - targets: ['10.8.5.79:8888', '10.8.15.102:8888']
```

It needs to be serialized yaml in yaml like this:

```yaml
- prometheus.yaml: "- \"job_name\": \"hydrabooster\"\n  \"scrape_interval\": \"10s\"\n  \"static_configs\":\n    - \"targets\": [\"10.8.5.79:8888\", \"10.8.15.102:8888\"]"
```

## Misc

I used the following script to generate the YAML config for the sybil ports:

```js
const begin = 30200
for (let i = 0; i < 200; i++) {
  console.log(`  - name: sybil-${i.toString().padStart(3, '0')}
    port: ${begin + i}
    nodePort: ${begin + i}
    protocol: TCP
    targetPort: ${begin + i}`)
}
```
