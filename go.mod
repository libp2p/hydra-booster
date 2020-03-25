module github.com/libp2p/hydra-booster

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/alanshaw/prom-metrics-client v0.3.0
	github.com/axiomhq/hyperloglog v0.0.0-20191112132149-a4c4c47bc57f
	github.com/dustin/go-humanize v1.0.0
	github.com/golang/snappy v0.0.1 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/ipfs/go-cid v0.0.5
	github.com/ipfs/go-datastore v0.4.4
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-ipns v0.0.2
	github.com/libp2p/go-libp2p v0.5.3-0.20200307065934-e641f58681af
	github.com/libp2p/go-libp2p-circuit v0.1.4
	github.com/libp2p/go-libp2p-connmgr v0.2.1
	github.com/libp2p/go-libp2p-core v0.5.0
	github.com/libp2p/go-libp2p-kad-dht v0.4.2-0.20191230184437-fd2e9b7e3db2
	github.com/libp2p/go-libp2p-kbucket v0.2.3
	github.com/libp2p/go-libp2p-record v0.1.2
	github.com/multiformats/go-multiaddr v0.2.1
	github.com/prometheus/client_golang v1.5.1
	go.opencensus.io v0.22.3
)

go 1.14
