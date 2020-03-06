package node

import (
	"context"
	"fmt"
	"math/rand"
	"time"

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
	hyopts "github.com/libp2p/hydra-booster/hydrabooster/opts"
	"github.com/multiformats/go-multiaddr"
)

func randBootstrapAddr(bootstrapPeers []multiaddr.Multiaddr) (*peer.AddrInfo, error) {
	addr := dht.DefaultBootstrapPeers[rand.Intn(len(dht.DefaultBootstrapPeers))]
	return peer.AddrInfoFromP2pAddr(addr)
}

// BootstrapStatus describes the status of connecting to a bootstrap node
type BootstrapStatus struct {
	Done bool
	Err  error
}

// HydraNode is a container for libp2p components used by a Hydra Booster node
type HydraNode struct {
	Host host.Host
	DHT  *dht.IpfsDHT
}

// NewHydraNode constructs a new Hydra Booster node
func NewHydraNode(options ...hyopts.Option) (*HydraNode, chan BootstrapStatus, error) {
	cfg := hyopts.Options{}
	cfg.Apply(append([]hyopts.Option{hyopts.Defaults}, options...)...)

	cmgr := connmgr.NewConnManager(1500, 2000, time.Minute)

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
		}

		if cfg.Limiter != nil {
			<-cfg.Limiter
		}

		bsCh <- BootstrapStatus{Done: true}
		close(bsCh)
	}()

	return &HydraNode{Host: node, DHT: dhtNode}, bsCh, nil
}
