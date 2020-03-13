package ui

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
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

// Buffer is a goroutine safe bytes.Buffer
type Buffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write appends the contents of p to the buffer, growing the buffer as needed. It returns
// the number of bytes written.
func (s *Buffer) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Write(p)
}

// String returns the contents of the unread portion of the buffer
// as a string.  If the Buffer is a nil pointer, it returns "<nil>".
func (s *Buffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.String()
}

func TestGooeyUI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, mux := newMockMetricsServeMux(t, "../testdata/metrics/1sybil.txt")
	go http.Serve(listener, mux)
	defer listener.Close()

	var b Buffer

	ui, err := NewUI(Gooey, opts.Writer(&b), opts.MetricsURL(fmt.Sprintf("http://%v/metrics", listener.Addr().String())))
	if err != nil {
		t.Fatal(err)
	}

	go ui.Render(ctx)

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, mux := newMockMetricsServeMux(t, "../testdata/metrics/2sybils.txt")
	go http.Serve(listener, mux)
	defer listener.Close()

	var b Buffer

	ui, err := NewUI(Logey, opts.Writer(&b), opts.MetricsURL(fmt.Sprintf("http://%v/metrics", listener.Addr().String())))
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

	for _, str := range expects {
		if !strings.Contains(b.String(), str) {
			t.Fatalf("%v not found in output", str)
		}
	}
}

func TestRefreshPeriod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener, mux := newMockMetricsServeMux(t, "../testdata/metrics/1sybil.txt")
	go http.Serve(listener, mux)
	defer listener.Close()

	var b Buffer

	ui, err := NewUI(
		Logey,
		opts.Writer(&b),
		opts.MetricsURL(fmt.Sprintf("http://%v/metrics", listener.Addr().String())),
		opts.RefreshPeriod(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	go ui.Render(ctx)

	// give it time to refresh
	time.Sleep(time.Second + (time.Millisecond * 100))

	fmt.Println(b.String())
	lines := strings.Split(b.String(), "\n")

	var logLines []string
	for _, l := range lines {
		if strings.Index(l, "[") == 0 {
			logLines = append(logLines, l)
		}
	}

	if len(logLines) < 2 {
		t.Fatal("expected 2 or more log lines")
	}
}
