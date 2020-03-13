package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	id "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/libp2p/hydra-booster/httpapi"
	"github.com/libp2p/hydra-booster/hydra"
	"github.com/libp2p/hydra-booster/metrics"
	hyui "github.com/libp2p/hydra-booster/ui"
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
	metricsPort := flag.Int("metrics-port", 8888, "Specify a port to run prometheus metrics and pprof http server on")
	relay := flag.Bool("relay", false, "Enable libp2p circuit relaying for this node")
	portBegin := flag.Int("portBegin", 0, "If set, begin port allocation here")
	bucketSize := flag.Int("bucketSize", defaultKValue, "Specify the bucket size")
	bootstrapConcurency := flag.Int("bootstrapConc", 32, "How many concurrent bootstraps to run")
	stagger := flag.Duration("stagger", 0*time.Second, "Duration to stagger nodes starts by")
	uiTheme := flag.String("ui-theme", "default", "UI theme, \"gooey\", \"logey\" or \"none\" (default \"gooey\" for 1 sybil otherwise \"logey\")")
	flag.Parse()
	// Set the protocol for Identify to report on handshake
	id.ClientVersion = "hydra-booster/1"

	if *relay {
		id.ClientVersion += "+relay"
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

	if *metricsPort > 0 {
		go func() {
			err := metrics.ListenAndServe(*metricsPort)
			if err != nil {
				log.Fatalln(err)
			}
		}()
		fmt.Printf("Prometheus metrics and pprof server listening on http://0.0.0.0:%d\n", *metricsPort)
	}

	hy, err := hydra.NewHydra(opts)
	if err != nil {
		log.Fatalln(err)
	}
	defer hy.Stop()

	var ui *hyui.UI
	if *uiTheme != "none" {
		if *uiTheme == "default" && len(hy.Sybils) == 1 {
			*uiTheme = "gooey"
		}
		var theme hyui.Theme
		if *uiTheme == "gooey" {
			theme = hyui.Gooey
		}

		ui, err = hyui.NewUI(theme, uiopts.Start(start), uiopts.MetricsURL(fmt.Sprintf("http://127.0.0.1:%v/metrics", *metricsPort)))
		if err != nil {
			log.Fatalln(err)
		}
		defer ui.Stop()

		go func() {
			err = ui.Render()
			if err != nil {
				log.Fatalln(err)
			}
		}()
	}

	go func() {
		err := httpapi.ListenAndServe(hy, httpAPIAddr)
		if err != nil {
			log.Fatalln(err)
		}
	}()
	fmt.Println(fmt.Sprintf("HTTP API listening on http://%s", httpAPIAddr))

	termChan := make(chan os.Signal)
	signal.Notify(termChan, os.Interrupt, syscall.SIGTERM)
	<-termChan // Blocks here until either SIGINT or SIGTERM is received.
	fmt.Println("Received interrupt signal, shutting down...")
}
