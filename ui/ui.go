package ui

import (
	"fmt"
	"io"
	"time"

	pmc "github.com/alanshaw/prom-metrics-client"
	"github.com/dustin/go-humanize"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/reports"
	uiopts "github.com/libp2p/hydra-booster/ui/opts"
)

const (
	logey = iota
	gooey
)

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

	switch uiType {
	case logey: // many node
		ticker := time.NewTicker(time.Second * 5)

		c := pmc.PromMetricsClient{
			URL: "http://localhost:8888/metrics",
		}

		for {
			<-ticker.C
			m, err := c.GetMetrics()
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("Gauges:")
			for _, gauge := range m.Gauges {
				fmt.Printf("%+v\n", gauge)
			}

			fmt.Println("Counters:")
			for _, counter := range m.Counters {
				fmt.Printf("%+v\n", counter)
			}
			// fmt.Sprintf("%s_%s", metrics.PrometheusNamespace, metrics.Sybils.Name())
		}
		// for {
		// 	r, ok := <-statusReports
		// 	if !ok {
		// 		break
		// 	}
		// 	printStatusLine(options.Writer, r, options.Start)
		// }
	case gooey: // 1 node
		// ga := &GooeyApp{Title: "Hydra Booster Node", Log: NewLog(options.Writer, 15, 15), writer: options.Writer}
		// ga.NewDataLine(3, "Peer ID", peers[0].Pretty())
		// econs := ga.NewDataLine(4, "Connections", "0")
		// uniqprs := ga.NewDataLine(5, "Unique Peers Seen", "0")
		// emem := ga.NewDataLine(6, "Memory Allocated", "0MB")
		// eprov := ga.NewDataLine(7, "Stored Provider Records", "0")
		// eprlat := ga.NewDataLine(8, "Store Provider Latency", "0s")
		// etime := ga.NewDataLine(9, "Uptime", "0h 0m 0s")
		// ga.Print()

		// var closed bool
		// second := time.NewTicker(time.Second)

		// for {
		// 	select {
		// 	// case m := <-messages:
		// 	// 	ga.Log.Add(m)
		// 	// 	ga.Log.Print()
		// 	case r, ok := <-statusReports:
		// 		if !ok {
		// 			second.Stop()
		// 			closed = true
		// 			break
		// 		}
		// 		emem.SetVal(humanize.Bytes(r.MemStats.Alloc))
		// 		econs.SetVal(fmt.Sprintf("%d peers", r.TotalConnectedPeers))
		// 		uniqprs.SetVal(fmt.Sprint(r.TotalUniquePeers))
		// 		eprov.SetVal(fmt.Sprint(r.TotalProvs))

		// 		if r.TotalProvs > 0 {
		// 			eprlat.SetVal(fmt.Sprint(r.TotalProvTime / time.Duration(r.TotalProvs)))
		// 		}

		// 		ga.Print()
		// 	case <-second.C:
		// 		t := time.Since(options.Start)
		// 		h := int(t.Hours())
		// 		m := int(t.Minutes()) % 60
		// 		s := int(t.Seconds()) % 60
		// 		etime.SetVal(fmt.Sprintf("%dh %dm %ds", h, m, s))
		// 		ga.Print()
		// 	}

		// 	if closed {
		// 		break
		// 	}
		// }
	}

	return nil
}

func printStatusLine(writer io.Writer, report reports.StatusReport, start time.Time) {
	fmt.Fprintf(
		writer,
		"[NumSybils: %d, Uptime: %s, Memory Usage: %s, PeersConnected: %d, TotalUniquePeersSeen: %d, Total Provs: %d, BootstrapsDone: %d]\n",
		report.TotalHydraNodes,
		time.Second*time.Duration(int(time.Since(start).Seconds())),
		humanize.Bytes(report.MemStats.Alloc),
		report.TotalConnectedPeers,
		report.TotalUniquePeers,
		report.TotalProvs,
		report.TotalBootstrappedHydraNodes,
	)
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
