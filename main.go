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
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
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
	dbpath := flag.String("db", "", "Datastore directory (for LevelDB store) or postgresql:// connection URI (for PostgreSQL store)")
	httpAPIAddr := flag.String("httpapi-addr", defaultHTTPAPIAddr, "Specify an IP and port to run prometheus metrics and pprof http server on")
	inmem := flag.Bool("mem", false, "Use an in-memory database. This overrides the -db option")
	metricsAddr := flag.String("metrics-addr", defaultMetricsAddr, "Specify an IP and port to run prometheus metrics and pprof http server on")
	enableRelay := flag.Bool("enable-relay", false, "Enable libp2p circuit relaying for this node (default false).")
	portBegin := flag.Int("port-begin", -1, "If set, begin port allocation here")
	protocolPrefix := flag.String("protocol-prefix", string(dht.DefaultPrefix), "Specify the DHT protocol prefix (default \"/ipfs\")")
	bucketSize := flag.Int("bucket-size", defaultBucketSize, "Specify the bucket size, note that for some protocols this must be a specific value i.e. for \"/ipfs\" it MUST be 20")
	bootstrapConcurrency := flag.Int("bootstrap-conc", 32, "How many concurrent bootstraps to run")
	stagger := flag.Duration("stagger", 0*time.Second, "Duration to stagger nodes starts by")
	uiTheme := flag.String("ui-theme", "default", "UI theme, \"gooey\", \"logey\" or \"none\" (default \"gooey\" for 1 head otherwise \"logey\")")
	name := flag.String("name", "", "A name for the Hydra (for use in metrics)")
	idgenAddr := flag.String("idgen-addr", "", "Address of an idgen HTTP API endpoint to use for generating private keys for heads")
	disableProvGC := flag.Bool("disable-prov-gc", false, "Disable provider record garbage collection (default false).")
	disableProviders := flag.Bool("disable-providers", false, "Disable storing and retrieving provider records, note that for some protocols, like \"/ipfs\", it MUST be false (default false).")
	disableValues := flag.Bool("disable-values", false, "Disable storing and retrieving value records, note that for some protocols, like \"/ipfs\", it MUST be false (default false).")
	enableV1Compat := flag.Bool("enable-v1-compat", false, "Enables DHT v1 compatibility (default false).")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "🐉 Hydra Booster starting up...\n")

	if *inmem {
		*dbpath = ""
	} else if *dbpath == "" {
		*dbpath = os.Getenv("HYDRA_DB")
		if *dbpath == "" {
			*dbpath = "hydra-belly"
		}
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
	if *disableProvGC == false {
		*disableProvGC = mustGetEnvBool("HYDRA_DISABLE_PROV_GC", false)
	}
	if *enableV1Compat == false {
		*enableV1Compat = mustGetEnvBool("HYDRA_ENABLE_V1_COMPAT", false)
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
		Name:             *name,
		DatastorePath:    *dbpath,
		EnableRelay:      *enableRelay,
		ProtocolPrefix:   protocol.ID(*protocolPrefix),
		BucketSize:       *bucketSize,
		GetPort:          utils.PortSelector(*portBegin),
		NHeads:           *nheads,
		BsCon:            *bootstrapConcurrency,
		Stagger:          *stagger,
		IDGenerator:      idGenerator,
		DisableProvGC:    *disableProvGC,
		DisableProviders: *disableProviders,
		DisableValues:    *disableValues,
		EnableV1Compat:   *enableV1Compat,
	}

	go func() {
		err := metrics.ListenAndServe(*metricsAddr)
		if err != nil {
			log.Fatalln(err)
		}
	}()
	fmt.Fprintf(os.Stderr, "📊 Prometheus metrics and pprof server listening on http://%v\n", *metricsAddr)

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
	fmt.Fprintf(os.Stderr, "🧩 HTTP API listening on http://%s\n", *httpAPIAddr)

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

func mustGetEnvBool(key string, def bool) bool {
	if os.Getenv(key) == "" {
		return def
	}
	val, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		log.Fatalln(fmt.Errorf("invalid %s env value: %w", key, err))
	}
	return val
}
