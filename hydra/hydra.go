package hydra

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/axiomhq/hyperloglog"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	levelds "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p-core/network"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/node"
	hynodeopts "github.com/libp2p/hydra-booster/node/opts"
	"github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

type Hydra struct {
	Sybils       []*node.HydraNode
	Datastore    datastore.Datastore
	RoutingTable *kbucket.RoutingTable

	hyperLock   *sync.Mutex
	hyperlog    *hyperloglog.Sketch
	statsTicker *time.Ticker
}

type Options struct {
	DatastorePath string
	GetPort       func() int
	NSybils       int
	BucketSize    int
	BsCon         int
	Relay         bool
	Stagger       time.Duration
}

func NewHydra(options Options) (*Hydra, error) {
	datastore, err := levelds.NewDatastore(options.DatastorePath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	var nodes []*node.HydraNode

	fmt.Fprintf(os.Stderr, "Running %d DHT Instances:\n", options.NSybils)

	var hyperLock sync.Mutex
	hyperlog := hyperloglog.New()

	// What is a limiter?
	limiter := make(chan struct{}, options.BsCon)

	for i := 0; i < options.NSybils; i++ {
		time.Sleep(options.Stagger)
		fmt.Fprintf(os.Stderr, ".")

		addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", options.GetPort()))
		nd, bsCh, err := node.NewHydraNode(
			hynodeopts.Datastore(datastore),
			hynodeopts.Addr(addr),
			hynodeopts.Relay(options.Relay),
			hynodeopts.BucketSize(options.BucketSize),
			hynodeopts.Limiter(limiter),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn node with swarm address %v: %w", addr, err)
		}

		ctx, err := tag.New(context.Background(), tag.Insert(metrics.KeyPeerID, nd.Host.ID().String()))
		if err != nil {
			return nil, err
		}

		stats.Record(ctx, metrics.Sybils.M(1))

		nd.Host.Network().Notify(&network.NotifyBundle{
			ConnectedF: func(n network.Network, v network.Conn) {
				hyperLock.Lock()
				hyperlog.Insert([]byte(v.RemotePeer()))
				hyperLock.Unlock()
				stats.Record(ctx, metrics.ConnectedPeers.M(1))
			},
			DisconnectedF: func(n network.Network, v network.Conn) {
				stats.Record(ctx, metrics.ConnectedPeers.M(-1))
			},
		})

		go handleBootstrapStatus(ctx, bsCh)

		nodes = append(nodes, nd)
	}
	fmt.Fprintf(os.Stderr, "\n")

	hydra := Hydra{
		Sybils:    nodes,
		Datastore: datastore,
		hyperLock: &hyperLock,
		hyperlog:  hyperlog,
	}

	hydra.statsTicker = recordPeriodicMetrics(&hydra, time.Second*5)

	return &hydra, nil
}

func handleBootstrapStatus(ctx context.Context, ch chan node.BootstrapStatus) {
	for status := range ch {
		if status.Err != nil {
			fmt.Println(status.Err)
		}
		if status.Done {
			stats.Record(ctx, metrics.BootstrappedSybils.M(1))
		}
	}
}

func recordPeriodicMetrics(hydra *Hydra, period time.Duration) *time.Ticker {
	ticker := time.NewTicker(period)

	go func() {
		for range ticker.C {
			var rts int
			for i := range hydra.Sybils {
				rts += hydra.Sybils[i].RoutingTable.Size()
			}
			stats.Record(context.Background(), metrics.RoutingTableSize.M(int64(rts)))

			prs, err := hydra.Datastore.Query(query.Query{Prefix: "/providers", KeysOnly: true})
			if err == nil {
				// TODO: make fast https://github.com/libp2p/go-libp2p-kad-dht/issues/487
				var provRecords int
				for range prs.Next() {
					provRecords++
				}
				stats.Record(context.Background(), metrics.ProviderRecords.M(int64(provRecords)))
			}

			hydra.hyperLock.Lock()
			uniqPeers := hydra.hyperlog.Estimate()
			hydra.hyperLock.Unlock()
			stats.Record(context.Background(), metrics.UniquePeers.M(int64(uniqPeers)))
		}
	}()

	return ticker
}
