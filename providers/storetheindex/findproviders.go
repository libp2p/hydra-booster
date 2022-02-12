package storetheindex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multihash"
)

var logger = logging.Logger("sti/client")

func (c *client) FindProviders(ctx context.Context, mh multihash.Multihash) ([]peer.AddrInfo, error) {
	// encode request in URL
	u := fmt.Sprint(c.endpoint.String(), "/", mh.B58String())
	httpReq, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return []peer.AddrInfo{}, nil
		}
		return nil, fmt.Errorf("http_%d", resp.StatusCode)
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
	ContextID []byte
	Metadata  json.RawMessage
	Provider  peer.AddrInfo
}
