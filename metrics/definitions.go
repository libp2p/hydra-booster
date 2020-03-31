package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Keys
var (
	KeyPeerID, _ = tag.NewKey("peer_id")
)

// Measures
var (
	Sybils             = stats.Int64("sybils", "Sybil nodes launched by Hydra", stats.UnitDimensionless)
	BootstrappedSybils = stats.Int64("bootstrapped_sybils", "Bootstrapped sybil nodes", stats.UnitDimensionless)
	ConnectedPeers     = stats.Int64("connected_peers", "Peers connected to all sybils", stats.UnitDimensionless)
	UniquePeers        = stats.Int64("unique_peers_total", "Total unique peers seen across all sybils", stats.UnitDimensionless)
	RoutingTableSize   = stats.Int64("routing_table_size", "Number of peers in the routing table", stats.UnitDimensionless)
	ProviderRecords    = stats.Int64("provider_records", "Number of provider records in the datastore shared by all sybils", stats.UnitDimensionless)
)

// Views
var (
	SybilsView = &view.View{
		Measure:     Sybils,
		TagKeys:     []tag.Key{KeyPeerID},
		Aggregation: view.Sum(),
	}
	BootstrappedSybilsView = &view.View{
		Measure:     BootstrappedSybils,
		TagKeys:     []tag.Key{KeyPeerID},
		Aggregation: view.Sum(),
	}
	ConnectedPeersView = &view.View{
		Measure:     ConnectedPeers,
		TagKeys:     []tag.Key{KeyPeerID},
		Aggregation: view.Sum(),
	}
	UniquePeersView = &view.View{
		Measure:     UniquePeers,
		Aggregation: view.LastValue(),
	}
	RoutingTableSizeView = &view.View{
		Measure:     RoutingTableSize,
		Aggregation: view.LastValue(),
	}
	ProviderRecordsView = &view.View{
		Measure:     ProviderRecords,
		Aggregation: view.Sum(),
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
}
