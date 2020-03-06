package hydrabooster

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-datastore"
)

func TestSpawnNode(t *testing.T) {
	node, _, bsCh, err := SpawnNode(SpawnNodeOptions{
		datastore:  datastore.NewMapDatastore(),
		addr:       "/ip4/0.0.0.0/tcp/0",
		bucketSize: 20,
	})

	if err != nil {
		t.Fatal(err)
	}

	for {
		status, ok := <-bsCh
		if !ok {
			t.Fatal(fmt.Errorf("channel closed before bootstrap complete"))
		}
		if status.err != nil {
			t.Fatal(status.err)
		}
		if status.done {
			break
		}
	}

	err = node.Close()

	if err != nil {
		t.Fatal(err)
	}
}
