package datastore

import (
	"context"
	"fmt"
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
	"github.com/multiformats/go-base32"
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
}

// option defaults
const (
	findProvidersQueueSize   = 1000
	findProvidersCount       = 1
	findProvidersConcurrency = 1
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

	ds, err := levelds.NewDatastore(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}

	return hook.NewBatching(ds, hopts.OnAfterQuery(newOnAfterQueryHook(ctx, getRouting, opts))), nil
}

func newOnAfterQueryHook(ctx context.Context, getRouting GetRouting, opts Options) func(query.Query, query.Results, error) (query.Results, error) {
	findProvsC := make(chan datastore.Key, opts.FindProvidersQueueSize)

	for i := 0; i < opts.FindProvidersConcurrency; i++ {
		go findProviders(ctx, findProvsC, getRouting, opts)
	}

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
			} else {
				fmt.Printf("query for %v finished with count %d\n", k, count)
				if count == 0 {
					// send to the find provs queue, if channel is full then discard...
					select {
					case findProvsC <- k:
					default:
					}
				}
			}
			return r, ok
		}))

		return res, nil
	}
}

func findProviders(ctx context.Context, findProvsC chan datastore.Key, getRouting GetRouting, opts Options) {
	for {
		select {
		case k := <-findProvsC:
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

			fmt.Printf("finding providers for %s\n", cid)
			start := time.Now()
			for ai := range routing.FindProvidersAsync(ctx, cid, opts.FindProvidersCount) {
				routing.ProviderManager.AddProvider(ctx, cid.Bytes(), ai.ID)
				fmt.Printf("added provider for %s -> %s (%v)\n", cid, ai.ID, time.Since(start))
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
