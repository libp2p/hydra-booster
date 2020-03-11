package hydra

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/axiomhq/hyperloglog"
	"github.com/ipfs/go-datastore"
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
	Heads        []*node.HydraNode
	Datastore    datastore.Datastore
	RoutingTable *kbucket.RoutingTable
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
				stats.Record(ctx, metrics.ConnectedPeers.M(int64(len(n.Peers()))))
			},
			DisconnectedF: func(n network.Network, v network.Conn) {
				stats.Record(ctx, metrics.ConnectedPeers.M(int64(len(n.Peers()))))
			},
		})

		go handleBootstrapStatus(ctx, bsCh)

		nodes = append(nodes, nd)
	}
	fmt.Fprintf(os.Stderr, "\n")

	return &Hydra{
		Heads:     nodes,
		Datastore: datastore,
	}, nil
}

func handleBootstrapStatus(ctx context.Context, ch chan node.BootstrapStatus) {
	for {
		status, ok := <-ch
		if !ok {
			break
		}
		if status.Err != nil {
			fmt.Println(status.Err)
		}
		if status.Done {
			stats.Record(ctx, metrics.BootstrappedSybils.M(1))
		}
	}
}
