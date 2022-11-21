package head

import (
	"context"
	"fmt"
	golog "log"
	"os"
	"sync"
	"time"

	"github.com/hnlq715/golang-lru/simplelru"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ipns"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	noise "github.com/libp2p/go-libp2p-noise"
	record "github.com/libp2p/go-libp2p-record"
	rcmgr "github.com/libp2p/go-libp2p-resource-manager"
	"github.com/libp2p/go-libp2p-resource-manager/obs"
	tls "github.com/libp2p/go-libp2p-tls"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/metricstasks"
	"github.com/libp2p/hydra-booster/periodictasks"
	hproviders "github.com/libp2p/hydra-booster/providers"
	"github.com/libp2p/hydra-booster/version"
	"github.com/multiformats/go-multiaddr"
)

const (
	providerRecordsTaskInterval = time.Minute * 5
	lowWater                    = 1200
	highWater                   = 1800
	gracePeriod                 = time.Minute
	provDisabledGCInterval      = time.Hour * 24 * 365 * 100 // set really high to be "disabled"
	provCacheSize               = 256
	provCacheExpiry             = time.Hour
)

var log = logging.Logger("hydra/hydra")

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

func buildRcmgr(ctx context.Context, disableRM bool, limitsFile string) (network.ResourceManager, error) {
	var limiter rcmgr.Limiter

	if disableRM {
		limiter = rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits)
	} else if limitsFile != "" {
		f, err := os.Open(limitsFile)
		if err != nil {
			return nil, fmt.Errorf("opening Resource Manager limits file: %w", err)
		}
		limiter, err = rcmgr.NewDefaultLimiterFromJSON(f)
		if err != nil {
			return nil, fmt.Errorf("creating Resource Manager limiter: %w", err)
		}
	} else {
		limits := rcmgr.DefaultLimits

		limits.SystemBaseLimit.ConnsOutbound = 128
		limits.SystemBaseLimit.ConnsInbound = 128
		limits.SystemBaseLimit.Conns = 256
		limits.SystemLimitIncrease.Conns = 1024
		limits.SystemLimitIncrease.ConnsInbound = 1024
		limits.SystemLimitIncrease.ConnsOutbound = 1024
		libp2p.SetDefaultServiceLimits(&limits)

		limiter = rcmgr.NewFixedLimiter(limits.AutoScale())
	}

	rcmgrMetrics, err := metrics.CreateRcmgrMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating Resource Manager metrics: %w", err)
	}
	mgr, err := rcmgr.NewResourceManager(
		limiter,
		rcmgr.WithTraceReporter(obs.StatsTraceReporter{}),
		rcmgr.WithMetrics(rcmgrMetrics),
	)
	if err != nil {
		return nil, fmt.Errorf("constructing resource manager: %w", err)
	}

	return mgr, nil
}

