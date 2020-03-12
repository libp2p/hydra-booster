package ui

import (
	"fmt"
	"time"

	pmc "github.com/alanshaw/prom-metrics-client"
	"github.com/dustin/go-humanize"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/metrics"
	uiopts "github.com/libp2p/hydra-booster/ui/opts"
	"go.opencensus.io/stats"
)

const (
	logey = iota
	gooey
)

// ErrMissingPeers is returned when no nodes are passed to the UI
var ErrMissingPeers = fmt.Errorf("ui needs at least one peer")

// Render displays and updates a "UI" for the Prometheus /metrics endpoint - CLI output based on the number of Hydra nodes
func Render(peers []peer.ID, opts ...uiopts.Option) error {
	if len(peers) == 0 {
		return ErrMissingPeers
	}

	options := uiopts.Options{}
	options.Apply(append([]uiopts.Option{uiopts.Defaults}, opts...)...)

	uiType := logey

	if len(peers) == 1 {
		uiType = gooey
	}

	client := pmc.PromMetricsClient{URL: fmt.Sprintf("http://127.0.0.1:%d/metrics", options.MetricsPort)}
	ch := make(chan *pmc.Metrics)

	go func() {
		for {
			time.Sleep(options.RefreshPeriod)
			m, err := client.GetMetrics()
			if err != nil {
				fmt.Println(err)
				continue
			}
			ch <- m
		}
	}()

	switch uiType {
	case logey: // many node
		for m := range ch {
			fmt.Fprintf(
				options.Writer,
				"[NumSybils: %v, Uptime: %s, MemoryUsage: %s, PeersConnected: %v, TotalUniquePeersSeen: %v, BootstrapsDone: %v, ProviderRecords: %v, RoutingTableSize: %v]\n",
				getCounterValue(m, nsName(metrics.Sybils)),
				time.Second*time.Duration(int(time.Since(options.Start).Seconds())),
				humanize.Bytes(uint64(getGaugeValue(m, "go_memstats_alloc_bytes"))),
				getCounterValue(m, nsName(metrics.ConnectedPeers)),
				getGaugeValue(m, nsName(metrics.UniquePeers)),
				getCounterValue(m, nsName(metrics.BootstrappedSybils)),
				getGaugeValue(m, nsName(metrics.ProviderRecords)),
				getGaugeValue(m, nsName(metrics.RoutingTableSize)),
			)
		}
	case gooey: // 1 node
		ga := &GooeyApp{Title: "Hydra Booster Sybil", Log: NewLog(options.Writer, 15, 15), writer: options.Writer}
		ga.NewDataLine(3, "Peer ID", peers[0].Pretty())
		econs := ga.NewDataLine(4, "Connections", "0")
		uniqprs := ga.NewDataLine(5, "Unique Peers Seen", "0")
		emem := ga.NewDataLine(6, "Memory Allocated", "0MB")
		eprov := ga.NewDataLine(7, "Stored Provider Records", "0")
		erts := ga.NewDataLine(8, "Routing Table Size", "0")
		etime := ga.NewDataLine(9, "Uptime", "0h 0m 0s")
		ga.Print()

		second := time.NewTicker(time.Second)

		for {
			select {
			// case m := <-messages:
			// 	ga.Log.Add(m)
			// 	ga.Log.Print()
			case m, ok := <-ch:
				if !ok {
					second.Stop()
					return nil
				}
				emem.SetVal(humanize.Bytes(uint64(getGaugeValue(m, "go_memstats_alloc_bytes"))))
				econs.SetVal(fmt.Sprintf("%v peers", getCounterValue(m, nsName(metrics.ConnectedPeers))))
				uniqprs.SetVal(fmt.Sprint(getGaugeValue(m, nsName(metrics.UniquePeers))))
				eprov.SetVal(fmt.Sprint(getGaugeValue(m, nsName(metrics.ProviderRecords))))
				erts.SetVal(fmt.Sprint(getGaugeValue(m, nsName(metrics.RoutingTableSize))))
				ga.Print()
			case <-second.C:
				t := time.Since(options.Start)
				h := int(t.Hours())
				m := int(t.Minutes()) % 60
				s := int(t.Seconds()) % 60
				etime.SetVal(fmt.Sprintf("%dh %dm %ds", h, m, s))
				ga.Print()
			}
		}
	}

	return nil
}

func nsName(m stats.Measure) string {
	return fmt.Sprintf("%s_%s", metrics.PrometheusNamespace, m.Name())
}

func getGaugeValue(m *pmc.Metrics, name string) int {
	for _, g := range m.Gauges {
		if g.Name == name {
			return int(sum(g.Values))
		}
	}
	return 0
}

func getCounterValue(m *pmc.Metrics, name string) int {
	for _, c := range m.Counters {
		if c.Name == name {
			return int(sum(c.Values))
		}
	}
	return 0
}

func sum(values []pmc.Value) float64 {
	var val float64
	for _, v := range values {
		val += v.Value
	}
	return val
}
