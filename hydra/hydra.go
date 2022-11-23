package hydra

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws/session"
	ddbv1 "github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/axiomhq/hyperloglog"
	"github.com/ipfs/go-datastore"
	ddbds "github.com/ipfs/go-ds-dynamodb"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/ipfs/go-libipfs/routing/http/client"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p-peerstore/pstoreds"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	hyds "github.com/libp2p/hydra-booster/datastore"
	"github.com/libp2p/hydra-booster/head"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/libp2p/hydra-booster/idgen"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/metricstasks"
	"github.com/libp2p/hydra-booster/periodictasks"
	hproviders "github.com/libp2p/hydra-booster/providers"
	"github.com/libp2p/hydra-booster/utils"
	"github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// Default intervals between periodic task runs, more cpu/memory intensive tasks are run less frequently
// TODO: expose these as command line options?
const (
	routingTableSizeTaskInterval = 5 * time.Second
	uniquePeersTaskInterval      = 5 * time.Second
	ipnsRecordsTaskInterval      = 15 * time.Minute
)

// Hydra is a container for heads and their shared belly bits.
type Hydra struct {
	Heads           []*head.Head
	SharedDatastore datastore.Datastore
	// SharedRoutingTable *kbucket.RoutingTable

	hyperLock *sync.Mutex
	hyperlog  *hyperloglog.Sketch
}

// Options are configuration for a new hydra.
type Options struct {
	Name                      string
	DatastorePath             string
	PeerstorePath             string
	ProviderStore             string
	DelegateTimeout           time.Duration
	GetPort                   func() int
	NHeads                    int
	ProtocolPrefix            protocol.ID
	BucketSize                int
	BsCon                     int
	EnableRelay               bool
	Stagger                   time.Duration
	IDGenerator               idgen.IdentityGenerator
	DisableProvGC             bool
	DisableProviders          bool
	DisableValues             bool
	BootstrapPeers            []multiaddr.Multiaddr
	DisablePrefetch           bool
	DisableProvCounts         bool
	DisableDBCreate           bool
	DisableResourceManager    bool
	ResourceManagerLimitsFile string
	ConnMgrHighWater          int
	ConnMgrLowWater           int
	ConnMgrGracePeriod        int
}

