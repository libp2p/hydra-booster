package providers

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-delegated-routing/client"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("hydra/providers")

func CombineProviders(backend ...providers.ProviderStore) providers.ProviderStore {
	return &CombinedProviderStore{backends: backend}
}

type CombinedProviderStore struct {
	backends []providers.ProviderStore
}

func (s *CombinedProviderStore) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	ch := make(chan error, len(s.backends))
	for _, b := range s.backends {
		go func(backend providers.ProviderStore) {
			ch <- backend.AddProvider(ctx, key, prov)
		}(b)
	}
	var errs *multierror.Error
	for range s.backends {
		if e := <-ch; e != nil {
			multierror.Append(errs, e)
		}
	}
	if len(errs.WrappedErrors()) > 0 {
		log.Errorf("some providers returned errors (%v)", errs)
	}
	if len(errs.WrappedErrors()) == len(s.backends) {
		return errs
	} else {
		return nil
	}
}

func (s *CombinedProviderStore) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	ch := make(chan client.FindProvidersAsyncResult, len(s.backends))
	for _, b := range s.backends {
		go func(backend providers.ProviderStore) {
			infos, err := backend.GetProviders(ctx, key)
			ch <- client.FindProvidersAsyncResult{AddrInfo: infos, Err: err}
		}(b)
	}
	infos := []peer.AddrInfo{}
	var errs *multierror.Error
	for range s.backends {
		r := <-ch
		if r.Err == nil {
			infos = append(infos, r.AddrInfo...)
		} else {
			multierror.Append(errs, r.Err)
		}
	}
	infos = mergeAddrInfos(infos)
	if len(errs.WrappedErrors()) > 0 {
		log.Errorf("some providers returned errors (%v)", errs)
	}
	if len(errs.WrappedErrors()) == len(s.backends) {
		return infos, errs
	} else {
		return infos, nil
	}
}

func mergeAddrInfos(infos []peer.AddrInfo) []peer.AddrInfo {
	m := map[peer.ID][]multiaddr.Multiaddr{}
	for _, info := range infos {
		m[info.ID] = mergeMultiaddrs(append(m[info.ID], info.Addrs...))
	}
	var r []peer.AddrInfo
	for k, v := range m {
		if k.Validate() == nil {
			r = append(r, peer.AddrInfo{ID: k, Addrs: v})
		}
	}
	return r
}

func mergeMultiaddrs(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
	m := map[string]multiaddr.Multiaddr{}
	for _, addr := range addrs {
		m[addr.String()] = addr
	}
	var r []multiaddr.Multiaddr
	for _, v := range m {
		r = append(r, v)
	}
	return r
}
