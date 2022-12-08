package providers

import (
	"context"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	"github.com/whyrusleeping/timecache"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

const (
	metricsPublishingInterval = 10 * time.Second
)

// ProvidersFinder finds providers for the given key using the given content router, passing each to the callback.
type ProvidersFinder interface {
	Find(ctx context.Context, router ReadContentRouting, key []byte, onProvider onProviderFunc) error
}

func NewAsyncProvidersFinder(timeout time.Duration, queueSize int, negativeCacheTTL time.Duration) *asyncProvidersFinder {
	clock := clock.New()
	return &asyncProvidersFinder{
		log:                logging.Logger("hydra/prefetch"),
		clock:              clock,
		metricsTicker:      clock.Ticker(metricsPublishingInterval),
		workQueueSize:      queueSize,
		workQueue:          make(chan findRequest, queueSize),
		pending:            map[string]bool{},
		timeout:            timeout,
		negativeCacheTTL:   negativeCacheTTL,
		negativeCache:      &idempotentTimeCache{cache: timecache.NewTimeCache(negativeCacheTTL)},
		onReqDone:          func(r findRequest) {},
		onMetricsPublished: func() {},
	}

}

type ReadContentRouting interface {
	FindProvidersAsync(ctx context.Context, cid cid.Cid, numResults int) <-chan peer.AddrInfo
}

type onProviderFunc func(peer.AddrInfo)

type findRequest struct {
	ctx        context.Context
	router     ReadContentRouting
	key        []byte
	onProvider onProviderFunc
}

// asyncProvidersFinder finds providers asynchronously using a bounded work queue and a bounded number of workers.
type asyncProvidersFinder struct {
	log              logging.EventLogger
	clock            clock.Clock
	metricsTicker    *clock.Ticker
	workQueueSize    int
	workQueue        chan findRequest
	pendingMut       sync.RWMutex
	pending          map[string]bool
	timeout          time.Duration
	negativeCacheTTL time.Duration
	negativeCache    *idempotentTimeCache
	ctx              context.Context

	// callbacks used for testing
	onReqDone          func(r findRequest)
	onMetricsPublished func()
}

// Find finds the providers for a given key using the passed content router asynchronously.
// It schedules work and returns immediately, invoking the callback concurrently as results are found.
// If the work queue is full, this does not block--it drops the request on the floor and immediately returns.
func (a *asyncProvidersFinder) Find(ctx context.Context, router ReadContentRouting, key []byte, onProvider onProviderFunc) error {
	a.pendingMut.Lock()
	defer a.pendingMut.Unlock()
	ks := string(key)
	pending := a.pending[ks]
	if pending {
		return nil
	}
	if a.negativeCache.Has(ks) {
		recordPrefetches(ctx, "failed-cached")
		return nil
	}
	select {
	case a.workQueue <- findRequest{ctx: ctx, router: router, key: key, onProvider: onProvider}:
		a.pending[ks] = true
		return nil
	default:
		recordPrefetches(ctx, "discarded")
		return nil
	}
}

// Run runs a set of goroutine workers that process Find() calls asynchronously.
// The workers shut down gracefully when the context is canceled.
func (a *asyncProvidersFinder) Run(ctx context.Context, numWorkers int) {
	a.ctx = ctx
	for i := 0; i < numWorkers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case req := <-a.workQueue:
					a.handleRequest(ctx, req)
				}
			}
		}()
	}
	// periodic metric publishing
	go func() {
		for {
			defer a.metricsTicker.Stop()
			select {
			case <-ctx.Done():
				return
			case <-a.metricsTicker.C:
				a.pendingMut.RLock()
				pending := len(a.pending)
				a.pendingMut.RUnlock()

				stats.Record(ctx, metrics.PrefetchesPending.M(int64(pending)))
				stats.Record(ctx, metrics.PrefetchNegativeCacheSize.M(int64(a.negativeCache.Len())))
				stats.Record(ctx, metrics.PrefetchNegativeCacheTTLSeconds.M(int64(a.negativeCacheTTL.Seconds())))
				stats.Record(ctx, metrics.PrefetchesPendingLimit.M(int64(a.workQueueSize)))

				a.onMetricsPublished()
			}
		}
	}()
}

func (a *asyncProvidersFinder) handleRequest(ctx context.Context, req findRequest) {
	defer func() {
		a.onReqDone(req)
		a.pendingMut.Lock()
		delete(a.pending, string(req.key))
		a.pendingMut.Unlock()
	}()

	// since this is async work, we don't want to use the deadline of the request's context
	ctx = tag.NewContext(ctx, tag.FromContext(req.ctx))

	mh := multihash.Multihash(req.key)
	// hack: we're using a raw encoding here so that we can construct a CIDv1 to make the type system happy
	// the DHT doesn't actually care about the CID, it cares about the multihash
	// ideally FindProvidersAsync would take in a multihash, not a CID
	cid := cid.NewCidV1(uint64(multicodec.Raw), mh)
	ctx, stop := context.WithTimeout(ctx, a.timeout)
	defer stop()
	foundProviders := false
	startTime := a.clock.Now()
	for addrInfo := range req.router.FindProvidersAsync(ctx, cid, 1) {
		req.onProvider(addrInfo)
		foundProviders = true
	}
	findTime := a.clock.Since(startTime)

	if !foundProviders {
		a.negativeCache.Add(string(req.key))
		recordPrefetches(ctx, "failed", metrics.PrefetchDuration.M(float64(findTime.Milliseconds())))
		return
	}

	recordPrefetches(ctx, "succeeded", metrics.PrefetchDuration.M(float64(findTime.Milliseconds())))
}

func recordPrefetches(ctx context.Context, status string, extraMeasures ...stats.Measurement) {
	stats.RecordWithTags(
		ctx,
		[]tag.Mutator{tag.Upsert(metrics.KeyStatus, status)},
		append([]stats.Measurement{metrics.Prefetches.M(1)}, extraMeasures...)...,
	)
}

// idempotentTimeCache wraps a timecache and adds thread safety and idempotency.
type idempotentTimeCache struct {
	mut   sync.RWMutex
	cache *timecache.TimeCache
}

// Add adds an element to the cache.
// If the element is already in the cache, it is left untouched (the time is not updated).
func (c *idempotentTimeCache) Add(s string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	if !c.cache.Has(s) {
		c.cache.Add(s)
	}
}

func (c *idempotentTimeCache) Len() int {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return len(c.cache.M)
}

func (c *idempotentTimeCache) Has(s string) bool {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cache.Has(s)
}
