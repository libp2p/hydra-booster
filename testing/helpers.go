package testing

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/hydra-booster/node"
	"github.com/libp2p/hydra-booster/opts"
)

var defaults = []opts.Option{opts.Datastore(datastore.NewMapDatastore()), opts.BootstrapPeers(nil)}

// SpawnNode a new Hydra nodes with an in memory datastore and 0 bootstrap peers by default.
// It also waits for bootstrapping to complete.
func SpawnNode(options ...opts.Option) (*node.HydraNode, error) {
	nd, bsCh, err := node.NewHydraNode(append(defaults, options...)...)
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

// SpawnNodes creates n new Hydra nodes with an in memory datastore and 0 bootstrap peers by default
func SpawnNodes(n int, options ...opts.Option) ([]*node.HydraNode, error) {
	var nodes []*node.HydraNode
	for i := 0; i < n; i++ {
		nd, err := SpawnNode(options...)
		if err != nil {
			for _, nd := range nodes {
				nd.Host.Close()
			}
			return nil, err
		}
		nodes = append(nodes, nd)
	}

	return nodes, nil
}
