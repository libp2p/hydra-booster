package hydra

import (
	"context"
	"time"

	"github.com/ipfs/go-datastore/query"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/periodictasks"
	"go.opencensus.io/stats"
)

func countProviderRecordsExactly(ctx context.Context, hy *Hydra) error {
	prs, err := hy.SharedDatastore.Query(query.Query{Prefix: "/providers", KeysOnly: true})
	if err != nil {
		return err
	}
	defer prs.Close()

	// TODO: make fast https://github.com/libp2p/go-libp2p-kad-dht/issues/487
	var provRecords int
	for {
		select {
		case r, ok := <-prs.Next():
			if !ok {
				stats.Record(ctx, metrics.ProviderRecords.M(int64(provRecords)))
				return nil
			}
			if r.Error == nil {
				provRecords++
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func countProviderRecordsApproximately(ctx context.Context, hy *Hydra) error {
	pgxPool := hy.SharedDatastore.(withPostgresBackend).PgxPool()
	const query = `SELECT
	(reltuples/relpages) * (
	  pg_relation_size('records') /
	  (current_setting('block_size')::integer)
	)
	FROM pg_class where relname = 'records';`
	row := pgxPool.QueryRow(ctx, query)
	var numProvRecords float64
	err := row.Scan(&numProvRecords)
	if err != nil {
		return err
	}
	stats.Record(ctx, metrics.ProviderRecords.M(int64(numProvRecords)))
	return nil
}

type withPostgresBackend interface {
	PgxPool() *pgxpool.Pool
}

func newProviderRecordsTask(hy *Hydra, d time.Duration) periodictasks.PeriodicTask {
	var task func(ctx context.Context) error
	if _, ok := hy.SharedDatastore.(withPostgresBackend); ok {
		task = func(ctx context.Context) error { return countProviderRecordsApproximately(ctx, hy) }
	} else {
		task = func(ctx context.Context) error { return countProviderRecordsExactly(ctx, hy) }
	}
	return periodictasks.PeriodicTask{
		Interval: d,
		Run:      task,
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
