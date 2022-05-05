package reframe

import (
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
	return server.DelegatedRoutingAsyncHandler(MockServer{Index: index})
}

type MockServer struct {
	Index map[cid.Cid][]peer.AddrInfo
}

func (x MockServer) FindProviders(key cid.Cid) (<-chan client.FindProvidersAsyncResult, error) {
	ch := make(chan client.FindProvidersAsyncResult)
	log.Infof("serving find providers request for %v", key.String())
	go func() {
		ch <- client.FindProvidersAsyncResult{AddrInfo: x.Index[key]}
		close(ch)
	}()
	return ch, nil
}

func (x MockServer) GetIPNS(id []byte) (<-chan client.GetIPNSAsyncResult, error) {
	return nil, routing.ErrNotSupported
}

func (x MockServer) PutIPNS(id []byte, record []byte) (<-chan client.PutIPNSAsyncResult, error) {
	return nil, routing.ErrNotSupported
}
