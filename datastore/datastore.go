package datastore

import (
	"context"
	"fmt"
	"sync"
	"time"

	hook "github.com/alanshaw/ipfs-hookds"
	hopts "github.com/alanshaw/ipfs-hookds/opts"
	hres "github.com/alanshaw/ipfs-hookds/query/results"
	hropts "github.com/alanshaw/ipfs-hookds/query/results/opts"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	levelds "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/multiformats/go-base32"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// root namespace of provider keys
var providersRoot = datastore.NewKey(providers.ProvidersKeyPrefix)

// AddProviderFunc adds a provider for a given CID to the datastore
type AddProviderFunc func(ctx context.Context, c cid.Cid, id peer.ID)

// GetRoutingFunc is a function that returns an appropriate routing module given a CID
type GetRoutingFunc func(cid.Cid) (routing.Routing, AddProviderFunc, error)

// Options are options for the Hydra datastore
type Options struct {
	// total number of find provider queries we should queue
	FindProvidersQueueSize int
	// number of providers to find when a provider record does not exist in the store
	FindProvidersCount int
	// number of find provider requests we will concurrently process
	FindProvidersConcurrency int
	// maximum time a find providers call is allowed to take
	FindProvidersTimeout time.Duration
}

// option defaults
const (
	findProvidersQueueSize   = 1000
	findProvidersCount       = 1
	findProvidersConcurrency = 1
	findProvidersTimeout     = time.Second * 10
)

// NewDatastore creates a new datastore that adds hooks to perform hydra things
func NewDatastore(ctx context.Context, path string, getRouting GetRoutingFunc, opts Options) (datastore.Batching, error) {
	opts = setOptionDefaults(opts)
	ds, err := levelds.NewDatastore(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	return hook.NewBatching(ds, hopts.OnAfterQuery(newOnAfterQueryHook(ctx, getRouting, opts))), nil
}

func setOptionDefaults(opts Options) Options {
	if opts.FindProvidersConcurrency == 0 {
		opts.FindProvidersConcurrency = findProvidersConcurrency
	}
	if opts.FindProvidersCount == 0 {
		opts.FindProvidersCount = findProvidersCount
	}
	if opts.FindProvidersQueueSize == 0 {
		opts.FindProvidersQueueSize = findProvidersQueueSize
	}
	if opts.FindProvidersTimeout == 0 {
		opts.FindProvidersTimeout = findProvidersTimeout
	}
	return opts
}

func newOnAfterQueryHook(ctx context.Context, getRouting GetRoutingFunc, opts Options) hopts.OnAfterQueryFunc {
	findC := make(chan datastore.Key, opts.FindProvidersQueueSize)
	foundC := make(chan datastore.Key)

	pending := make(map[string]bool)
	var pendingLock sync.Mutex

	for i := 0; i < opts.FindProvidersConcurrency; i++ {
		go findProviders(ctx, findC, foundC, getRouting, opts)
	}

	go func() {
		for {
			select {
			case k := <-foundC:
				pendingLock.Lock()
				delete(pending, k.String())
				stats.Record(ctx, metrics.FindProvsQueueSize.M(-1))
				pendingLock.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	return func(q query.Query, res query.Results, err error) (query.Results, error) {
		if err != nil {
			return res, err
		}

		k := datastore.NewKey(q.Prefix)

		// not interested if this is not a query for providers
		if !providersRoot.IsAncestorOf(k) {
			return res, err
		}

		var count int
		onAfterNextSync := func(r query.Result, ok bool) (query.Result, bool) {
			if ok {
				count++
				return r, ok
			}
			if count > 0 { // query has ended and there were found records
				return r, ok
			}

			pendingLock.Lock()
			isPending, _ := pending[k.String()]
			if !isPending {
				// send to the find provs queue, if channel is full then discard...
				select {
				case findC <- k:
					pending[k.String()] = true
					stats.Record(ctx, metrics.FindProvsQueueSize.M(1))
				case <-ctx.Done():
				default:
					stats.RecordWithTags(
						ctx,
						[]tag.Mutator{tag.Upsert(metrics.KeyStatus, "discarded")},
						metrics.FindProvs.M(1),
					)
				}
			}
			pendingLock.Unlock()

			return r, ok
		}

		return hres.NewResults(res, hropts.OnAfterNextSync(onAfterNextSync)), nil
	}
}

func findProviders(ctx context.Context, findC chan datastore.Key, foundC chan datastore.Key, getRouting GetRoutingFunc, opts Options) {
	done := func(k datastore.Key) {
		select {
		case foundC <- k:
		case <-ctx.Done():
		}
	}

	for {
		select {
		case k := <-findC:
			cid, err := providerKeyToCID(k)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to create CID from provider record key: %w", err))
				done(k)
				continue
			}

			routing, addProvider, err := getRouting(cid)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to get routing for CID: %w", err))
				done(k)
				continue
			}

			count := 0
			start := time.Now()
			fctx, cancel := context.WithTimeout(ctx, opts.FindProvidersTimeout)

			for ai := range routing.FindProvidersAsync(fctx, cid, opts.FindProvidersCount) {
				addProvider(ctx, cid, ai.ID)
				count++
			}

			cancel()

			if count == 0 {
				stats.RecordWithTags(
					ctx,
					[]tag.Mutator{tag.Upsert(metrics.KeyStatus, "failed")},
					metrics.FindProvs.M(1),
					metrics.FindProvsDuration.M(float64(time.Since(start)/1e+9)),
				)
			} else {
				stats.RecordWithTags(
					ctx,
					[]tag.Mutator{tag.Upsert(metrics.KeyStatus, "succeeded")},
					metrics.FindProvs.M(1),
					metrics.FindProvsDuration.M(float64(time.Since(start)/1e+9)),
				)
			}

			done(k)
		case <-ctx.Done():
			return
		}
	}
}

var errInvalidKeyNamespaces = fmt.Errorf("not enough namespaces in provider record key")

func providerKeyToCID(k datastore.Key) (cid.Cid, error) {
	nss := k.Namespaces()
	if len(nss) < 2 {
		return cid.Undef, errInvalidKeyNamespaces
	}

	b, err := base32.RawStdEncoding.DecodeString(nss[1])
	if err != nil {
		return cid.Undef, err
	}

	_, c, err := cid.CidFromBytes(b)
	if err != nil {
		return cid.Undef, err
	}

	return c, nil
}
