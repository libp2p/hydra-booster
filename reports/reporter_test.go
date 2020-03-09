package reports

import (
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/hydra-booster/node"
	"github.com/libp2p/hydra-booster/node/opts"
	hytesting "github.com/libp2p/hydra-booster/testing"
	"github.com/multiformats/go-multiaddr"
)

func TestReporterRequiresNodesToReportOn(t *testing.T) {
	_, err := NewReporter([]*node.HydraNode{}, time.Second)
	if err != ErrMissingNodes {
		t.Fatal("created a reporter with no hydra nodes to report on")
	}
}

func TestReporterPublishesReports(t *testing.T) {
	nodes, err := hytesting.SpawnNodes(2)
	if err != nil {
		t.Fatal(err)
	}

	reporter, err := NewReporter(nodes, time.Millisecond*50)
	if err != nil {
		t.Fatal(err)
	}

	var reports []StatusReport

	for i := 0; i < 3; i++ {
		r, ok := <-reporter.StatusReports
		if !ok {
			t.Fatalf("reports channel closed before a report was received")
		}
		reports = append(reports, r)
	}

	reporter.Stop()

	for _, report := range reports {
		if report.TotalHydraNodes != len(nodes) {
			t.Fatalf("invalid total nodes, wanted %d got %d", len(nodes), report.TotalHydraNodes)
		}
		if report.TotalBootstrappedHydraNodes != 0 {
			t.Fatalf("invalid bootstrapped nodes, wanted 0 got %d", report.TotalBootstrappedHydraNodes)
		}
		if report.TotalConnectedPeers != 0 {
			t.Fatalf("invalid connected peers, wanted 0 got %d", report.TotalConnectedPeers)
		}
	}
}

func TestReporterPublishesReportsWithBootstrappedNodes(t *testing.T) {
	n0, err := hytesting.SpawnNode()
	if err != nil {
		t.Fatal(err)
	}

	var bsAddrs []multiaddr.Multiaddr
	for _, addr := range n0.Host.Addrs() {
		p2p, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", n0.Host.ID()))
		bsAddrs = append(bsAddrs, addr.Encapsulate(p2p))
	}

	n1, err := hytesting.SpawnNode(opts.BootstrapPeers(bsAddrs))
	if err != nil {
		t.Fatal(err)
	}

	reporter, err := NewReporter([]*node.HydraNode{n0, n1}, time.Millisecond*50)
	if err != nil {
		t.Fatal(err)
	}

	var reports []StatusReport

	for i := 0; i < 3; i++ {
		r, ok := <-reporter.StatusReports
		if !ok {
			t.Fatalf("reports channel closed before a report was received")
		}
		reports = append(reports, r)
	}

	reporter.Stop()

	for _, report := range reports {
		if report.TotalHydraNodes != 2 {
			t.Fatalf("invalid total nodes, wanted 2 got %d", report.TotalHydraNodes)
		}
		if report.TotalBootstrappedHydraNodes != 1 {
			t.Fatalf("invalid bootstrapped nodes, wanted 1 got %d", report.TotalBootstrappedHydraNodes)
		}
		if report.TotalConnectedPeers != 2 {
			t.Fatalf("invalid connected peers, wanted 2 got %d", report.TotalConnectedPeers)
		}
	}
}
