package metrics

import (
	dhtmetrics "github.com/libp2p/go-libp2p-kad-dht/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	defaultBytesDistribution        = view.Distribution(1024, 2048, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216, 67108864, 268435456, 1073741824, 4294967296)
	defaultMillisecondsDistribution = view.Distribution(0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)
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
	// DHT views
	ReceivedMessagesView = &view.View{
		Measure:     dhtmetrics.ReceivedMessages,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: view.Count(),
	}
	ReceivedMessageErrorsView = &view.View{
		Measure:     dhtmetrics.ReceivedMessageErrors,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: view.Count(),
	}
	ReceivedBytesView = &view.View{
		Measure:     dhtmetrics.ReceivedBytes,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: defaultBytesDistribution,
	}
	InboundRequestLatencyView = &view.View{
		Measure:     dhtmetrics.InboundRequestLatency,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: defaultMillisecondsDistribution,
	}
	OutboundRequestLatencyView = &view.View{
		Measure:     dhtmetrics.OutboundRequestLatency,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: defaultMillisecondsDistribution,
	}
	SentMessagesView = &view.View{
		Measure:     dhtmetrics.SentMessages,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: view.Count(),
	}
	SentMessageErrorsView = &view.View{
		Measure:     dhtmetrics.SentMessageErrors,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: view.Count(),
	}
	SentRequestsView = &view.View{
		Measure:     dhtmetrics.SentRequests,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: view.Count(),
	}
	SentRequestErrorsView = &view.View{
		Measure:     dhtmetrics.SentRequestErrors,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: view.Count(),
	}
	SentBytesView = &view.View{
		Measure:     dhtmetrics.SentBytes,
		TagKeys:     []tag.Key{dhtmetrics.KeyMessageType},
		Aggregation: defaultBytesDistribution,
	}
)

// DefaultViews with all views in it.
var DefaultViews = []*view.View{
	// Hydra views
	HeadsView,
	BootstrappedHeadsView,
	ConnectedPeersView,
	UniquePeersView,
	RoutingTableSizeView,
	ProviderRecordsView,
	FindProvsView,
	FindProvsDurationView,
	FindProvsQueueSizeView,
	// DHT views
	ReceivedMessagesView,
	ReceivedMessageErrorsView,
	ReceivedBytesView,
	InboundRequestLatencyView,
	OutboundRequestLatencyView,
	SentMessagesView,
	SentMessageErrorsView,
	SentRequestsView,
	SentRequestErrorsView,
	SentBytesView,
}
