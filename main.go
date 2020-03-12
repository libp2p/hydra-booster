package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	id "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/libp2p/hydra-booster/httpapi"
	"github.com/libp2p/hydra-booster/hydra"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/reports"
	"github.com/libp2p/hydra-booster/ui"
	uiopts "github.com/libp2p/hydra-booster/ui/opts"
	"github.com/libp2p/hydra-booster/utils"
)

const (
	defaultKValue = 20
	httpAPIAddr   = "127.0.0.1:7779"
)

func main() {
	start := time.Now()
	many := flag.Int("many", 1, "Instead of running one dht, run many!")
	dbpath := flag.String("db", "hydra-belly", "Datastore folder path")
	inmem := flag.Bool("mem", false, "Use an in-memory database. This overrides the -db option")
	pprofport := flag.Int("pprof-port", -1, "Specify a port to run pprof http server on")
	relay := flag.Bool("relay", false, "Enable libp2p circuit relaying for this node")
	portBegin := flag.Int("portBegin", 0, "If set, begin port allocation here")
	bucketSize := flag.Int("bucketSize", defaultKValue, "Specify the bucket size")
	bootstrapConcurency := flag.Int("bootstrapConc", 32, "How many concurrent bootstraps to run")
	stagger := flag.Duration("stagger", 0*time.Second, "Duration to stagger nodes starts by")
	noUI := flag.Bool("no-ui", false, "Disable UI")
	flag.Parse()
	// Set the protocol for Identify to report on handshake
	id.ClientVersion = "hydra-booster/1"

	if *relay {
		id.ClientVersion += "+relay"
	}

	if *pprofport > 0 {
		fmt.Printf("Running metrics server on port: %d\n", *pprofport)
		go metrics.SetupMetrics(*pprofport)
	}

	if *inmem {
		*dbpath = ""
	}

	// Allow short keys. Otherwise, we'll refuse connections from the bootsrappers and break the network.
	// TODO: Remove this when we shut those bootstrappers down.
	crypto.MinRsaKeyBits = 1024

	opts := hydra.Options{
		DatastorePath: *dbpath,
		Relay:         *relay,
		BucketSize:    *bucketSize,
		GetPort:       utils.PortSelector(*portBegin),
		NSybils:       *many,
		BsCon:         *bootstrapConcurency,
		Stagger:       *stagger,
	}

	hy, err := hydra.NewHydra(opts)
	if err != nil {
		log.Fatalln(err)
	}

	if !*noUI {
		reporter, err := reports.NewReporter(hy.Sybils, time.Second*3)
		if err != nil {
			log.Fatalln(err)
		}

		var peers []peer.ID
		for _, syb := range hy.Sybils {
			peers = append(peers, syb.Host.ID())
		}

		go ui.NewUI(peers, reporter.StatusReports, uiopts.Start(start))
	}

	fmt.Println(fmt.Sprintf("HTTP API listening on http://%s", httpAPIAddr))
	httpapi.ListenAndServe(hy, httpAPIAddr)
}
