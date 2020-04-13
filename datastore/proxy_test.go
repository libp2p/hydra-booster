package datastore

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/multiformats/go-base32"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

type FindProvidersAsyncFunc func(context.Context, cid.Cid, int) <-chan peer.AddrInfo

type MockDHT struct {
	dht.IpfsDHT
	mockFindProvidersAsync FindProvidersAsyncFunc
}

func (mdht *MockDHT) FindProvidersAsync(ctx context.Context, c cid.Cid, n int) <-chan peer.AddrInfo {
	return mdht.mockFindProvidersAsync(ctx, c, n)
}

func NewMockDHT(findProvs FindProvidersAsyncFunc) *MockDHT {
	return &MockDHT{mockFindProvidersAsync: findProvs}
}

func TestProviderKeyToCIDNamespacesError(t *testing.T) {
	_, err := providerKeyToCID(datastore.NewKey("invalid"))
	if err != errInvalidKeyNamespaces {
		t.Fatal("expected invalid key namespaces error")
	}
}

func TestProviderKeyToCIDEncodingBase32Error(t *testing.T) {
	_, err := providerKeyToCID(datastore.NewKey("/providers/8"))
	if err == nil {
		t.Fatal("expected invalid base32 encoding error")
	}
}

func TestProviderKeyToCIDEncodingCIDError(t *testing.T) {
	_, err := providerKeyToCID(datastore.NewKey("/providers/base32notcid"))
	if err == nil {
		t.Fatal("expected invalid CID encoding error")
	}
}

func TestNotFoundProvidersNetwork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mh, err := multihash.FromB58String("QmbWqxBEKC3P8tqsKc98xmWNzrzDtRLMiMPL8wBuTGsMnR")
	if err != nil {
		t.Fatal(err)
	}

	pfx := providers.ProvidersKeyPrefix + base32.RawStdEncoding.EncodeToString(mh)

	wg := sync.WaitGroup{}
	wg.Add(1)

	findProvs := func(_ context.Context, _ cid.Cid, _ int) <-chan peer.AddrInfo {
		ch := make(chan peer.AddrInfo)
		go func() {
			close(ch)
			wg.Done()
		}()
		return ch
	}

	dht := NewMockDHT(findProvs)

	addProvider := func(ctx context.Context, c cid.Cid, id peer.ID) {
		t.Fatal("unexpected provider")
	}

	getRouting := func(_ cid.Cid) (routing.Routing, AddProviderFunc, error) {
		return dht, addProvider, nil
	}

	ds := NewProxy(ctx, datastore.NewMapDatastore(), getRouting, Options{})
	defer ds.Close()

	res, err := ds.Query(query.Query{Prefix: pfx})
	if err != nil {
		t.Fatal(err)
	}

	_, ok := res.NextSync()
	if ok {
		t.Fatal("unexpectedly found a result")
	}

	wg.Wait()
}

func TestFoundProvidersNetwork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mh, err := multihash.FromB58String("QmbWqxBEKC3P8tqsKc98xmWNzrzDtRLMiMPL8wBuTGsMnR")
	if err != nil {
		t.Fatal(err)
	}

	pfx := providers.ProvidersKeyPrefix + base32.RawStdEncoding.EncodeToString(mh)

	wg := sync.WaitGroup{}
	wg.Add(1)

	provAddr, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/5001/p2p/QmeChhUxoWo2CQSdNjcbLemQHecVAuganTDoVejaZGJFKb")
	if err != nil {
		t.Fatal(err)
	}

	prov, err := peer.AddrInfoFromP2pAddr(provAddr)
	if err != nil {
		t.Fatal(err)
	}

	findProvs := func(_ context.Context, _ cid.Cid, _ int) <-chan peer.AddrInfo {
		ch := make(chan peer.AddrInfo)
		go func() {
			ch <- *prov
			close(ch)
		}()
		return ch
	}

	dht := NewMockDHT(findProvs)

	var addedProvID peer.ID
	addProvider := func(ctx context.Context, c cid.Cid, id peer.ID) {
		addedProvID = id
		wg.Done()
	}

	getRouting := func(_ cid.Cid) (routing.Routing, AddProviderFunc, error) {
		return dht, addProvider, nil
	}

	ds := NewProxy(ctx, datastore.NewMapDatastore(), getRouting, Options{})
	defer ds.Close()

	res, err := ds.Query(query.Query{Prefix: pfx})
	if err != nil {
		t.Fatal(err)
	}

	_, ok := res.NextSync()
	if ok {
		t.Fatal("unexpectedly found a result")
	}

	wg.Wait()

	if addedProvID != prov.ID {
		t.Fatalf("%v was not the expected provider ID", addedProvID)
	}
}

func TestIgnoresNonProviderKeys(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pfx := "/notproviderskey/"

	getRouting := func(_ cid.Cid) (routing.Routing, AddProviderFunc, error) {
		t.Fatal("did not ignore key")
		return nil, nil, nil
	}

	ds := NewProxy(ctx, datastore.NewMapDatastore(), getRouting, Options{})
	defer ds.Close()

	ds.Put(datastore.NewKey(pfx+"test"), []byte("test"))

	res, err := ds.Query(query.Query{Prefix: pfx})
	if err != nil {
		t.Fatal(err)
	}

	_, ok := res.NextSync()
	if !ok {
		t.Fatal("did not find result")
	}

	time.Sleep(time.Second) // Give the queue a second to potentially process this
}

func TestFoundProvidersLocal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mh, err := multihash.FromB58String("QmbWqxBEKC3P8tqsKc98xmWNzrzDtRLMiMPL8wBuTGsMnR")
	if err != nil {
		t.Fatal(err)
	}

	pfx := providers.ProvidersKeyPrefix + base32.RawStdEncoding.EncodeToString(mh)

	getRouting := func(_ cid.Cid) (routing.Routing, AddProviderFunc, error) {
		t.Fatal("was not found locally")
		return nil, nil, nil
	}

	ds := NewProxy(ctx, datastore.NewMapDatastore(), getRouting, Options{})
	defer ds.Close()

	ds.Put(datastore.NewKey(pfx+"/test"), []byte("test"))

	res, err := ds.Query(query.Query{Prefix: pfx})
	if err != nil {
		t.Fatal(err)
	}

	_, ok := res.NextSync()
	if !ok {
		t.Fatal("did not find result")
	}

	_, ok = res.NextSync()
	if ok {
		t.Fatal("expected only one result")
	}

	time.Sleep(time.Second) // Give the queue a second to potentially process this
}
