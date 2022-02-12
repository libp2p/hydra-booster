package providers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/providers/storetheindex"
	"github.com/multiformats/go-multihash"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func StoreTheIndexProvider(endpointURL string, timeout time.Duration) (providers.ProviderStore, error) {
	c, err := storetheindex.New(endpointURL, storetheindex.WithHTTPClient(&http.Client{Timeout: timeout}))
	if err != nil {
		return nil, err
	}
	return &StoreTheIndexProviderStore{c: c}, nil
}

type StoreTheIndexProviderStore struct {
	c storetheindex.Client
}

func (s *StoreTheIndexProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return fmt.Errorf("adding providers not supported")
}

func (s *StoreTheIndexProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	h, err := multihash.Cast(key)
	if err != nil {
		return nil, err
	}
	t0 := time.Now()
	infos, err := s.c.FindProviders(ctx, h)
	dur := time.Now().Sub(t0)
	status := "success"
	if err != nil {
		status = err.Error()
	}
	recordSTIFindProvsComplete(ctx, status, metrics.STIFindProvsDuration.M(float64(dur)))
	return infos, err
}

func recordSTIFindProvsComplete(ctx context.Context, status string, extraMeasures ...stats.Measurement) {
	stats.RecordWithTags(
		ctx,
		[]tag.Mutator{tag.Upsert(metrics.KeyStatus, status)},
		append([]stats.Measurement{metrics.STIFindProvs.M(1)}, extraMeasures...)...,
	)
}
