package httpapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/hydra"
	hytesting "github.com/libp2p/hydra-booster/testing"
)

func TestHTTPAPISybils(t *testing.T) {
	nodes, err := hytesting.SpawnNodes(2)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewServeMux(&hydra.Hydra{Sybils: nodes}))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/sybils", listener.Addr().String())
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
