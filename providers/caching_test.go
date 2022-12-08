package providers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

type mockProviderStore struct {
	providers map[string][]peer.AddrInfo
	err       error
}

func (m *mockProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	if m.err != nil {
		return m.err
	}
	if m.providers == nil {
		m.providers = map[string][]peer.AddrInfo{}
	}
	m.providers[string(key)] = append(m.providers[string(key)], prov)
	return nil
}

func (m *mockProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.providers[string(key)], nil
}

type mockFinder struct {
	providers map[string][]peer.AddrInfo
}

func (m *mockFinder) Find(ctx context.Context, router ReadContentRouting, key []byte, onProvider onProviderFunc) error {
	for _, ai := range m.providers[string(key)] {
		onProvider(ai)
	}
	return nil
}

func TestCachingProviderStore_GetProviders(t *testing.T) {
	cases := []struct {
		name string
		mh   string

		delegateErr     error
		readProviders   map[string][]peer.AddrInfo
		routerProviders map[string][]peer.AddrInfo
		finderProviders map[string][]peer.AddrInfo

		expProviders      []peer.AddrInfo
		expWriteProviders map[string][]peer.AddrInfo
		expErr            error
	}{
		{
			name: "returns providers when delegate has them",
			mh:   "mh1",
			readProviders: map[string][]peer.AddrInfo{
				"mh1": {peer.AddrInfo{ID: peer.ID([]byte("peer1"))}},
			},
			expProviders: []peer.AddrInfo{
				{ID: peer.ID([]byte("peer1"))},
			},
		},
		{
			name:          "finds and caches providers when delegate doesn't have them",
			mh:            "mh1",
			readProviders: map[string][]peer.AddrInfo{},
			finderProviders: map[string][]peer.AddrInfo{
				"mh1": {peer.AddrInfo{ID: peer.ID([]byte("peer1"))}},
			},
			expWriteProviders: map[string][]peer.AddrInfo{
				"mh1": {peer.AddrInfo{ID: peer.ID([]byte("peer1"))}},
			},
		},
		{
			name:        "returns error on delegate error",
			delegateErr: errors.New("boom"),
			expErr:      errors.New("boom"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
			defer stop()
			writePS := &mockProviderStore{
				err: c.delegateErr,
			}
			readPS := &mockProviderStore{
				providers: c.readProviders,
				err:       c.delegateErr,
			}
			finder := &mockFinder{
				providers: c.finderProviders,
			}

			ps := NewCachingProviderStore(readPS, writePS, finder, nil)

			provs, err := ps.GetProviders(ctx, []byte(c.mh))
			assert.Equal(t, c.expErr, err)
			assert.Equal(t, c.expProviders, provs)
			assert.Equal(t, c.expWriteProviders, writePS.providers)
		})
	}
}
