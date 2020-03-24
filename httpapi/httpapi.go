package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	cid "github.com/ipfs/go-cid"
	dsq "github.com/ipfs/go-datastore/query"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/hydra-booster/hydra"
)

// ListenAndServe instructs a Hydra HTTP API server to listen and serve on the passed address
func ListenAndServe(hy *hydra.Hydra, addr string) error {
	srv := &http.Server{
		Addr: addr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      NewRouter(hy),
	}
	return srv.ListenAndServe()
}

// NewRouter creates a new Hydra Booster HTTP API Gorilla Mux
func NewRouter(hy *hydra.Hydra) *mux.Router {
	// mux := http.NewServeMux()
	mux := mux.NewRouter()
	mux.HandleFunc("/sybils", sybilsHandler(hy))
	mux.HandleFunc("/records/fetch/{key}", recordFetchHandler(hy))
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
		vars := mux.Vars(r)
		cidStr := vars["key"]
		cid, err := cid.Decode(cidStr)
		if err != nil {
			fmt.Printf("Received invalid CID, got %s\n", cidStr)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		first := true
		nProviders := 1
		nProvidersStr := r.FormValue("nProviders")
		if nProvidersStr != "" {
			nProviders, err = strconv.Atoi(nProvidersStr)
			if err != nil {
				fmt.Printf("Received invalid nProviders, got %s\n", nProvidersStr)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		enc := json.NewEncoder(w)
		ctx := r.Context()
		for peerAddrInfo := range hy.Sybils[0].Routing.FindProvidersAsync(ctx, cid, nProviders) {
			// fmt.Printf("Got one provider %s\n", peerAddrInfo.String())
			// Store the Provider locally
			hy.Sybils[0].Routing.ProviderManager.AddProvider(ctx, cid.Bytes(), peerAddrInfo.ID)
			// AddProvider doesn't automatically flush it to the datastore.
			// A GetProviders helps wash it down. Useful for monitoring number of records stored
			res := hy.Sybils[0].Routing.ProviderManager.GetProviders(ctx, cid.Bytes())
			fmt.Printf("Stored new Provider Record for CID: %s with providers: %s\n", cidStr, res)

			if first {
				w.WriteHeader(http.StatusOK)
				first = false
			}
			enc.Encode(peerAddrInfo)
		}
		if first {
			w.WriteHeader(http.StatusNotFound)
			return
		}
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
