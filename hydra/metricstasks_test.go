package hydra

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/axiomhq/hyperloglog"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/hydra-booster/head"
	"github.com/libp2p/hydra-booster/head/opts"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/multiformats/go-multiaddr"
	"go.opencensus.io/stats/view"
)

func TestNewProviderRecordsTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ds := datastore.NewMapDatastore()
	defer ds.Close()

	rand.Seed(time.Now().UTC().UnixNano())
	count := rand.Intn(100) + 1

	for i := 0; i < count; i++ {
		err := ds.Put(datastore.NewKey(fmt.Sprintf("/providers/%d", i)), []byte{})
		if err != nil {
			t.Fatal(err)
		}
	}

	hy := Hydra{SharedDatastore: ds}
	pt := newProviderRecordsTask(&hy, time.Second)

	if pt.Interval != time.Second {
		t.Fatal("invalid interval")
	}

	if err := view.Register(metrics.ProviderRecordsView); err != nil {
		t.Fatal(err)
	}
	defer view.Unregister(metrics.ProviderRecordsView)

	err := pt.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := view.RetrieveData(metrics.ProviderRecordsView.Name)
	if err != nil {
		t.Fatal(err)
	}

	if len(rows) == 0 {
		t.Fatal("no data was recorded")
	}

	data := rows[0].Data
	dis, ok := data.(*view.LastValueData)
	if !ok {
		t.Fatalf("want LastValueData, got %+v\n", data)
	}

	if int(dis.Value) != count {
		t.Fatal("incorrect value recorded")
	}
}

func TestNewRoutingTableSizeTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hd, _, err := head.NewHead(ctx, opts.Datastore(datastore.NewMapDatastore()), opts.BootstrapPeers([]multiaddr.Multiaddr{}))
	if err != nil {
		t.Fatal(err)
	}

	hy := Hydra{Heads: []*head.Head{hd}}

	rt := newRoutingTableSizeTask(&hy, time.Second)

	if rt.Interval != time.Second {
		t.Fatal("invalid interval")
	}

	if err := view.Register(metrics.RoutingTableSizeView); err != nil {
		t.Fatal(err)
	}
	defer view.Unregister(metrics.RoutingTableSizeView)

	err = rt.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := view.RetrieveData(metrics.RoutingTableSizeView.Name)
	if err != nil {
		t.Fatal(err)
	}

	if len(rows) == 0 {
		t.Fatal("no data was recorded")
	}

	data := rows[0].Data
	dis, ok := data.(*view.LastValueData)
	if !ok {
		t.Fatalf("want LastValueData, got %+v\n", data)
	}

	if int(dis.Value) != 0 {
		t.Fatal("incorrect value recorded")
	}
}

func TestNewUniquePeersTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var hyperLock sync.Mutex
	hyperlog := hyperloglog.New()

	rand.Seed(time.Now().UTC().UnixNano())
	count := rand.Intn(100) + 1

	for i := 0; i < count; i++ {
		hyperlog.Insert([]byte(fmt.Sprintf("peer%d", i)))
	}

	hy := Hydra{hyperLock: &hyperLock, hyperlog: hyperlog}
	rt := newUniquePeersTask(&hy, time.Second)

	if rt.Interval != time.Second {
		t.Fatal("invalid interval")
	}

	if err := view.Register(metrics.UniquePeersView); err != nil {
		t.Fatal(err)
	}
	defer view.Unregister(metrics.UniquePeersView)

	err := rt.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := view.RetrieveData(metrics.UniquePeersView.Name)
	if err != nil {
		t.Fatal(err)
	}

	if len(rows) == 0 {
		t.Fatal("no data was recorded")
	}

	data := rows[0].Data
	dis, ok := data.(*view.LastValueData)
	if !ok {
		t.Fatalf("want LastValueData, got %+v\n", data)
	}

	if int(dis.Value) != count {
		t.Fatal("incorrect value recorded")
	}
}
