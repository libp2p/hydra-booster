package reports

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/axiomhq/hyperloglog"
	"github.com/ipfs/go-datastore/query"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	// logging "github.com/ipfs/go-log"
	// logwriter "github.com/ipfs/go-log/writer"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/libp2p/hydra-booster/node"
)

// var _ = logwriter.WriterGroup
// var log = logging.Logger("hydrabooster")

// StatusReport represents a captured snapshot of Hydra Node status data
type StatusReport struct {
	MemStats                    runtime.MemStats
	TotalHydraNodes             int
	TotalBootstrappedHydraNodes int
	TotalConnectedPeers         int
	TotalUniquePeers            uint64
	TotalProvs                  int
	TotalProvTime               time.Duration
}

// ProvInfo contains information about provider records
type ProvInfo struct {
	Key      string
	Duration time.Duration
}

// Reporter collects status reports on Hydra Nodes and publishes them to a channel
type Reporter struct {
	StatusReports chan StatusReport
	ticker        *time.Ticker
	provs         chan *ProvInfo
	waitGroup     *sync.WaitGroup
}

// Stop halts all status report collection and reporting, will wait for pending report to be published and consumed
func (r *Reporter) Stop() {
	close(r.provs)
	r.ticker.Stop()
	r.waitGroup.Wait() // wait for pending report publishes to complete
	close(r.StatusReports)
}

// ErrMissingNodes is returned when no nodes are passed to a reporter
var ErrMissingNodes = fmt.Errorf("reporter needs at least one node")

// NewReporter creates a new reporter that immediately starts collecting status reports for the passed Hydra nodes and publishes them to a channel
func NewReporter(nodes []*node.HydraNode, reportInterval time.Duration) (*Reporter, error) {
	if len(nodes) == 0 {
		return nil, ErrMissingNodes
	}

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

	var wg sync.WaitGroup
	reports := make(chan StatusReport)
	ticker := time.NewTicker(reportInterval)
	reporter := Reporter{StatusReports: reports, ticker: ticker, provs: provs, waitGroup: &wg}

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
				wg.Add(1)

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

					rts := nodes[i].RoutingTable.Size()

					prs, err := nodes[i].Datastore.Query(query.Query{Prefix: "/providers", KeysOnly: true})
					if err != nil {
						fmt.Println(err)
						continue
					}

					// TODO: make fast https://github.com/libp2p/go-libp2p-kad-dht/issues/487
					var totalProvRecords int
					for _ = range prs.Next() {
						totalProvRecords++
					}

					ctx, err := tag.New(context.Background(), tag.Insert(metrics.KeyPeerID, nodes[i].Host.ID().String()))
					if err != nil {
						fmt.Println(err)
						continue
					}

					stats.Record(
						ctx,
						metrics.RoutingTableSize.M(int64(rts)),
						metrics.ProviderRecords.M(int64(totalProvRecords)),
					)
				}

				stats.Record(
					tag.NewContext(context.Background(), nil),
					metrics.Sybils.M(int64(len(nodes))),
					metrics.BootstrappedSybils.M(int64(totalBootstrappedHydraNodes)),
					metrics.ConnectedPeers.M(int64(totalConnectedPeers)),
					metrics.UniquePeers.M(int64(totalUniqPeers)),
				)

				reports <- StatusReport{
					MemStats:                    mstat,
					TotalHydraNodes:             len(nodes),
					TotalBootstrappedHydraNodes: totalBootstrappedHydraNodes,
					TotalConnectedPeers:         totalConnectedPeers,
					TotalUniquePeers:            totalUniqPeers,
					TotalProvs:                  totalProvs,
					TotalProvTime:               totalProvTime,
				}

				wg.Done()
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
