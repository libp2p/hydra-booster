package ui

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/reports"
	hytesting "github.com/libp2p/hydra-booster/testing"
	"github.com/libp2p/hydra-booster/ui/opts"
)

func TestUIRequiresPeers(t *testing.T) {
	err := NewUI([]peer.ID{}, make(chan reports.StatusReport))
	if err != ErrMissingPeers {
		t.Fatal("created a UI with no peers")
	}
}

func TestGooeyUI(t *testing.T) {
	var b bytes.Buffer

	peerId, _, _, err := hytesting.GeneratePeerID()
	if err != nil {
		t.Fatal(err)
	}

	srs := make(chan reports.StatusReport)

	go func() {
		srs <- reports.StatusReport{}
		time.Sleep(time.Second * 2) // Wait for uptime to update
		close(srs)
	}()

	NewUI([]peer.ID{peerId}, srs, opts.Writer(&b))

	if !strings.Contains(b.String(), peerId.Pretty()) {
		t.Fatalf("%v not found in output", peerId.Pretty())
	}

	// ensure uptime got updated
	if !strings.Contains(b.String(), "0h 0m 1s") {
		t.Fatalf("%v not found in output", "0h 0m 1s")
	}
}

func TestLogeyUI(t *testing.T) {
	var b bytes.Buffer

	peerId0, _, _, err := hytesting.GeneratePeerID()
	if err != nil {
		t.Fatal(err)
	}

	peerId1, _, _, err := hytesting.GeneratePeerID()
	if err != nil {
		t.Fatal(err)
	}

	srs := make(chan reports.StatusReport)

	rand.Seed(time.Now().UnixNano())
	r := reports.StatusReport{
		TotalHydraNodes:             rand.Int(),
		TotalBootstrappedHydraNodes: rand.Int(),
		TotalConnectedPeers:         rand.Int(),
		TotalUniquePeers:            rand.Uint64(),
	}

	go func() {
		srs <- r
		close(srs)
	}()

	NewUI([]peer.ID{peerId0, peerId1}, srs, opts.Writer(&b))

	expects := []string{
		fmt.Sprintf("NumSybils: %v", r.TotalHydraNodes),
		fmt.Sprintf("BootstrapsDone: %v", r.TotalBootstrappedHydraNodes),
		fmt.Sprintf("PeersConnected: %v", r.TotalConnectedPeers),
		fmt.Sprintf("TotalUniquePeersSeen: %v", r.TotalUniquePeers),
	}

	for _, str := range expects {
		if !strings.Contains(b.String(), str) {
			t.Fatalf("%v not found in output", str)
		}
	}
}
