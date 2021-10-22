package providers

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-delegated-routing/client"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
)

func CombineProviders(backend ...providers.ProviderStore) providers.ProviderStore {
	return &CombinedProviderStore{backends: backend}
}

type CombinedProviderStore struct {
	backends []providers.ProviderStore
}

func (s *CombinedProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	ch := make(chan error, len(s.backends))
	for _, b := range s.backends {
		go func(backend providers.ProviderStore) {
			ch <- backend.AddProvider(ctx, key, prov)
		}(b)
	}
	var errs *multierror.Error
	for range s.backends {
		if e := <-ch; e != nil {
			multierror.Append(errs, e)
		}
	}
	if len(errs.WrappedErrors()) == len(s.backends) {
		return errs
	} else {
		return nil
	}
}

func (s *CombinedProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	ch := make(chan client.FindProvidersAsyncResult, len(s.backends))
	for _, b := range s.backends {
		go func(backend providers.ProviderStore) {
			infos, err := backend.GetProviders(ctx, key)
			ch <- client.FindProvidersAsyncResult{AddrInfo: infos, Err: err}
		}(b)
	}
	infos := []peer.AddrInfo{}
	var errs *multierror.Error
	for range s.backends {
		r := <-ch
		if r.Err == nil {
			infos = append(infos, r.AddrInfo...)
		} else {
			multierror.Append(errs, r.Err)
		}
	}
	if len(errs.WrappedErrors()) == len(s.backends) {
		return infos, errs
	} else {
		return infos, nil
	}
}
