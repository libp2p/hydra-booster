module github.com/libp2p/hydra-booster

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/alanshaw/ipfs-hookds v0.2.0
	github.com/alanshaw/prom-metrics-client v0.3.0
	github.com/axiomhq/hyperloglog v0.0.0-20191112132149-a4c4c47bc57f
	github.com/dustin/go-humanize v1.0.0
	github.com/golang/snappy v0.0.1 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hnlq715/golang-lru v0.2.1-0.20200422024707-82ba7badf9a6
	github.com/ipfs/go-cid v0.0.5
	github.com/ipfs/go-datastore v0.4.4
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-ds-sql v0.1.1-0.20200423142616-ada9b9f97230
	github.com/ipfs/go-ipns v0.0.2
	github.com/libp2p/go-libp2p v0.8.2
	github.com/libp2p/go-libp2p-circuit v0.2.2
	github.com/libp2p/go-libp2p-connmgr v0.2.1
	github.com/libp2p/go-libp2p-core v0.5.2
	github.com/libp2p/go-libp2p-kad-dht v0.7.10
	github.com/libp2p/go-libp2p-kbucket v0.4.1
	github.com/libp2p/go-libp2p-mplex v0.2.3
	github.com/libp2p/go-libp2p-quic-transport v0.3.5
	github.com/libp2p/go-libp2p-record v0.1.2
	github.com/libp2p/go-mplex v0.1.3-0.20200424022829-dabf4b3e113f // indirect
	github.com/libp2p/go-tcp-transport v0.2.0
	github.com/multiformats/go-base32 v0.0.3
	github.com/multiformats/go-multiaddr v0.2.1
	github.com/multiformats/go-multihash v0.0.13
	github.com/prometheus/client_golang v1.5.1
	github.com/whyrusleeping/timecache v0.0.0-20160911033111-cfcb2f1abfee
	go.opencensus.io v0.22.3
)

go 1.13
