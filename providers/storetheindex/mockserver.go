package storetheindex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

type mockServer struct {
	cids map[cid.Cid][]peer.AddrInfo
}

func NewMockServer(cids map[cid.Cid][]peer.AddrInfo) *mockServer {
	return &mockServer{cids: cids}
}

func (m *mockServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	cs := path.Base(request.URL.Path)
	c, err := cid.Parse(cs)
	if err != nil {
		fmt.Printf("invalid cid: %v\n", err)
		writer.WriteHeader(400)
		return
	}

	ais, ok := m.cids[c]
	if !ok {
		fmt.Printf("unknown cid: %s\n", cs)
		writer.WriteHeader(404)
		return
	}

	provResults := []indexProviderResult{}
	for _, ai := range ais {
		provResults = append(provResults, indexProviderResult{
			Metadata: bitswapPrefix,
			Provider: ai,
		})
	}

	resp := indexFindResponse{
		MultihashResults: []indexMultihashResult{{
			Multihash:       c.Hash(),
			ProviderResults: provResults,
		}},
	}

	b, err := json.Marshal(&resp)
	if err != nil {
		fmt.Printf("error encoding json: %v\n", err)
		writer.WriteHeader(500)
		return
	}

	_, err = writer.Write(b)
	if err != nil {
		fmt.Printf("error writing response: %v\n", err)
	}
}
