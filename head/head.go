package head

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hnlq715/golang-lru/simplelru"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ipns"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	noise "github.com/libp2p/go-libp2p-noise"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	record "github.com/libp2p/go-libp2p-record"
	tls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-tcp-transport"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/libp2p/hydra-booster/version"
	"github.com/multiformats/go-multiaddr"
)

const (
	lowWater               = 1200
	highWater              = 1800
	gracePeriod            = time.Minute
	provDisabledGCInterval = time.Hour * 24 * 365 * 100 // set really high to be "disabled"
	provCacheSize          = 256
	provCacheExpiry        = time.Hour
)

// BootstrapStatus describes the status of connecting to a bootstrap node.
type BootstrapStatus struct {
	Done bool
	Err  error
}

// Head is a container for ipfs/libp2p components used by a Hydra head.
type Head struct {
	Host      host.Host
	Datastore datastore.Datastore
	Routing   routing.Routing
}

// NewHead constructs a new Hydra Booster head node
func NewHead(ctx context.Context, options ...opts.Option) (*Head, chan BootstrapStatus, error) {
	cfg := opts.Options{}
	cfg.Apply(append([]opts.Option{opts.Defaults}, options...)...)

	cmgr := connmgr.NewConnManager(lowWater, highWater, gracePeriod)

	priv, err := cfg.IDGenerator.AddBalanced()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate balanced private key: %w", err)
	}

	ua := version.UserAgent
	if cfg.EnableRelay {
		ua += "+relay"
	}

	libp2pOpts := []libp2p.Option{
		libp2p.UserAgent(version.UserAgent),
		libp2p.ListenAddrs(cfg.Addrs...),
		libp2p.ConnectionManager(cmgr),
		libp2p.Identity(priv),
		libp2p.EnableNATService(),
		libp2p.AutoNATServiceRateLimit(0, 3, time.Minute),
		libp2p.DefaultMuxers,
		libp2p.Transport(quic.NewTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New),
	}
	if cfg.Peerstore != nil {
		libp2pOpts = append(libp2pOpts, libp2p.Peerstore(cfg.Peerstore))
	}
	if cfg.EnableRelay {
		libp2pOpts = append(libp2pOpts, libp2p.EnableRelay(circuit.OptHop))
	}

	node, err := libp2p.New(ctx, libp2pOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to spawn libp2p node: %w", err)
	}

	dhtOpts := []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(cfg.ProtocolPrefix),
		dht.BucketSize(cfg.BucketSize),
		dht.Datastore(cfg.Datastore),
		dht.QueryFilter(dht.PublicQueryFilter),
		dht.RoutingTableFilter(dht.PublicRoutingTableFilter),
	}

	if cfg.DisableProvGC {
		cache, _ := simplelru.NewLRUWithExpire(provCacheSize, provCacheExpiry, nil)
		dhtOpts = append(dhtOpts, dht.ProvidersOptions([]providers.Option{
			providers.CleanupInterval(provDisabledGCInterval),
			providers.Cache(cache),
		}))
	}
	if cfg.DisableValues {
		dhtOpts = append(dhtOpts, dht.DisableValues())
	} else {
		dhtOpts = append(dhtOpts, dht.Validator(record.NamespacedValidator{
			"pk":   record.PublicKeyValidator{},
			"ipns": ipns.Validator{KeyBook: node.Peerstore()},
		}))
	}
	if cfg.DisableProviders {
		dhtOpts = append(dhtOpts, dht.DisableProviders())
	}

	dhtNode, err := dht.New(ctx, node, dhtOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to instantiate DHT: %w", err)
	}

	// bootstrap in the background
	// it's safe to start doing this _before_ establishing any connections
	// as we'll trigger a boostrap round as soon as we get a connection anyways.
	dhtNode.Bootstrap(ctx)

	bsCh := make(chan BootstrapStatus)
	hd := Head{
		Host:      node,
		Datastore: cfg.Datastore,
		Routing:   dhtNode,
	}

	go func() {
		// ❓ what is this limiter for?
		if cfg.Limiter != nil {
			select {
			case cfg.Limiter <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}

		// Connect to all bootstrappers, and protect them.
		if len(cfg.BootstrapPeers) > 0 {
			var wg sync.WaitGroup
			wg.Add(len(cfg.BootstrapPeers))
			for _, addr := range cfg.BootstrapPeers {
				go func(addr multiaddr.Multiaddr) {
					defer wg.Done()
					ai, err := peer.AddrInfoFromP2pAddr(addr)
					if err != nil {
						select {
						case bsCh <- BootstrapStatus{Err: fmt.Errorf("failed to get random bootstrap multiaddr: %w", err)}:
						case <-ctx.Done():
						}
						return
					}
					if err := node.Connect(context.Background(), *ai); err != nil {
						select {
						case bsCh <- BootstrapStatus{Err: fmt.Errorf("bootstrap connect failed with error: %w. Trying again", err)}:
						case <-ctx.Done():
						}
						return
					}
					node.ConnManager().Protect(ai.ID, "bootstrap-peer")
				}(addr)
			}
			wg.Wait()

			if ctx.Err() != nil {
				return
			}

			select {
			case bsCh <- BootstrapStatus{Done: true}:
			case <-ctx.Done():
				return
			}
		}

		if cfg.Limiter != nil {
			<-cfg.Limiter
		}

		close(bsCh)
	}()

	return &hd, bsCh, nil
}

// RoutingTable returns the underlying RoutingTable for this head
func (s *Head) RoutingTable() *kbucket.RoutingTable {
	dht, _ := s.Routing.(*dht.IpfsDHT)
	return dht.RoutingTable()
}

// AddProvider adds the given provider to the datastore
func (s *Head) AddProvider(ctx context.Context, c cid.Cid, id peer.ID) {
	dht, _ := s.Routing.(*dht.IpfsDHT)
	dht.ProviderManager.AddProvider(ctx, c.Bytes(), id)
}
