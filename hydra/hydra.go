package hydra

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/axiomhq/hyperloglog"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	noise "github.com/libp2p/go-libp2p-noise"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	secio "github.com/libp2p/go-libp2p-secio"
	tls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-tcp-transport"
	hyds "github.com/libp2p/hydra-booster/datastore"
	"github.com/libp2p/hydra-booster/head"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/libp2p/hydra-booster/idgen"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/periodictasks"
	"github.com/libp2p/hydra-booster/version"
	"github.com/multiformats/go-multiaddr"
	mafmt "github.com/multiformats/go-multiaddr-fmt"
	"github.com/whyrusleeping/timecache"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// Default intervals between periodic task runs, more cpu/memory intensive tasks are run less frequently
// TODO: expose these as command line options?
const (
	providerRecordsTaskInterval  = time.Minute * 5
	routingTableSizeTaskInterval = time.Second * 5
	uniquePeersTaskInterval      = time.Second * 5
	lowWater                     = 1200
	highWater                    = 1800
	gracePeriod                  = time.Minute
)

const (
	agentVersionKey      = "AgentVersion"
	dialedPeersCacheSpan = 8 * time.Hour
)

// Hydra is a container for heads and their shared belly bits.
type Hydra struct {
	Heads           []*head.Head
	SharedDatastore datastore.Datastore
	// SharedRoutingTable *kbucket.RoutingTable

	hyperLock *sync.Mutex
	hyperlog  *hyperloglog.Sketch
}

// Options are configuration for a new hydra.
type Options struct {
	Name              string
	DatastorePath     string
	PeerstorePath     string
	GetPort           func() int
	NHeads            int
	ProtocolPrefix    protocol.ID
	BucketSize        int
	BsCon             int
	EnableRelay       bool
	Stagger           time.Duration
	IDGenerator       idgen.IdentityGenerator
	DisableProvGC     bool
	DisableProviders  bool
	DisableValues     bool
	BootstrapPeers    []multiaddr.Multiaddr
	DisablePrefetch   bool
	DisableProvCounts bool
	DisableDBCreate   bool
}

