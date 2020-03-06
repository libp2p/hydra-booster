package hydrabooster

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	ds "github.com/ipfs/go-datastore"
	ipns "github.com/ipfs/go-ipns"
	libp2p "github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	record "github.com/libp2p/go-libp2p-record"
)

func randBootstrapAddr() (*peer.AddrInfo, error) {
	addr := dht.DefaultBootstrapPeers[rand.Intn(len(dht.DefaultBootstrapPeers))]
	return peer.AddrInfoFromP2pAddr(addr)
}

// BootstrapStatus describes the status of connecting to a bootstrap node
type BootstrapStatus struct {
	done bool
	err  error
}

// SpawnNodeOptions func options
type SpawnNodeOptions struct {
	datastore  ds.Batching
	relay      bool
	addr       string
	bucketSize int
	limiter    chan struct{}
}

// SpawnNode ...
func SpawnNode(opts SpawnNodeOptions) (host.Host, *dht.IpfsDHT, chan BootstrapStatus, error) {
	cmgr := connmgr.NewConnManager(1500, 2000, time.Minute)

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	libp2pOpts := []libp2p.Option{libp2p.ListenAddrStrings(opts.addr), libp2p.ConnectionManager(cmgr), libp2p.Identity(priv)}

	if opts.relay {
		libp2pOpts = append(libp2pOpts, libp2p.EnableRelay(circuit.OptHop))
	}

	node, err := libp2p.New(context.Background(), libp2pOpts...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to spawn libp2p node: %w", err)
	}

	dhtNode, err := dht.New(context.Background(), node, dhtopts.BucketSize(opts.bucketSize), dhtopts.Datastore(opts.datastore), dhtopts.Validator(record.NamespacedValidator{
		"pk":   record.PublicKeyValidator{},
		"ipns": ipns.Validator{KeyBook: node.Peerstore()},
	}))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to instantiate DHT: %w", err)
	}

	// bootstrap in the background
	// it's safe to start doing this _before_ establishing any connections
	// as we'll trigger a boostrap round as soon as we get a connection anyways.
	dhtNode.Bootstrap(context.Background())

	bsCh := make(chan BootstrapStatus, 1)

	go func() {
		// ❓ what is this limiter for?
		if opts.limiter != nil {
			opts.limiter <- struct{}{}
		}

		for {
			addr, err := randBootstrapAddr()
			if err != nil {
				bsCh <- BootstrapStatus{err: fmt.Errorf("failed to get random bootstrap multiaddr: %w", err)}
				continue
			}
			if err := node.Connect(context.Background(), *addr); err != nil {
				bsCh <- BootstrapStatus{err: fmt.Errorf("bootstrap connect failed with error: %w. Trying again", err)}
				continue
			}
			break
		}

		if opts.limiter != nil {
			<-opts.limiter
		}

		bsCh <- BootstrapStatus{done: true}
		close(bsCh)
	}()
	return node, dhtNode, bsCh, nil
}
