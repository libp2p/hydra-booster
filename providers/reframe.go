package providers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-delegated-routing/client"
	"github.com/ipfs/go-delegated-routing/gen/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/multiformats/go-multihash"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func NewReframeProviderStore(httpClient *http.Client, endpointURL string) (*reframeProvider, error) {
	q, err := proto.New_DelegatedRouting_Client(endpointURL, proto.DelegatedRouting_Client_WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}
	return &reframeProvider{
		reframe: client.NewClient(q),
	}, nil
}

type reframeProvider struct {
	reframe *client.Client
}

func (x *reframeProvider) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return fmt.Errorf("reframe does not support adding providers")
}

func (x *reframeProvider) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	mh, err := multihash.Cast(key)
	if err != nil {
		return nil, err
	}
	cid1 := cid.NewCidV1(cid.Raw, mh)
	start := time.Now()
	peers, err := x.reframe.FindProviders(ctx, cid1)
	if err != nil {
		recordReframeFindProvsComplete(ctx, "reframe server returned an error", time.Since(start))
	} else {
		recordReframeFindProvsComplete(ctx, "success", time.Since(start))
	}
	return peers, err
}

func recordReframeFindProvsComplete(ctx context.Context, status string, duration time.Duration) {
	stats.RecordWithTags(
		ctx,
		[]tag.Mutator{tag.Upsert(metrics.KeyStatus, status)},
		[]stats.Measurement{
			metrics.STIFindProvs.M(1),
			metrics.STIFindProvsDuration.M(float64(duration)),
		}...,
	)
}
