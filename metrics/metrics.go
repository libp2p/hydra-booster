package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	defaultMillisecondsDistribution = view.Distribution(0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)
)

// Keys
var (
	KeyPeerID, _ = tag.NewKey("peer_id")
)

// Measures
var (
	Sybils             = stats.Int64("sybils_total", "Total sybil nodes launched by Hydra", stats.UnitDimensionless)
	BootstrappedSybils = stats.Int64("bootstrapped_sybils_total", "Total bootstrapped sybil nodes", stats.UnitDimensionless)
	ConnectedPeers     = stats.Int64("connected_peers_total", "Total peers connected to all sybils", stats.UnitDimensionless)
	UniquePeers        = stats.Int64("unique_peers_total", "Total unique peers seen across all sybils", stats.UnitDimensionless)
	RoutingTableSize   = stats.Int64("routing_table_size", "Total number of peers in the routing table", stats.UnitDimensionless)
	ProviderRecords    = stats.Int64("provider_records_total", "Total number of provider records in the datastore shared by all sybils", stats.UnitDimensionless)
	Provides           = stats.Int64("provides", "Provides and their durations", stats.UnitMilliseconds)
)

// Views
var (
	SybilsView = &view.View{
		Measure:     Sybils,
		Aggregation: view.Sum(),
	}
	BootstrappedSybilsView = &view.View{
		Measure:     BootstrappedSybils,
		Aggregation: view.Sum(),
	}
	ConnectedPeersView = &view.View{
		Measure:     ConnectedPeers,
		Aggregation: view.LastValue(),
	}
	UniquePeersView = &view.View{
		Measure:     UniquePeers,
		Aggregation: view.LastValue(),
	}
	RoutingTableSizeView = &view.View{
		Measure:     RoutingTableSize,
		TagKeys:     []tag.Key{KeyPeerID}, // Eventually this will be shared and this filter will not apply
		Aggregation: view.LastValue(),
	}
	ProviderRecordsView = &view.View{
		Measure:     ProviderRecords,
		Aggregation: view.LastValue(),
	}
	ProvidesView = &view.View{
		Name:        "provides_total",
		Description: "Total number of provides made",
		Measure:     Provides,
		TagKeys:     []tag.Key{KeyPeerID},
		Aggregation: view.Count(),
	}
	ProvidesDurationView = &view.View{
		Name:        "provides_duration_total",
		Description: "Total duration (latency) of all provides made",
		Measure:     Provides,
		TagKeys:     []tag.Key{KeyPeerID},
		Aggregation: view.Sum(),
	}
	ProvidesLatencyView = &view.View{
		Name:        "provides_latency",
		Description: "Histogram distribution of provide latency",
		Measure:     Provides,
		Aggregation: defaultMillisecondsDistribution,
	}
)

// DefaultViews with all views in it.
var DefaultViews = []*view.View{
	SybilsView,
	BootstrappedSybilsView,
	ConnectedPeersView,
	UniquePeersView,
	RoutingTableSizeView,
	ProviderRecordsView,
	ProvidesView,
	ProvidesDurationView,
	ProvidesLatencyView,
}
