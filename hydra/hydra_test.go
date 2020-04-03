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
		NSybils: 2,
		GetPort: utils.PortSelector(3000),
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(hy.Sybils) != 2 {
		t.Fatal("expected hydra to spawn 2 sybils")
	}
}

func TestGetUniquePeersCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hy, err := NewHydra(ctx, Options{
		NSybils: 2,
		GetPort: utils.PortSelector(3000),
	})
	if err != nil {
		t.Fatal(err)
	}

	syb0Addr := hy.Sybils[0].Host.Addrs()[0]
	syb0ID := hy.Sybils[0].Host.ID()
	syb0p2pAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("%s/p2p/%s", syb0Addr, syb0ID))
	if err != nil {
		t.Fatal(err)
	}
	syb0AddrInfo, err := peer.AddrInfoFromP2pAddr(syb0p2pAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = hy.Sybils[1].Host.Connect(ctx, *syb0AddrInfo)
	if err != nil {
		t.Fatal(err)
	}

	c := hy.GetUniquePeersCount()
	if c <= 0 {
		t.Fatal("expected unique peers count to be greater than 0")
	}
}
