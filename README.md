<p align="center">
  <img src="https://ipfs.io/ipfs/QmfRfm5EWe5hLT1XTS6ZURDo8Bg61z9RDzFiRYA1J9uY7J" width="360" alt="Hydra Booster Logo" />
</p>
<h1 align="center">Hydra Booster</h1>

<p align="center">
  <a href="http://protocol.ai"><img src="https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg" /></a>
  <a href="http://libp2p.io/"><img src="https://img.shields.io/badge/project-libp2p-yellow.svg" /></a>
  <a href="http://webchat.freenode.net/?channels=%23libp2p"><img src="https://img.shields.io/badge/freenode-%23libp2p-yellow.svg" /></a>
  <a href="https://discuss.libp2p.io"><img src="https://img.shields.io/discourse/https/discuss.libp2p.io/posts.svg"/></a>
</p>

<p align="center">
  <a href="https://travis-ci.com/libp2p/hydra-booster"><img src="https://travis-ci.com/libp2p/hydra-booster.svg?branch=master"></a>
  <a href="https://codecov.io/gh/libp2p/hydra-booster"><img src="https://codecov.io/gh/libp2p/hydra-booster/branch/master/graph/badge.svg"></a>
  <a href="https://goreportcard.com/report/github.com/libp2p/hydra-booster"><img src="https://goreportcard.com/badge/github.com/libp2p/hydra-booster" /></a>  
  <a href="https://github.com/RichardLitt/standard-readme"><img src="https://img.shields.io/badge/readme%20style-standard-brightgreen.svg" /></a>
  <a href="https://godoc.org/github.com/libp2p/hydra-booster"><img src="http://img.shields.io/badge/godoc-reference-5272B4.svg" /></a>
  <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.13.8-orange.svg" /></a>
  <br>
</p>

> A DHT Indexer node & Peer Router

A new type of DHT node designed to accelerate the Content Resolution & Content Providing on the IPFS Network. A (cute) Hydra with one belly full of records and many heads (Peer IDs) to tell other nodes about them, charged with rocket boosters to transport other nodes to their destination faster.

