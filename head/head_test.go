package head

import (
	"context"
	"fmt"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	"github.com/libp2p/hydra-booster/head/opts"
	hydratesting "github.com/libp2p/hydra-booster/testing"
)

func TestSpawnHead(t *testing.T) { // TODO spawn a node to bootstrap from so we don't hit the public bootstrappers
	ctx, cancel := context.WithCancel(hydratesting.NewContext())
	defer cancel()

	_, bsCh, err := NewHead(ctx, opts.Datastore(datastore.NewMapDatastore()))
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

func TestSpawnHeadWithDisabledProviderGC(t *testing.T) { // TODO spawn a node to bootstrap from so we don't hit the public bootstrappers
	ctx, cancel := context.WithCancel(hydratesting.NewContext())
	defer cancel()

	_, bsCh, err := NewHead(
		ctx,
		opts.Datastore(datastore.NewMapDatastore()),
		opts.DisableProvGC(),
	)
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

func TestSpawnHeadWithCustomProtocolPrefix(t *testing.T) { // TODO spawn a node to bootstrap from so we don't hit the public bootstrappers
	ctx, cancel := context.WithCancel(hydratesting.NewContext())
	defer cancel()

	_, bsCh, err := NewHead(
		ctx,
		opts.Datastore(datastore.NewMapDatastore()),
		opts.ProtocolPrefix("/myapp"),
		opts.DisableProviders(),
		opts.DisableValues(),
	)
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

func TestSpawnHeadWithCustomPeerstore(t *testing.T) {
	ctx, cancel := context.WithCancel(hydratesting.NewContext())
	defer cancel()

	pstore, err := pstoreds.NewPeerstore(ctx, sync.MutexWrap(datastore.NewMapDatastore()), pstoreds.DefaultOpts())
	if err != nil {
		t.Fatal(err)
	}

	_, bsCh, err := NewHead(
		ctx,
		opts.Datastore(datastore.NewMapDatastore()),
		opts.Peerstore(pstore),
	)
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

func TestGetRoutingTable(t *testing.T) { // TODO spawn a node to bootstrap from so we don't hit the public bootstrappers
	ctx, cancel := context.WithCancel(hydratesting.NewContext())
	defer cancel()

	hd, _, err := NewHead(ctx, opts.Datastore(datastore.NewMapDatastore()))
	if err != nil {
		t.Fatal(err)
	}

	hd.RoutingTable()
}
