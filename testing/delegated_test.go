package testing

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	vole "github.com/aschmahmann/vole/lib"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-delegated-routing/client"
	"github.com/ipfs/go-delegated-routing/server"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/multiformats/go-multiaddr"
)

func TestDelegatedRoutingEndToEnd(t *testing.T) {
	// start mock delegated routing server
	s := httptest.NewServer(server.FindProvidersAsyncHandler(testFindProvidersAsyncFunc))
	defer s.Close()

	// start hydra head
	headTcpAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 35121))
	_, err := SpawnHead(
		context.Background(),
		opts.Addrs([]multiaddr.Multiaddr{headTcpAddr}),
		opts.Delegate(s.URL),
	)
	if err != nil {
		t.Fatal(err)
	}

	// query hydra head
	vole.DhtGetProvs(context.Background(), []byte{0x00}, protocol.ID("/ipfs"), headTcpAddr)
}

// testFindProvidersAsyncFunc responds with the same provider for any query key.
func testFindProvidersAsyncFunc(key cid.Cid, ch chan<- client.FindProvidersAsyncResult) error {
	go func() {
		ch <- client.FindProvidersAsyncResult{AddrInfo: []peer.AddrInfo{testAddrInfo}}
		close(ch)
	}()
	return nil
}

var testAddrInfo peer.AddrInfo

func init() {
	ma := multiaddr.StringCast("/ip4/7.7.7.7/tcp/4242/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	ai, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		panic(err)
	}
	testAddrInfo = *ai
}
