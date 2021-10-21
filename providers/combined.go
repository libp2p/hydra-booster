package providers

import (
	"context"

	"github.com/ipfs/go-delegated-routing/client"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
)

type CombinedProviderStore struct {
	Backends []providers.ProviderStore
}

func (s *CombinedProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	ch := make(chan error, len(s.Backends))
	for _, b := range s.Backends {
		go func(backend providers.ProviderStore) {
			ch <- backend.AddProvider(ctx, key, prov)
		}(b)
	}
	for range s.Backends {
		<-ch
	}
	return nil
}

func (s *CombinedProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	ch := make(chan client.FindProvidersAsyncResult, len(s.Backends))
	for _, b := range s.Backends {
		go func(backend providers.ProviderStore) {
			infos, err := backend.GetProviders(ctx, key)
			ch <- client.FindProvidersAsyncResult{AddrInfo: infos, Err: err}
		}(b)
	}
	infos := []peer.AddrInfo{}
	for range s.Backends {
		r := <-ch
		if r.Err == nil {
			infos = append(infos, r.AddrInfo...)
		}
	}
	return infos, nil
}
