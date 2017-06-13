package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"

	human "github.com/dustin/go-humanize"
	ds "github.com/ipfs/go-datastore"
	levelds "github.com/ipfs/go-ds-leveldb"
	golog "github.com/ipfs/go-log"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	id "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	testutil "github.com/libp2p/go-testutil"
	ma "github.com/multiformats/go-multiaddr"
)

var log = golog.Logger("dhtbooster")

func makeBasicHost(listen string) (host.Host, error) {
	addr, err := ma.NewMultiaddr(listen)
	if err != nil {
		return nil, err
	}

	ps := pstore.NewPeerstore()
	var pid peer.ID

	ident, err := testutil.RandIdentity()
	if err != nil {
		return nil, err
	}

	ident.PrivateKey()
	ps.AddPrivKey(ident.ID(), ident.PrivateKey())
	ps.AddPubKey(ident.ID(), ident.PublicKey())
	pid = ident.ID()
	fmt.Println("I am peer: ", pid.Pretty())

	ctx := context.Background()

	// create a new swarm to be used by the service host
	netw, err := swarm.NewNetwork(ctx, []ma.Multiaddr{addr}, pid, ps, nil)
	if err != nil {
		return nil, err
	}

	return bhost.New(netw), nil
}

type Event struct {
	Event  string
	System string
	Time   string
}

type provInfo struct {
	Key      string
	Duration time.Duration
}

func waitForNotifications(r io.Reader, provs chan *provInfo, mesout chan string) {
	var e map[string]interface{}
	dec := json.NewDecoder(r)
	for {
		err := dec.Decode(&e)
		if err != nil {
			panic(err)
		}
		event := e["event"]
		if event == "handleAddProvider" {
			provs <- &provInfo{
				Key:      e["key"].(string),
				Duration: time.Duration(e["duration"].(float64)),
			}
		}
	}
}

func boostrapper() pstore.PeerInfo {
	a, err := ma.NewMultiaddr("/ip4/128.199.219.111/tcp/4001")
	if err != nil {
		panic(err)
	}

	pid, err := peer.IDB58Decode("QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu")
	if err != nil {
		panic(err)
	}

	return pstore.PeerInfo{
		ID:    pid,
		Addrs: []ma.Multiaddr{a},
	}
}

func makeAndStartDHT(ds ds.Batching, addr string) (host.Host, *dht.IpfsDHT, error) {
	h, err := makeBasicHost(addr)
	if err != nil {
		panic(err)
	}

	d := dht.NewDHT(context.Background(), h, ds)
	if err != nil {
		panic(err)
	}

	go func() {
		err = h.Connect(context.Background(), boostrapper())
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Second)
		d.FindPeer(context.Background(), peer.ID("foo"))
	}()
	return h, d, nil
}

func main() {
	many := flag.Int("many", -1, "Instead of running one dht, run many!")
	dbpath := flag.String("db", "dht-data", "Database folder")
	inmem := flag.Bool("mem", false, "Use an in-memory database. This overrides the -db option")
	pprofport := flag.Int("pprof-port", -1, "Specify a port to run pprof http server on")
	flag.Parse()
	id.ClientVersion = "dhtbooster/1"

	if *pprofport >= 0 {
		go func() {
			fmt.Printf("Http server listening on port: %d\n", *pprofport)
			panic(http.ListenAndServe(fmt.Sprintf(":%d", *pprofport), nil))
		}()
	}

	if *inmem {
		*dbpath = ""
	}
	if *many == -1 {
		runSingleDHTWithUI(*dbpath)
	}

	ds, err := levelds.NewDatastore(*dbpath, nil)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	var hosts []host.Host
	var dhts []*dht.IpfsDHT
	uniqpeers := make(map[peer.ID]struct{})
	fmt.Fprintf(os.Stderr, "Running %d DHT Instances...", *many)
	for i := 0; i < *many; i++ {
		h, d, err := makeAndStartDHT(ds, "/ip4/0.0.0.0/tcp/0")
		if err != nil {
			panic(err)
		}
		hosts = append(hosts, h)
		dhts = append(dhts, d)
	}

	for range time.Tick(time.Second * 5) {
		printStatusLine(*many, start, hosts, dhts, uniqpeers)
	}
}

func printStatusLine(ndht int, start time.Time, hosts []host.Host, dhts []*dht.IpfsDHT, uniqprs map[peer.ID]struct{}) {
	uptime := time.Second * time.Duration(int(time.Since(start).Seconds()))
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)
	var totalpeers int
	for _, h := range hosts {
		peers := h.Network().Peers()
		totalpeers += len(peers)
		for _, p := range peers {
			uniqprs[p] = struct{}{}
		}
	}

	fmt.Fprintf(os.Stderr, "[NumDhts: %d, Uptime: %s, Memory Usage: %s, TotalPeers: %d/%d]\n", ndht, uptime, human.Bytes(mstat.Alloc), totalpeers, len(uniqprs))
}

func runSingleDHTWithUI(path string) {
	ds, err := levelds.NewDatastore(path, nil)
	if err != nil {
		panic(err)
	}
	h, _, err := makeAndStartDHT(ds, "/ip4/0.0.0.0/tcp/19264")
	if err != nil {
		panic(err)
	}

	uniqpeers := make(map[peer.ID]struct{})
	messages := make(chan string, 16)
	provs := make(chan *provInfo, 16)
	r, w := io.Pipe()
	golog.WriterGroup.AddWriter(w)
	go waitForNotifications(r, provs, messages)

	ga := &GooeyApp{Title: "Libp2p DHT Node", Log: NewLog(15, 15)}
	ga.NewDataLine(3, "Peer ID", h.ID().Pretty())
	econs := ga.NewDataLine(4, "Connections", "0")
	uniqprs := ga.NewDataLine(5, "Unique Peers Seen", "0")
	emem := ga.NewDataLine(6, "Memory Allocated", "0MB")
	eprov := ga.NewDataLine(7, "Stored Provider Records", "0")
	eprlat := ga.NewDataLine(8, "Store Provider Latency", "0s")
	etime := ga.NewDataLine(9, "Uptime", "0h 0m 0s")
	ga.Print()
	mt := time.NewTicker(time.Second * 3)
	second := time.NewTicker(time.Second)
	start := time.Now()
	var totalprovs int
	var totalprovtime time.Duration
	for {
		select {
		case m := <-messages:
			ga.Log.Add(m)
			ga.Log.Print()
		case <-mt.C:
			ga.Print()
			var mstat runtime.MemStats
			runtime.ReadMemStats(&mstat)
			emem.SetVal(human.Bytes(mstat.Alloc))
			peers := h.Network().Peers()
			econs.SetVal(fmt.Sprintf("%d peers", len(peers)))
			for _, p := range peers {
				uniqpeers[p] = struct{}{}
			}
			uniqprs.SetVal(fmt.Sprint(len(uniqpeers)))
		case p := <-provs:
			totalprovs++
			totalprovtime += p.Duration
			eprov.SetVal(fmt.Sprint(totalprovs))
			eprlat.SetVal(fmt.Sprint(totalprovtime / time.Duration(totalprovs)))
		case <-second.C:
			t := time.Since(start)
			h := int(t.Hours())
			m := int(t.Minutes()) % 60
			s := int(t.Seconds()) % 60
			etime.SetVal(fmt.Sprintf("%dh %dm %ds", h, m, s))
		}
	}
}
