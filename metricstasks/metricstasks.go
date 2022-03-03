package metricstasks

import (
	"context"
	"fmt"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	hydrads "github.com/libp2p/hydra-booster/datastore"

	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/periodictasks"
	"go.opencensus.io/stats"
)

func countProviderRecordsExactly(ctx context.Context, datastore ds.Datastore) error {
	fmt.Println("counting provider records")
	prs, err := datastore.Query(ctx, query.Query{Prefix: "/providers", KeysOnly: true})
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

func countProviderRecordsApproximately(ctx context.Context, pgxPool *pgxpool.Pool) error {
	var approxCountSql = `SELECT
	(reltuples/relpages) * (
	  pg_relation_size($1) /
	  (current_setting('block_size')::integer)
	)
	FROM pg_class where relname = $2;`
	fmt.Println("approximating provider records")
	row := pgxPool.QueryRow(ctx, approxCountSql, hydrads.TableName, hydrads.TableName)
	var numProvRecords float64
	err := row.Scan(&numProvRecords)
	if err != nil {
		return err
	}
	fmt.Printf("found %v provider records\n", int64(numProvRecords))
	stats.Record(ctx, metrics.ProviderRecords.M(int64(numProvRecords)))
	return nil
}

type providerRecordCounter interface {
	CountProviderRecords(ctx context.Context) (int64, error)
}

func recordFromProviderRecordCounter(ctx context.Context, c providerRecordCounter) error {
	n, err := c.CountProviderRecords(ctx)
	if err != nil {
		return err
	}
	stats.Record(ctx, metrics.ProviderRecords.M(n))
	return nil
}

func NewProviderRecordsTask(datastore ds.Datastore, providerstore providers.ProviderStore, d time.Duration) periodictasks.PeriodicTask {
	var task func(ctx context.Context) error
	if pgBackend, ok := datastore.(hydrads.WithPgxPool); ok {
		task = func(ctx context.Context) error { return countProviderRecordsApproximately(ctx, pgBackend.PgxPool()) }
	} else if counter, ok := providerstore.(providerRecordCounter); ok {
		task = func(ctx context.Context) error { return recordFromProviderRecordCounter(ctx, counter) }
	} else {
		task = func(ctx context.Context) error { return countProviderRecordsExactly(ctx, datastore) }
	}
	return periodictasks.PeriodicTask{
		Interval: d,
		Run:      task,
	}
}

func NewRoutingTableSizeTask(getRoutingTableSize func() int, d time.Duration) periodictasks.PeriodicTask {
	return periodictasks.PeriodicTask{
		Interval: d,
		Run: func(ctx context.Context) error {
			stats.Record(ctx, metrics.RoutingTableSize.M(int64(getRoutingTableSize())))
			return nil
		},
	}
}

func NewUniquePeersTask(getUniquePeersCount func() uint64, d time.Duration) periodictasks.PeriodicTask {
	return periodictasks.PeriodicTask{
		Interval: d,
		Run: func(ctx context.Context) error {
			stats.Record(ctx, metrics.UniquePeers.M(int64(getUniquePeersCount())))
			return nil
		},
	}
}

type entryCounter interface {
	EntryCount(context.Context) (uint64, error)
}

func NewIPNSRecordsTask(c entryCounter, d time.Duration) periodictasks.PeriodicTask {
	return periodictasks.PeriodicTask{
		Interval: d,
		Run: func(ctx context.Context) error {
			count, err := c.EntryCount(ctx)
			if err != nil {
				return err
			}
			stats.Record(ctx, metrics.IPNSRecords.M(int64(count)))
			return nil
		},
	}
}
