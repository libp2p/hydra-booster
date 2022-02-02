package storetheindex

import (
	"context"
	"net/http"
	"net/url"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multihash"
)

type Client interface {
	FindProviders(ctx context.Context, mh multihash.Multihash) ([]peer.AddrInfo, error)
}

type Option func(*client) error

type client struct {
	client   *http.Client
	endpoint *url.URL
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *client) error {
		c.client = hc
		return nil
	}
}

func New(endpoint string, opts ...Option) (*client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	c := &client{endpoint: u, client: http.DefaultClient}
	for _, o := range opts {
		if err := o(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}
