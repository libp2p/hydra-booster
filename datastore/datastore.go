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
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/multiformats/go-base32"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// root namespace of provider keys
var providersRoot = datastore.NewKey(providers.ProvidersKeyPrefix)

// GetRouting is a function that returns an appropriate routing module given a CID
type GetRouting func(cid.Cid) (*dht.IpfsDHT, error)

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
	findProvidersTimeout     = time.Minute
)

// NewDatastore creates a new datastore that adds hooks to perform hydra things
func NewDatastore(ctx context.Context, path string, getRouting GetRouting, opts Options) (datastore.Batching, error) {
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

	ds, err := levelds.NewDatastore(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	return hook.NewBatching(ds, hopts.OnAfterQuery(newOnAfterQueryHook(ctx, getRouting, opts))), nil
}

func newOnAfterQueryHook(ctx context.Context, getRouting GetRouting, opts Options) hopts.OnAfterQueryFunc {
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
				stats.Record(ctx, metrics.FindProvsQueueSize.M(int64(len(pending))))
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
		res = hres.NewResults(res, hropts.OnAfterNextSync(func(r query.Result, ok bool) (query.Result, bool) {
			if ok {
				count++
			} else if count == 0 {
				pendingLock.Lock()
				isPending, _ := pending[k.String()]
				if !isPending {
					pending[k.String()] = true
					stats.Record(ctx, metrics.FindProvsQueueSize.M(int64(len(pending))))
					// send to the find provs queue, if channel is full then discard...
					select {
					case findC <- k:
					default:
						stats.RecordWithTags(
							ctx,
							[]tag.Mutator{tag.Upsert(metrics.KeyStatus, "discarded")},
							metrics.FindProvs.M(1),
						)
					}
				}
				pendingLock.Unlock()
			}
			return r, ok
		}))

		return res, nil
	}
}

func findProviders(ctx context.Context, findC chan datastore.Key, foundC chan datastore.Key, getRouting GetRouting, opts Options) {
	for {
		select {
		case k := <-findC:
			cid, err := providerKeyToCID(k)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to create CID from provider record key: %w", err))
				continue
			}

			routing, err := getRouting(cid)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to get routing for CID: %w", err))
				continue
			}

			count := 0
			start := time.Now()
			fctx, cancel := context.WithTimeout(ctx, opts.FindProvidersTimeout)

			for ai := range routing.FindProvidersAsync(fctx, cid, opts.FindProvidersCount) {
				routing.ProviderManager.AddProvider(ctx, cid.Bytes(), ai.ID)
				// fmt.Printf("added provider for %s -> %s (%v)\n", cid, ai.ID, time.Since(start))
				count++
			}

			cancel()

			if count == 0 {
				// fmt.Printf("no providers found for %s (%v)\n", cid, time.Since(start))
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

			select {
			case foundC <- k:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func providerKeyToCID(k datastore.Key) (cid.Cid, error) {
	nss := k.Namespaces()
	if len(nss) < 2 {
		return cid.Undef, fmt.Errorf("not enough namespaces in provider record key")
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
