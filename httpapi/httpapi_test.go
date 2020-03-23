package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"

	dsq "github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/hydra"
	hytesting "github.com/libp2p/hydra-booster/testing"
)

func TestHTTPAPISybils(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sybils, err := hytesting.SpawnSybils(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Sybils: sybils}))
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sybils, err := hytesting.SpawnSybils(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Sybils: sybils, SharedDatastore: sybils[0].Datastore}))
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

func TestHTTPAPIRecordsFetch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sybils, err := hytesting.SpawnSybils(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Sybils: sybils, SharedDatastore: sybils[0].Datastore}))
	defer listener.Close()

	// Missing CID
	url := fmt.Sprintf("http://%s/records/fetch", listener.Addr().String())
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 404 {
		t.Fatal(fmt.Errorf("Should have got a 404, got %d: %s", res.StatusCode, url))
	}

	// Malformed CID
	url = fmt.Sprintf("http://%s/records/fetch/notacid", listener.Addr().String())
	res, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatal(fmt.Errorf("Should have got a 400, got %d: %s", res.StatusCode, url))
	}

	// Malformed queryString
	url = fmt.Sprintf("http://%s/records/fetch/notacid?nProviders=bananas", listener.Addr().String())
	res, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatal(fmt.Errorf("Should have got a 400, got %d: %s", res.StatusCode, url))
	}

	// Valid
	url = fmt.Sprintf("http://%s/records/fetch/QmVBEq6nnXQR2Ueb6etMFMUVhGM5vu34Y2KfHW5FVdGFok", listener.Addr().String())
	res, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Fatal(fmt.Errorf("got non-2XX status code %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	entries := []peer.AddrInfo{}

	for {
		var e peer.AddrInfo
		if err := dec.Decode(&e); err != nil {
			break
		}
		entries = append(entries, e)
	}

	// We can ensure how many we will get as we are testing this with live network
	// if len(entries) >= 0 {
	// 	t.Fatal(fmt.Errorf("Expected to found 0 or more records, found %d", len(entries)))
	// }

	// Valid with queryString
	url = fmt.Sprintf("http://%s/records/fetch/QmVBEq6nnXQR2Ueb6etMFMUVhGM5vu34Y2KfHW5FVdGFok?nProviders=2", listener.Addr().String())
	res, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Fatal(fmt.Errorf("got non-2XX status code %d: %s", res.StatusCode, url))
	}

	dec = json.NewDecoder(res.Body)
	entries = []peer.AddrInfo{}

	for {
		var e peer.AddrInfo
		if err := dec.Decode(&e); err != nil {
			break
		}
		entries = append(entries, e)
	}

	// We can ensure how many we will get as we are testing this with live network
	// if len(entries) >= 0 {
	// 	t.Fatal(fmt.Errorf("Expected to found 0 or more records, found %d", len(entries)))
	// }
}
