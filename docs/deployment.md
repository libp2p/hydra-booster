# Hydra Booster Deployment Notes

The Hydra nodes are deployed to DigitalOcean as a Kubenetes cluster. The [Kubenetes configuration files and instructions on applying the config](../k8s) are kept in this repo.

## Deployment config

There are the environment variables that can be tweaked to affect the deployment:

* `HYDRA_NAME` - a name for the Hydra that is used to more easily distinguish between Hydras in metrics
* `HYDRA_NHEADS` - controls the number of heads that are spawned by a Hydra
* `HYDRA_PORT_BEGIN` - controls the port that Hydra heads listen on. Each head is allocated a port sequentially beginning from the port specified here. See [Cluster Setup](#cluster-setup) below for what this value should be for each Hydra
* `HYDRA_IDGEN_ADDR` - the address of a Hydra HTTP API server that provides the `/idgen/*` endpoints. In the current deployment, Alasybil acts as the idgen server and the other nodes obtain their Peer IDs from it.

These environment variables are not yet used in production but will be soon (and possibly by the time you read this):

* `HYDRA_DB` - a PostgreSQL database connection string that can be shared by all Hydras in the swarm.
* `HYDRA_DISABLE_PROV_GC` - disables provider record garbage collection (when used in combination with `HYDRA_DB` it should be `true` on all but one Hydra).

## Cluster setup

We have one _cluster_  in DigitalOcean's `SFO2` region with a _deployment_ for each Hydra. Deployments have a application name picked from [petnames.net](http://www.petnames.net/unusual-pet-names.html) and live in the `hydra-boosters` namespace. Each deployment has _one pod_ and a `NodePort` service that forwards external ports to internal ports on the pod.

This [blog post](https://medium.com/google-cloud/kubernetes-nodeport-vs-loadbalancer-vs-ingress-when-should-i-use-what-922f010849e0) has some good info and diagrams on the differences between the different types of "services" that Kubernetes has.

TLDR; `NodePort` restricts you to exposing public services on ports in the range `30000-32767`. For people wanting to expose HTTP services on port `80` this is problematic but we don't care. We also do not need any actual load balancing to happen, we just need ports exposed publically.

Hydra head swarm listening ports are allocated as such:

|          | Port range    |
| -------- | ------------- |
| Alasybil | `30000-30199` |
| Bubbles  | `30200-30399` |
| Chumpy   | `30400-30599` |
| Domino   | `30600-30799` |
| Euclid   | `30800-30999` |
| ...      | ...           |

This gives us **up to 13 hydras and 2,600 heads per cluster**. It assumes we can run up to 200 heads on a single node. We may want to revist these allocations if the hardware is not capable.

Ports `32600-32767` are reserved for misc other services. We currently have 2 per hydra (httpapi and metrics).

Misc service ports are allocated as such:

|          | HTTP API port | Metrics port | ...     |
| -------- | ------------- | ------------ | ------- |
| Alasybil | `32600`       | `32601`      | `32602` |
| Bubbles  | `32610`       | `32611`      | `32612` |
| Chumpy   | `32620`       | `32621`      | `32622` |
| Domino   | `32630`       | `32631`      | `32632` |
| Euclid   | `32640`       | `32641`      | `32642` |
| ...      | ...           | ...          | ...     |

This gives us **up to 10 misc service ports per hydra**.

There is one firewall rule ("Networking" => "Firewalls") that opens up ports `30000-32767` (the ports that `NodePort` allows us to bind to).

We're currently running **5 Hydras** with the following head counts:

|          | Heads | 
| -------- | ----- | 
| Alasybil | `25`  |
| Bubbles  | `50`  |
| Chumpy   | `100` |
| Domino   | `150` |
| Euclid   | `200` |
| ...      | ...   |

## Metrics and reporting

Metrics are available at the [PL Grafana](https://protocollabs.grafana.net).

### Grafana Prometheus config

The Grafana Prometheus config you need is (substitute the `10.` IPs for the actual exposed load balancer IPs):

```yaml
- job_name: 'hydrabooster'
  scrape_interval: 10s
  static_configs:
    - targets: ['10.8.5.79:8888', '10.8.15.102:8888', '10.8.10.98:8888', '10.8.5.238:8888', '10.8.15.157:8888']
```

## Misc

I used the following script to generate the YAML config for the head ports:

```js
const begin = 30200
for (let i = 0; i < 200; i++) {
  console.log(`  - name: head-${i.toString().padStart(3, '0')}
    port: ${begin + i}
    nodePort: ${begin + i}
    protocol: TCP
    targetPort: ${begin + i}`)
}
```
