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
	Heads             = stats.Int64("heads", "Heads launched by Hydra", stats.UnitDimensionless)
	BootstrappedHeads = stats.Int64("bootstrapped_heads", "Bootstrapped heads", stats.UnitDimensionless)
	ConnectedPeers    = stats.Int64("connected_peers", "Peers connected to all heads", stats.UnitDimensionless)
	UniquePeers       = stats.Int64("unique_peers_total", "Total unique peers seen across all heads", stats.UnitDimensionless)
	RoutingTableSize  = stats.Int64("routing_table_size", "Number of peers in the routing table", stats.UnitDimensionless)
	ProviderRecords   = stats.Int64("provider_records", "Number of provider records in the datastore shared by all heads", stats.UnitDimensionless)
	// Augmented with "status" label:
	// "local" (found locally)
	// "succeeded" (found at least 1 provider on the network)
	// "failed" (not found any providers on the network)
	// "discarded" (not local and queue was full)
	FindProvs = stats.Int64("find_provs_total", "Total find provider attempts that were found locally, or not found locally and succeeded, failed or were discarded", stats.UnitDimensionless)
	// Augmented with "status" label:
	// "succeeded" (found at least 1 provider on the network)
	// "failed" (not found any providers on the network)
	FindProvsDuration  = stats.Float64("find_provs_duration_seconds", "The time it took find provider attempts from the network to succeed or fail because of timeout or completion", stats.UnitSeconds)
	FindProvsQueueSize = stats.Int64("find_provs_queue_size", "The current size of the queue for finding providers", stats.UnitDimensionless)
)

// Views
var (
	HeadsView = &view.View{
		Measure:     Heads,
		TagKeys:     []tag.Key{KeyName, KeyPeerID},
		Aggregation: view.Sum(),
	}
	BootstrappedHeadsView = &view.View{
		Measure:     BootstrappedHeads,
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
		Aggregation: view.Sum(),
	}
)

// DefaultViews with all views in it.
var DefaultViews = []*view.View{
	HeadsView,
	BootstrappedHeadsView,
	ConnectedPeersView,
	UniquePeersView,
	RoutingTableSizeView,
	ProviderRecordsView,
	FindProvsView,
	FindProvsDurationView,
	FindProvsQueueSizeView,
}
