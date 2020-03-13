package ui

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/hydra-booster/ui/opts"
)

func newMockMetricsServeMux(t *testing.T, name string) (net.Listener, *http.ServeMux) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, name)
	})

	return listener, mux
}

func TestGooeyUI(t *testing.T) {
	listener, mux := newMockMetricsServeMux(t, "../testdata/metrics/1sybil.txt")
	go http.Serve(listener, mux)
	defer listener.Close()

	var b bytes.Buffer

	ui, err := NewUI(Gooey, opts.Writer(&b), opts.MetricsURL(fmt.Sprintf("http://%v/metrics", listener.Addr().String())))
	if err != nil {
		t.Fatal(err)
	}

	go ui.Render()
	defer ui.Stop()

	// wait for output after just over 1s
	time.Sleep(time.Second + (time.Millisecond * 100))

	if !strings.Contains(b.String(), "12D3KooWETMx8cDb7JtmpUjPrhXv27mRi7rLmENoK5JT2FYogZvo") {
		t.Fatalf("12D3KooWETMx8cDb7JtmpUjPrhXv27mRi7rLmENoK5JT2FYogZvo not found in output")
	}

	// ensure uptime got updated
	if !strings.Contains(b.String(), "0h 0m 1s") {
		t.Fatalf("%v not found in output", "0h 0m 1s")
	}
}

func TestLogeyUI(t *testing.T) {
	listener, mux := newMockMetricsServeMux(t, "../testdata/metrics/2sybils.txt")
	go http.Serve(listener, mux)
	defer listener.Close()

	var b bytes.Buffer

	ui, err := NewUI(Logey, opts.Writer(&b), opts.MetricsURL(fmt.Sprintf("http://%v/metrics", listener.Addr().String())))
	if err != nil {
		t.Fatal(err)
	}

	go ui.Render()
	defer ui.Stop()

	// give it time to render once!
	time.Sleep(time.Millisecond * 100)

	expects := []string{
		"NumSybils: 2",
		"BootstrapsDone: 2",
		"PeersConnected: 11",
		"TotalUniquePeersSeen: 9",
	}

	for _, str := range expects {
		if !strings.Contains(b.String(), str) {
			t.Fatalf("%v not found in output", str)
		}
	}
}
