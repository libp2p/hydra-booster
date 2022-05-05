package testing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	ipfsutil "github.com/ipfs/go-ipfs-util"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	noise "github.com/libp2p/go-libp2p-noise"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	tls "github.com/libp2p/go-libp2p-tls"
	"github.com/libp2p/go-tcp-transport"
	"github.com/libp2p/hydra-booster/head"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/libp2p/hydra-booster/testing/reframe"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestDelegatedRoutingEndToEnd(t *testing.T) {
	key := cid.NewCidV0(ipfsutil.Hash([]byte("testkey")))
	fmt.Printf("cid: %s\n", key.String())
	// start mock delegated routing server

	mockServer := reframe.NewMockServer(map[cid.Cid][]peer.AddrInfo{
		key: {testAddrInfo},
	})
	s := httptest.NewServer(mockServer)
	defer s.Close()

	// start hydra head
	headTcpAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 35121))
	head, err := head.SpawnTestHead(
		context.Background(),
		opts.Addrs([]multiaddr.Multiaddr{headTcpAddr}),
		opts.ReframeAddr(s.URL),
		opts.DelegateHTTPClient(&http.Client{Timeout: 1 * time.Second}),
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
	host, err := libp2p.New(
		libp2p.Transport(quic.NewTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New),
	)
	require.NoError(t, err)
	defer host.Close()

	d, err := dht.New(dhtCtx, host, dhtOpts...)
	require.NoError(t, err)
	t.Logf("started dht %v", host.ID())

	// add the hydra head to the DHT routing table
	err = host.Connect(context.Background(), peer.AddrInfo{ID: head.Host.ID(), Addrs: head.Host.Addrs()})
	if err != nil {
		t.Fatalf("connecting dht to head (%v)", err)
	}
	_, err = d.RoutingTable().TryAddPeer(head.Host.ID(), true, false)
	if err != nil {
		t.Fatalf("cannot add peer to table")
	}
	t.Logf("connected dht to hydra")

	// query hydra head
	infos, err := d.FindProviders(dhtCtx, key)
	require.NoError(t, err)
	if len(infos) < 1 {
		t.Fatalf("expecting at least one provider, got %d: %v", len(infos), infos)
	}
	for _, ai := range infos {
		if equalAddrInfos(ai, testAddrInfo) {
			return
		}
	}
	t.Errorf("expecting %v in %v", testAddrInfo, infos)
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
