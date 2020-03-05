package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	id "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/libp2p/hydra-booster/hydrabooster"
)

const defaultKValue = 20

func main() {
	many := flag.Int("many", -1, "Instead of running one dht, run many!")
	dbpath := flag.String("db", "dht-data", "Database folder")
	inmem := flag.Bool("mem", false, "Use an in-memory database. This overrides the -db option")
	pprofport := flag.Int("pprof-port", -1, "Specify a port to run pprof http server on")
	relay := flag.Bool("relay", false, "Enable libp2p circuit relaying for this node")
	portBegin := flag.Int("portBegin", 0, "If set, begin port allocation here")
	bucketSize := flag.Int("bucketSize", defaultKValue, "Specify the bucket size")
	bootstrapConcurency := flag.Int("bootstrapConc", 32, "How many concurrent bootstraps to run")
	stagger := flag.Duration("stagger", 0*time.Second, "Duration to stagger nodes starts by")
	flag.Parse()
	// Set the protocol for Identify to report on handshake
	id.ClientVersion = "hydra-booster/1"

	if *relay {
		id.ClientVersion += "+relay"
	}

	if *pprofport > 0 {
		fmt.Printf("Running metrics server on port: %d\n", *pprofport)
		go hydrabooster.SetupMetrics(*pprofport)
	}

	if *inmem {
		*dbpath = ""
	}

	if *many == -1 {
		err := hydrabooster.RunSingleDHTWithUI(*dbpath, *relay, *bucketSize)
		if err != nil {
			log.Fatalln(err)
		}

		return
	}

	getPort := hydrabooster.PortSelector(*portBegin)
	err := hydrabooster.RunMany(*dbpath, getPort, *many, *bucketSize, *bootstrapConcurency, *relay, *stagger)
	if err != nil {
		log.Fatalln(err)
	}
}
