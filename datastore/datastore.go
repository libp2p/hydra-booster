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

const (
	// total number of find provider queries we should queue
	findProvidersQueueSize = 1000
	// number of providers to find when a provider record does not exist in the store
	findProvidersCount = 1
)

// root namespace of provider keys
var providersRoot = datastore.NewKey(providers.ProvidersKeyPrefix)

// GetRouting is a function that returns an appropriate routing module given a CID
type GetRouting = func(cid.Cid) (*dht.IpfsDHT, error)

// NewDatastore creates a new datastore that adds hooks to perform hydra things
func NewDatastore(ctx context.Context, path string, getRouting GetRouting) (datastore.Batching, error) {
	ds, err := levelds.NewDatastore(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore: %w", err)
	}
	return hook.NewBatching(ds, hopts.OnAfterGet(newOnAfterGetHook(ctx, getRouting))), nil
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

func newOnAfterGetHook(ctx context.Context, getRouting GetRouting) func(datastore.Key, []byte, error) ([]byte, error) {
	findProvsC := make(chan datastore.Key, findProvidersQueueSize)

	// TODO: maybe we can process more than one at a time?
	go func() {
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
				for ai := range routing.FindProvidersAsync(ctx, cid, findProvidersCount) {
					routing.ProviderManager.AddProvider(ctx, cid.Bytes(), ai.ID)
					fmt.Printf("added provider for %s -> %s (%v)\n", cid, ai.ID, time.Since(start))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return func(k datastore.Key, v []byte, err error) ([]byte, error) {
		if err != nil && err != datastore.ErrNotFound {
			return nil, err
		}
		if !providersRoot.IsAncestorOf(k) {
			return v, nil
		}

		// Send to the find provs queue, if channel is full then discard...
		select {
		case findProvsC <- k:
		default:
		}

		return v, nil
	}
}
