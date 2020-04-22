package head

import (
	"context"
	"fmt"
	"math/rand"
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
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/libp2p/hydra-booster/version"
	"github.com/multiformats/go-multiaddr"
)

const (
	lowWater               = 1500
	highWater              = 2000
	gracePeriod            = time.Minute
	provDisabledGCInterval = time.Hour * 24 * 365 * 100 // set really high to be "disabled"
	provCacheSize          = 256
	provCacheExpiry        = time.Hour
)

func randBootstrapAddr(bootstrapPeers []multiaddr.Multiaddr) (*peer.AddrInfo, error) {
	addr := bootstrapPeers[rand.Intn(len(bootstrapPeers))]
	ai, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s to AddrInfo: %w", addr, err)
	}
	return ai, nil
}

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
		libp2p.ListenAddrs(cfg.Addr),
		libp2p.ConnectionManager(cmgr),
		libp2p.Identity(priv),
		libp2p.EnableNATService(),
		libp2p.AutoNATServiceRateLimit(0, 3, time.Minute),
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

		if len(cfg.BootstrapPeers) > 0 {
			for {
				addr, err := randBootstrapAddr(cfg.BootstrapPeers)
				if err != nil {
					select {
					case bsCh <- BootstrapStatus{Err: fmt.Errorf("failed to get random bootstrap multiaddr: %w", err)}:
						continue
					case <-ctx.Done():
						return
					}
				}
				if err := node.Connect(context.Background(), *addr); err != nil {
					select {
					case bsCh <- BootstrapStatus{Err: fmt.Errorf("bootstrap connect failed with error: %w. Trying again", err)}:
						continue
					case <-ctx.Done():
						return
					}
				}
				break
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
