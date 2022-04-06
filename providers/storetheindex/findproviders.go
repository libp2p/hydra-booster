package storetheindex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/metrics"
	"github.com/multiformats/go-multihash"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var log = logging.Logger("hydra/storetheindex")

func (c *client) FindProviders(ctx context.Context, mh multihash.Multihash) ([]peer.AddrInfo, error) {
	httpStatusCode := 0
	start := time.Now()
	defer func() {
		recordSTIFindProvsComplete(ctx, httpStatusCode, time.Since(start))
	}()
	// encode request in URL
	u := fmt.Sprint(c.endpoint.String(), "/", mh.B58String())
	httpReq, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if resp.StatusCode == 0 {
			log.Errorw("received non-HTTP error from StoreTheIndex", "Error", err)
		}
		return nil, err
	}
	httpStatusCode = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return []peer.AddrInfo{}, nil
		}
		return nil, fmt.Errorf("find query failed: %v", http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parsedResponse := indexFindResponse{}
	if err := json.Unmarshal(body, &parsedResponse); err != nil {
		return nil, err
	}

	if len(parsedResponse.MultihashResults) != 1 {
		return nil, fmt.Errorf("unexpected number of responses")
	}
	result := make([]peer.AddrInfo, len(parsedResponse.MultihashResults[0].ProviderResults))
	for _, m := range parsedResponse.MultihashResults[0].ProviderResults {
		result = append(result, m.Provider)
	}

	return result, nil
}

type indexFindResponse struct {
	MultihashResults []indexMultihashResult
}

type indexMultihashResult struct {
	Multihash       multihash.Multihash
	ProviderResults []indexProviderResult
}

type indexProviderResult struct {
	Provider peer.AddrInfo
}

func recordSTIFindProvsComplete(ctx context.Context, statusCode int, duration time.Duration) {
	stats.RecordWithTags(
		ctx,
		[]tag.Mutator{tag.Upsert(metrics.KeyStatus, strconv.Itoa(statusCode))},
		[]stats.Measurement{
			metrics.STIFindProvs.M(1),
			metrics.STIFindProvsDuration.M(float64(duration)),
		}...,
	)
}
