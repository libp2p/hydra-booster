package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/ipfs/go-cid"
	dsq "github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/hydra-booster/hydra"
	"github.com/libp2p/hydra-booster/idgen"
	hytesting "github.com/libp2p/hydra-booster/testing"
)

func TestHTTPAPIHeads(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds}))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/heads", listener.Addr().String())
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
		for _, hd := range hds {
			if ai.ID == hd.Host.ID() {
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

	hds, err := hytesting.SpawnHeads(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds, SharedDatastore: hds[0].Datastore}))
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

	hds, err := hytesting.SpawnHeads(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds, SharedDatastore: hds[0].Datastore}))
	defer listener.Close()

	cidStr := "QmVBEq6nnXQR2Ueb6etMFMUVhGM5vu34Y2KfHW5FVdGFok"
	cid, err := cid.Decode(cidStr)
	if err != nil {
		t.Fatal(err)
	}

	// Add the provider as itself for the test
	// In an ideal testing scenario, we would spawn multiple nodes and see that they can indeed
	// fetch from each other
	hds[0].AddProvider(ctx, cid, hds[0].Host.ID())

	// Valid CID
	url := fmt.Sprintf("http://%s/records/fetch/%s", listener.Addr().String(), cidStr)
	res, err := http.Get(url)
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
	if len(entries) < 1 {
		t.Fatal(fmt.Errorf("Expected to found 1 or more records, found %d", len(entries)))
	}

	// Valid with queryString
	url = fmt.Sprintf("http://%s/records/fetch/%s?nProviders=2", listener.Addr().String(), cidStr)
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
	if len(entries) < 1 {
		t.Fatal(fmt.Errorf("Expected to found 1 or more records, found %d", len(entries)))
	}
}

func TestHTTPAPIRecordsFetchErrorStates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds, SharedDatastore: hds[0].Datastore}))
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
}

func TestHTTPAPIRecordsAdd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	cidStr := "QmVBEq6nnXQR2Ueb6etMFMUVhGM5vu34Y2KfHW5FVdGFok"
	cid, err := cid.Decode(cidStr)
	if err != nil {
		t.Fatal(err)
	}

	peerIdStr := "12D3KooWHacdCMnm4YKDJHn72HPTxc6LRGNzbrbyVEnuLFA3FXCZ"
	peerId, err := peer.Decode(peerIdStr)
	if err != nil {
		t.Fatal(err)
	}

	providerRouter := hds[0].Routing.(*dht.IpfsDHT)

	results := providerRouter.ProviderManager.GetProviders(ctx, cid.Bytes())
	if len(results) > 0 {
		t.Fatal("The node datastore is not empty at the beginning of the test.")
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds, SharedDatastore: hds[0].Datastore}))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/records/add", listener.Addr().String())
	res, err := http.Post(
		url, "application/json",
		bytes.NewReader([]byte(fmt.Sprintf("[{\"CID\": \"%s\", \"PeerID\": \"%s\"}]", cidStr, peerIdStr))))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 202 {
		t.Fatal(fmt.Errorf("Should have got a 202, got %d: %s", res.StatusCode, url))
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	} else if len(body) > 0 {
		t.Fatal(fmt.Errorf("The response has non empty body: %s", string(body)))
	}

	results = providerRouter.ProviderManager.GetProviders(ctx, cid.Bytes())
	if len(results) == 0 {
		t.Fatal("The node datastore is empty at the beginning of the test.")
	}

	if results[0] != peerId {
		t.Fatal(fmt.Sprintf("Should have received PeerID %s, got %s", results[0], peerIdStr))
	}
}

func TestHTTPAPIRecordsAddInvalidContentType(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds, SharedDatastore: hds[0].Datastore}))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/records/add", listener.Addr().String())
	res, err := http.Post(url, "text/plain", bytes.NewReader([]byte("{{")))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 415 {
		t.Fatal(fmt.Errorf("Should have got a 415, got %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	var apiError ApiError

	if err := dec.Decode(&apiError); err != nil {
		t.Fatal(err)
	}

	if apiError.Error != "Request must specify the Content-Type header as \"application/json\"." {
		t.Fatal(fmt.Errorf("Found unexpected API Error: %s", apiError.Error))
	}
}

func TestHTTPAPIRecordsAddMalformedJSON(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds, SharedDatastore: hds[0].Datastore}))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/records/add", listener.Addr().String())
	res, err := http.Post(url, "application/json", bytes.NewReader([]byte("{{")))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatal(fmt.Errorf("Should have got a 400, got %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	var apiError ApiError

	if err := dec.Decode(&apiError); err != nil {
		t.Fatal(err)
	}

	if apiError.Error != "Invalid request payload." {
		t.Fatal(fmt.Errorf("Found unexpected API Error: %s", apiError.Error))
	}
}

func TestHTTPAPIRecordsAddInvalidJSON(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds, SharedDatastore: hds[0].Datastore}))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/records/add", listener.Addr().String())
	res, err := http.Post(url, "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatal(fmt.Errorf("Should have got a 400, got %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	var apiError ApiError

	if err := dec.Decode(&apiError); err != nil {
		t.Fatal(err)
	}

	if apiError.Error != "Invalid request payload." {
		t.Fatal(fmt.Errorf("Found unexpected API Error: %s", apiError))
	}
}

