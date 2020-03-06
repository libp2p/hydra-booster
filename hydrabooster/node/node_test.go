package node

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-datastore"
	hyopts "github.com/libp2p/hydra-booster/hydrabooster/opts"
)

func TestSpawnNode(t *testing.T) { // TODO spawn a node to bootstrap from so we don't hit the public bootstrappers
	node, bsCh, err := NewHydraNode(hyopts.Datastore(datastore.NewMapDatastore()))

	if err != nil {
		t.Fatal(err)
	}

	for {
		status, ok := <-bsCh
		if !ok {
			t.Fatal(fmt.Errorf("channel closed before bootstrap complete"))
		}
		if status.Err != nil {
			t.Fatal(status.Err)
		}
		if status.Done {
			break
		}
	}

	err = node.Host.Close()

	if err != nil {
		t.Fatal(err)
	}
}
