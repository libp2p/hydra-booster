package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	"github.com/multiformats/go-multiaddr"
)

const (
	defaultBucketSize  = 20
	defaultMetricsAddr = "127.0.0.1:9758"
	defaultHTTPAPIAddr = "127.0.0.1:7779"
)

func main() {
	start := time.Now()
	nheads := flag.Int("nheads", -1, "Specify the number of Hydra heads to create.")
	randomSeed := flag.String("random-seed", "", "Seed to use to generate IDs (useful if you want to have persistent IDs). Should be Base64 encoded and 256bits")
	idOffset := flag.Int("id-offset", -1, "What offset in the sequence of keys generated from random-seed to start from")
	dbpath := flag.String("db", "", "Datastore directory (for LevelDB store) or postgresql:// connection URI (for PostgreSQL store) or 'dynamodb://table=<string>'")
	pstorePath := flag.String("pstore", "", "Peerstore directory for LevelDB store (defaults to in-memory store)")
	providerStore := flag.String("provider-store", "", "A non-default provider store to use, either \"none\" or \"dynamodb://table=<string>,ttl=<ttl-in-seconds>,queryLimit=<int>\"")
	httpAPIAddr := flag.String("httpapi-addr", defaultHTTPAPIAddr, "Specify an IP and port to run the HTTP API server on")
	delegateTimeout := flag.Int("delegate-timeout", 0, "Timeout for delegated routing in milliseconds")
	reframeAddr := flag.String("reframe-addr", "", "Reframe API endpoint for delegated routing")
	inmem := flag.Bool("mem", false, "Use an in-memory database. This overrides the -db option")
	metricsAddr := flag.String("metrics-addr", defaultMetricsAddr, "Specify an IP and port to run Prometheus metrics and pprof HTTP server on")
	enableRelay := flag.Bool("enable-relay", false, "Enable libp2p circuit relaying for this node (default false).")
	portBegin := flag.Int("port-begin", -1, "If set, begin port allocation here")
	protocolPrefix := flag.String("protocol-prefix", string(dht.DefaultPrefix), "Specify the DHT protocol prefix (default \"/ipfs\")")
	bucketSize := flag.Int("bucket-size", defaultBucketSize, "Specify the bucket size, note that for some protocols this must be a specific value i.e. for \"/ipfs\" it MUST be 20")
	bootstrapConcurrency := flag.Int("bootstrap-conc", 32, "How many concurrent bootstraps to run")
	bootstrapPeers := flag.String("bootstrap-peers", "", "A CSV list of peer addresses to bootstrap from.")
	stagger := flag.Duration("stagger", 0*time.Second, "Duration to stagger nodes starts by")
	uiTheme := flag.String("ui-theme", "logey", "UI theme, \"logey\", \"gooey\" or \"none\" (default \"logey\")")
	name := flag.String("name", "", "A name for the Hydra (for use in metrics)")
	idgenAddr := flag.String("idgen-addr", "", "Address of an idgen HTTP API endpoint to use for generating private keys for heads")
	disableProvGC := flag.Bool("disable-prov-gc", false, "Disable provider record garbage collection (default false).")
	disableProviders := flag.Bool("disable-providers", false, "Disable storing and retrieving provider records, note that for some protocols, like \"/ipfs\", it MUST be false (default false).")
	disableValues := flag.Bool("disable-values", false, "Disable storing and retrieving value records, note that for some protocols, like \"/ipfs\", it MUST be false (default false).")
	disablePrefetch := flag.Bool("disable-prefetch", false, "Disables pre-fetching of discovered provider records (default false).")
	disableProvCounts := flag.Bool("disable-prov-counts", false, "Disable counting provider records for metrics reporting (default false).")
	disableDBCreate := flag.Bool("disable-db-create", false, "Don't create table and index in the target database (default false).")
	disableResourceManager := flag.Bool("disable-rcmgr", false, "Disable libp2p Resource Manager by configuring it with infinite limits (default false).")
	resourceManagerLimits := flag.String("rcmgr-limits", "", "Resource Manager limits JSON config (default none).")
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
	if *randomSeed == "" {
		*randomSeed = os.Getenv("HYDRA_RANDOM_SEED")
	}
	if *idOffset == -1 {
		*idOffset = mustGetEnvInt("HYDRA_ID_OFFSET", 0)
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
	if !*disableProvGC {
		*disableProvGC = mustGetEnvBool("HYDRA_DISABLE_PROV_GC", false)
	}
	if *bootstrapPeers == "" {
		*bootstrapPeers = os.Getenv("HYDRA_BOOTSTRAP_PEERS")
	}
	if !*disablePrefetch {
		*disablePrefetch = mustGetEnvBool("HYDRA_DISABLE_PREFETCH", false)
	}
	if !*disableDBCreate {
		*disableDBCreate = mustGetEnvBool("HYDRA_DISABLE_DBCREATE", false)
	}
	if !*disableProvCounts {
		*disableProvCounts = mustGetEnvBool("HYDRA_DISABLE_PROV_COUNTS", false)
	}
	if *pstorePath == "" {
		*pstorePath = os.Getenv("HYDRA_PSTORE")
	}
	if *providerStore == "" {
		*providerStore = os.Getenv("HYDRA_PROVIDER_STORE")
	}
	if *delegateTimeout == 0 {
		*delegateTimeout = mustGetEnvInt("HYDRA_DELEGATED_ROUTING_TIMEOUT", 1000)
	}
	if *reframeAddr == "" {
		*reframeAddr = os.Getenv("HYDRA_REFRAME_ADDR")
	}
	if !*disableResourceManager {
		*disableResourceManager = mustGetEnvBool("DISABLE_RCMGR", false)
	}
	if *resourceManagerLimits == "" {
		*resourceManagerLimits = os.Getenv("RCMGR_LIMITS")
	}

	// Allow short keys. Otherwise, we'll refuse connections from the bootsrappers and break the network.
	// TODO: Remove this when we shut those bootstrappers down.
	crypto.MinRsaKeyBits = 1024

	// Seed the random number generator used by Hydra heads to select a bootstrap peer
	rand.Seed(time.Now().UTC().UnixNano())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var idGenerator idgen.IdentityGenerator
	if *randomSeed != "" && *idgenAddr != "" {
		log.Fatalln("error: Should not set both idgen-addr and random-seed")
	}
	if *randomSeed != "" {
		seed, err := base64.StdEncoding.DecodeString(*randomSeed)
		if err != nil {
			log.Fatalln("error: Could not base64 decode seed")
		}
		if len(seed) != 32 {
			log.Fatalln("error: Seed should be 256bit in base64")
		}
		idGenerator = idgen.NewBalancedIdentityGeneratorFromSeed(seed, *idOffset)
	}
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
		Name:                      *name,
		DatastorePath:             *dbpath,
		PeerstorePath:             *pstorePath,
		ProviderStore:             *providerStore,
		DelegateTimeout:           time.Millisecond * time.Duration(*delegateTimeout),
		ReframeAddr:               *reframeAddr,
		EnableRelay:               *enableRelay,
		ProtocolPrefix:            protocol.ID(*protocolPrefix),
		BucketSize:                *bucketSize,
		GetPort:                   utils.PortSelector(*portBegin),
		NHeads:                    *nheads,
		BsCon:                     *bootstrapConcurrency,
		Stagger:                   *stagger,
		IDGenerator:               idGenerator,
		DisableProvGC:             *disableProvGC,
		DisableProviders:          *disableProviders,
		DisableValues:             *disableValues,
		BootstrapPeers:            mustConvertToMultiaddr(*bootstrapPeers),
		DisablePrefetch:           *disablePrefetch,
		DisableProvCounts:         *disableProvCounts,
		DisableDBCreate:           *disableDBCreate,
		DisableResourceManager:    *disableResourceManager,
		ResourceManagerLimitsFile: *resourceManagerLimits,
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

	termChan := make(chan os.Signal, 1)
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

func mustConvertToMultiaddr(csv string) []multiaddr.Multiaddr {
	var peers []multiaddr.Multiaddr
	if csv != "" {
		addrs := strings.Split(csv, ",")
		for _, addr := range addrs {
			ma, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				log.Fatalln(fmt.Errorf("invalid multiaddr %s: %w", addr, err))
			}
			peers = append(peers, ma)
		}
	}
	return peers
}
