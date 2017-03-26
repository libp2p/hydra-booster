# Libp2p DHT Node
Run a libp2p dht node for the good of the network!

## Installation
Currently, you must build from source. To do so, make sure you have go installed and run:
```
go get -u github.com/ipfs/dht-node
```

## Usage
`dht-node` has two modes. A 'single dht' mode that has a nicer UI, this is intended to be run in a tmux window or something so you can see statistics about your contribution to the network.

The second mode is called 'many mode'. Passing the `-many=N` allows you to run N dhts at a time in the same process. It periodically prints out a status line with information about total peers, uptime, and memory usage.

## Best Practices
Try to only run a dht-node on machines with public IP addresses. Having more
dht nodes behind NATs makes dht queries in general slower, as connecting in
generally takes longer and sometimes doesnt even work (resulting in a timeout).

When running with `-many`, please make sure to bump the ulimit to something
fairly high. Expect ~500 connections per node youre running (so with
`-many=10`, try setting `ulimit -n 5000`)

## License
MIT - @whyrusleeping
