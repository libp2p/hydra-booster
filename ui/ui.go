package ui

import (
	"context"
	"fmt"
	"time"

	pmc "github.com/alanshaw/prom-metrics-client"
	"github.com/dustin/go-humanize"
	"github.com/libp2p/hydra-booster/metrics"
	uiopts "github.com/libp2p/hydra-booster/ui/opts"
	"go.opencensus.io/stats"
)

// Theme is the style of UI to render
type Theme int

const (
	// Logey is a UI theme that simply logs data periodically to stdout
	Logey Theme = iota
	// Gooey is a UI theme that refreshes values in place
	Gooey
)

// UI is a simple command line interface to the Prometheus /metrics endpoint
type UI struct {
	theme   Theme
	options uiopts.Options
}

// NewUI constructs a new "UI" for the Prometheus /metrics endpoint
func NewUI(theme Theme, opts ...uiopts.Option) (*UI, error) {
	options := uiopts.Options{}
	options.Apply(append([]uiopts.Option{uiopts.Defaults}, opts...)...)
	return &UI{theme: theme, options: options}, nil
}

// Render displays and updates a "UI" for the Prometheus /metrics endpoint
func (ui *UI) Render(ctx context.Context) error {
	client := pmc.PromMetricsClient{URL: ui.options.MetricsURL}
	mC := make(chan []*pmc.Metric)

	go func() {
		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				ms, err := client.GetMetrics()
				if err != nil {
					fmt.Println(err)
				} else {
					mC <- ms
				}
				timer.Reset(ui.options.RefreshPeriod)
			case <-ctx.Done():
				return
			}
		}
	}()

	switch ui.theme {
	case Logey:
		for {
			select {
			case ms := <-mC:
				fmt.Fprintf(
					ui.options.Writer,
					"[NumHeads: %v, Uptime: %s, MemoryUsage: %s, PeersConnected: %v, TotalUniquePeersSeen: %v, BootstrapsDone: %v, ProviderRecords: %v, RoutingTableSize: %v]\n",
					sumSamples(findByName(ms, nsName(metrics.Heads))),
					time.Second*time.Duration(int(time.Since(ui.options.Start).Seconds())),
					humanize.Bytes(uint64(sumSamples(findByName(ms, "go_memstats_alloc_bytes")))),
					sumSamples(findByName(ms, nsName(metrics.ConnectedPeers))),
					sumSamples(findByName(ms, nsName(metrics.UniquePeers))),
					sumSamples(findByName(ms, nsName(metrics.BootstrappedHeads))),
					sumSamples(findByName(ms, nsName(metrics.ProviderRecords))),
					sumSamples(findByName(ms, nsName(metrics.RoutingTableSize))),
				)
			case <-ctx.Done():
				return nil
			}
		}
	case Gooey:
		ga := &GooeyApp{Title: "Hydra Booster", Log: NewLog(ui.options.Writer, 15, 15), writer: ui.options.Writer}
		ehds := ga.NewDataLine(3, "Head ID(s)", "")
		econs := ga.NewDataLine(4, "Connections", "0")
		uniqprs := ga.NewDataLine(5, "Unique Peers Seen", "0")
		emem := ga.NewDataLine(6, "Memory Allocated", "0MB")
		eprov := ga.NewDataLine(7, "Stored Provider Records", "0")
		erts := ga.NewDataLine(8, "Routing Table Size", "0")
		etime := ga.NewDataLine(9, "Uptime", "0h 0m 0s")
		ga.Print()

		seconds := time.NewTicker(time.Second)
		defer seconds.Stop()

		for {
			select {
			// case m := <-messages:
			// 	ga.Log.Add(m)
			// 	ga.Log.Print()
			case ms := <-mC:
				ehds.SetVal(fmt.Sprintf("%v", labelValues(findByName(ms, nsName(metrics.Heads)), "peer_id")))
				emem.SetVal(humanize.Bytes(uint64(sumSamples(findByName(ms, "go_memstats_alloc_bytes")))))
				econs.SetVal(fmt.Sprintf("%v peers", sumSamples(findByName(ms, nsName(metrics.ConnectedPeers)))))
				uniqprs.SetVal(fmt.Sprint(sumSamples(findByName(ms, nsName(metrics.UniquePeers)))))
				eprov.SetVal(fmt.Sprint(sumSamples(findByName(ms, nsName(metrics.ProviderRecords)))))
				erts.SetVal(fmt.Sprint(sumSamples(findByName(ms, nsName(metrics.RoutingTableSize)))))
			case <-seconds.C:
				t := time.Since(ui.options.Start)
				h := int(t.Hours())
				m := int(t.Minutes()) % 60
				s := int(t.Seconds()) % 60
				etime.SetVal(fmt.Sprintf("%dh %dm %ds", h, m, s))
			case <-ctx.Done():
				return nil
			}
			ga.Print()
		}
	}

	return nil
}

func nsName(m stats.Measure) string {
	return fmt.Sprintf("%s_%s", metrics.PrometheusNamespace, m.Name())
}

func findByName(ms []*pmc.Metric, metricName string) *pmc.Metric {
	for _, m := range ms {
		if m.Name == metricName {
			return m
		}
	}
	return nil
}

func labelValues(m *pmc.Metric, labelKey string) []string {
	var vals []string
	if m != nil {
		for _, s := range m.Samples {
			vals = append(vals, s.Labels[labelKey])
		}
	}
	return vals
}

func sumSamples(m *pmc.Metric) float64 {
	var val float64
	if m != nil {
		for _, s := range m.Samples {
			val += s.Value
		}
	}
	return val
}
