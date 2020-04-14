package hydra

import (
	"context"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/utils"
	"github.com/multiformats/go-multiaddr"
)

func TestSpawnHydra(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hy, err := NewHydra(ctx, Options{
		NHeads:  2,
		GetPort: utils.PortSelector(3000),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(hy.Heads) != 2 {
		t.Fatal("expected hydra to spawn 2 heads")
	}
}

func TestGetUniquePeersCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hy, err := NewHydra(ctx, Options{
		NHeads:  2,
		GetPort: utils.PortSelector(3000),
	})
	if err != nil {
		t.Fatal(err)
	}

	hd0Addr := hy.Heads[0].Host.Addrs()[0]
	hd0ID := hy.Heads[0].Host.ID()
	hd0p2pAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("%s/p2p/%s", hd0Addr, hd0ID))
	if err != nil {
		t.Fatal(err)
	}
	hd0AddrInfo, err := peer.AddrInfoFromP2pAddr(hd0p2pAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = hy.Heads[1].Host.Connect(ctx, *hd0AddrInfo)
	if err != nil {
		t.Fatal(err)
	}

	c := hy.GetUniquePeersCount()
	if c <= 0 {
		t.Fatal("expected unique peers count to be greater than 0")
	}
}
