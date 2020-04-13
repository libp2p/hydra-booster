package hydra

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/axiomhq/hyperloglog"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/routing"
	hyds "github.com/libp2p/hydra-booster/datastore"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/periodictasks"
	"github.com/libp2p/hydra-booster/sybil"
	"github.com/libp2p/hydra-booster/sybil/opts"
	"github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// Default intervals between periodic task runs, more cpu/memory intensive tasks are run less frequently
// TODO: expose these as command line options?
const (
	providerRecordsTaskInterval  = time.Minute
	routingTableSizeTaskInterval = time.Second * 5
	uniquePeersTaskInterval      = time.Second * 5
)

// Hydra is a container for heads (sybils) and their shared belly bits.
type Hydra struct {
	Sybils          []*sybil.Sybil
	SharedDatastore datastore.Datastore
	// SharedRoutingTable *kbucket.RoutingTable

	hyperLock *sync.Mutex
	hyperlog  *hyperloglog.Sketch
}

// Options are configuration for a new hydra.
type Options struct {
	Name          string
	DatastorePath string
	GetPort       func() int
	NSybils       int
	BucketSize    int
	BsCon         int
	Relay         bool
	Stagger       time.Duration
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

	lds, err := leveldb.NewDatastore(options.DatastorePath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	var sybils []*sybil.Sybil

	ds := hyds.NewProxy(ctx, lds, func(_ cid.Cid) (routing.Routing, hyds.AddProviderFunc, error) {
		if len(sybils) == 0 {
			return nil, nil, fmt.Errorf("no sybils available")
		}
		s := sybils[rand.Intn(len(sybils))]
		// we should ask the closest sybil, but later they'll all share the same routing table so it won't matter which one we pick
		return s.Routing, s.AddProvider, nil
	}, hyds.Options{
		FindProvidersConcurrency:    options.NSybils,
		FindProvidersCount:          1,
		FindProvidersQueueSize:      options.NSybils * 12,
		FindProvidersTimeout:        time.Second * 20,
		FindProvidersFailureBackoff: time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Running %d DHT Sybils:\n", options.NSybils)

	var hyperLock sync.Mutex
	hyperlog := hyperloglog.New()

	// What is a limiter?
	limiter := make(chan struct{}, options.BsCon)

	for i := 0; i < options.NSybils; i++ {
		time.Sleep(options.Stagger)
		fmt.Fprintf(os.Stderr, ".")

		addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", options.GetPort()))
		syb, bsCh, err := sybil.NewSybil(
			ctx,
			opts.Datastore(ds),
			opts.Addr(addr),
			opts.Relay(options.Relay),
			opts.BucketSize(options.BucketSize),
			opts.Limiter(limiter),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn node with swarm address %v: %w", addr, err)
		}

		sybCtx, err := tag.New(ctx, tag.Insert(metrics.KeyPeerID, syb.Host.ID().String()))
		if err != nil {
			return nil, err
		}

		stats.Record(sybCtx, metrics.Sybils.M(1))

		syb.Host.Network().Notify(&network.NotifyBundle{
			ConnectedF: func(n network.Network, v network.Conn) {
				hyperLock.Lock()
				hyperlog.Insert([]byte(v.RemotePeer()))
				hyperLock.Unlock()
				stats.Record(sybCtx, metrics.ConnectedPeers.M(1))
			},
			DisconnectedF: func(n network.Network, v network.Conn) {
				stats.Record(sybCtx, metrics.ConnectedPeers.M(-1))
			},
		})

		go handleBootstrapStatus(sybCtx, bsCh)

		sybils = append(sybils, syb)
	}
	fmt.Fprintf(os.Stderr, "\n")

	hydra := Hydra{
		Sybils:          sybils,
		SharedDatastore: ds,
		hyperLock:       &hyperLock,
		hyperlog:        hyperlog,
	}

	periodictasks.RunTasks(ctx, []periodictasks.PeriodicTask{
		newProviderRecordsTask(&hydra, providerRecordsTaskInterval),
		newRoutingTableSizeTask(&hydra, routingTableSizeTaskInterval),
		newUniquePeersTask(&hydra, uniquePeersTaskInterval),
	})

	return &hydra, nil
}

func handleBootstrapStatus(ctx context.Context, ch chan sybil.BootstrapStatus) {
	for status := range ch {
		if status.Err != nil {
			fmt.Println(status.Err)
		}
		if status.Done {
			stats.Record(ctx, metrics.BootstrappedSybils.M(1))
		}
	}
}

// GetUniquePeersCount retrieves the current total for unique peers
func (hy *Hydra) GetUniquePeersCount() uint64 {
	hy.hyperLock.Lock()
	defer hy.hyperLock.Unlock()
	return hy.hyperlog.Estimate()
}
