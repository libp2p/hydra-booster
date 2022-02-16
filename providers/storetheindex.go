package providers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/providers/storetheindex"
	"github.com/multiformats/go-multihash"
)

// NewStoreTheIndexProviderStore creates a new STI provider store.
// If the cache writer is not nil, then this will ask it to find and cache providers that STI doesn't know about,
// and AddProvider() calls will also be forwarded to it.
// If the cache writer is nil, then AddProvider() calls will go to the delegate.
func NewStoreTheIndexProviderStore(httpClient *http.Client, endpointURL string) (*storeTheIndexProviderStore, error) {
	c, err := storetheindex.New(endpointURL, storetheindex.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}
	return &storeTheIndexProviderStore{c: c}, nil
}

type storeTheIndexProviderStore struct {
	c storetheindex.Client
}

// AddProvider adds the provider to the cache writer. If there is no cache writer, it's added to the delegate instead.
func (s *storeTheIndexProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return fmt.Errorf("StoreTheIndex does not support adding providers")
}

func (s *storeTheIndexProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	mh, err := multihash.Cast(key)
	if err != nil {
		return nil, err
	}

	return s.c.FindProviders(ctx, mh)
}
