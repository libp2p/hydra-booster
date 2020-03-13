package ui

import (
	"fmt"
	"strings"
	"sync"
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

// ErrMissingPeers is returned when no nodes are passed to the UI
var ErrMissingPeers = fmt.Errorf("ui needs at least one peer")

// UI is a simple command line interface to the Prometheus /metrics endpoint
type UI struct {
	theme        Theme
	options      uiopts.Options
	refreshTicks *time.Ticker
	secondTicks  *time.Ticker
	metricsC     chan *pmc.Metrics
	stopC        chan bool
	wg           sync.WaitGroup
}

// NewUI constructs a new "UI" for the Prometheus /metrics endpoint
func NewUI(theme Theme, opts ...uiopts.Option) (*UI, error) {
	options := uiopts.Options{}
	options.Apply(append([]uiopts.Option{uiopts.Defaults}, opts...)...)
	return &UI{
		theme:        theme,
		options:      options,
		refreshTicks: time.NewTicker(options.RefreshPeriod),
		secondTicks:  time.NewTicker(time.Second),
		metricsC:     make(chan *pmc.Metrics),
		stopC:        make(chan bool),
	}, nil
}

// Stop halts UI rendering after the next render cycle (note this function returns before that happens)
func (ui *UI) Stop() {
	ui.refreshTicks.Stop() // prevent further ticks
	ui.secondTicks.Stop()  // prevent further ticks
	ui.wg.Wait()           // wait for current metrics fetch and/or write to complete
	close(ui.metricsC)     // close the metrics channel
	close(ui.stopC)        // signal the UI is stopping
}

// Render displays and updates a "UI" for the Prometheus /metrics endpoint
func (ui *UI) Render() error {
	client := pmc.PromMetricsClient{URL: ui.options.MetricsURL}

	go func() {
		ui.wg.Add(1)
		m, err := client.GetMetrics()
		if err != nil {
			fmt.Println(err)
		}
		ui.metricsC <- m
		ui.wg.Done()

		for {
			select {
			case <-ui.refreshTicks.C:
				ui.wg.Add(1)
				m, err := client.GetMetrics()
				if err != nil {
					fmt.Println(err)
				} else {
					ui.metricsC <- m
				}
				ui.wg.Done()
			case <-ui.stopC:
				return
			}
		}
	}()

	switch ui.theme {
	case Logey:
		for m := range ui.metricsC {
			ui.wg.Add(1)
			fmt.Fprintf(
				ui.options.Writer,
				"[NumSybils: %v, Uptime: %s, MemoryUsage: %s, PeersConnected: %v, TotalUniquePeersSeen: %v, BootstrapsDone: %v, ProviderRecords: %v, RoutingTableSize: %v]\n",
				getCounterValue(m, nsName(metrics.Sybils)),
				time.Second*time.Duration(int(time.Since(ui.options.Start).Seconds())),
				humanize.Bytes(uint64(getGaugeValue(m, "go_memstats_alloc_bytes"))),
				getCounterValue(m, nsName(metrics.ConnectedPeers)),
				getGaugeValue(m, nsName(metrics.UniquePeers)),
				getCounterValue(m, nsName(metrics.BootstrappedSybils)),
				getGaugeValue(m, nsName(metrics.ProviderRecords)),
				getGaugeValue(m, nsName(metrics.RoutingTableSize)),
			)
			ui.wg.Done()
		}
	case Gooey:
		ga := &GooeyApp{Title: "Hydra Booster", Log: NewLog(ui.options.Writer, 15, 15), writer: ui.options.Writer}
		esybs := ga.NewDataLine(3, "Sybil ID(s)", "")
		econs := ga.NewDataLine(4, "Connections", "0")
		uniqprs := ga.NewDataLine(5, "Unique Peers Seen", "0")
		emem := ga.NewDataLine(6, "Memory Allocated", "0MB")
		eprov := ga.NewDataLine(7, "Stored Provider Records", "0")
		erts := ga.NewDataLine(8, "Routing Table Size", "0")
		etime := ga.NewDataLine(9, "Uptime", "0h 0m 0s")
		ga.Print()

		for {
			select {
			// case m := <-messages:
			// 	ga.Log.Add(m)
			// 	ga.Log.Print()
			case m, ok := <-ui.metricsC:
				if !ok {
					return nil
				}
				esybs.SetVal(fmt.Sprintf("%v", getCounterTagValues(m, nsName(metrics.Sybils), "peer_id")))
				emem.SetVal(humanize.Bytes(uint64(getGaugeValue(m, "go_memstats_alloc_bytes"))))
				econs.SetVal(fmt.Sprintf("%v peers", getCounterValue(m, nsName(metrics.ConnectedPeers))))
				uniqprs.SetVal(fmt.Sprint(getGaugeValue(m, nsName(metrics.UniquePeers))))
				eprov.SetVal(fmt.Sprint(getGaugeValue(m, nsName(metrics.ProviderRecords))))
				erts.SetVal(fmt.Sprint(getGaugeValue(m, nsName(metrics.RoutingTableSize))))
			case <-ui.secondTicks.C:
				t := time.Since(ui.options.Start)
				h := int(t.Hours())
				m := int(t.Minutes()) % 60
				s := int(t.Seconds()) % 60
				etime.SetVal(fmt.Sprintf("%dh %dm %ds", h, m, s))
			}
			ui.wg.Add(1)
			ga.Print()
			ui.wg.Done()
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

func getCounterTagValues(m *pmc.Metrics, metricName string, tagName string) []string {
	var vals []string
	for _, c := range m.Counters {
		if c.Name == metricName {
			for _, v := range c.Values {
				// TODO add tag parsing to prom-metrics-client
				if strings.Index(v.Name, tagName+"=\"") > -1 {
					val := strings.Split(v.Name, tagName+"=\"")[1]
					vals = append(vals, strings.Split(val, "\"")[0])
				}
			}
		}
	}
	return vals
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
