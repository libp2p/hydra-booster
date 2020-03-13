package ui

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	hytesting "github.com/libp2p/hydra-booster/testing"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, mux := newMockMetricsServeMux(t, "../testdata/metrics/1sybil.txt")
	go http.Serve(listener, mux)
	defer listener.Close()

	cw := hytesting.NewChanWriter()

	ui, err := NewUI(Gooey, opts.Writer(cw), opts.MetricsURL(fmt.Sprintf("http://%v/metrics", listener.Addr().String())))
	if err != nil {
		t.Fatal(err)
	}

	go ui.Render(ctx)

	var chunks bytes.Buffer
	for chunk := range cw.Chan() {
		chunks.Write(chunk)
		if !strings.Contains(chunks.String(), "12D3KooWETMx8cDb7JtmpUjPrhXv27mRi7rLmENoK5JT2FYogZvo") {
			continue
		}
		// ensure uptime got updated
		if !strings.Contains(chunks.String(), "0h 0m 1s") {
			continue
		}
		break
	}
}

func TestLogeyUI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, mux := newMockMetricsServeMux(t, "../testdata/metrics/2sybils.txt")
	go http.Serve(listener, mux)
	defer listener.Close()

	cw := hytesting.NewChanWriter()

	ui, err := NewUI(Logey, opts.Writer(cw), opts.MetricsURL(fmt.Sprintf("http://%v/metrics", listener.Addr().String())))
	if err != nil {
		t.Fatal(err)
	}

	go ui.Render(ctx)

	// give it time to render once!
	time.Sleep(time.Millisecond * 100)

	expects := []string{
		"NumSybils: 2",
		"BootstrapsDone: 2",
		"PeersConnected: 11",
		"TotalUniquePeersSeen: 9",
	}

	var chunks bytes.Buffer
	for chunk := range cw.Chan() {
		chunks.Write(chunk)
		found := true
		for _, str := range expects {
			if !strings.Contains(chunks.String(), str) {
				found = false
				break
			}
		}

		if found {
			break
		}
	}
}

func TestRefreshPeriod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, mux := newMockMetricsServeMux(t, "../testdata/metrics/1sybil.txt")
	go http.Serve(listener, mux)
	defer listener.Close()

	cw := hytesting.NewChanWriter()

	ui, err := NewUI(
		Logey,
		opts.Writer(cw),
		opts.MetricsURL(fmt.Sprintf("http://%v/metrics", listener.Addr().String())),
		opts.RefreshPeriod(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	go ui.Render(ctx)

	var chunks bytes.Buffer
	for chunk := range cw.Chan() {
		chunks.Write(chunk)
		lines := strings.Split(chunks.String(), "\n")

		var logLines []string
		for _, l := range lines {
			if strings.Index(l, "[") == 0 {
				logLines = append(logLines, l)
			}
		}

		if len(logLines) >= 2 {
			break
		}
	}
}
