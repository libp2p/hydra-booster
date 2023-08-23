package providers

import (
	"context"

	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/peer"
)

func AddProviderNotSupported(backend providers.ProviderStore) providers.ProviderStore {
	return &AddProviderNotSupportedProviderStore{backend: backend}
}

type AddProviderNotSupportedProviderStore struct {
	backend providers.ProviderStore
}

func (s *AddProviderNotSupportedProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	return nil
}

func (s *AddProviderNotSupportedProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	return s.backend.GetProviders(ctx, key)
}
