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
	ipns "github.com/ipfs/go-ipns"
	logging "github.com/ipfs/go-log"
	logwriter "github.com/ipfs/go-log/writer"
	libp2p "github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	record "github.com/libp2p/go-libp2p-record"
	id "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	ma "github.com/multiformats/go-multiaddr"
)

var _ = circuit.P_CIRCUIT

var log = logging.Logger("dhtbooster")

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
		// TODO: this is broken now, the log format for go-log eventlogs has changed
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

func makeAndStartNode(ds ds.Batching, addr string, relay bool, retries int) (host.Host, *dht.IpfsDHT, error) {
	opts := []libp2p.Option{libp2p.ListenAddrStrings(addr)}
	if relay {
		opts = append(opts, libp2p.EnableRelay(circuit.OptHop))
	}

	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		panic(err)
	}

	d := dht.NewDHT(context.Background(), h, ds)
	if err != nil {
		panic(err)
	}

	d.Validator = record.NamespacedValidator{
		"pk":   record.PublicKeyValidator{},
		"ipns": ipns.Validator{KeyBook: h.Peerstore()},
	}

	go func() {
		var err error
		for r := 0; r < retries; r++ {
			err = h.Connect(context.Background(), boostrapper())
			if err == nil {
				break
			}
			time.Sleep(6000 * time.Millisecond)
			fmt.Fprintf(os.Stderr, "Error starting node: %s\n", err.Error())
		}
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
	retries := flag.Int("retries", 1, "Number of times to retry starting nodes")
	dbpath := flag.String("db", "dht-data", "Database folder")
	inmem := flag.Bool("mem", false, "Use an in-memory database. This overrides the -db option")
	pprofport := flag.Int("pprof-port", -1, "Specify a port to run pprof http server on")
	relay := flag.Bool("relay", false, "Enable libp2p circuit relaying for this node")
	flag.Parse()
	id.ClientVersion = "dhtbooster/2"

	if *relay {
		id.ClientVersion += "+relay"
	}

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
		runSingleDHTWithUI(*dbpath, *relay, *retries)
	}

	ds, err := levelds.NewDatastore(*dbpath, nil)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	var hosts []host.Host
	var dhts []*dht.IpfsDHT
	uniqpeers := make(map[peer.ID]struct{})
	fmt.Fprintf(os.Stderr, "Running %d DHT Instances...\n", *many)
	for i := 0; i < *many; i++ {
		h, d, err := makeAndStartNode(ds, "/ip4/0.0.0.0/tcp/0", *relay, *retries)
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

func runSingleDHTWithUI(path string, relay bool, retries int) {
	ds, err := levelds.NewDatastore(path, nil)
	if err != nil {
		panic(err)
	}
	h, _, err := makeAndStartNode(ds, "/ip4/0.0.0.0/tcp/19264", relay, retries)
	if err != nil {
		panic(err)
	}

	uniqpeers := make(map[peer.ID]struct{})
	messages := make(chan string, 16)
	provs := make(chan *provInfo, 16)
	r, w := io.Pipe()
	logwriter.WriterGroup.AddWriter(w)
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
