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
	"github.com/libp2p/hydra-booster/reports"
	"github.com/libp2p/hydra-booster/sybil"
	sybopts "github.com/libp2p/hydra-booster/sybil/opts"
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

func handleBootstrapStatus(ch chan sybil.BootstrapStatus) {
	for status := range ch {
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

	var sybils []*sybil.Sybil
	var peers []peer.ID

	fmt.Fprintf(os.Stderr, "Running %d DHT Instances:\n", opts.NSybils)

	// What is a limiter?
	limiter := make(chan struct{}, opts.BsCon)

	for i := 0; i < opts.NSybils; i++ {
		time.Sleep(opts.Stagger)
		fmt.Fprintf(os.Stderr, ".")

		addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", opts.GetPort()))
		syb, bsCh, err := sybil.NewSybil(
			sybopts.Datastore(sharedDatastore),
			sybopts.Addr(addr),
			sybopts.Relay(opts.Relay),
			sybopts.BucketSize(opts.BucketSize),
			sybopts.Limiter(limiter),
		)
		if err != nil {
			return fmt.Errorf("failed to spawn node with swarm address %v: %w", addr, err)
		}
		go handleBootstrapStatus(bsCh)
		sybils = append(sybils, syb)
		peers = append(peers, syb.Host.ID())
	}
	fmt.Fprintf(os.Stderr, "\n")

	reporter, err := reports.NewReporter(sybils, time.Second*5)
	if err != nil {
		return err
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
	syb, bsCh, err := sybil.NewSybil(
		sybopts.Datastore(datastore),
		sybopts.Addr(addr),
		sybopts.Relay(opts.Relay),
		sybopts.BucketSize(opts.BucketSize),
	)
	if err != nil {
		return fmt.Errorf("failed to spawn node with swarm address %v: %w", singleDHTSwarmAddr, err)
	}

	go handleBootstrapStatus(bsCh)

	// Launch HTTP API
	go httpapi.ListenAndServe([]*sybil.Sybil{syb}, httpAPIAddr)

	nodes := []*sybil.Sybil{syb}
	reporter, err := reports.NewReporter(nodes, time.Second*3)
	if err != nil {
		return err
	}

	return ui.NewUI([]peer.ID{syb.Host.ID()}, reporter.StatusReports, uiopts.Start(start))
}
