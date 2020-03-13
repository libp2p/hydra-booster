package hydra

import (
	"context"
	"testing"

	"github.com/libp2p/hydra-booster/utils"
)

func TestSpawnHydra(t *testing.T) { // TODO spawn a node to bootstrap from so we don't hit the public bootstrappers
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
