package datastore

import (
	hook "github.com/alanshaw/ipfs-hookds"
	"github.com/alanshaw/ipfs-hookds/opts"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/hydra-booster/metrics"
	"go.opencensus.io/stats"
)

// NewDatastore creates a new datastore instrumented with hydra specific functionality
func NewDatastore(ds datastore.Batching) datastore.Batching {
	hds := hook.Batching(ds, opts.OnAfterPut(onAfterPut(ds)), opts.OnAfterDelete(onAfterDelete()))
	return hds
}

func isProviderRecord(k datastore.Key) bool {
	return k.String()[0:11] == "/providers/"
}

func onAfterPut(ds datastore.Batching) func(k datastore.Key, v []byte, err error) error {
	return func(k datastore.Key, v []byte, err error) error {
		if isProviderRecord(k) {
			exists, err := ds.Has(k)
			if err != nil && !exists {
				stats.Record(ctx, metrics.ProviderRecords.M(1))
			}
		}
		return err
	}
}

func onAfterDelete() func(k datastore.Key, err error) error {
	return func(k datastore.Key, err error) error {
		if isProviderRecord(k) {
			stats.Record(ctx, metrics.ProviderRecords.M(-1))
		}
		return err
	}
}