// NewHydra creates a new Hydra with the passed options.
func NewHydra(ctx context.Context, options Options) (*Hydra, error) {
	if options.Name != "" {
		nctx, err := tag.New(ctx, tag.Insert(metrics.KeyName, options.Name))
		if err != nil {
			return nil, err
		}
		ctx = nctx
	}

	var ds datastore.Batching
	var err error
	if strings.HasPrefix(options.DatastorePath, "postgresql://") {
		fmt.Fprintf(os.Stderr, "🐘 Using PostgreSQL datastore\n")
		ds, err = hyds.NewPostgreSQLDatastore(ctx, options.DatastorePath, !options.DisableDBCreate)
	} else if strings.HasPrefix(options.DatastorePath, "dynamodb://") {
		optsStr := strings.TrimPrefix(options.DatastorePath, "dynamodb://")
		table, err := parseDDBTable(optsStr)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "Using DynamoDB datastore with table '%s'\n", table)
		ddbClient := ddbv1.New(session.Must(session.NewSession()))
		ddbDS := ddbds.New(ddbClient, table, ddbds.WithScanParallelism(5))
		ds = ddbDS
		periodictasks.RunTasks(ctx, []periodictasks.PeriodicTask{metricstasks.NewIPNSRecordsTask(ddbDS, ipnsRecordsTaskInterval)})
	} else {
		fmt.Fprintf(os.Stderr, "🥞 Using LevelDB datastore\n")
		ds, err = leveldb.NewDatastore(options.DatastorePath, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	var hds []*head.Head

	if options.PeerstorePath == "" {
		fmt.Fprintf(os.Stderr, "💭 Using in-memory peerstore\n")
	} else {
		fmt.Fprintf(os.Stderr, "🥞 Using LevelDB peerstore (EXPERIMENTAL)\n")
	}

	if options.IDGenerator == nil {
		options.IDGenerator = idgen.HydraIdentityGenerator
	}
	fmt.Fprintf(os.Stderr, "🐲 Spawning %d heads: \n", options.NHeads)

	var hyperLock sync.Mutex
	hyperlog := hyperloglog.New()

	// What is a limiter?
	limiter := make(chan struct{}, options.BsCon)

	// Increase per-host connection pool since we are making lots of concurrent requests to a small number of hosts.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 500
	transport.MaxIdleConnsPerHost = 100
	limitedTransport := &client.ResponseBodyLimitedTransport{RoundTripper: transport, LimitBytes: 1 << 20}

	delegateHTTPClient := &http.Client{
		Timeout:   options.DelegateTimeout,
		Transport: limitedTransport,
	}

	providerStoreBuilder, err := newProviderStoreBuilder(ctx, delegateHTTPClient, options)
	if err != nil {
		return nil, err
	}

	providersFinder := hproviders.NewAsyncProvidersFinder(5*time.Second, 1000, 1*time.Hour)
	providersFinder.Run(ctx, 1000)

	// Reuse the HTTP client across all the heads.
	for i := 0; i < options.NHeads; i++ {
		time.Sleep(options.Stagger)
		fmt.Fprintf(os.Stderr, ".")

		port := options.GetPort()
		tcpAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
		quicAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port))
		priv, err := options.IDGenerator.AddBalanced()
		if err != nil {
			return nil, fmt.Errorf("failed to generate balanced private key %w", err)
		}
		hdOpts := []opts.Option{
			opts.Datastore(ds),
			opts.ProviderStoreBuilder(providerStoreBuilder),
			opts.Addrs([]multiaddr.Multiaddr{tcpAddr, quicAddr}),
			opts.ProtocolPrefix(options.ProtocolPrefix),
			opts.BucketSize(options.BucketSize),
			opts.Limiter(limiter),
			opts.ID(priv),
			opts.BootstrapPeers(options.BootstrapPeers),
			opts.DelegateHTTPClient(delegateHTTPClient),
			opts.DisableResourceManager(options.DisableResourceManager),
			opts.ResourceManagerLimitsFile(options.ResourceManagerLimitsFile),
			opts.ConnMgrHighWater(options.ConnMgrHighWater),
			opts.ConnMgrLowWater(options.ConnMgrLowWater),
			opts.ConnMgrGracePeriod(options.ConnMgrGracePeriod),
		}
		if options.EnableRelay {
			hdOpts = append(hdOpts, opts.EnableRelay())
		}
		if options.DisableProviders {
			hdOpts = append(hdOpts, opts.DisableProviders())
		}
		if options.DisableValues {
			hdOpts = append(hdOpts, opts.DisableValues())
		}
		if options.DisableProvGC || i > 0 {
			// the first head GCs, if it's enabled
			hdOpts = append(hdOpts, opts.DisableProvGC())
		}
		if options.DisableProvCounts || i > 0 {
			// the first head counts providers, if it's enabled
			hdOpts = append(hdOpts, opts.DisableProvCounts())
		}
		if !options.DisablePrefetch {
			hdOpts = append(hdOpts, opts.ProvidersFinder(providersFinder))
		}
		if options.PeerstorePath != "" {
			pstoreDs, err := leveldb.NewDatastore(fmt.Sprintf("%s/head-%d", options.PeerstorePath, i), nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create peerstore datastore: %w", err)
			}
			pstore, err := pstoreds.NewPeerstore(ctx, pstoreDs, pstoreds.DefaultOpts())
			if err != nil {
				return nil, fmt.Errorf("failed to create peerstore: %w", err)
			}
			hdOpts = append(hdOpts, opts.Peerstore(pstore))
		}

		hd, bsCh, err := head.NewHead(ctx, hdOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn node with swarm addresses %v %v: %w", tcpAddr, quicAddr, err)
		}

		hdCtx, err := tag.New(ctx, tag.Insert(metrics.KeyPeerID, hd.Host.ID().String()))
		if err != nil {
			return nil, err
		}

		stats.Record(hdCtx, metrics.Heads.M(1))

		hd.Host.Network().Notify(&network.NotifyBundle{
			ConnectedF: func(n network.Network, v network.Conn) {
				hyperLock.Lock()
				hyperlog.Insert([]byte(v.RemotePeer()))
				hyperLock.Unlock()
				stats.Record(hdCtx, metrics.ConnectedPeers.M(1))
			},
			DisconnectedF: func(n network.Network, v network.Conn) {
				stats.Record(hdCtx, metrics.ConnectedPeers.M(-1))
			},
		})

		go handleBootstrapStatus(hdCtx, bsCh)

		hds = append(hds, hd)
	}
	fmt.Fprintf(os.Stderr, "\n")

	for _, hd := range hds {
		fmt.Fprintf(os.Stderr, "🆔 %v\n", hd.Host.ID())
		for _, addr := range hd.Host.Addrs() {
			fmt.Fprintf(os.Stderr, "🐝 Swarm listening on %v\n", addr)
		}
	}

	hydra := Hydra{
		Heads:           hds,
		SharedDatastore: ds,
		hyperLock:       &hyperLock,
		hyperlog:        hyperlog,
	}

	tasks := []periodictasks.PeriodicTask{
		metricstasks.NewRoutingTableSizeTask(hydra.GetRoutingTableSize, routingTableSizeTaskInterval),
		metricstasks.NewUniquePeersTask(hydra.GetUniquePeersCount, uniquePeersTaskInterval),
	}

	periodictasks.RunTasks(ctx, tasks)

	return &hydra, nil
}

