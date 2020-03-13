package testing

import (
	"context"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/hydra-booster/sybil"
	"github.com/libp2p/hydra-booster/sybil/opts"
)

// Defaults are the SpawnNode defaults
var defaults = []opts.Option{
	opts.Datastore(datastore.NewMapDatastore()),
	opts.BootstrapPeers(nil),
}

// SpawnSybil creates a new Hydra sybil with an in memory datastore and 0 bootstrap peers by default.
// It also waits for bootstrapping to complete.
func SpawnSybil(ctx context.Context, options ...opts.Option) (*sybil.Sybil, error) {
	nd, bsCh, err := sybil.NewSybil(ctx, append(defaults, options...)...)
	if err != nil {
		return nil, err
	}

	for {
		status, ok := <-bsCh
		if !ok {
			break
		}
		if status.Err != nil {
			fmt.Println(status.Err)
		}
	}

	return nd, nil
}

// SpawnSybils creates n new Hydra nodes with an in memory datastore and 0 bootstrap peers by default
func SpawnSybils(ctx context.Context, n int, options ...opts.Option) ([]*sybil.Sybil, error) {
	var sybils []*sybil.Sybil
	for i := 0; i < n; i++ {
		syb, err := SpawnSybil(ctx, options...)
		if err != nil {
			for _, nd := range sybils {
				nd.Host.Close()
			}
			return nil, err
		}
		sybils = append(sybils, syb)
	}

	return sybils, nil
}

// ChanWriter is a writer that writes to a channel
type ChanWriter struct {
	C chan []byte
}

// NewChanWriter creates a new channel writer
func NewChanWriter() *ChanWriter {
	return &ChanWriter{make(chan []byte)}
}

// Write writes to the channel
func (w *ChanWriter) Write(p []byte) (int, error) {
	d := make([]byte, len(p))
	copy(d, p)
	w.C <- d
	return len(p), nil
}
