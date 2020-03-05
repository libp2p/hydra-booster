<p align="center">
  <img src="https://ipfs.io/ipfs/QmfRfm5EWe5hLT1XTS6ZURDo8Bg61z9RDzFiRYA1J9uY7J" width="360" />
</p>
<h1 align="center">Hydra Booster</h1>

> A DHT Indexer node & Peer Router

A new type of DHT node designed to accelerate the Content Resolution & Content Providing on the IPFS Network. A (cute) Hydra with one belly full of records and many heads (PeerIds) to tell other nodes about them, charged with rocket boosters to transport other nodes to their destination faster.

[**Read the RFC**](https://docs.google.com/document/d/1yA2fY5c0WIv3LCtJCPVesHzvCWt14OPv7QlHdV3ghgU)

## Installation

```
[openssl support (lower CPU usage)]
go get -u -tags=openssl github.com/libp2p/hydra-booster

[standard (sub-optimal)]
go get -u github.com/libp2p/hydra-booster
```

## Usage

`hydra-booster` has two modes. A 'single dht' mode that has a nicer UI, this is intended to be run in a tmux window or something so you can see statistics about your contribution to the network.

The second mode is called 'many mode'. Passing the `-many=N` allows you to run N dhts at a time in the same process. It periodically prints out a status line with information about total peers, uptime, and memory usage.

## Best Practices

Only run a hydra-booster on machines with public IP addresses. Having more dht nodes behind NATs makes dht queries in general slower, as connecting in generally takes longer and sometimes doesnt even work (resulting in a timeout).

When running with `-many`, please make sure to bump the ulimit to something fairly high. Expect ~500 connections per node youre running (so with `-many=10`, try setting `ulimit -n 5000`)

## License

MIT - @whyrusleeping
