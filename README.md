<p align="center">
  <img src="https://ipfs.io/ipfs/QmfRfm5EWe5hLT1XTS6ZURDo8Bg61z9RDzFiRYA1J9uY7J" width="360" alt="Hydra Booster Logo" />
</p>
<h1 align="center">Hydra Booster</h1>

<p align="center">
  <a href="http://protocol.ai"><img src="https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square" /></a>
  <a href="http://libp2p.io/"><img src="https://img.shields.io/badge/project-libp2p-yellow.svg?style=flat-square" /></a>
  <a href="http://webchat.freenode.net/?channels=%23libp2p"><img src="https://img.shields.io/badge/freenode-%23libp2p-yellow.svg?style=flat-square" /></a>
  <a href="https://discuss.libp2p.io"><img src="https://img.shields.io/discourse/https/discuss.libp2p.io/posts.svg?style=flat-square"/></a>
</p>

<p align="center">
  <a href="https://travis-ci.com/libp2p/hydra-booster"><img src="https://img.shields.io/travis/com/libp2p/hydra-booster/master?style=flat-square"></a>
  <a href="https://codecov.io/gh/libp2p/hydra-booster"><img src="https://img.shields.io/codecov/c/github/libp2p/hydra-booster?style=flat-square"></a>
  <a href="https://github.com/RichardLitt/standard-readme"><img src="https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square" /></a>
  <a href="https://godoc.org/github.com/libp2p/hydra-booster"><img src="http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square" /></a>
  <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.14.0-orange.svg?style=flat-square" /></a>
  <br>
</p>

> A DHT Indexer node & Peer Router

A new type of DHT node designed to accelerate the Content Resolution & Content Providing on the IPFS Network. A (cute) Hydra with one belly full of records and many heads (PeerIds) to tell other nodes about them, charged with rocket boosters to transport other nodes to their destination faster.

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

`hydra-booster` has two modes. A 'single dht' mode that has a nicer UI, this is intended to be run in a tmux window or something so you can see statistics about your contribution to the network.

```sh
go run ./main.go
```

The second mode is called 'many mode'. Passing the `-many=N` allows you to run N dhts at a time in the same process. It periodically prints out a status line with information about total peers, uptime, and memory usage.


```sh
go run ./main.go -many=5
```

### Best Practices

Only run a hydra-booster on machines with public IP addresses. Having more dht nodes behind NATs makes dht queries in general slower, as connecting in generally takes longer and sometimes doesnt even work (resulting in a timeout).

When running with `-many`, please make sure to bump the ulimit to something fairly high. Expect ~500 connections per node youre running (so with `-many=10`, try setting `ulimit -n 5000`)

## API

### HTTP API

By default the HTTP API is available at http://127.0.0.1:7779.

#### `GET /sybils`

Returns an ndjson list of peers created by the Hydra: their IDs and mulitaddrs. Example output:

```json
{"Addrs":["/ip4/127.0.0.1/tcp/50277","/ip4/192.168.0.3/tcp/50277"],"ID":"12D3KooWHacdCMnm4YKDJHn72HPTxc6LRGNzbrbyVEnuLFA3FXCZ"}
{"Addrs":["/ip4/127.0.0.1/tcp/50278","/ip4/192.168.0.3/tcp/50278","/ip4/90.198.150.147/tcp/50278"],"ID":"12D3KooWDGFCMQYpRHJ5BkVf842Fqnt3sCUAbvUw26ABuTo9Q1Gt"}
{"Addrs":["/ip4/127.0.0.1/tcp/50279","/ip4/192.168.0.3/tcp/50279","/ip4/90.198.150.147/tcp/50279"],"ID":"12D3KooWNYBmyyFmktyna9WPBT1UAgGLKqTJqbkZYmJF8fBKmMqd"}
{"Addrs":["/ip4/127.0.0.1/tcp/50280","/ip4/192.168.0.3/tcp/50280","/ip4/90.198.150.147/tcp/50280"],"ID":"12D3KooWQnUpnw6xS2VrJw3WuCP8e92fsEDnh4tbqyrXW5AVJ7oe"}
{"Addrs":["/ip4/127.0.0.1/tcp/50281","/ip4/192.168.0.3/tcp/50281"],"ID":"12D3KooWBmgW3i8vZaD49DDWJ3dRRb6KCG42UubpJDPHpzwKDXB9"}
```

## License

MIT - @whyrusleeping
