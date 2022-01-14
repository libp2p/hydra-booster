package providers

import (
	"context"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/hydra-booster/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// CachingProviderStore wraps a providerstore, and finds providers for CIDs that were requested
// but not found in the underlying providerstore, and then caches them in the providerstore.
type CachingProviderStore struct {
	Delegate        providers.ProviderStore
	Finder          ProvidersFinder
	Router          ReadContentRouting
	ProvidersToFind int
	log             logging.EventLogger
}

func NewCachingProviderStore(delegate providers.ProviderStore, finder ProvidersFinder, router ReadContentRouting) *CachingProviderStore {
	return &CachingProviderStore{
		Delegate: delegate,
		Finder:   finder,
		Router:   router,
		log:      logging.Logger("hydra/providersn"),
	}
}

func (s *CachingProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return s.Delegate.AddProvider(ctx, key, prov)
}

// GetProviders gets providers for the given key from the providerstore.
// If the providerstore does not have providers for the key, then the ProvidersFinder is queried and the results are cached.
func (d *CachingProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	addrInfos, err := d.Delegate.GetProviders(ctx, key)
	if err != nil {
		return addrInfos, err
	}

	if len(addrInfos) > 0 {
		recordPrefetches(ctx, "local")
		return addrInfos, nil
	}

	return nil, d.Finder.Find(ctx, d.Router, key, func(ai peer.AddrInfo) {
		err := d.Delegate.AddProvider(ctx, key, ai)
		if err != nil {
			d.log.Errorf("failed to add provider to providerstore: %s", err)
			stats.Record(ctx, metrics.PrefetchFailedToCache.M(1))
		}
	})
}

func recordPrefetches(ctx context.Context, status string, extraMeasures ...stats.Measurement) {
	stats.RecordWithTags(
		ctx,
		[]tag.Mutator{tag.Upsert(metrics.KeyStatus, status)},
		append([]stats.Measurement{metrics.Prefetches.M(1)}, extraMeasures...)...,
	)
}
