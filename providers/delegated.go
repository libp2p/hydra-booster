package providers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-delegated-routing/client"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/multiformats/go-multihash"
)

func DelegateProvider(endpointURL string, timeout time.Duration) (providers.ProviderStore, error) {
	c, err := client.New(endpointURL, client.WithHTTPClient(&http.Client{Timeout: timeout}))
	if err != nil {
		return nil, err
	}
	return &DelegatedProviderStore{c: c}, nil
}

type DelegatedProviderStore struct {
	c client.Client
}

func (s *DelegatedProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return fmt.Errorf("adding providers not supported")
}

func (s *DelegatedProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	h, err := multihash.Cast(key)
	if err != nil {
		return nil, err
	}
	return s.c.FindProviders(ctx, cid.NewCidV1(cid.Raw, h))
}