// NewHead constructs a new Hydra Booster head node
func NewHead(ctx context.Context, options ...opts.Option) (*Head, chan BootstrapStatus, error) {
	cfg := opts.Options{}
	cfg.Apply(append([]opts.Option{opts.Defaults}, options...)...)

	cmgr := connmgr.NewConnManager(lowWater, highWater, gracePeriod)

	ua := version.UserAgent
	if cfg.EnableRelay {
		ua += "+relay"
	}

	rm, err := buildRcmgr(ctx, cfg.DisableResourceManager, cfg.ResourceManagerLimitsFile)
	if err != nil {
		return nil, nil, err
	}

	libp2pOpts := []libp2p.Option{
		libp2p.UserAgent(version.UserAgent),
		libp2p.ListenAddrs(cfg.Addrs...),
		libp2p.ConnectionManager(cmgr),
		libp2p.Identity(cfg.ID),
		libp2p.EnableNATService(),
		libp2p.AutoNATServiceRateLimit(0, 3, time.Minute),
		libp2p.DefaultMuxers,
		libp2p.Transport(quic.NewTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.ResourceManager(rm),
	}
	if cfg.Peerstore != nil {
		libp2pOpts = append(libp2pOpts, libp2p.Peerstore(cfg.Peerstore))
	}
	if cfg.EnableRelay {
		libp2pOpts = append(libp2pOpts, libp2p.EnableRelay())
	}

	golog.Println("libp2p new")
	node, err := libp2p.New(libp2pOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to spawn libp2p node: %w", err)
	}
	go func() {
		<-ctx.Done()
		node.Close()
	}()

	dhtOpts := []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(cfg.ProtocolPrefix),
		dht.BucketSize(cfg.BucketSize),
		dht.Datastore(cfg.Datastore),
		dht.QueryFilter(dht.PublicQueryFilter),
		dht.RoutingTableFilter(dht.PublicRoutingTableFilter),
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

	var providerStore providers.ProviderStore
	if cfg.ProviderStoreBuilder == nil {
		ps, err := newDefaultProviderStore(ctx, cfg, node)
		if err != nil {
			return nil, nil, err
		}
		providerStore = ps
	} else {
		ps, err := cfg.ProviderStoreBuilder(cfg, node)
		if err != nil {
			return nil, nil, err
		}
		providerStore = ps
	}

	if !cfg.DisableProvCounts {
		periodictasks.RunTasks(ctx, []periodictasks.PeriodicTask{metricstasks.NewProviderRecordsTask(cfg.Datastore, providerStore, providerRecordsTaskInterval)})
	}

	var cachingProviderStore *hproviders.CachingProviderStore
	if cfg.ProvidersFinder != nil && cfg.ReframeAddr == "" {
		cachingProviderStore = hproviders.NewCachingProviderStore(providerStore, providerStore, cfg.ProvidersFinder, nil)
		providerStore = cachingProviderStore
	}
	if cfg.ProvidersFinder != nil && cfg.ReframeAddr != "" {
		reframeProviderStore, err := hproviders.NewReframeProviderStore(cfg.DelegateHTTPClient, cfg.ReframeAddr)
		if err != nil {
			return nil, nil, fmt.Errorf("creating Reframe providerstore: %w", err)
		}

		cachingProviderStore = hproviders.NewCachingProviderStore(
			hproviders.CombineProviders(providerStore, reframeProviderStore),
			providerStore,
			cfg.ProvidersFinder,
			nil,
		)

		// we still want to use the caching provider store instead of the provider store directly b/c it publishes cache metrics
		fmt.Printf("Will delegate to %v with timeout %v.\n", cfg.ReframeAddr, cfg.DelegateHTTPClient.Timeout)
		providerStore = cachingProviderStore
	}
	if cfg.ProvidersFinder == nil && cfg.ReframeAddr != "" {
		reframePS, err := hproviders.NewReframeProviderStore(cfg.DelegateHTTPClient, cfg.ReframeAddr)
		if err != nil {
			return nil, nil, fmt.Errorf("creating Reframe provider store: %w", err)
		}
		fmt.Printf("Will delegate to %v with timeout %v.\n", cfg.ReframeAddr, cfg.DelegateHTTPClient.Timeout)
		providerStore = reframePS
	}

	dhtOpts = append(dhtOpts, dht.ProviderStore(providerStore))

	golog.Println("dhtnew")
	dhtNode, err := dht.New(ctx, node, dhtOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to instantiate DHT: %w", err)
	}

	// if we are using the caching provider store, we need to give it the content router to use (the DHT)
	if cachingProviderStore != nil {
		cachingProviderStore.Router = dhtNode
	}

	// bootstrap in the background
	// it's safe to start doing this _before_ establishing any connections
	// as we'll trigger a boostrap round as soon as we get a connection anyways.
	golog.Println("bootstrap")
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

func newDefaultProviderStore(ctx context.Context, options opts.Options, h host.Host) (providers.ProviderStore, error) {
	fmt.Fprintf(os.Stderr, "🥞 Using default providerstore\n")
	var provMgrOpts []providers.Option
	if options.DisableProvGC {
		cache, err := simplelru.NewLRUWithExpire(provCacheSize, provCacheExpiry, nil)
		if err != nil {
			return nil, err
		}
		provMgrOpts = append(provMgrOpts,
			providers.CleanupInterval(provDisabledGCInterval),
			providers.Cache(cache),
		)
	}
	var ps providers.ProviderStore
	ps, err := providers.NewProviderManager(ctx, h.ID(), h.Peerstore(), options.Datastore, provMgrOpts...)
	if err != nil {
		return nil, err
	}
	return ps, nil
}

// RoutingTable returns the underlying RoutingTable for this head
func (s *Head) RoutingTable() *kbucket.RoutingTable {
	dht, _ := s.Routing.(*dht.IpfsDHT)
	return dht.RoutingTable()
}

// AddProvider adds the given provider to the datastore
func (s *Head) AddProvider(ctx context.Context, c cid.Cid, id peer.ID) {
	dht, _ := s.Routing.(*dht.IpfsDHT)
	dht.ProviderStore().AddProvider(ctx, c.Hash(), peer.AddrInfo{ID: id})
}
