package httpapi

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/node"
)

// NewServeMux creates a new Hydra Booster HTTP API ServeMux
func NewServeMux(nodes []*node.HydraNode) *http.ServeMux {
	mux := http.NewServeMux()

	// Get the peers created by hydra booster (ndjson)
	mux.HandleFunc("/peers", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		for _, n := range nodes {
			enc.Encode(peer.AddrInfo{
				ID:    n.Host.ID(),
				Addrs: n.Host.Addrs(),
			})
		}
	})

	return mux
}

// ListenAndServe instructs a Hydra HTTP API server to listen and serve on the passed address
func ListenAndServe(nodes []*node.HydraNode, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return http.Serve(listener, NewServeMux(nodes))
}