func newProviderStoreBuilder(ctx context.Context, httpClient *http.Client, options Options) (opts.ProviderStoreBuilderFunc, error) {
	if options.ProviderStore == "none" {
		return func(opts opts.Options, host host.Host) (providers.ProviderStore, error) {
			return &hproviders.NoopProviderStore{}, nil
		}, nil
	}
	if strings.HasPrefix(options.ProviderStore, "https://") {
		return func(opts opts.Options, host host.Host) (providers.ProviderStore, error) {
			fmt.Printf("Using HTTP provider store\n")
			return hproviders.NewHTTPProviderStore(httpClient, options.ProviderStore)
		}, nil
	}
	if strings.HasPrefix(options.ProviderStore, "dynamodb://") {
		// dynamodb,table=<table>,ttl=<ttl>,queryLimit=<queryLimit>
		ddbOpts, err := utils.ParseOptsString(strings.TrimPrefix(options.ProviderStore, "dynamodb://"))
		if err != nil {
			return nil, fmt.Errorf("parsing DynamoDB config string: %w", err)
		}
		table := ddbOpts["table"]
		if table == "" {
			return nil, errors.New("DynamoDB table must be specified")
		}
		ttlStr := ddbOpts["ttl"]
		if ttlStr == "" {
			return nil, errors.New("DynamoDB TTL must be specified")
		}
		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			return nil, fmt.Errorf("parsing DynamoDB TTL: %w", err)
		}

		queryLimitStr := ddbOpts["queryLimit"]
		if queryLimitStr == "" {
			return nil, errors.New("DynamoDB query limit must be specified")
		}
		queryLimit64, err := strconv.ParseInt(queryLimitStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parsing DynamoDB query limit: %w", err)
		}
		queryLimit := int32(queryLimit64)

		fmt.Fprintf(os.Stderr, "🥞 Using DynamoDB providerstore with table=%s, ttl=%s, queryLimit=%d\n", table, ttl, queryLimit)
		awsCfg, err := config.LoadDefaultConfig(ctx,
			config.WithRetryer(func() aws.Retryer {
				return retry.NewStandard(func(so *retry.StandardOptions) { so.MaxAttempts = 1 })
			}))
		if err != nil {
			return nil, fmt.Errorf("loading AWS config: %w", err)
		}
		awsCfg.APIOptions = append(awsCfg.APIOptions, metrics.AddAWSSDKMiddleware)

		// reuse the client across all the heads
		ddbClient := dynamodb.NewFromConfig(awsCfg)

		return func(opts opts.Options, h host.Host) (providers.ProviderStore, error) {
			return hproviders.NewDynamoDBProviderStore(h.ID(), h.Peerstore(), ddbClient, table, ttl, queryLimit), nil
		}, nil
	}
	return nil, nil
}

func handleBootstrapStatus(ctx context.Context, ch chan head.BootstrapStatus) {
	for status := range ch {
		if status.Err != nil {
			fmt.Println(status.Err)
		}
		if status.Done {
			stats.Record(ctx, metrics.BootstrappedHeads.M(1))
		}
	}
}

func parseDDBTable(optsStr string) (string, error) {
	opts, err := utils.ParseOptsString(optsStr)
	if err != nil {
		return "", fmt.Errorf("parsing DynamoDB config string: %w", err)
	}
	table, ok := opts["table"]
	if !ok {
		return "", errors.New("must specify table in DynamoDB opts string")
	}
	return table, nil
}

// GetUniquePeersCount retrieves the current total for unique peers
func (hy *Hydra) GetUniquePeersCount() uint64 {
	hy.hyperLock.Lock()
	defer hy.hyperLock.Unlock()
	return hy.hyperlog.Estimate()
}

func (hy *Hydra) GetRoutingTableSize() int {
	var rts int
	for i := range hy.Heads {
		rts += hy.Heads[i].RoutingTable().Size()
	}
	return rts
}
