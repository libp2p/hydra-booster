package ui

import (
	"fmt"
	"time"

	pmc "github.com/alanshaw/prom-metrics-client"
	"github.com/dustin/go-humanize"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/metrics"
	uiopts "github.com/libp2p/hydra-booster/ui/opts"
)

const (
	logey = iota
	gooey
)

type UIData struct {
	MemAllocBytes      int
	Sybils             int
	BootstrappedSybils int
	ConnectedPeers     int
	UniquePeers        int
	ProviderRecords    int
	RoutingTableSize   int
}

// ErrMissingPeers is returned when no nodes are passed to the UI
var ErrMissingPeers = fmt.Errorf("ui needs at least one peer")

// NewUI creates a "UI" for status reports - CLI output based on the number of Hydra nodes
func NewUI(peers []peer.ID, opts ...uiopts.Option) error {
	options := uiopts.Options{}
	options.Apply(append([]uiopts.Option{uiopts.Defaults}, opts...)...)

	uiType := logey

	if len(peers) == 0 {
		return ErrMissingPeers
	}

	if len(peers) == 1 {
		uiType = gooey
	}

	dataC := make(chan *UIData)
	client := pmc.PromMetricsClient{URL: fmt.Sprintf("http://127.0.0.1:%d/metrics", options.MetricsPort)}

	go func() {
		for {
			time.Sleep(time.Second * 5)
			m, err := client.GetMetrics()
			if err != nil {
				fmt.Println(err)
				continue
			}
			dataC <- metricsToUIData(m)
		}
	}()

	switch uiType {
	case logey: // many node
		for d := range dataC {
			fmt.Fprintf(
				options.Writer,
				"[NumSybils: %v, Uptime: %s, MemoryUsage: %s, PeersConnected: %v, TotalUniquePeersSeen: %v, BootstrapsDone: %v, ProviderRecords: %v, RoutingTableSize: %v]\n",
				d.Sybils,
				time.Second*time.Duration(int(time.Since(options.Start).Seconds())),
				humanize.Bytes(uint64(d.MemAllocBytes)),
				d.ConnectedPeers,
				d.UniquePeers,
				d.BootstrappedSybils,
				d.ProviderRecords,
				d.RoutingTableSize,
			)
		}
	case gooey: // 1 node
		ga := &GooeyApp{Title: "Hydra Booster Node", Log: NewLog(options.Writer, 15, 15), writer: options.Writer}
		ga.NewDataLine(3, "Peer ID", peers[0].Pretty())
		econs := ga.NewDataLine(4, "Connections", "0")
		uniqprs := ga.NewDataLine(5, "Unique Peers Seen", "0")
		emem := ga.NewDataLine(6, "Memory Allocated", "0MB")
		eprov := ga.NewDataLine(7, "Stored Provider Records", "0")
		erts := ga.NewDataLine(8, "Routing Table Size", "0")
		etime := ga.NewDataLine(9, "Uptime", "0h 0m 0s")
		ga.Print()

		var closed bool
		second := time.NewTicker(time.Second)

		for {
			select {
			// case m := <-messages:
			// 	ga.Log.Add(m)
			// 	ga.Log.Print()
			case d, ok := <-dataC:
				if !ok {
					second.Stop()
					closed = true
					break
				}
				emem.SetVal(humanize.Bytes(uint64(d.MemAllocBytes)))
				econs.SetVal(fmt.Sprintf("%v peers", d.ConnectedPeers))
				uniqprs.SetVal(fmt.Sprint(d.UniquePeers))
				eprov.SetVal(fmt.Sprint(d.ProviderRecords))
				erts.SetVal(fmt.Sprint(d.RoutingTableSize))

				ga.Print()
			case <-second.C:
				t := time.Since(options.Start)
				h := int(t.Hours())
				m := int(t.Minutes()) % 60
				s := int(t.Seconds()) % 60
				etime.SetVal(fmt.Sprintf("%dh %dm %ds", h, m, s))
				ga.Print()
			}

			if closed {
				break
			}
		}
	}

	return nil
}

func metricsToUIData(m *pmc.Metrics) *UIData {
	gns := []string{ // Gauge names
		fmt.Sprintf("%s_%s", metrics.PrometheusNamespace, metrics.ProviderRecords.Name()),
		fmt.Sprintf("%s_%s", metrics.PrometheusNamespace, metrics.RoutingTableSize.Name()),
		fmt.Sprintf("%s_%s", metrics.PrometheusNamespace, metrics.UniquePeers.Name()),
		"go_memstats_alloc_bytes",
	}
	cns := []string{ // Counter names
		fmt.Sprintf("%s_%s", metrics.PrometheusNamespace, metrics.Sybils.Name()),
		fmt.Sprintf("%s_%s", metrics.PrometheusNamespace, metrics.BootstrappedSybils.Name()),
		fmt.Sprintf("%s_%s", metrics.PrometheusNamespace, metrics.ConnectedPeers.Name()),
	}

	var gvs, cvs []float64

	for _, name := range gns {
		g := findGauge(m.Gauges, name)
		if g != nil {
			var val float64
			for _, v := range g.Values {
				val += v.Value
			}
			gvs = append(gvs, val)
		}
	}

	for _, name := range cns {
		c := findCounter(m.Counters, name)
		if c != nil {
			var val float64
			for _, v := range c.Values {
				val += v.Value
			}
			cvs = append(cvs, val)
		}
	}

	return &UIData{
		MemAllocBytes:      int(gvs[3]),
		Sybils:             int(cvs[0]),
		BootstrappedSybils: int(cvs[1]),
		ConnectedPeers:     int(cvs[1]),
		UniquePeers:        int(gvs[2]),
		ProviderRecords:    int(gvs[0]),
		RoutingTableSize:   int(gvs[1]),
	}
}

func findGauge(gauges []pmc.Gauge, name string) *pmc.Gauge {
	for _, g := range gauges {
		if g.Name == name {
			return &g
		}
	}
	return nil
}

func findCounter(counters []pmc.Counter, name string) *pmc.Counter {
	for _, c := range counters {
		if c.Name == name {
			return &c
		}
	}
	return nil
}
