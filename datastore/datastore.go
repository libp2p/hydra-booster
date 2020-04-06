package datastore

import (
	"context"
	"fmt"
	"time"

	hook "github.com/alanshaw/ipfs-hookds"
	hopts "github.com/alanshaw/ipfs-hookds/opts"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
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

	return hook.NewBatching(ds, hopts.OnAfterGet(newOnAfterGetHook(ctx, getRouting, opts))), nil
}

func newOnAfterGetHook(ctx context.Context, getRouting GetRouting, opts Options) func(datastore.Key, []byte, error) ([]byte, error) {
	findProvsC := make(chan datastore.Key, opts.FindProvidersQueueSize)

	for i := 0; i < opts.FindProvidersConcurrency; i++ {
		go findProviders(ctx, findProvsC, getRouting, opts)
	}

	return func(k datastore.Key, v []byte, err error) ([]byte, error) {
		// if key was not found and the key is for a provider record...
		if err == datastore.ErrNotFound && providersRoot.IsAncestorOf(k) {
			// Send to the find provs queue, if channel is full then discard...
			select {
			case findProvsC <- k:
			default:
			}
		}
		return v, err
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
