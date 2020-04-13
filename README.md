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

The second mode is called 'many heads'. Passing the `-nsybils=N` allows you to run N heads (called [sybils](https://en.wikipedia.org/wiki/Sybil_attack)) at a time in the same process. It periodically prints out a status line with information about total peers, uptime, and memory usage.

```sh
go run ./main.go -nsybils=5
```

Alternatively you can use the `HYDRA_NSYBILS` environment var to specify the number of sybils. Note the `-nsybils` flag takes precedence.

### Best Practices

Only run a `hydra-booster` on machines with public IP addresses. Having more DHT nodes behind NATs makes DHT queries in general slower, as connecting in generally takes longer and sometimes doesnt even work (resulting in a timeout).

When running with `-nsybils`, please make sure to bump the ulimit to something fairly high. Expect ~500 connections per node youre running (so with `-nsybils=10`, try setting `ulimit -n 5000`)

## Developers

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
go run ./main.go -nsybils=5 -metrics-port=8888
```

You should now be able to access metrics at http://127.0.0.1:9090.

## API

### HTTP API

By default the HTTP API is available at http://127.0.0.1:7779.

#### `GET /sybils`

Returns an ndjson list of peers created by the Hydra: their IDs and mulitaddrs. Example output:

```json
{"Addrs":["/ip4/127.0.0.1/tcp/50277","/ip4/192.168.0.3/tcp/50277"],"ID":"12D3KooWHacdCMnm4YKDJHn72HPTxc6LRGNzbrbyVEnuLFA3FXCZ"}
{"Addrs":["/ip4/127.0.0.1/tcp/50280","/ip4/192.168.0.3/tcp/50280","/ip4/90.198.150.147/tcp/50280"],"ID":"12D3KooWQnUpnw6xS2VrJw3WuCP8e92fsEDnh4tbqyrXW5AVJ7oe"}
...
```

#### `GET /records/list`

Returns an ndjson list of records stored by the Hydra Booster node.

#### `GET /records/fetch`

Fetches a record available on the network by CID. `NOT IMPLEMENTED YET`

#### `POST /idgen/add`

Generate a balanced Peer ID for use by another Hydra Booster peer. Example output:

```json
"CAESQNcYNr0ENfml2IaiE97Kf3hGTqfB5k5W+C2/dW0o0sJ7b7zsvxWMedz64vKpS2USpXFBKKM9tWDmcc22n3FBnow="
```

## License

The hydra-booster project is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](./LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](./LICENSE-MIT) or http://opensource.org/licenses/MIT)
