package head

import (
	"context"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/hydra-booster/head/opts"
)

// SpawnHead creates a new Hydra head with an in memory datastore and 0 bootstrap peers by default.
// It also waits for bootstrapping to complete.
func SpawnTestHead(ctx context.Context, options ...opts.Option) (*Head, error) {
	defaults := []opts.Option{
		opts.Datastore(datastore.NewMapDatastore()),
		opts.BootstrapPeers(nil),
	}
	hd, bsCh, err := NewHead(ctx, append(defaults, options...)...)
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
func SpawnTestHeads(ctx context.Context, n int, options ...opts.Option) ([]*Head, error) {
	var hds []*Head
	for i := 0; i < n; i++ {
		hd, err := SpawnTestHead(ctx, options...)
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
