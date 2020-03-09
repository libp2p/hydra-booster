package runner

import (
	"fmt"
	"os"
	"time"

	levelds "github.com/ipfs/go-ds-leveldb"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/crypto"
	dhtmetrics "github.com/libp2p/go-libp2p-kad-dht/metrics"
	"github.com/libp2p/hydra-booster/httpapi"
	"github.com/libp2p/hydra-booster/node"
	"github.com/libp2p/hydra-booster/opts"
	"github.com/libp2p/hydra-booster/reports"
	"github.com/libp2p/hydra-booster/ui"
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

// RunMany ...
func RunMany(dbpath string, getPort func() int, many, bucketSize, bsCon int, relay bool, stagger time.Duration) error {
	start := time.Now()

	sharedDatastore, err := levelds.NewDatastore(dbpath, nil)
	if err != nil {
		return fmt.Errorf("failed to create datastore: %w", err)
	}

	var nodes []*node.HydraNode

	fmt.Fprintf(os.Stderr, "Running %d DHT Instances:\n", many)

	limiter := make(chan struct{}, bsCon)
	for i := 0; i < many; i++ {
		time.Sleep(stagger)
		fmt.Fprintf(os.Stderr, ".")

		addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", getPort()))
		nd, bsCh, err := node.NewHydraNode(
			opts.Datastore(sharedDatastore),
			opts.Addr(addr),
			opts.Relay(relay),
			opts.BucketSize(bucketSize),
			opts.Limiter(limiter),
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

	return ui.NewUI(nodes, reporter.StatusReports, start)
}

// RunSingle ...
func RunSingle(path string, relay bool, bucketSize int) error {
	start := time.Now()

	datastore, err := levelds.NewDatastore(path, nil)
	if err != nil {
		return fmt.Errorf("failed to create datastore: %w", err)
	}

	addr, _ := multiaddr.NewMultiaddr(singleDHTSwarmAddr)
	nd, bsCh, err := node.NewHydraNode(
		opts.Datastore(datastore),
		opts.Addr(addr),
		opts.Relay(relay),
		opts.BucketSize(bucketSize),
	)
	if err != nil {
		return fmt.Errorf("failed to spawn node with swarm address %v: %w", singleDHTSwarmAddr, err)
	}

	go handleBootstrapStatus(bsCh)

	// Simple endpoint to report the addrs of the sybils that were launched
	go httpapi.ListenAndServe([]*node.HydraNode{nd}, httpAPIAddr)

	nodes := []*node.HydraNode{nd}
	reporter, err := reports.NewReporter(nodes, time.Second*3)
	if err != nil {
		return err
	}

	return ui.NewUI(nodes, reporter.StatusReports, start)
}
