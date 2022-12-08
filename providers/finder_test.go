package providers

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/stretchr/testify/assert"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type mockRouter struct {
	addrInfos map[string][]peer.AddrInfo
}

func (r *mockRouter) FindProvidersAsync(ctx context.Context, cid cid.Cid, results int) <-chan peer.AddrInfo {
	ch := make(chan peer.AddrInfo)
	go func() {
		ais := r.addrInfos[string(cid.Hash())]
		if len(ais) != 0 {
			for _, ai := range ais {
				ch <- ai
			}
		}
		close(ch)
	}()
	return ch
}

func TestAsyncProvidersFinder_Find(t *testing.T) {
	ttl := 20 * time.Second
	queueSize := 10
	cases := []struct {
		name            string
		key             string
		routerAddrInfos map[string][]peer.AddrInfo // key: multihash bytes

		expAIs    []peer.AddrInfo
		expCached []string

		expMetricRows map[string][]view.Row
	}{
		{
			name: "single matching addrinfo",
			key:  "foo",
			routerAddrInfos: map[string][]peer.AddrInfo{
				"foo": {{ID: peer.ID("peer1")}},
				"bar": {{ID: peer.ID("peer2")}},
			},
			expAIs: []peer.AddrInfo{{ID: peer.ID("peer1")}},
			expMetricRows: map[string][]view.Row{
				metrics.Prefetches.Name(): {{
					Data: &view.SumData{Value: 1},
					Tags: []tag.Tag{
						{Key: metrics.KeyStatus, Value: "succeeded"},
					},
				}},
				metrics.PrefetchesPending.Name(): {{
					Data: &view.LastValueData{Value: 0},
				}},
				metrics.PrefetchNegativeCacheSize.Name(): {{
					Data: &view.LastValueData{Value: 0},
				}},
				metrics.PrefetchNegativeCacheTTLSeconds.Name(): {{
					Data: &view.LastValueData{Value: float64(ttl.Seconds())},
				}},
				metrics.PrefetchesPendingLimit.Name(): {{
					Data: &view.LastValueData{Value: float64(queueSize)},
				}},
			},
		},
		{
			name:      "failed lookups should be cached",
			key:       "foo",
			expCached: []string{"foo"},
			expMetricRows: map[string][]view.Row{
				metrics.Prefetches.Name(): {{
					Data: &view.SumData{Value: 1},
					Tags: []tag.Tag{
						{Key: metrics.KeyStatus, Value: "failed"},
					},
				}},
				metrics.PrefetchNegativeCacheSize.Name(): {{
					Data: &view.LastValueData{Value: 1},
				}},
				metrics.PrefetchesPending.Name(): {{
					Data: &view.LastValueData{Value: 0},
				}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()

			views := []*view.View{
				metrics.PrefetchesView,
				metrics.PrefetchesPendingView,
				metrics.PrefetchNegativeCacheSizeView,
				metrics.PrefetchNegativeCacheTTLSecondsView,
				metrics.PrefetchesPendingLimitView,
			}
			view.Register(views...)
			defer view.Unregister(views...)

			finder := NewAsyncProvidersFinder(10*time.Second, queueSize, ttl)

			// set a mock clock so we can control the timing of things
			clock := clock.NewMock()
			finder.clock = clock
			finder.metricsTicker = clock.Ticker(metricsPublishingInterval)

			router := &mockRouter{addrInfos: c.routerAddrInfos}

			// wait group so we know when all Find() reqs have been processed
			reqWG := &sync.WaitGroup{}
			reqWG.Add(1)
			finder.onReqDone = func(r findRequest) { reqWG.Done() }

			ais := []peer.AddrInfo{}
			// wait group so we know when all AIs have been processed
			aiWG := &sync.WaitGroup{}
			aiWG.Add(len(c.expAIs))

			// wait group so we know when metric publishing has occurred
			metricsWG := &sync.WaitGroup{}
			metricsWG.Add(1)
			finder.onMetricsPublished = func() { metricsWG.Done() }

			finder.Run(ctx, 10)
			err := finder.Find(ctx, router, []byte(c.key), func(ai peer.AddrInfo) {
				ais = append(ais, ai)
				aiWG.Done()
			})
			wait(t, ctx, "addrinfos", aiWG)
			wait(t, ctx, "requests", reqWG)

			assert.NoError(t, err)
			assert.Equal(t, len(c.expAIs), len(ais))

			for i, ai := range c.expAIs {
				assert.EqualValues(t, ai.ID, ais[i].ID)
			}

			for _, k := range c.expCached {
				assert.True(t, finder.negativeCache.Has(k))
			}

			// trigger metric publishing and verify
			clock.Add(metricsPublishingInterval + time.Second)
			wait(t, ctx, "metrics", metricsWG)

			for name, expRows := range c.expMetricRows {
				rows, err := view.RetrieveData(name)
				assert.NoError(t, err)
				assert.True(t, subsetRowVals(expRows, rows))
			}
		})

	}
}

// wait waits on a waitgroup with a timeout
func wait(t *testing.T, ctx context.Context, name string, wg *sync.WaitGroup) {
	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()

	select {
	case <-ch:
		return
	case <-ctx.Done():
		t.Fatalf("timeout waiting for %s", name)
	}
}

// rowsEqual returns true if two OpenCensus view rows are equal, excluding timestamps
func rowsEqual(row1 view.Row, row2 view.Row) bool {
	if !reflect.DeepEqual(row1.Tags, row2.Tags) {
		return false
	}

	switch row1Data := row1.Data.(type) {
	case *view.CountData:
		if row2Data, ok := row2.Data.(*view.CountData); ok {
			return row1Data.Value == row2Data.Value
		}
	case *view.SumData:
		if row2Data, ok := row2.Data.(*view.SumData); ok {
			return row1Data.Value == row2Data.Value
		}
	case *view.LastValueData:
		if row2Data, ok := row2.Data.(*view.LastValueData); ok {
			return row1Data.Value == row2Data.Value
		}
	}
	return false
}

func containsRowVal(row view.Row, rows []*view.Row) bool {
	for _, r := range rows {
		if rowsEqual(*r, row) {
			return true
		}
	}
	return false
}

func subsetRowVals(subset []view.Row, rows []*view.Row) bool {
	for _, expRow := range subset {
		if !containsRowVal(expRow, rows) {
			return false
		}
	}
	return true
}
