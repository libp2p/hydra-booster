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

	mux.HandleFunc("/sybils", sybilsHandler(nodes))
	mux.HandleFunc("/records/fetch", recordFetchHandler(nodes))
	mux.HandleFunc("/records/list", recordListHandler(nodes))
	return mux
}

// "/sybils" Get the peers created by hydra booster (ndjson)
func sybilsHandler(nodes []*node.HydraNode) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		for _, n := range nodes {
			enc.Encode(peer.AddrInfo{
				ID:    n.Host.ID(),
				Addrs: n.Host.Addrs(),
			})
		}
	}
}

// "/records/fetch" Receive a record and fetch it from the network, if available
func recordFetchHandler(nodes []*node.HydraNode) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO
	}
}

// "/records/list" Receive a record and fetch it from the network, if available
func recordListHandler(nodes []*node.HydraNode) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO
	}
}

// ListenAndServe instructs a Hydra HTTP API server to listen and serve on the passed address
func ListenAndServe(nodes []*node.HydraNode, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return http.Serve(listener, NewServeMux(nodes))
}
