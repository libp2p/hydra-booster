package providers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-delegated-routing/client"
	"github.com/ipfs/go-delegated-routing/gen/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multihash"
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
	return x.reframe.FindProviders(ctx, cid1)
}