// NewHydra creates a new Hydra with the passed options.
func NewHydra(ctx context.Context, options Options) (*Hydra, error) {
	if options.Name != "" {
		nctx, err := tag.New(ctx, tag.Insert(metrics.KeyName, options.Name))
		if err != nil {
			return nil, err
		}
		ctx = nctx
	}

	var ds datastore.Batching
	var err error
	if strings.HasPrefix(options.DatastorePath, "postgresql://") {
		fmt.Fprintf(os.Stderr, "🐘 Using PostgreSQL datastore\n")
		ds, err = hyds.NewPostgreSQLDatastore(ctx, options.DatastorePath, !options.DisableDBCreate)
	} else {
		fmt.Fprintf(os.Stderr, "🥞 Using LevelDB datastore\n")
		ds, err = leveldb.NewDatastore(options.DatastorePath, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	var hds []*head.Head

	if !options.DisablePrefetch {
		ds = hyds.NewProxy(ctx, ds, func(_ cid.Cid) (routing.Routing, hyds.AddProviderFunc, error) {
			if len(hds) == 0 {
				return nil, nil, fmt.Errorf("no heads available")
			}
			s := hds[rand.Intn(len(hds))]
			// we should ask the closest head, but later they'll all share the same routing table so it won't matter which one we pick
			return s.Routing, s.AddProvider, nil
		}, hyds.Options{
			FindProvidersConcurrency:    options.NHeads,
			FindProvidersCount:          1,
			FindProvidersQueueSize:      options.NHeads * 10,
			FindProvidersTimeout:        time.Second * 20,
			FindProvidersFailureBackoff: time.Hour,
		})
	}

	if options.PeerstorePath == "" {
		fmt.Fprintf(os.Stderr, "💭 Using in-memory peerstore\n")
	} else {
		fmt.Fprintf(os.Stderr, "🥞 Using LevelDB peerstore (EXPERIMENTAL)\n")
	}

	fmt.Fprintf(os.Stderr, "🐲 Spawning %d heads: ", options.NHeads)

	var hyperLock sync.Mutex
	hyperlog := hyperloglog.New()

	// What is a limiter?
	limiter := make(chan struct{}, options.BsCon)

	//  create QUIC dial back host
	qh, err := getDialBackHost(ctx, libp2p.Transport(quic.NewTransport))
	if err != nil {
		return nil, fmt.Errorf("failed to create dial back host for quic: %w", err)
	}

	// create TCP dial back host
	th, err := getDialBackHost(ctx, libp2p.Transport(tcp.NewTCPTransport))
	if err != nil {
		return nil, fmt.Errorf("failed to create dial back host for tcp: %w", err)
	}

	hdPeerIds := make(map[peer.ID]struct{}, len(hds))
	for i := 0; i < options.NHeads; i++ {
		time.Sleep(options.Stagger)
		fmt.Fprintf(os.Stderr, ".")

		port := options.GetPort()
		tcpAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
		quicAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port))

		hdOpts := []opts.Option{
			opts.Datastore(ds),
			opts.Addrs([]multiaddr.Multiaddr{tcpAddr, quicAddr}),
			opts.ProtocolPrefix(options.ProtocolPrefix),
			opts.BucketSize(options.BucketSize),
			opts.Limiter(limiter),
			opts.IDGenerator(options.IDGenerator),
			opts.BootstrapPeers(options.BootstrapPeers),
		}
		if options.EnableRelay {
			hdOpts = append(hdOpts, opts.EnableRelay())
		}
		// only the first head should GC, or none of them if it's disabled
		if options.DisableProvGC || i > 0 {
			hdOpts = append(hdOpts, opts.DisableProvGC())
		}
		if options.DisableProviders {
			hdOpts = append(hdOpts, opts.DisableProviders())
		}
		if options.DisableValues {
			hdOpts = append(hdOpts, opts.DisableValues())
		}
		if options.PeerstorePath != "" {
			pstoreDs, err := leveldb.NewDatastore(fmt.Sprintf("%s/head-%d", options.PeerstorePath, i), nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create peerstore datastore: %w", err)
			}
			pstore, err := pstoreds.NewPeerstore(ctx, pstoreDs, pstoreds.DefaultOpts())
			if err != nil {
				return nil, fmt.Errorf("failed to create peerstore: %w", err)
			}
			hdOpts = append(hdOpts, opts.Peerstore(pstore))
		}

		hd, bsCh, err := head.NewHead(ctx, hdOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn node with swarm addresses %v %v: %w", tcpAddr, quicAddr, err)
		}

		hdCtx, err := tag.New(ctx, tag.Insert(metrics.KeyPeerID, hd.Host.ID().String()))
		if err != nil {
			return nil, err
		}
		hd.HeadCtx = hdCtx

		stats.Record(hdCtx, metrics.Heads.M(1))

		hd.Host.Network().Notify(&network.NotifyBundle{
			ConnectedF: func(n network.Network, v network.Conn) {
				hyperLock.Lock()
				hyperlog.Insert([]byte(v.RemotePeer()))
				hyperLock.Unlock()
				stats.Record(hdCtx, metrics.ConnectedPeers.M(1))
			},
			DisconnectedF: func(n network.Network, v network.Conn) {
				stats.Record(hdCtx, metrics.ConnectedPeers.M(-1))
			},
		})

		go handleBootstrapStatus(hdCtx, bsCh)

		hds = append(hds, hd)
		hdPeerIds[hd.Host.ID()] = struct{}{}
	}

	fmt.Fprintf(os.Stderr, "\n")

	var dp sync.Mutex
	dialedPeers := timecache.NewTimeCache(dialedPeersCacheSpan)

	for _, hd := range hds {
		fmt.Fprintf(os.Stderr, "🆔 %v\n", hd.Host.ID())
		for _, addr := range hd.Host.Addrs() {
			fmt.Fprintf(os.Stderr, "🐝 Swarm listening on %v\n", addr)
		}

		// dial back hooks
		evts := []interface{}{
			new(event.EvtPeerIdentificationCompleted),
		}

		subs, err := hd.Host.EventBus().Subscribe(evts, eventbus.BufSize(256))
		if err != nil {
			return nil, fmt.Errorf("head could not subscribe to eventbus events; err: %w", err)
		}
		go func(hd *head.Head, subs event.Subscription) {
			defer subs.Close()
			for {
				select {
				case evt := <-subs.Out():
					ev := evt.(event.EvtPeerIdentificationCompleted)
					p := ev.Peer

					// do not dial back the peer if we've already dialled it
					dp.Lock()
					seen := dialedPeers.Has(p.String())
					dp.Unlock()
					if seen {
						continue
					}

					// do not dial back our own heads
					if _, ok := hdPeerIds[p]; ok {
						continue
					}

					// do not dial back other Hydras
					v, err := hd.Host.Peerstore().Get(p, agentVersionKey)
					if err != nil {
						continue
					}
					if s := v.(string); s == version.UserAgent {
						continue
					}

					addrs := hd.Host.Peerstore().Addrs(p)

					// dial back on quic if peer advertises a quic address
					for _, a := range addrs {
						// ignore relay addrs
						_, err := a.ValueForProtocol(multiaddr.P_CIRCUIT)
						if err != nil && mafmt.QUIC.Matches(a) {
							stats.Record(ctx, metrics.QuicConns.M(1))

							if err := qh.Connect(hd.HeadCtx, peer.AddrInfo{ID: p, Addrs: addrs}); err == nil {
								stats.Record(ctx, metrics.QuicDialBacks.M(1))

								// close the connection as we don't need it anymore
								if err := qh.Network().ClosePeer(p); err != nil {
									fmt.Fprintf(os.Stderr, "\n quic dial back: failed to close connection to peer %s, err: %s", p.Pretty(),
										err)
								}
							} else {
								fmt.Fprintf(os.Stderr, "\n quic dial failed because of error: %+v", err)
								stats.Record(ctx, metrics.QuicDialBackFailures.M(1))
							}

							break
						}
					}
					qh.Peerstore().ClearAddrs(p)

					// dial back on tcp if peer advertises a tcp address
					for _, a := range addrs {
						_, err := a.ValueForProtocol(multiaddr.P_CIRCUIT)
						if err != nil && mafmt.TCP.Matches(a) {
							stats.Record(ctx, metrics.TCPConns.M(1))

							if err := th.Connect(hd.HeadCtx, peer.AddrInfo{ID: p, Addrs: addrs}); err == nil {
								stats.Record(ctx, metrics.TCPDialBacks.M(1))

								// close the connection as we don't need it anymore
								if err := th.Network().ClosePeer(p); err != nil {
									fmt.Fprintf(os.Stderr, "\n tcp dial back: failed to close connection to peer %s, err: %s", p.Pretty(), err)
								}
							} else {
								fmt.Fprintf(os.Stderr, "\n tcp dial failed because of error: %+v", err)
								stats.Record(ctx, metrics.TCPDialBackFailures.M(1))
							}

							break
						}
					}

					// mark peer as seen
					dp.Lock()
					if !dialedPeers.Has(p.String()) {
						dialedPeers.Add(p.String())
					}
					dp.Unlock()

					th.Peerstore().ClearAddrs(p)

				case <-hd.HeadCtx.Done():
					return
				}
			}
		}(hd, subs)
	}

	hydra := Hydra{
		Heads:           hds,
		SharedDatastore: ds,
		hyperLock:       &hyperLock,
		hyperlog:        hyperlog,
	}

	tasks := []periodictasks.PeriodicTask{
		newRoutingTableSizeTask(&hydra, routingTableSizeTaskInterval),
		newUniquePeersTask(&hydra, uniquePeersTaskInterval),
	}

	if !options.DisableProvCounts {
		tasks = append(tasks, newProviderRecordsTask(&hydra, providerRecordsTaskInterval))
	}

	periodictasks.RunTasks(ctx, tasks)

	return &hydra, nil
}

func getDialBackHost(ctx context.Context, transportOpt libp2p.Option) (host.Host, error) {
	cmgr := connmgr.NewConnManager(lowWater, highWater, gracePeriod)

	libp2pOpts := []libp2p.Option{
		libp2p.UserAgent(version.UserAgent),
		libp2p.ConnectionManager(cmgr),
		transportOpt,
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(secio.ID, secio.New),
		libp2p.NoListenAddrs,
	}

	node, err := libp2p.New(ctx, libp2pOpts...)
	if err != nil {
		return nil, fmt.Errorf("dialback host: failed to spawn libp2p node: %w", err)
	}

	return node, err
}

func handleBootstrapStatus(ctx context.Context, ch chan head.BootstrapStatus) {
	for status := range ch {
		if status.Err != nil {
			fmt.Println(status.Err)
		}
		if status.Done {
			stats.Record(ctx, metrics.BootstrappedHeads.M(1))
		}
	}
}

// GetUniquePeersCount retrieves the current total for unique peers
func (hy *Hydra) GetUniquePeersCount() uint64 {
	hy.hyperLock.Lock()
	defer hy.hyperLock.Unlock()
	return hy.hyperlog.Estimate()
}
