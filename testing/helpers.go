package testing

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/node"
	"github.com/libp2p/hydra-booster/node/opts"
)

// Defaults are the SpawnNode defaults
var defaults = []opts.Option{
	opts.Datastore(datastore.NewMapDatastore()),
	opts.BootstrapPeers(nil),
}

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

// GeneratePeerID ...
func GeneratePeerID() (peer.ID, crypto.PrivKey, crypto.PubKey, error) {
	priv, pub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return "", nil, nil, err
	}

	peerId, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return "", nil, nil, err
	}

	return peerId, priv, pub, nil
}
