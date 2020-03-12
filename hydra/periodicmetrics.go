package hydra

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/hydra-booster/metrics"
	"go.opencensus.io/stats"
)

// PeriodicMetrics periodically collects and records statistics with prometheus.
type PeriodicMetrics struct {
	hydra *Hydra
}

// NewPeriodicMetrics creates a new PeriodicMetrics that immeidately begins to periodically collect and record statistics with prometheus.
func NewPeriodicMetrics(hy *Hydra, period time.Duration) *PeriodicMetrics {
	pm := PeriodicMetrics{hydra: hy}

	go func() {
		for {
			time.Sleep(period)
			err := pm.periodicCollectAndRecord()
			if err != nil {
				fmt.Println(fmt.Errorf("failed to collect and record stats: %w", err))
			}
		}
	}()

	return &pm
}

func (pm *PeriodicMetrics) periodicCollectAndRecord() error {
	var rts int
	for i := range pm.hydra.Sybils {
		rts += pm.hydra.Sybils[i].RoutingTable.Size()
	}
	stats.Record(context.Background(), metrics.RoutingTableSize.M(int64(rts)))

	prs, err := pm.hydra.SharedDatastore.Query(query.Query{Prefix: "/providers", KeysOnly: true})
	if err == nil {
		// TODO: make fast https://github.com/libp2p/go-libp2p-kad-dht/issues/487
		var provRecords int
		for range prs.Next() {
			provRecords++
		}
		stats.Record(context.Background(), metrics.ProviderRecords.M(int64(provRecords)))
	}

	stats.Record(context.Background(), metrics.UniquePeers.M(int64(pm.hydra.GetUniquePeersCount())))

	return err
}
