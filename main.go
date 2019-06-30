package main

import (
	"context"
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/axiomhq/hyperloglog"
	human "github.com/dustin/go-humanize"
	ds "github.com/ipfs/go-datastore"
	levelds "github.com/ipfs/go-ds-leveldb"
	ipns "github.com/ipfs/go-ipns"
	logging "github.com/ipfs/go-log"
	logwriter "github.com/ipfs/go-log/writer"
	libp2p "github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	network "github.com/libp2p/go-libp2p-core/network"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	dhtmetrics "github.com/libp2p/go-libp2p-kad-dht/metrics"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	record "github.com/libp2p/go-libp2p-record"
	id "github.com/libp2p/go-libp2p/p2p/protocol/identify"
	ma "github.com/multiformats/go-multiaddr"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
)

var _ = dhtmetrics.DefaultViews
var _ = circuit.P_CIRCUIT
var _ = logwriter.WriterGroup

var (
	log           = logging.Logger("dhtbooster")
	defaultKValue = 20
)

// Event is an event.
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
			fmt.Printf("waitForNotifications error: %s\n", err)
			close(provs)
			return
		}

		event := e["Operation"]
		if event == "handleAddProvider" {
			provs <- &provInfo{
				Key:      (e["Tags"].(map[string]interface{}))["key"].(string),
				Duration: time.Duration(e["Duration"].(float64)),
			}
		}
	}
}

var bootstrappers = []string{
	"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",  // mars.i.ipfs.io
	"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM", // pluto.i.ipfs.io
	"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu", // saturn.i.ipfs.io
	"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",   // venus.i.ipfs.io
	"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",  // earth.i.ipfs.io
}

func bootstrapper() pstore.PeerInfo {
	bsa := bootstrappers[rand.Intn(len(bootstrappers))]

	a, err := ma.NewMultiaddr(bsa)
	if err != nil {
		panic(err)
	}

	ai, err := pstore.InfoFromP2pAddr(a)
	if err != nil {
		panic(err)
	}

	return *ai
}

var bootstrapDone int64

func makeAndStartNode(ds ds.Batching, addr string, relay bool, bucketSize int, limiter chan struct{}) (host.Host, *dht.IpfsDHT, error) {
	cmgr := connmgr.NewConnManager(1500, 2000, time.Minute)

	priv, _, _ := crypto.GenerateKeyPair(crypto.Ed25519, 0)

	opts := []libp2p.Option{libp2p.ListenAddrStrings(addr), libp2p.ConnectionManager(cmgr), libp2p.Identity(priv)}
	if relay {
		opts = append(opts, libp2p.EnableRelay(circuit.OptHop))
	}

	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		panic(err)
	}

	d, err := dht.New(context.Background(), h, dhtopts.BucketSize(bucketSize), dhtopts.Datastore(ds))
	if err != nil {
		panic(err)
	}

	d.Validator = record.NamespacedValidator{
		"pk":   record.PublicKeyValidator{},
		"ipns": ipns.Validator{KeyBook: h.Peerstore()},
	}

	go func() {
		if limiter != nil {
			limiter <- struct{}{}
		}

		for i := 0; i < 2; i++ {
			if err := h.Connect(context.Background(), bootstrapper()); err != nil {
				fmt.Println("bootstrap connect failed: ", err)
				i--
			}
		}

		time.Sleep(time.Second)

		timeout := time.Minute * 5
		tctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		d.BootstrapOnce(tctx, dht.BootstrapConfig{Queries: 4, Timeout: timeout})

		if limiter != nil {
			<-limiter
		}
		atomic.AddInt64(&bootstrapDone, 1)

	}()
	return h, d, nil
}

func portSelector(beg int) func() int {
	port := beg
	return func() int {
		if port == 0 {
			return 0
		}

		out := port
		port++
		return out + 1
	}
}

func main() {
	many := flag.Int("many", -1, "Instead of running one dht, run many!")
	dbpath := flag.String("db", "dht-data", "Database folder")
	inmem := flag.Bool("mem", false, "Use an in-memory database. This overrides the -db option")
	pprofport := flag.Int("pprof-port", -1, "Specify a port to run pprof http server on")
	relay := flag.Bool("relay", false, "Enable libp2p circuit relaying for this node")
	portBegin := flag.Int("portBegin", 0, "If set, begin port allocation here")
	bucketSize := flag.Int("bucketSize", defaultKValue, "Specify the bucket size")
	bootstrapConcurency := flag.Int("bootstrapConc", 32, "How many concurrent bootstraps to run")
	stagger := flag.Duration("stagger", 0*time.Second, "Duration to stagger nodes starts by")
	flag.Parse()
	id.ClientVersion = "dhtbooster/2"

	if *relay {
		id.ClientVersion += "+relay"
	}

	if *pprofport > 0 {
		fmt.Printf("Running metrics server on port: %d\n", *pprofport)
		go setupMetrics(*pprofport)
	}

	getPort := portSelector(*portBegin)

	if *inmem {
		*dbpath = ""
	}
	if *many == -1 {
		runSingleDHTWithUI(*dbpath, *relay, *bucketSize)
		return
	}

	runMany(*dbpath, getPort, *many, *bucketSize, *bootstrapConcurency, *relay, *stagger)
}

