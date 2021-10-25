package testing

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-delegated-routing/client"
	"github.com/ipfs/go-delegated-routing/server"
	ipfsutil "github.com/ipfs/go-ipfs-util"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestDelegatedRoutingEndToEnd(t *testing.T) {
	// start mock delegated routing server
	s := httptest.NewServer(server.FindProvidersAsyncHandler(testFindProvidersAsyncFunc))
	defer s.Close()

	// start hydra head
	headTcpAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 35121))
	head, err := SpawnHead(
		context.Background(),
		opts.Addrs([]multiaddr.Multiaddr{headTcpAddr}),
		opts.Delegate(s.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("started hydra %v at %v", head.Host.ID(), headTcpAddr)

	// create DHT client
	dhtCtx := context.Background()
	dhtOpts := []dht.Option{
		// dht.NamespacedValidator("v", blankValidator{}),
		dht.DisableAutoRefresh(),
		dht.Mode(dht.ModeClient),
	}
	host, err := bhost.NewHost(dhtCtx, swarmt.GenSwarm(t, dhtCtx, swarmt.OptDisableReuseport), new(bhost.HostOpts))
	require.NoError(t, err)

	d, err := dht.New(dhtCtx, host, dhtOpts...)
	require.NoError(t, err)
	t.Logf("started dht %v", host.ID())

	// add the hydra head to the DHT routing table
	err = host.Connect(context.Background(), peer.AddrInfo{ID: head.Host.ID(), Addrs: head.Host.Addrs()})
	if err != nil {
		t.Fatalf("connecting dht to head (%v)", err)
	}
	ok, err := d.RoutingTable().TryAddPeer(head.Host.ID(), true, false)
	if !ok || err != nil {
		t.Fatalf("cannot add peer to table")
	}

	// query hydra head
	key := cid.NewCidV1(cid.Raw, ipfsutil.Hash([]byte("testkey")))
	infos, err := d.FindProviders(dhtCtx, key)
	require.NoError(t, err)
	if len(infos) != 1 {
		t.Fatalf("expecting a single provider")
	}
	if !equalAddrInfos(infos[0], testAddrInfo) {
		t.Errorf("expecting %v, got %v", testAddrInfo, infos[0])
	}
}

// testFindProvidersAsyncFunc responds with the same provider for any query key.
func testFindProvidersAsyncFunc(key cid.Cid, ch chan<- client.FindProvidersAsyncResult) error {
	go func() {
		ch <- client.FindProvidersAsyncResult{AddrInfo: []peer.AddrInfo{testAddrInfo}}
		close(ch)
	}()
	return nil
}

func equalAddrInfos(x, y peer.AddrInfo) bool {
	if x.ID != y.ID {
		return false
	}
	if len(x.Addrs) != len(y.Addrs) {
		return false
	}
	for i := range x.Addrs {
		if !x.Addrs[i].Equal(y.Addrs[i]) {
			return false
		}
	}
	return true
}

var testAddrInfo peer.AddrInfo

func init() {
	ma := multiaddr.StringCast("/ip4/127.0.0.1/tcp/14242/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	ai, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		panic(err)
	}
	testAddrInfo = *ai
}
