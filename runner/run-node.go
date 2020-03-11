package runner

import (
	"fmt"
	"os"
	"time"

	levelds "github.com/ipfs/go-ds-leveldb"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	dhtmetrics "github.com/libp2p/go-libp2p-kad-dht/metrics"
	"github.com/libp2p/hydra-booster/httpapi"
	"github.com/libp2p/hydra-booster/node"
	hydraNodeOpts "github.com/libp2p/hydra-booster/node/opts"
	"github.com/libp2p/hydra-booster/reports"
	"github.com/libp2p/hydra-booster/ui"
	uiopts "github.com/libp2p/hydra-booster/ui/opts"
	"github.com/multiformats/go-multiaddr"
)

func init() {
	// Allow short keys. Otherwise, we'll refuse connections from the bootsrappers and break the network.
	// TODO: Remove this when we shut those bootstrappers down.
	crypto.MinRsaKeyBits = 1024
}

var _ = dhtmetrics.DefaultViews
var _ = circuit.P_CIRCUIT

const singleDHTSwarmAddr = "/ip4/0.0.0.0/tcp/19264"
const httpAPIAddr = "127.0.0.1:7779"

func handleBootstrapStatus(ch chan node.BootstrapStatus) {
	for {
		status, ok := <-ch
		if !ok {
			return
		}
		if status.Err != nil {
			fmt.Println(status.Err)
		}
	}
}

// Options for RunMany and RunSingle
type Options struct {
	DatastorePath string
	GetPort       func() int
	NSybils       int
	BucketSize    int
	BsCon         int
	Relay         bool
	Stagger       time.Duration
}

// RunMany ...
func RunMany(opts Options) error {
	start := time.Now()

	sharedDatastore, err := levelds.NewDatastore(opts.DatastorePath, nil)
	if err != nil {
		return fmt.Errorf("failed to create datastore: %w", err)
	}

	var nodes []*node.HydraNode

	fmt.Fprintf(os.Stderr, "Running %d DHT Instances:\n", opts.NSybils)

	// What is a limiter?
	limiter := make(chan struct{}, opts.BsCon)

	for i := 0; i < opts.NSybils; i++ {
		time.Sleep(opts.Stagger)
		fmt.Fprintf(os.Stderr, ".")

		addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", opts.GetPort()))
		nd, bsCh, err := node.NewHydraNode(
			hydraNodeOpts.Datastore(sharedDatastore),
			hydraNodeOpts.Addr(addr),
			hydraNodeOpts.Relay(opts.Relay),
			hydraNodeOpts.BucketSize(opts.BucketSize),
			hydraNodeOpts.Limiter(limiter),
		)
		if err != nil {
			return fmt.Errorf("failed to spawn node with swarm address %v: %w", addr, err)
		}
		go handleBootstrapStatus(bsCh)
		nodes = append(nodes, nd)
	}
	fmt.Fprintf(os.Stderr, "\n")

	reporter, err := reports.NewReporter(nodes, time.Second*5)
	if err != nil {
		return err
	}

	var peers []peer.ID
	for _, nd := range nodes {
		peers = append(peers, nd.Host.ID())
	}

	return ui.NewUI(peers, reporter.StatusReports, uiopts.Start(start))
}

// RunSingle ...
func RunSingle(opts Options) error {
	start := time.Now()

	datastore, err := levelds.NewDatastore(opts.DatastorePath, nil)
	if err != nil {
		return fmt.Errorf("failed to create datastore: %w", err)
	}

	addr, _ := multiaddr.NewMultiaddr(singleDHTSwarmAddr)
	nd, bsCh, err := node.NewHydraNode(
		hydraNodeOpts.Datastore(datastore),
		hydraNodeOpts.Addr(addr),
		hydraNodeOpts.Relay(opts.Relay),
		hydraNodeOpts.BucketSize(opts.BucketSize),
	)
	if err != nil {
		return fmt.Errorf("failed to spawn node with swarm address %v: %w", singleDHTSwarmAddr, err)
	}

	go handleBootstrapStatus(bsCh)

	// Launch HTTP API
	go httpapi.ListenAndServe([]*node.HydraNode{nd}, httpAPIAddr)

	nodes := []*node.HydraNode{nd}
	reporter, err := reports.NewReporter(nodes, time.Second*3)
	if err != nil {
		return err
	}

	return ui.NewUI([]peer.ID{nd.Host.ID()}, reporter.StatusReports, uiopts.Start(start))
}
