package testing

import (
	"context"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
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

// GeneratePeerID ...
func GeneratePeerID() (peer.ID, crypto.PrivKey, crypto.PubKey, error) {
	priv, pub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return "", nil, nil, err
	}

	peerID, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return "", nil, nil, err
	}

	return peerID, priv, pub, nil
}
