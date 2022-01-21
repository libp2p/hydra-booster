package testing

import (
	"context"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/hydra-booster/head"
	"github.com/libp2p/hydra-booster/head/opts"
)

// SpawnHead creates a new Hydra head with an in memory datastore and 0 bootstrap peers by default.
// It also waits for bootstrapping to complete.
func SpawnHead(ctx context.Context, options ...opts.Option) (*head.Head, error) {
	/*
		Defaults are the SpawnNode defaults.
		It is not defined as a global so that the datastore is not shared between tests.
	*/
	defaults := []opts.Option{
		opts.Datastore(datastore.NewMapDatastore()),
		opts.BootstrapPeers(nil),
	}

	hd, bsCh, err := head.NewHead(ctx, append(defaults, options...)...)
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

	return hd, nil
}

// SpawnHeads creates n new Hydra nodes with an in memory datastore and 0 bootstrap peers by default
func SpawnHeads(ctx context.Context, n int, options ...opts.Option) ([]*head.Head, error) {
	defaults := []opts.Option{
		opts.Datastore(datastore.NewMapDatastore()),
		opts.BootstrapPeers(nil),
	}

	var hds []*head.Head
	for i := 0; i < n; i++ {
		hd, err := SpawnHead(ctx, append(defaults, options...)...)
		if err != nil {
			for _, nd := range hds {
				nd.Host.Close()
			}
			return nil, err
		}
		hds = append(hds, hd)
	}

	return hds, nil
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
