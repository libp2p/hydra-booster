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
	KeyHydraID, _ = tag.NewKey("hydra_id")
	KeySybilID, _ = tag.NewKey("sybil_id")
)

// Measures
var (
	MSybil             = stats.Int64("libp2p.io/hydra/sybil", "Sybil node(s) arrived on the network", stats.UnitDimensionless)
	MBootstrappedSybil = stats.Int64("libp2p.io/hydra/bootstrapped_sybils", "A sybil node(s) became bootstrapped", stats.UnitDimensionless)
	MConnectedPeer     = stats.Int64("libp2p.io/hydra/connected_peer", "Peer(s) became connected (1) or disconnected (-1)", stats.UnitDimensionless)
	MUniquePeer        = stats.Int64("libp2p.io/hydra/total_unique_peers", "Total unique peers seen across all sybils", stats.UnitDimensionless)
	MProvide           = stats.Int64("libp2p.io/hydra/provide", "A provide and it's duration", stats.UnitMilliseconds)
)

// Views
var (
	TotalSybilsView = &view.View{
		Name:        "libp2p.io/hydra/total_sybils",
		Measure:     MSybil,
		TagKeys:     []tag.Key{KeyHydraID, KeySybilID},
		Aggregation: view.Sum(),
	}
	TotalBootstrappedSybilsView = &view.View{
		Name:        "libp2p.io/hydra/total_bootstrapped_sybils",
		Measure:     MBootstrappedSybil,
		TagKeys:     []tag.Key{KeyHydraID, KeySybilID},
		Aggregation: view.Sum(),
	}
	TotalConnectedPeersView = &view.View{
		Name:        "libp2p.io/hydra/total_connected_peers",
		Measure:     MConnectedPeer,
		TagKeys:     []tag.Key{KeyHydraID, KeySybilID},
		Aggregation: view.Sum(),
	}
	// TotalUniquePeersView = &view.View{
	// 	Measure:     MUniquePeer,
	// 	Aggregation: view.LastValue(),
	// }
	TotalProvidesView = &view.View{
		Name:        "libp2p.io/hydra/total_provides",
		Description: "Total number of provides made",
		Measure:     MProvide,
		TagKeys:     []tag.Key{KeyHydraID, KeySybilID},
		Aggregation: view.Count(),
	}
	TotalProvidesDurationView = &view.View{
		Name:        "libp2p.io/hydra/total_provides_duration",
		Description: "Total duration (latency) of all provides made",
		Measure:     MProvide,
		TagKeys:     []tag.Key{KeyHydraID, KeySybilID},
		Aggregation: view.Sum(),
	}
	ProvidesLatencyView = &view.View{
		Name:        "libp2p.io/hydra/provides_latency",
		Description: "Histogram distribution of provide latency",
		Measure:     MProvide,
		Aggregation: defaultMillisecondsDistribution,
	}
)

// DefaultViews with all views in it.
var DefaultViews = []*view.View{
	TotalSybilsView,
	TotalBootstrappedSybilsView,
	TotalConnectedPeersView,
	// TotalUniquePeersView,
	TotalProvidesView,
	TotalProvidesDurationView,
	ProvidesLatencyView,
}
