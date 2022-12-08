package providers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ipfs/go-cid"
	drc "github.com/ipfs/go-libipfs/routing/http/client"
	"github.com/ipfs/go-libipfs/routing/http/contentrouter"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

type readContentRouter interface {
	FindProvidersAsync(context.Context, cid.Cid, int) <-chan peer.AddrInfo
}

func NewHTTPProviderStore(httpClient *http.Client, endpointURL string) (*httpProvider, error) {
	drClient, err := drc.New(endpointURL, drc.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("building delegated routing HTTP client: %w", err)
	}
	cr := contentrouter.NewContentRoutingClient(drClient)

	return &httpProvider{
		cr: cr,
	}, nil
}

type httpProvider struct {
	cr readContentRouter
}

func (p *httpProvider) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return nil
}

func (p *httpProvider) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	mh, err := multihash.Cast(key)
	if err != nil {
		return nil, err
	}
	c := cid.NewCidV1(cid.Raw, mh)
	provChan := p.cr.FindProvidersAsync(ctx, c, 100)
	var provs []peer.AddrInfo
	for {
		select {
		case <-ctx.Done():
			return provs, ctx.Err()
		case prov, ok := <-provChan:
			if !ok {
				return provs, nil
			}
			provs = append(provs, prov)
		}
	}
}
