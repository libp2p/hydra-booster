package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/libp2p/hydra-booster/node"
	"github.com/libp2p/hydra-booster/reports"
)

const (
	logey = iota
	gooey
)

func NewUI(nodes []*node.HydraNode, statusReports chan reports.StatusReport, start time.Time) error {
	uiType := logey

	if len(nodes) == 0 {
		return fmt.Errorf("ui needs at least one node")
	}

	if len(nodes) == 1 {
		uiType = gooey
	}

	switch uiType {
	case logey: // many node
		for {
			r, ok := <-statusReports
			if !ok {
				break
			}
			printStatusLine(r, start)
		}
	case gooey: // 1 node
		ga := &GooeyApp{Title: "Hydra Booster Node", Log: NewLog(15, 15)}
		ga.NewDataLine(3, "Peer ID", nodes[0].Host.ID().Pretty())
		econs := ga.NewDataLine(4, "Connections", "0")
		uniqprs := ga.NewDataLine(5, "Unique Peers Seen", "0")
		emem := ga.NewDataLine(6, "Memory Allocated", "0MB")
		eprov := ga.NewDataLine(7, "Stored Provider Records", "0")
		eprlat := ga.NewDataLine(8, "Store Provider Latency", "0s")
		etime := ga.NewDataLine(9, "Uptime", "0h 0m 0s")
		ga.Print()

		second := time.NewTicker(time.Second)

		for {
			select {
			// case m := <-messages:
			// 	ga.Log.Add(m)
			// 	ga.Log.Print()
			case r, ok := <-statusReports:
				if !ok {
					second.Stop()
					statusReports = nil
					break
				}
				emem.SetVal(humanize.Bytes(r.MemStats.Alloc))
				econs.SetVal(fmt.Sprintf("%d peers", r.TotalConnectedPeers))
				uniqprs.SetVal(fmt.Sprint(r.TotalUniquePeers))
				eprov.SetVal(fmt.Sprint(r.TotalProvs))

				if r.TotalProvs > 0 {
					eprlat.SetVal(fmt.Sprint(r.TotalProvTime / time.Duration(r.TotalProvs)))
				}

				ga.Print()
			case <-second.C:
				t := time.Since(start)
				h := int(t.Hours())
				m := int(t.Minutes()) % 60
				s := int(t.Seconds()) % 60
				etime.SetVal(fmt.Sprintf("%dh %dm %ds", h, m, s))
				ga.Print()
			}
		}
	default:
		return fmt.Errorf("unknown UI type %v", uiType)
	}

	return nil
}

func printStatusLine(report reports.StatusReport, start time.Time) {
	fmt.Fprintf(
		os.Stderr,
		"[NumDhts: %d, Uptime: %s, Memory Usage: %s, TotalPeers: %d/%d, Total Provs: %d, BootstrapsDone: %d]\n",
		report.TotalHydraNodes,
		time.Second*time.Duration(int(time.Since(start).Seconds())),
		humanize.Bytes(report.MemStats.Alloc),
		report.TotalConnectedPeers,
		report.TotalUniquePeers,
		report.TotalProvs,
		report.TotalBootstrappedHydraNodes,
	)
}
