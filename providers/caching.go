package providers

import (
	"context"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/hydra-booster/metrics"
	"go.opencensus.io/stats"
)

// CachingProviderStore checks the ReadProviderStore for providers. If no providers are returned,
// then the Finder is used to find providers, which are then added to the WriteProviderStore.
type CachingProviderStore struct {
	ReadProviderStore  providers.ProviderStore
	WriteProviderStore providers.ProviderStore
	Finder             ProvidersFinder
	Router             ReadContentRouting
	log                logging.EventLogger
}

func NewCachingProviderStore(getDelegate providers.ProviderStore, addDelegate providers.ProviderStore, finder ProvidersFinder, router ReadContentRouting) *CachingProviderStore {
	return &CachingProviderStore{
		ReadProviderStore:  getDelegate,
		WriteProviderStore: addDelegate,
		Finder:             finder,
		Router:             router,
		log:                logging.Logger("hydra/providers"),
	}
}

func (s *CachingProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return s.WriteProviderStore.AddProvider(ctx, key, prov)
}

// GetProviders gets providers for the given key from the providerstore.
// If the providerstore does not have providers for the key, then the ProvidersFinder is queried and the results are cached.
func (d *CachingProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	addrInfos, err := d.ReadProviderStore.GetProviders(ctx, key)
	if err != nil {
		return addrInfos, err
	}

	if len(addrInfos) > 0 {
		return addrInfos, nil
	}

	return nil, d.Finder.Find(ctx, d.Router, key, func(ai peer.AddrInfo) {
		err := d.WriteProviderStore.AddProvider(ctx, key, ai)
		if err != nil {
			d.log.Errorf("failed to add provider to providerstore: %s", err)
			stats.Record(ctx, metrics.PrefetchFailedToCache.M(1))
		}
	})
}
