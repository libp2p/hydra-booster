package hydra

import (
	"fmt"
	"os"
	"time"

	"github.com/ipfs/go-datastore"
	levelds "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/hydra-booster/sybil"
	"github.com/libp2p/hydra-booster/sybil/opts"
	"github.com/multiformats/go-multiaddr"
)

// Hydra is a container for heads (sybils) and their shared belly bits.
type Hydra struct {
	Sybils          []*sybil.Sybil
	SharedDatastore datastore.Datastore
	// SharedRoutingTable *kbucket.RoutingTable
}

// Options are configuration for a new hydra
type Options struct {
	DatastorePath string
	GetPort       func() int
	NSybils       int
	BucketSize    int
	BsCon         int
	Relay         bool
	Stagger       time.Duration
}

// NewHydra creates a new Hydra with the passed options.
func NewHydra(options Options) (*Hydra, error) {
	datastore, err := levelds.NewDatastore(options.DatastorePath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	var sybils []*sybil.Sybil

	fmt.Fprintf(os.Stderr, "Running %d DHT Instances:\n", options.NSybils)

	// What is a limiter?
	limiter := make(chan struct{}, options.BsCon)

	for i := 0; i < options.NSybils; i++ {
		time.Sleep(options.Stagger)
		fmt.Fprintf(os.Stderr, ".")

		addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", options.GetPort()))
		syb, bsCh, err := sybil.NewSybil(
			opts.Datastore(datastore),
			opts.Addr(addr),
			opts.Relay(options.Relay),
			opts.BucketSize(options.BucketSize),
			opts.Limiter(limiter),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn node with swarm address %v: %w", addr, err)
		}

		go handleBootstrapStatus(bsCh)

		sybils = append(sybils, syb)
	}
	fmt.Fprintf(os.Stderr, "\n")

	hydra := Hydra{
		Sybils:          sybils,
		SharedDatastore: datastore,
	}

	return &hydra, nil
}

func handleBootstrapStatus(ch chan sybil.BootstrapStatus) {
	for status := range ch {
		if status.Err != nil {
			fmt.Println(status.Err)
		}
	}
}
