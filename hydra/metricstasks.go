package hydra

import (
	"context"
	"time"

	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/periodictasks"
	"go.opencensus.io/stats"
)

func newProviderRecordsTask(hy *Hydra, d time.Duration) periodictasks.PeriodicTask {
	return periodictasks.PeriodicTask{
		Interval: d,
		Run: func(ctx context.Context) error {
			prs, err := hy.SharedDatastore.Query(query.Query{Prefix: "/providers", KeysOnly: true})
			if err != nil {
				return err
			}

			// TODO: make fast https://github.com/libp2p/go-libp2p-kad-dht/issues/487
			var provRecords int
			for range prs.Next() {
				provRecords++
			}
			prs.Close()

			stats.Record(ctx, metrics.ProviderRecords.M(int64(provRecords)))
			return nil
		},
	}
}

func newRoutingTableSizeTask(hy *Hydra, d time.Duration) periodictasks.PeriodicTask {
	return periodictasks.PeriodicTask{
		Interval: d,
		Run: func(ctx context.Context) error {
			var rts int
			for i := range hy.Heads {
				rts += hy.Heads[i].RoutingTable().Size()
			}
			stats.Record(ctx, metrics.RoutingTableSize.M(int64(rts)))
			return nil
		},
	}
}

func newUniquePeersTask(hy *Hydra, d time.Duration) periodictasks.PeriodicTask {
	return periodictasks.PeriodicTask{
		Interval: d,
		Run: func(ctx context.Context) error {
			stats.Record(ctx, metrics.UniquePeers.M(int64(hy.GetUniquePeersCount())))
			return nil
		},
	}
}
