package sybil

import (
	"context"
	"fmt"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/hydra-booster/sybil/opts"
)

func TestSpawnSybil(t *testing.T) { // TODO spawn a node to bootstrap from so we don't hit the public bootstrappers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, bsCh, err := NewSybil(ctx, opts.Datastore(datastore.NewMapDatastore()))
	if err != nil {
		t.Fatal(err)
	}

	for {
		status, ok := <-bsCh
		if !ok {
			t.Fatal(fmt.Errorf("channel closed before bootstrap complete"))
		}
		if status.Err != nil {
			fmt.Println(status.Err)
		}
		if status.Done {
			break
		}
	}
}