func TestHTTPAPIRecordsAddInvalidCID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds, SharedDatastore: hds[0].Datastore}))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/records/add", listener.Addr().String())
	res, err := http.Post(url, "application/json", bytes.NewReader([]byte("[{\"CID\": \"INVALID\"}]")))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatal(fmt.Errorf("Should have got a 400, got %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	var apiError ApiError

	if err := dec.Decode(&apiError); err != nil {
		t.Fatal(err)
	}

	if apiError.Error != "Invalid CID provided on record[0]." {
		t.Fatal(fmt.Errorf("Found unexpected API Error: %s", apiError.Error))
	}
}

func TestHTTPAPIRecordsAddInvalidPeerID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds, SharedDatastore: hds[0].Datastore}))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/records/add", listener.Addr().String())
	res, err := http.Post(
		url, "application/json",
		bytes.NewReader([]byte("[{\"CID\": \"QmdmQXB2mzChmMeKY47C43LxUdg1NDJ5MWcKMKxDu7RgQm\", \"PeerID\": \"12D3KooWHacdCMnm4YKDJHn72HPTxc6LRGNzbrbyVEnuLFA3FXCZ\"}, {\"CID\": \"QmdmQXB2mzChmMeKY47C43LxUdg1NDJ5MWcKMKxDu7RgQm\", \"PeerID\": \"INVALID\"}]")))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 400 {
		t.Fatal(fmt.Errorf("Should have got a 400, got %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	var apiError ApiError

	if err := dec.Decode(&apiError); err != nil {
		t.Fatal(err)
	}

	if apiError.Error != "Invalid PeerID provided on record[1]." {
		t.Fatal(fmt.Errorf("Found unexpected API Error: %s", apiError.Error))
	}
}

func TestIDGeneratorAdd(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(nil))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/idgen/add", listener.Addr().String())
	res, err := http.Post(url, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatal(fmt.Errorf("unexpected status %d", res.StatusCode))
	}

	dec := json.NewDecoder(res.Body)
	var b64 string
	if err := dec.Decode(&b64); err != nil {
		t.Fatal(err)
	}

	bytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}

	_, err = crypto.UnmarshalPrivateKey(bytes)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIDGeneratorRemove(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(nil))
	defer listener.Close()

	pk, err := idgen.HydraIdentityGenerator.AddBalanced()
	if err != nil {
		t.Fatal(err)
	}

	b, err := crypto.MarshalPrivateKey(pk)
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(base64.StdEncoding.EncodeToString(b))
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("http://%s/idgen/remove", listener.Addr().String())
	res, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 204 {
		t.Fatal(fmt.Errorf("unexpected status %d", res.StatusCode))
	}
}

func TestIDGeneratorRemoveInvalidJSON(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(nil))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/idgen/remove", listener.Addr().String())
	res, err := http.Post(url, "application/json", bytes.NewReader([]byte("{{")))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatal(fmt.Errorf("unexpected status %d", res.StatusCode))
	}
}

func TestIDGeneratorRemoveInvalidBase64(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(nil))
	defer listener.Close()

	url := fmt.Sprintf("http://%s/idgen/remove", listener.Addr().String())
	res, err := http.Post(url, "application/json", bytes.NewReader([]byte("\"! invalid b64 !\"")))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatal(fmt.Errorf("unexpected status %d", res.StatusCode))
	}
}

func TestIDGeneratorRemoveInvalidPrivateKey(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(nil))
	defer listener.Close()

	data, err := json.Marshal(base64.StdEncoding.EncodeToString([]byte("invalid private key")))
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("http://%s/idgen/remove", listener.Addr().String())
	res, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatal(fmt.Errorf("unexpected status %d", res.StatusCode))
	}
}

type hostPeer struct {
	ID   peer.ID
	Peer struct {
		ID        peer.ID
		Addr      string
		Direction int
	}
}

func TestHTTPAPISwarmPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds}))
	defer listener.Close()

	err = hds[0].Host.Connect(ctx, peer.AddrInfo{
		ID:    hds[1].Host.ID(),
		Addrs: hds[1].Host.Addrs(),
	})
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("http://%s/swarm/peers", listener.Addr().String())
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Fatal(fmt.Errorf("got non-2XX status code %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	hps := []hostPeer{}

	for {
		var hp hostPeer
		if err := dec.Decode(&hp); err != nil {
			break
		}
		hps = append(hps, hp)
	}

	found := false
	for _, hp := range hps {
		if hp.Peer.ID == hds[1].Host.ID() {
			found = true
			break
		}
	}

	if !found {
		t.Fatal(fmt.Errorf("head %s not in peer list", hds[1].Host.ID()))
	}
}

func TestHTTPAPISwarmPeersHeadFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hds, err := hytesting.SpawnHeads(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	go http.Serve(listener, NewRouter(&hydra.Hydra{Heads: hds}))
	defer listener.Close()

	err = hds[0].Host.Connect(ctx, peer.AddrInfo{
		ID:    hds[1].Host.ID(),
		Addrs: hds[1].Host.Addrs(),
	})
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("http://%s/swarm/peers?head=%s", listener.Addr().String(), hds[0].Host.ID())
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Fatal(fmt.Errorf("got non-2XX status code %d: %s", res.StatusCode, url))
	}

	dec := json.NewDecoder(res.Body)
	hps := []hostPeer{}

	for {
		var hp hostPeer
		if err := dec.Decode(&hp); err != nil {
			break
		}
		hps = append(hps, hp)
	}

	for _, hp := range hps {
		if hp.ID != hds[0].Host.ID() {
			t.Fatal(fmt.Errorf("unexpectedly found head %s in peer list", hp.ID))
		}
	}
}
