package metricstasks

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/hydra-booster/metrics"
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
		err := ds.Put(ctx, datastore.NewKey(fmt.Sprintf("/providers/%d", i)), []byte{})
		if err != nil {
			t.Fatal(err)
		}
	}

	pt := NewProviderRecordsTask(ds, nil, time.Second)

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

	rt := NewRoutingTableSizeTask(func() int { return 1 }, time.Second)

	if rt.Interval != time.Second {
		t.Fatal("invalid interval")
	}

	if err := view.Register(metrics.RoutingTableSizeView); err != nil {
		t.Fatal(err)
	}
	defer view.Unregister(metrics.RoutingTableSizeView)

	err := rt.Run(ctx)
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

	if int(dis.Value) != 1 {
		t.Fatal("incorrect value recorded")
	}
}

func TestNewUniquePeersTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rt := NewUniquePeersTask(func() uint64 { return 1 }, time.Second)

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

	if int(dis.Value) != 1 {
		t.Fatal("incorrect value recorded")
	}
}