[**Read the RFC**](https://docs.google.com/document/d/1yA2fY5c0WIv3LCtJCPVesHzvCWt14OPv7QlHdV3ghgU).
Disclaimer: We are at Stage 1 of the RFC. [**Kanban**](https://app.zenhub.com/workspaces/hydra-booster-5e64ef0d1fa19e698b659cec/board?repos=245123455)

## Install

```
[openssl support (lower CPU usage)]
go get -u -tags=openssl github.com/libp2p/hydra-booster

[standard (sub-optimal)]
go get -u github.com/libp2p/hydra-booster
```

## Usage

`hydra-booster` has two modes. A 'single head' mode that has a nicer UI, this is intended to be run in a tmux window or something so you can see statistics about your contribution to the network.

```sh
go run ./main.go
```

The second mode is called 'many heads'. Passing the `-nheads=N` allows you to run N heads at a time in the same process. It periodically prints out a status line with information about total peers, uptime, and memory usage.

```sh
go run ./main.go -nheads=5
```

Alternatively you can use the `HYDRA_NHEADS` environment var to specify the number of heads. Note the `-nheads` flag takes precedence.

### Flags

```console
Usage of hydra-booster:
  -bootstrap-conc int
        How many concurrent bootstraps to run (default 32)
  -bucket-size int
        Specify the bucket size, note that for some protocols this must be a specific value i.e. for "/ipfs" it MUST be 20 (default 20)
  -db string
        Datastore directory (for LevelDB store) or postgresql:// connection URI (for PostgreSQL store)
  -disable-prov-gc
        Disable provider record garbage collection (default false).
  -disable-providers
        Disable storing and retrieving provider records, note that for some protocols, like "/ipfs", it MUST be false (default false).
  -disable-values
        Disable storing and retrieving value records, note that for some protocols, like "/ipfs", it MUST be false (default false).
  -enable-relay
        Enable libp2p circuit relaying for this node
  -httpapi-addr string
        Specify an IP and port to run prometheus metrics and pprof http server on (default "127.0.0.1:7779")
  -idgen-addr string
        Address of an idgen HTTP API endpoint to use for generating private keys for heads
  -mem
        Use an in-memory database. This overrides the -db option
  -metrics-addr string
        Specify an IP and port to run prometheus metrics and pprof http server on (default "0.0.0.0:8888")
  -name string
        A name for the Hydra (for use in metrics)
  -nheads int
        Specify the number of Hydra heads to create. (default -1)
  -port-begin int
        If set, begin port allocation here (default -1)
  -protocol-prefix string
        Specify the DHT protocol prefix (default "/ipfs") (default "/ipfs")
  -stagger duration
        Duration to stagger nodes starts by
  -ui-theme string
        UI theme, "gooey", "logey" or "none" (default "gooey" for 1 head otherwise "logey")
```

### Environment variables

Alternatively, some flags can be set via environment variables. Note that flags take precedence over environment variables.

```console
Usage of hydra-booster:
  HYDRA_DB string
        Datastore directory (for LevelDB store) or postgresql:// connection URI (for PostgreSQL store)
  HYDRA_DISABLE_PROV_GC
        Disable provider record garbage collection (default false).
  HYDRA_IDGEN_ADDR string
        Address of an idgen HTTP API endpoint to use for generating private keys for heads
  HYDRA_NAME string
        A name for the Hydra (for use in metrics)
  HYDRA_NHEADS int
        Specify the number of Hydra heads to create. (default -1)
  HYDRA_PORT_BEGIN int
        If set, begin port allocation here (default -1)
```

### Best Practices

Only run a `hydra-booster` on machines with public IP addresses. Having more DHT nodes behind NATs makes DHT queries in general slower, as connecting in generally takes longer and sometimes doesnt even work (resulting in a timeout).

When running with `-nheads`, please make sure to bump the ulimit to something fairly high. Expect ~500 connections per node you're running (so with `-nheads=10`, try setting `ulimit -n 5000`)

### Running Multiple Hydras

The total number of heads a single Hydra can have depends on the resources of the machine it's running on. To get the desired number of heads you may need to run multiple Hydras on multiple machines. There's a couple of challenges with this:

* Peer IDs of Hydra heads are balanced in the DHT. When running multiple Hydras it's necessary to designate one of the Hydras to be the "idgen server" and the rest to be "idgen clients" so that all Peer IDs in the Hydra swarm are balanced. Use the `-idgen-addr` flag or `HYDRA_IDGEN_ADDR` environment variable to ensure all Peer IDs in the Hydra swarm are balanced perfectly.
* A datastore is shared by all Hydra heads but not by all Hydras. Use the `-db` flag or `HYDRA_DB` environment variable to specify a PostgreSQL database connection string that can be shared by all Hydras in the swarm.
* When sharing a datastore between multiple _Hydras_ ensure only one Hydra in the swarm is performing GC on provider records by using the `-disable-prov-gc` flag or `HYDRA_DISABLE_PROV_GC` environment variable.

## Developers

### Release a new version

1. Update version number in [`version.go`](version/version.go).
2. Create a semver tag with "v" prefix e.g. `git tag v0.1.7`.
3. See [`deployment.md#continuous-deployment`](docs/deployment.md#continuous-deployment) for what happens next.

### Publish a new image

```console
# Build your container
docker build -t hydra-booster .

# Get it to run
docker run hydra-booster

# Commit new version
docker commit -m="some commit message" <CONTAINER_ID> libp2p/hydra-booster

# Push to docker hub (must be logged in, do docker login)
docker push libp2p/hydra-booster
```

### Metrics collection with Prometheus

Install [Prometheus](https://prometheus.io/) and then start it using the provided config:

```console
prometheus --config.file=promconfig.yaml --storage.tsdb.path=prometheus-data
```

Next start the Hydra Booster, specifying the port to run metrics on:

```console
go run ./main.go -nheads=5 -metrics-port=8888
```

You should now be able to access metrics at http://127.0.0.1:9090.

## API

### HTTP API

By default the HTTP API is available at http://127.0.0.1:7779.

#### `GET /heads`

Returns an ndjson list of peers created by the Hydra: their IDs and mulitaddrs. Example output:

```json
{"Addrs":["/ip4/127.0.0.1/tcp/50277","/ip4/192.168.0.3/tcp/50277"],"ID":"12D3KooWHacdCMnm4YKDJHn72HPTxc6LRGNzbrbyVEnuLFA3FXCZ"}
{"Addrs":["/ip4/127.0.0.1/tcp/50280","/ip4/192.168.0.3/tcp/50280","/ip4/90.198.150.147/tcp/50280"],"ID":"12D3KooWQnUpnw6xS2VrJw3WuCP8e92fsEDnh4tbqyrXW5AVJ7oe"}
...
```

#### `GET /records/list`

Returns an ndjson list of provider records stored by the Hydra Booster node.

#### `GET /records/fetch/{cid}?nProviders=1`

Fetches provider record(s) available on the network by CID. Use the `nProviders` query string parameter to signal the number of provider records to find. Returns an ndjson list of provider peers: their IDs and mulitaddrs. Will return HTTP status code 404 if no records were found.

#### `POST /idgen/add`

Generate and add a balanced Peer ID to the server's xor trie and return it for use by another Hydra Booster peer. Returns a base64 encoded JSON string. Example output:

```json
"CAESQNcYNr0ENfml2IaiE97Kf3hGTqfB5k5W+C2/dW0o0sJ7b7zsvxWMedz64vKpS2USpXFBKKM9tWDmcc22n3FBnow="
```

#### `POST /idgen/remove`

Remove a balanced Peer ID from the server's xor trie. Accepts a base64 encoded JSON string.

## License

The hydra-booster project is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](./LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](./LICENSE-MIT) or http://opensource.org/licenses/MIT)
