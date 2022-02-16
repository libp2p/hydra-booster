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
	// a coarser-grained milliseconds distribution for metrics with higher cardinality and where we don't need a more fine-grained distribution
	coarseMillisecondsDistribution = view.Distribution(0, 1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000)
	defaultProvidersDistribution   = view.Distribution(0, 1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000)
)

// Keys
var (
	KeyName, _      = tag.NewKey("name")
	KeyPeerID, _    = tag.NewKey("peer_id")
	KeyStatus, _    = tag.NewKey("status")
	KeyHTTPCode, _  = tag.NewKey("http_code")
	KeyOperation, _ = tag.NewKey("operation")
)

// Measures
var (
	Heads                 = stats.Int64("heads", "Heads launched by Hydra", stats.UnitDimensionless)
	BootstrappedHeads     = stats.Int64("bootstrapped_heads", "Bootstrapped heads", stats.UnitDimensionless)
	ConnectedPeers        = stats.Int64("connected_peers", "Peers connected to all heads", stats.UnitDimensionless)
	UniquePeers           = stats.Int64("unique_peers_total", "Total unique peers seen across all heads", stats.UnitDimensionless)
	RoutingTableSize      = stats.Int64("routing_table_size", "Number of peers in the routing table", stats.UnitDimensionless)
	ProviderRecords       = stats.Int64("provider_records", "Number of provider records in the datastore shared by all heads", stats.UnitDimensionless)
	ProviderRecordsPerKey = stats.Int64("provider_records_per_key", "Number of provider records returned per key", stats.UnitDimensionless)
	// Augmented with "status" label:
	// "local" (found locally)
	// "succeeded" (found at least 1 provider on the network)
	// "failed" (not found any providers on the network)
	// "failed-cached" (no providers found locally, and did not attempt to try network due to negative cache)
	// "discarded" (not local and queue was full)
	Prefetches = stats.Int64("prov_prefetches", "Total find provider prefetch attempts that were found locally, or not found locally and succeeded, failed or were discarded", stats.UnitDimensionless)
	// Augmented with "status" label:
	// "succeeded" (found at least 1 provider on the network)
	// "failed" (not found any providers on the network)
	PrefetchDuration                = stats.Float64("prov_prefetch_duration", "The time it took  provider prefetching attempts from the network to succeed or fail because of timeout or completion", stats.UnitMilliseconds)
	PrefetchNegativeCacheHits       = stats.Int64("prov_prefetch_neg_cache_hits", "Total provider prefetch negative cache hits (lookups skipped due to previous recent failed lookups)", stats.UnitDimensionless)
	PrefetchNegativeCacheSize       = stats.Int64("prov_prefetch_neg_cache_size", "Total size of the provider prefetch negative cache", stats.UnitDimensionless)
	PrefetchNegativeCacheTTLSeconds = stats.Int64("prov_prefetch_neg_cache_ttl", "The TTL duration for negative cache entries", stats.UnitDimensionless)
	PrefetchFailedToCache           = stats.Int64("prov_prefetch_failed_to_cache", "Number of times the provider prefetcher failed to cache a result", stats.UnitDimensionless)
	PrefetchesPending               = stats.Int64("prov_prefetch_pending", "Total number of async provider prefetches pending (queued or in progress)", stats.UnitDimensionless)
	PrefetchesPendingLimit          = stats.Int64("prov_prefetch_pending_limit", "The limit of the number of pending prefetches", stats.UnitDimensionless)

	// Augmented with "status" label:
	// "succeeded" if a response with no error was received from the source.
	// "failed" if an error was encountered and the request failed.
	DelegatedFindProvs         = stats.Int64("delegated_find_provs_total", "Total delegated find provider attempts that were found locally, or not found locally and succeeded, failed or were discarded", stats.UnitDimensionless)
	DelegatedFindProvsDuration = stats.Float64("delegated_find_provs_duration", "The time it took delegated find provider attempts from the network to succeed or fail because of timeout or completion", stats.UnitMilliseconds)

	STIFindProvs         = stats.Int64("sti_find_provs_total", "Total store the index find provider attempts that were found locally, or not found locally and succeeded, failed or were discarded", stats.UnitDimensionless)
	STIFindProvsDuration = stats.Float64("sti_find_provs_duration_nanoseconds", "The time it took storetheindex finds from the network to succeed or fail because of timeout or completion", stats.UnitSeconds)

	AWSRequests              = stats.Int64("aws_requests", "Requests made to AWS", stats.UnitDimensionless)
	AWSRequestDurationMillis = stats.Float64("aws_request_duration", "The time it took to make an AWS request and receive a response", stats.UnitMilliseconds)
	AWSRequestRetries        = stats.Int64("aws_retries", "Retried requests to AWS", stats.UnitDimensionless)
	ProviderDDBCollisions    = stats.Int64("prov_ddb_collisions", "Number of key collisions when writing provider records into DynamoDB", stats.UnitDimensionless)
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
	ProviderRecordsPerKeyView = &view.View{
		Measure:     ProviderRecordsPerKey,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: defaultProvidersDistribution,
	}
	PrefetchesView = &view.View{
		Measure:     Prefetches,
		TagKeys:     []tag.Key{KeyName, KeyStatus},
		Aggregation: view.Sum(),
	}
	PrefetchDurationMillisView = &view.View{
		Measure:     PrefetchDuration,
		TagKeys:     []tag.Key{KeyName, KeyStatus},
		Aggregation: coarseMillisecondsDistribution,
	}
	PrefetchNegativeCacheHitsView = &view.View{
		Measure:     PrefetchNegativeCacheHits,
		Aggregation: view.Sum(),
	}
	PrefetchNegativeCacheSizeView = &view.View{
		Measure:     PrefetchNegativeCacheSize,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.LastValue(),
	}
	PrefetchNegativeCacheTTLSecondsView = &view.View{
		Measure:     PrefetchNegativeCacheTTLSeconds,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.LastValue(),
	}
	PrefetchFailedToCacheView = &view.View{
		Measure:     PrefetchFailedToCache,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.Sum(),
	}
	PrefetchesPendingView = &view.View{
		Measure:     PrefetchesPending,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.LastValue(),
	}
	PrefetchesPendingLimitView = &view.View{
		Measure:     PrefetchesPendingLimit,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.LastValue(),
	}
	AWSRequestsView = &view.View{
		Measure:     AWSRequests,
		TagKeys:     []tag.Key{KeyName, KeyOperation, KeyHTTPCode},
		Aggregation: view.Sum(),
	}
	AWSRequestsDurationView = &view.View{
		Measure:     AWSRequestDurationMillis,
		TagKeys:     []tag.Key{KeyName, KeyOperation, KeyHTTPCode},
		Aggregation: coarseMillisecondsDistribution,
	}
	AWSRequestRetriesView = &view.View{
		Measure:     AWSRequestRetries,
		TagKeys:     []tag.Key{KeyName, KeyOperation, KeyHTTPCode},
		Aggregation: view.Sum(),
	}
	ProviderDDBCollisionsView = &view.View{
		Measure:     ProviderDDBCollisions,
		TagKeys:     []tag.Key{KeyName},
		Aggregation: view.Sum(),
	}
	STIFindProvsView = &view.View{
		Measure:     STIFindProvs,
		TagKeys:     []tag.Key{KeyName, KeyStatus},
		Aggregation: view.Sum(),
	}
	STIFindProvsDurationView = &view.View{
		Measure:     STIFindProvsDuration,
		TagKeys:     []tag.Key{KeyName, KeyStatus},
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
		Aggregation: coarseMillisecondsDistribution,
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
	STIFindProvsView,
	STIFindProvsDurationView,
	ProviderRecordsPerKeyView,
	PrefetchesView,
	PrefetchDurationMillisView,
	PrefetchNegativeCacheHitsView,
	PrefetchNegativeCacheSizeView,
	PrefetchNegativeCacheTTLSecondsView,
	PrefetchFailedToCacheView,
	PrefetchesPendingView,
	PrefetchesPendingLimitView,
	AWSRequestsView,
	AWSRequestsDurationView,
	AWSRequestRetriesView,
	ProviderDDBCollisionsView,
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
