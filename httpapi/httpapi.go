package httpapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	dsq "github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/hydra"
)

// ListenAndServe instructs a Hydra HTTP API server to listen and serve on the passed address
func ListenAndServe(hy *hydra.Hydra, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return http.Serve(listener, NewServeMux(hy))
}

// NewServeMux creates a new Hydra Booster HTTP API ServeMux
func NewServeMux(hy *hydra.Hydra) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/sybils", sybilsHandler(hy))
	mux.HandleFunc("/records/fetch", recordFetchHandler(hy))
	mux.HandleFunc("/records/list", recordListHandler(hy))
	return mux
}

// "/sybils" Get the peers created by hydra booster (ndjson)
func sybilsHandler(hy *hydra.Hydra) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		for _, syb := range hy.Sybils {
			enc.Encode(peer.AddrInfo{
				ID:    syb.Host.ID(),
				Addrs: syb.Host.Addrs(),
			})
		}
	}
}

// "/records/fetch" Receive a record and fetch it from the network, if available
func recordFetchHandler(hy *hydra.Hydra) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO needs functionality to be implemented in libp2p
		// See: https://discuss.libp2p.io/t/does-a-findproviders-replicate-the-provider-records-to-the-node-issuing-the-query/452
		httpNotImplemented := 501
		w.WriteHeader(httpNotImplemented)
	}
}

// "/records/list" Receive a record and fetch it from the network, if available
func recordListHandler(hy *hydra.Hydra) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO Improve this handler once ProvideManager gets exposed
		// https://discuss.libp2p.io/t/list-provider-records/450
		// for now, enumerate the Provider Records in the datastore

		providersKeyPrefix := "/providers/" // https://github.com/libp2p/go-libp2p-kad-dht/blob/master/providers/providers.go#L76
		ds := hy.SharedDatastore
		results, err := ds.Query(dsq.Query{
			Prefix: providersKeyPrefix,
		})
		if err != nil {
			fmt.Printf("Error on retrieving provider records: %s\n", err)
			w.WriteHeader(500)
			return
		}

		enc := json.NewEncoder(w)

		for result := range results.Next() {
			enc.Encode(result.Entry)
		}
	}
}
