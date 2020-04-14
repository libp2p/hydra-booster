package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/hydra-booster/httpapi"
	"github.com/libp2p/hydra-booster/hydra"
	"github.com/libp2p/hydra-booster/idgen"
	"github.com/libp2p/hydra-booster/metrics"
	hyui "github.com/libp2p/hydra-booster/ui"
	uiopts "github.com/libp2p/hydra-booster/ui/opts"
	"github.com/libp2p/hydra-booster/utils"
)

const (
	defaultBucketSize  = 20
	defaultMetricsAddr = "0.0.0.0:8888"
	defaultHTTPAPIAddr = "127.0.0.1:7779"
)

func main() {
	start := time.Now()
	nheads := flag.Int("nheads", -1, "Specify the number of Hydra heads to create.")
	dbpath := flag.String("db", "hydra-belly", "Datastore folder path")
	httpAPIAddr := flag.String("httpapi-addr", defaultHTTPAPIAddr, "Specify an IP and port to run prometheus metrics and pprof http server on")
	inmem := flag.Bool("mem", false, "Use an in-memory database. This overrides the -db option")
	metricsAddr := flag.String("metrics-addr", defaultMetricsAddr, "Specify an IP and port to run prometheus metrics and pprof http server on")
	relay := flag.Bool("relay", false, "Enable libp2p circuit relaying for this node")
	portBegin := flag.Int("port-begin", -1, "If set, begin port allocation here")
	bucketSize := flag.Int("bucket-size", defaultBucketSize, "Specify the bucket size")
	bootstrapConcurrency := flag.Int("bootstrap-conc", 32, "How many concurrent bootstraps to run")
	stagger := flag.Duration("stagger", 0*time.Second, "Duration to stagger nodes starts by")
	uiTheme := flag.String("ui-theme", "default", "UI theme, \"gooey\", \"logey\" or \"none\" (default \"gooey\" for 1 head otherwise \"logey\")")
	name := flag.String("name", "", "A name for the Hydra (for use in metrics)")
	idgenAddr := flag.String("idgen-addr", "", "Address of an idgen HTTP API endpoint to use for generating private keys for heads")
	flag.Parse()

	if *inmem {
		*dbpath = ""
	}

	if *nheads == -1 {
		*nheads = mustGetEnvInt("HYDRA_NHEADS", 1)
	}

	if *portBegin == -1 {
		*portBegin = mustGetEnvInt("HYDRA_PORT_BEGIN", 0)
	}

	if *name == "" {
		*name = os.Getenv("HYDRA_NAME")
	}

	if *idgenAddr == "" {
		*idgenAddr = os.Getenv("HYDRA_IDGEN_ADDR")
	}

	// Allow short keys. Otherwise, we'll refuse connections from the bootsrappers and break the network.
	// TODO: Remove this when we shut those bootstrappers down.
	crypto.MinRsaKeyBits = 1024

	// Seed the random number generator used by Hydra heads to select a bootstrap peer
	rand.Seed(time.Now().UTC().UnixNano())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var idGenerator idgen.IdentityGenerator
	if *idgenAddr != "" {
		dg := idgen.NewCleaningIDGenerator(idgen.NewDelegatedIDGenerator(*idgenAddr))
		defer func() {
			err := dg.Clean()
			if err != nil {
				fmt.Println(err)
			}
		}()
		idGenerator = dg
	}

	opts := hydra.Options{
		Name:          *name,
		DatastorePath: *dbpath,
		Relay:         *relay,
		BucketSize:    *bucketSize,
		GetPort:       utils.PortSelector(*portBegin),
		NHeads:        *nheads,
		BsCon:         *bootstrapConcurrency,
		Stagger:       *stagger,
		IDGenerator:   idGenerator,
	}

	go func() {
		err := metrics.ListenAndServe(*metricsAddr)
		if err != nil {
			log.Fatalln(err)
		}
	}()
	fmt.Printf("Prometheus metrics and pprof server listening on http://%v\n", *metricsAddr)

	hy, err := hydra.NewHydra(ctx, opts)
	if err != nil {
		log.Fatalln(err)
	}

	var ui *hyui.UI
	if *uiTheme != "none" {
		if *uiTheme == "default" && len(hy.Heads) == 1 {
			*uiTheme = "gooey"
		}
		var theme hyui.Theme
		if *uiTheme == "gooey" {
			theme = hyui.Gooey
		}

		ui, err = hyui.NewUI(theme, uiopts.Start(start), uiopts.MetricsURL(fmt.Sprintf("http://%v/metrics", *metricsAddr)))
		if err != nil {
			log.Fatalln(err)
		}

		go func() {
			err = ui.Render(ctx)
			if err != nil {
				log.Fatalln(err)
			}
		}()
	}

	go func() {
		err := httpapi.ListenAndServe(hy, *httpAPIAddr)
		if err != nil {
			log.Fatalln(err)
		}
	}()
	fmt.Println(fmt.Sprintf("HTTP API listening on http://%s", *httpAPIAddr))

	termChan := make(chan os.Signal)
	signal.Notify(termChan, os.Interrupt, syscall.SIGTERM)
	<-termChan // Blocks here until either SIGINT or SIGTERM is received.
	fmt.Println("Received interrupt signal, shutting down...")
}

func mustGetEnvInt(key string, def int) int {
	if os.Getenv(key) == "" {
		return def
	}
	val, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		log.Fatalln(fmt.Errorf("invalid %s env value: %w", key, err))
	}
	return val
}
