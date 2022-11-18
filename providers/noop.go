package providers

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
)

type NoopProviderStore struct{}

func (s *NoopProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return nil
}

func (s *NoopProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	return nil, nil
}
