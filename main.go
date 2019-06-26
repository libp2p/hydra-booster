package main

import (
	"context"
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
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

var _ = circuit.P_CIRCUIT

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
			panic(err)
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

var bootstrapDone int64

func makeAndStartNode(ds ds.Batching, addr string, relay bool, bucketSize int, limiter chan struct{}) (host.Host, *dht.IpfsDHT, error) {
	opts := []libp2p.Option{libp2p.ListenAddrStrings(addr)}
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
		err = h.Connect(context.Background(), boostrapper())
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Second)

		d.BootstrapOnce(context.Background(), dht.BootstrapConfig{Queries: 4})
		d.FindPeer(context.Background(), peer.ID("foo"))
		if limiter != nil {
			<-limiter
		}
		atomic.AddInt64(&bootstrapDone, 1)

	}()
	return h, d, nil
}

func main() {
	many := flag.Int("many", -1, "Instead of running one dht, run many!")
	dbpath := flag.String("db", "dht-data", "Database folder")
	inmem := flag.Bool("mem", false, "Use an in-memory database. This overrides the -db option")
	pprofport := flag.Int("pprof-port", -1, "Specify a port to run pprof http server on")
	relay := flag.Bool("relay", false, "Enable libp2p circuit relaying for this node")
	portBegin := flag.Int("portBegin", 0, "If set, begin port allocation here")
	bucketSize := flag.Int("bucketSize", defaultKValue, "Specify the bucket size")
	bootstrapConcurency := flag.Int("bootstrapConc", 128, "How many concurrent bootstraps to run")
	flag.Parse()
	id.ClientVersion = "dhtbooster/2"

	if *relay {
		id.ClientVersion += "+relay"
	}

	port := *portBegin
	getPort := func() int {
		if port == 0 {
			return 0
		}

		out := port
		port++
		return out
	}

	if *inmem {
		*dbpath = ""
	}
	if *many == -1 {
		runSingleDHTWithUI(*dbpath, *relay, *bucketSize)
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

	provs := make(chan *provInfo, 16)
	r, w := io.Pipe()
	logwriter.WriterGroup.AddWriter(w)
	go waitForNotifications(r, provs, nil)

	limiter := make(chan struct{}, *bootstrapConcurency)
	for i := 0; i < *many; i++ {
		laddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", getPort())
		h, d, err := makeAndStartNode(ds, laddr, *relay, *bucketSize, limiter)
		if err != nil {
			panic(err)
		}
		hosts = append(hosts, h)
		dhts = append(dhts, d)
	}

	if *pprofport > 0 {
		fmt.Println("Running metrics server on port: %d", *pprofport)
		go setupMetrics(*pprofport)
	}

	totalprovs := 0
	reportInterval := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-provs:
			totalprovs++
		case <-reportInterval.C:
			printStatusLine(*many, start, hosts, dhts, uniqpeers, totalprovs, limiter)
		}
	}
}

func printStatusLine(ndht int, start time.Time, hosts []host.Host, dhts []*dht.IpfsDHT, uniqprs map[peer.ID]struct{}, totalprovs int, limiter chan struct{}) {
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

	fmt.Fprintf(os.Stderr, "[NumDhts: %d, Uptime: %s, Memory Usage: %s, TotalPeers: %d/%d, Total Provs: %d, BootstrapLimiter: %d/%d, BootstrapsDone: %d]\n", ndht, uptime, human.Bytes(mstat.Alloc), totalpeers, len(uniqprs), totalprovs, len(limiter), cap(limiter), atomic.LoadInt64(&bootstrapDone))
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

	// register prometheus with opencensus
	view.RegisterExporter(pe)
	view.SetReportingPeriod(2)

	// register the metrics views of interest
	if err := view.Register(dhtmetrics.DefaultViews...); err != nil {
		return err
	}

	go func() {
		mux := http.NewServeMux()
		zpages.Handle(mux, "/debug")
		mux.Handle("/metrics", pe)
		mux.Handle("/debug/vars", expvar.Handler())
		mux.HandleFunc("/debug/pprof", pprof.Index)
		// mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		// mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		// mux.Handle("/debug/pprof/block", pprof.Handler("block"))
		mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		// mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
		// mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux); err != nil {
			log.Fatalf("Failed to run Prometheus /metrics endpoint: %v", err)
		}
	}()
	return nil
}
