package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/multiformats/go-multiaddr"

	"github.com/axiomhq/hyperloglog"
	human "github.com/dustin/go-humanize"
	levelds "github.com/ipfs/go-ds-leveldb"
	logging "github.com/ipfs/go-log"
	logwriter "github.com/ipfs/go-log/writer"
	circuit "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	network "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	dhtmetrics "github.com/libp2p/go-libp2p-kad-dht/metrics"
	"github.com/libp2p/hydra-booster/httpapi"
	"github.com/libp2p/hydra-booster/node"
	"github.com/libp2p/hydra-booster/opts"
	"github.com/libp2p/hydra-booster/ui"
)

func init() {
	// Allow short keys. Otherwise, we'll refuse connections from the bootsrappers and break the network.
	// TODO: Remove this when we shut those bootstrappers down.
	crypto.MinRsaKeyBits = 1024
}

var _ = dhtmetrics.DefaultViews
var _ = circuit.P_CIRCUIT
var _ = logwriter.WriterGroup

var log = logging.Logger("hydrabooster")

// Event ...
type Event struct {
	Event  string
	System string
	Time   string
}

type provInfo struct {
	Key      string
	Duration time.Duration
}

const singleDHTSwarmAddr = "/ip4/0.0.0.0/tcp/19264"
const httpAPIAddr = "127.0.0.1:7779"

var bootstrapDone int64

func handleBootstrapStatus(ch chan node.BootstrapStatus) {
	status, ok := <-ch
	if !ok {
		return
	}
	if status.Err != nil {
		fmt.Println(status.Err)
	}
	if status.Done {
		atomic.AddInt64(&bootstrapDone, 1)
	}
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

// RunMany ...
func RunMany(dbpath string, getPort func() int, many, bucketSize, bsCon int, relay bool, stagger time.Duration) error {
	sharedDatastore, err := levelds.NewDatastore(dbpath, nil)
	if err != nil {
		return fmt.Errorf("failed to create datastore: %w", err)
	}

	start := time.Now()
	var nodes []*node.HydraNode

	var hyperLock sync.Mutex
	hyperlog := hyperloglog.New()
	var peersConnected int64

	notifiee := &network.NotifyBundle{
		ConnectedF: func(_ network.Network, v network.Conn) {
			hyperLock.Lock()
			hyperlog.Insert([]byte(v.RemotePeer()))
			hyperLock.Unlock()

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

		addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", getPort()))
		nd, bsCh, err := node.NewHydraNode(
			opts.Datastore(sharedDatastore),
			opts.Addr(addr),
			opts.Relay(relay),
			opts.BucketSize(bucketSize),
			opts.Limiter(limiter),
		)
		if err != nil {
			return fmt.Errorf("failed to spawn node with swarm address %v: %w", addr, err)
		}
		go handleBootstrapStatus(bsCh)
		nd.Host.Network().Notify(notifiee)
		nodes = append(nodes, nd)
	}
	fmt.Fprintf(os.Stderr, "\n")

	provs := make(chan *provInfo, 16)
	//r, w := io.Pipe()
	//logwriter.WriterGroup.AddWriter(w)
	//go waitForNotifications(r, provs, nil)

	// Simple endpoint to report the addrs of the sybils that were launched
	go httpapi.ListenAndServe(nodes, httpAPIAddr)

	// Reporting interval for provs
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
			hyperLock.Lock()
			uniqpeers := hyperlog.Estimate()
			hyperLock.Unlock()
			printStatusLine(many, start, atomic.LoadInt64(&peersConnected), uniqpeers, totalprovs)
		}
	}
}

func printStatusLine(ndht int, start time.Time, totalpeers int64, uniqpeers uint64, totalprovs int) {
	uptime := time.Second * time.Duration(int(time.Since(start).Seconds()))
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	fmt.Fprintf(os.Stderr, "[NumDhts: %d, Uptime: %s, Memory Usage: %s, TotalPeers: %d/%d, Total Provs: %d, BootstrapsDone: %d]\n", ndht, uptime, human.Bytes(mstat.Alloc), totalpeers, uniqpeers, totalprovs, atomic.LoadInt64(&bootstrapDone))
}

// RunSingleDHTWithUI ...
func RunSingleDHTWithUI(path string, relay bool, bucketSize int) error {
	datastore, err := levelds.NewDatastore(path, nil)
	if err != nil {
		return fmt.Errorf("failed to create datastore: %w", err)
	}

	addr, _ := multiaddr.NewMultiaddr(singleDHTSwarmAddr)
	nd, bsCh, err := node.NewHydraNode(
		opts.Datastore(datastore),
		opts.Addr(addr),
		opts.Relay(relay),
		opts.BucketSize(bucketSize),
	)
	if err != nil {
		return fmt.Errorf("failed to spawn node with swarm address %v: %w", singleDHTSwarmAddr, err)
	}

	go handleBootstrapStatus(bsCh)

	// Simple endpoint to report the addrs of the sybils that were launched
	go httpapi.ListenAndServe([]*node.HydraNode{nd}, httpAPIAddr)

	uniqpeers := make(map[peer.ID]struct{})
	messages := make(chan string, 16)
	provs := make(chan *provInfo, 16)
	//r, w := io.Pipe()
	//logwriter.WriterGroup.AddWriter(w)
	//go waitForNotifications(r, provs, messages)

	ga := &ui.GooeyApp{Title: "Libp2p DHT Node", Log: ui.NewLog(15, 15)}
	ga.NewDataLine(3, "Peer ID", nd.Host.ID().Pretty())
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
			peers := nd.Host.Network().Peers()
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
