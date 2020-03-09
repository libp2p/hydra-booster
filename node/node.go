package node

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ipfs/go-ipns"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/hydra-booster/opts"
	"github.com/multiformats/go-multiaddr"
)

const lowWater = 1500
const highWater = 2000
const gracePeriod = time.Minute

func randBootstrapAddr(bootstrapPeers []multiaddr.Multiaddr) (*peer.AddrInfo, error) {
	addr := bootstrapPeers[rand.Intn(len(bootstrapPeers))]
	ai, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s to AddrInfo: %w", addr, err)
	}
	return ai, nil
}

// BootstrapStatus describes the status of connecting to a bootstrap node
type BootstrapStatus struct {
	Done bool
	Err  error
}

// HydraNode is a container for libp2p components used by a Hydra Booster node
type HydraNode struct {
	Host         host.Host
	Routing      routing.Routing
	Bootstrapped bool
}

// NewHydraNode constructs a new Hydra Booster node
func NewHydraNode(options ...opts.Option) (*HydraNode, chan BootstrapStatus, error) {
	cfg := opts.Options{}
	cfg.Apply(append([]opts.Option{opts.Defaults}, options...)...)

	cmgr := connmgr.NewConnManager(lowWater, highWater, gracePeriod)

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	libp2pOpts := []libp2p.Option{libp2p.ListenAddrs(cfg.Addr), libp2p.ConnectionManager(cmgr), libp2p.Identity(priv)}

	if cfg.Relay {
		libp2pOpts = append(libp2pOpts, libp2p.EnableRelay(circuit.OptHop))
	}

	node, err := libp2p.New(context.Background(), libp2pOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to spawn libp2p node: %w", err)
	}

	dhtNode, err := dht.New(context.Background(), node, dhtopts.BucketSize(cfg.BucketSize), dhtopts.Datastore(cfg.Datastore), dhtopts.Validator(record.NamespacedValidator{
		"pk":   record.PublicKeyValidator{},
		"ipns": ipns.Validator{KeyBook: node.Peerstore()},
	}))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to instantiate DHT: %w", err)
	}

	// bootstrap in the background
	// it's safe to start doing this _before_ establishing any connections
	// as we'll trigger a boostrap round as soon as we get a connection anyways.
	dhtNode.Bootstrap(context.Background())

	bsCh := make(chan BootstrapStatus, 1)
	hyNode := HydraNode{Host: node, Routing: dhtNode}

	go func() {
		// ❓ what is this limiter for?
		if cfg.Limiter != nil {
			cfg.Limiter <- struct{}{}
		}

		if len(cfg.BootstrapPeers) > 0 {
			for {
				addr, err := randBootstrapAddr(cfg.BootstrapPeers)
				if err != nil {
					bsCh <- BootstrapStatus{Err: fmt.Errorf("failed to get random bootstrap multiaddr: %w", err)}
					continue
				}
				if err := node.Connect(context.Background(), *addr); err != nil {
					bsCh <- BootstrapStatus{Err: fmt.Errorf("bootstrap connect failed with error: %w. Trying again", err)}
					continue
				}
				break
			}
			hyNode.Bootstrapped = true
		}

		if cfg.Limiter != nil {
			<-cfg.Limiter
		}

		bsCh <- BootstrapStatus{Done: true}
		close(bsCh)
	}()

	return &hyNode, bsCh, nil
}
