package httpapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/node"
	"github.com/libp2p/hydra-booster/opts"
	"github.com/multiformats/go-multiaddr"
)

var noBootstrappers = []multiaddr.Multiaddr{}

func spawnNodes(n int) ([]*node.HydraNode, error) {
	var nodes []*node.HydraNode
	for i := 0; i < n; i++ {
		nd, _, err := node.NewHydraNode(opts.Datastore(datastore.NewMapDatastore()), opts.BootstrapPeers(noBootstrappers))
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

func TestHTTPAPIPeers(t *testing.T) {
	nodes, err := spawnNodes(2)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewServeMux(nodes))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/peers", listener.Addr().String())
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Fatal(fmt.Errorf("got non-2XX status code %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	ais := []peer.AddrInfo{}

	for {
		var ai peer.AddrInfo
		if err := dec.Decode(&ai); err != nil {
			break
		}
		ais = append(ais, ai)
	}

	for _, ai := range ais {
		found := false
		for _, node := range nodes {
			if ai.ID == node.Host.ID() {
				found = true
				break
			}
		}
		if !found {
			t.Fatal(fmt.Errorf("%s not found in spawned node peer IDs", ai.ID))
		}
	}
}
