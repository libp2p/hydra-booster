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
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	hyds "github.com/libp2p/hydra-booster/datastore"
	"github.com/libp2p/hydra-booster/head"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/libp2p/hydra-booster/idgen"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/periodictasks"
	"github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// Default intervals between periodic task runs, more cpu/memory intensive tasks are run less frequently
// TODO: expose these as command line options?
const (
	providerRecordsTaskInterval  = time.Minute * 5
	routingTableSizeTaskInterval = time.Second * 5
	uniquePeersTaskInterval      = time.Second * 5
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
	EnableV1Compat    bool
	BootstrapPeers    []multiaddr.Multiaddr
	DisablePrefetch   bool
	DisableProvCounts bool
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
		ds, err = hyds.NewPostgreSQLDatastore(ctx, options.DatastorePath)
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

	fmt.Fprintf(os.Stderr, "🐲 Spawning %d heads: ", options.NHeads)

	var hyperLock sync.Mutex
	hyperlog := hyperloglog.New()

	// What is a limiter?
	limiter := make(chan struct{}, options.BsCon)

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
		if options.EnableV1Compat {
			hdOpts = append(hdOpts, opts.EnableV1Compat())
		}

		hd, bsCh, err := head.NewHead(ctx, hdOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn node with swarm addresses %v %v: %w", tcpAddr, quicAddr, err)
		}

		hdCtx, err := tag.New(ctx, tag.Insert(metrics.KeyPeerID, hd.Host.ID().String()))
		if err != nil {
			return nil, err
		}

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
	}
	fmt.Fprintf(os.Stderr, "\n")

	for _, hd := range hds {
		fmt.Fprintf(os.Stderr, "🆔 %v\n", hd.Host.ID())
		for _, addr := range hd.Host.Addrs() {
			fmt.Fprintf(os.Stderr, "🐝 Swarm listening on %v\n", addr)
		}
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
