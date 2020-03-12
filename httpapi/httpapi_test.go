package httpapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"

	dsq "github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-core/peer"
	hytesting "github.com/libp2p/hydra-booster/testing"
)

func TestHTTPAPISybils(t *testing.T) {
	sybils, err := hytesting.SpawnNodes(2)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewServeMux(sybils))
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
		for _, syb := range sybils {
			if ai.ID == syb.Host.ID() {
				found = true
				break
			}
		}
		if !found {
			t.Fatal(fmt.Errorf("%s not found in spawned node peer IDs", ai.ID))
		}
	}
}

func TestHTTPAPIRecordsListWithoutRecords(t *testing.T) {
	sybils, err := hytesting.SpawnNodes(1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewServeMux(sybils))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/records/list", listener.Addr().String())
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Fatal(fmt.Errorf("got non-2XX status code %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	entries := []dsq.Entry{}

	for {
		var e dsq.Entry
		if err := dec.Decode(&e); err != nil {
			break
		}
		entries = append(entries, e)
	}

	if len(entries) > 0 {
		t.Fatal(fmt.Errorf("Expected to have 0 records stored, found %d", len(entries)))
	}
}
