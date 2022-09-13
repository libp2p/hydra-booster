package reframe

import (
	"context"
	"net/http"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-delegated-routing/client"
	"github.com/ipfs/go-delegated-routing/server"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	routing "github.com/libp2p/go-libp2p-routing"
)

var log = logging.Logger("hydra/test-reframe-server")

func NewMockServer(index map[cid.Cid][]peer.AddrInfo) http.HandlerFunc {
	// rewrite the cids to v1/raw, because reframe sends v1/raw cids on the wire
	index2 := map[cid.Cid][]peer.AddrInfo{}
	for k, v := range index {
		index2[cid.NewCidV1(cid.Raw, k.Hash())] = v
	}
	return server.DelegatedRoutingAsyncHandler(MockServer{Index: index2})
}

type MockServer struct {
	Index map[cid.Cid][]peer.AddrInfo
}

func (x MockServer) FindProviders(ctx context.Context, key cid.Cid) (<-chan client.FindProvidersAsyncResult, error) {
	ch := make(chan client.FindProvidersAsyncResult)
	log.Infof("serving find providers request for %v", key.String())
	go func() {
		ch <- client.FindProvidersAsyncResult{AddrInfo: x.Index[key]}
		close(ch)
	}()
	return ch, nil
}

func (x MockServer) GetIPNS(ctx context.Context, id []byte) (<-chan client.GetIPNSAsyncResult, error) {
	return nil, routing.ErrNotSupported
}

func (x MockServer) PutIPNS(ctx context.Context, id []byte, record []byte) (<-chan client.PutIPNSAsyncResult, error) {
	return nil, routing.ErrNotSupported
}

func (x MockServer) Provide(ctx context.Context, req *client.ProvideRequest) (<-chan client.ProvideAsyncResult, error) {
	panic("not implemented")
}
