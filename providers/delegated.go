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
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/multiformats/go-multihash"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
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
	t0 := time.Now()
	infos, err := s.c.FindProviders(ctx, cid.NewCidV1(cid.Raw, h))
	dur := time.Now().Sub(t0)
	recordFindProvsComplete(ctx, statusFromErr(err), metrics.DelegatedFindProvsDuration.M(float64(dur)))
	return infos, err
}

func recordFindProvsComplete(ctx context.Context, status string, extraMeasures ...stats.Measurement) {
	stats.RecordWithTags(
		ctx,
		[]tag.Mutator{tag.Upsert(metrics.KeyStatus, status)},
		append([]stats.Measurement{metrics.DelegatedFindProvs.M(1)}, extraMeasures...)...,
	)
}

func statusFromErr(err error) string {
	if err != nil {
		return "failure"
	}
	return "success"
}
