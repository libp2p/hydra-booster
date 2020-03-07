package reports

import (
	"runtime"
	"sync"
	"time"

	"github.com/axiomhq/hyperloglog"
	// logging "github.com/ipfs/go-log"
	// logwriter "github.com/ipfs/go-log/writer"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/hydra-booster/node"
)

// var _ = logwriter.WriterGroup
// var log = logging.Logger("hydrabooster")

type StatusReport struct {
	MemStats                    runtime.MemStats
	TotalHydraNodes             int
	TotalBootstrappedHydraNodes int
	TotalConnectedPeers         int
	TotalUniquePeers            uint64
	TotalProvs                  int
	TotalProvTime               time.Duration
}

type ProvInfo struct {
	Key      string
	Duration time.Duration
}

type Reporter struct {
	StatusReports chan StatusReport
	ticker        *time.Ticker
	provs         chan *ProvInfo
}

func (r *Reporter) Stop() {
	close(r.provs)
	r.ticker.Stop()
	close(r.StatusReports)
}

func NewReporter(nodes []*node.HydraNode, reportInterval time.Duration) (*Reporter, error) {
	var hyperLock sync.Mutex
	hyperlog := hyperloglog.New()

	notifiee := &network.NotifyBundle{
		ConnectedF: func(_ network.Network, v network.Conn) {
			hyperLock.Lock()
			hyperlog.Insert([]byte(v.RemotePeer()))
			hyperLock.Unlock()
		},
	}

	for i := range nodes {
		nodes[i].Host.Network().Notify(notifiee)
	}

	provs := make(chan *ProvInfo, 16)
	//r, w := io.Pipe()
	//logwriter.WriterGroup.AddWriter(w)
	//go waitForNotifications(r, provs, nil)

	reports := make(chan StatusReport)
	ticker := time.NewTicker(reportInterval)
	reporter := Reporter{StatusReports: reports, ticker: ticker}

	totalProvs := 0
	var totalProvTime time.Duration

	go func() {
		for {
			select {
			case p, ok := <-provs:
				if !ok {
					totalProvs = -1
					provs = nil
				} else {
					totalProvs++
					totalProvTime += p.Duration
				}
			case <-ticker.C:
				hyperLock.Lock()
				totalUniqPeers := hyperlog.Estimate()
				hyperLock.Unlock()

				var mstat runtime.MemStats
				runtime.ReadMemStats(&mstat)

				var totalBootstrappedHydraNodes int
				var totalConnectedPeers int
				for i := range nodes {
					if nodes[i].Bootstrapped {
						totalBootstrappedHydraNodes++
					}
					totalConnectedPeers += len(nodes[i].Host.Network().Peers())
				}

				reports <- StatusReport{
					MemStats:                    mstat,
					TotalHydraNodes:             len(nodes),
					TotalBootstrappedHydraNodes: totalBootstrappedHydraNodes,
					TotalConnectedPeers:         totalConnectedPeers,
					TotalUniquePeers:            totalUniqPeers,
					TotalProvs:                  totalProvs,
					TotalProvTime:               totalProvTime,
				}
			}
		}
	}()

	return &reporter, nil
}

// FIXME: unused in original code
// func waitForNotifications(r io.Reader, provs chan *provInfo, mesout chan string) {
// 	var e map[string]interface{}
// 	dec := json.NewDecoder(r)
// 	for {
// 		err := dec.Decode(&e)
// 		if err != nil {
// 			fmt.Printf("waitForNotifications error: %s\n", err)
// 			close(provs)
// 			return
// 		}

// 		event := e["Operation"]
// 		if event == "handleAddProvider" {
// 			provs <- &provInfo{
// 				Key:      (e["Tags"].(map[string]interface{}))["key"].(string),
// 				Duration: time.Duration(e["Duration"].(float64)),
// 			}
// 		}
// 	}
// }