func runMany(dbpath string, getPort func() int, many, bucketSize, bsCon int, relay bool, stagger time.Duration) {
	ds, err := levelds.NewDatastore(dbpath, nil)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	var hosts []host.Host
	var dhts []*dht.IpfsDHT
	hyperlog := hyperloglog.New()

	var peersConnected int64

	notifiee := &network.NotifyBundle{
		ConnectedF: func(_ network.Network, v network.Conn) {
			hyperlog.Insert([]byte(v.RemotePeer()))
			atomic.AddInt64(&peersConnected, 1)
		},
		DisconnectedF: func(_ network.Network, v network.Conn) {
			atomic.AddInt64(&peersConnected, -1)
		},
	}

	fmt.Fprintf(os.Stderr, "Running %d DHT Instances:\n", many)

	limiter := make(chan struct{}, bsCon)
	for i := 0; i < many; i++ {
		time.Sleep(stagger)
		fmt.Fprintf(os.Stderr, ".")

		laddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", getPort())
		h, d, err := makeAndStartNode(ds, laddr, relay, bucketSize, limiter)
		if err != nil {
			panic(err)
		}
		h.Network().Notify(notifiee)
		hosts = append(hosts, h)
		dhts = append(dhts, d)
	}
	fmt.Fprintf(os.Stderr, "\n")

	provs := make(chan *provInfo, 16)
	//r, w := io.Pipe()
	//logwriter.WriterGroup.AddWriter(w)
	//go waitForNotifications(r, provs, nil)

	totalprovs := 0
	reportInterval := time.NewTicker(time.Second * 5)
	for {
		select {
		case _, ok := <-provs:
			if !ok {
				totalprovs = -1
				provs = nil
			} else {
				totalprovs++
			}
		case <-reportInterval.C:
			printStatusLine(many, start, atomic.LoadInt64(&peersConnected), hyperlog.Estimate(), totalprovs)
		}
	}
}

func printStatusLine(ndht int, start time.Time, totalpeers int64, uniqpeers uint64, totalprovs int) {
	uptime := time.Second * time.Duration(int(time.Since(start).Seconds()))
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	fmt.Fprintf(os.Stderr, "[NumDhts: %d, Uptime: %s, Memory Usage: %s, TotalPeers: %d/%d, Total Provs: %d, BootstrapsDone: %d]\n", ndht, uptime, human.Bytes(mstat.Alloc), totalpeers, uniqpeers, totalprovs, atomic.LoadInt64(&bootstrapDone))
}

func runSingleDHTWithUI(path string, relay bool, bucketSize int) {
	ds, err := levelds.NewDatastore(path, nil)
	if err != nil {
		panic(err)
	}
	h, _, err := makeAndStartNode(ds, "/ip4/0.0.0.0/tcp/19264", relay, bucketSize, nil)
	if err != nil {
		panic(err)
	}

	uniqpeers := make(map[peer.ID]struct{})
	messages := make(chan string, 16)
	provs := make(chan *provInfo, 16)
	//r, w := io.Pipe()
	//logwriter.WriterGroup.AddWriter(w)
	//go waitForNotifications(r, provs, messages)

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

func setupMetrics(port int) error {
	// setup Prometheus
	registry := prom.NewRegistry()
	goCollector := prom.NewGoCollector()
	procCollector := prom.NewProcessCollector(prom.ProcessCollectorOpts{})
	registry.MustRegister(goCollector, procCollector)
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "dht_node",
		Registry:  registry,
	})
	if err != nil {
		return err
	}

	_ = view.RegisterExporter
	/* Disabling opencensus for now, it allocates too much
	// register prometheus with opencensus
	view.RegisterExporter(pe)
	view.SetReportingPeriod(2)

	// register the metrics views of interest
	if err := view.Register(dhtmetrics.DefaultViews...); err != nil {
		return err
	}
	*/

	go func() {
		mux := http.NewServeMux()
		zpages.Handle(mux, "/debug")
		mux.Handle("/metrics", pe)
		mux.Handle("/debug/vars", expvar.Handler())

		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux); err != nil {
			log.Fatalf("Failed to run Prometheus /metrics endpoint: %v", err)
		}
	}()
	return nil
}
