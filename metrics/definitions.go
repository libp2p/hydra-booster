package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Keys
var (
	KeyName, _   = tag.NewKey("name")
	KeyPeerID, _ = tag.NewKey("peer_id")
	KeyStatus, _ = tag.NewKey("status")
)

// Measures
var (
	Sybils             = stats.Int64("sybils", "Sybil nodes launched by Hydra", stats.UnitDimensionless)
	BootstrappedSybils = stats.Int64("bootstrapped_sybils", "Bootstrapped sybil nodes", stats.UnitDimensionless)
	ConnectedPeers     = stats.Int64("connected_peers", "Peers connected to all sybils", stats.UnitDimensionless)
	UniquePeers        = stats.Int64("unique_peers_total", "Total unique peers seen across all sybils", stats.UnitDimensionless)
	RoutingTableSize   = stats.Int64("routing_table_size", "Number of peers in the routing table", stats.UnitDimensionless)
	ProviderRecords    = stats.Int64("provider_records", "Number of provider records in the datastore shared by all sybils", stats.UnitDimensionless)
	// With "status" label: "succeeded" (found at least 1 provider), "failed" (not found any providers), "discarded" (queue was full)
	FindProvs = stats.Int64("find_provs_total", "Total find provider attempts that succeeded, failed or were discarded", stats.UnitDimensionless)
	// With "status" label: "succeeded" (found at least 1 provider), "failed" (not found any providers)
	FindProvsDuration  = stats.Float64("find_provs_duration_seconds", "The time it took find provider attempts to succeed or fail because of timeout or completion", stats.UnitSeconds)
	FindProvsQueueSize = stats.Int64("find_provs_queue_size", "The current size of the queue for finding providers", stats.UnitDimensionless)
)

// Views
var (
	SybilsView = &view.View{
		Measure:     Sybils,
		TagKeys:     []tag.Key{KeyName, KeyPeerID},
		Aggregation: view.Sum(),
	}
	BootstrappedSybilsView = &view.View{
		Measure:     BootstrappedSybils,
		TagKeys:     []tag.Key{KeyName, KeyPeerID},
		Aggregation: view.Sum(),
	}
	ConnectedPeersView = &view.View{
		Measure:     ConnectedPeers,
		TagKeys:     []tag.Key{KeyName, KeyPeerID},
		Aggregation: view.Sum(),
	}
	UniquePeersView = &view.View{
		Measure:     UniquePeers,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.LastValue(),
	}
	RoutingTableSizeView = &view.View{
		Measure:     RoutingTableSize,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.LastValue(),
	}
	ProviderRecordsView = &view.View{
		Measure:     ProviderRecords,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.LastValue(),
	}
	FindProvsView = &view.View{
		Measure:     FindProvs,
		TagKeys:     []tag.Key{KeyName, KeyStatus},
		Aggregation: view.Sum(),
	}
	FindProvsDurationView = &view.View{
		Measure:     FindProvsDuration,
		TagKeys:     []tag.Key{KeyName, KeyStatus},
		Aggregation: view.Sum(),
	}
	FindProvsQueueSizeView = &view.View{
		Measure:     FindProvsQueueSize,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.LastValue(),
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
	FindProvsView,
	FindProvsDurationView,
	FindProvsQueueSizeView,
}
