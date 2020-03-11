package httpapi

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/hydra"
)

// NewServeMux creates a new Hydra Booster HTTP API ServeMux
func NewServeMux(hy *hydra.Hydra) *http.ServeMux {
	mux := http.NewServeMux()

	// Get the peers created by hydra booster (ndjson)
	mux.HandleFunc("/sybils", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		for _, n := range hy.Sybils {
			enc.Encode(peer.AddrInfo{
				ID:    n.Host.ID(),
				Addrs: n.Host.Addrs(),
			})
		}
	})

	return mux
}

// ListenAndServe instructs a Hydra HTTP API server to listen and serve on the passed address
func ListenAndServe(hy *hydra.Hydra, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return http.Serve(listener, NewServeMux(hy))
}
