package main

import (
	"flag"
	"net/http"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-delegated-routing/client"
	"github.com/ipfs/go-delegated-routing/server"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
)

var log = logging.Logger("hydra/test-routing-server")

const (
	defaultHTTPAPIAddr = "127.0.0.1:9999"
	defaultHTTPAPIPath = "/"
)

var (
	httpAPIAddr = flag.String("httpapi-addr", defaultHTTPAPIAddr, "IP and port to bind the HTTP API server on")
	httpAPIPath = flag.String("httpapi-path", defaultHTTPAPIPath, "Path to delegated routing API")
)

func main() {
	flag.Parse()
	log.Info("starting test routing server")

	mx := http.NewServeMux()
	mx.HandleFunc(*httpAPIPath, server.DelegatedRoutingAsyncHandler(mockServer{}))

	s := &http.Server{
		Addr:    *httpAPIAddr,
		Handler: mx,
	}
	err := s.ListenAndServe()
	log.Errorf("server died with error (%v)", err)
}

type mockServer struct{}

// FindProviders fulfills find provider requests by returning no results.
// NOTE: Since it is intended to run in production, as a placeholder delegated routing server for hydra,
// we probably don't want to return any results as this would:
//	(a) degrade user experience by pointing to an unresponsive destination
//	(b) create heavy connection request load (from clients trying to download content) in some IP network.
//	This could be a problem, if this system is not ours.
func (mockServer) FindProviders(key cid.Cid) (<-chan client.FindProvidersAsyncResult, error) {
	ch := make(chan client.FindProvidersAsyncResult)
	// ma := multiaddr.StringCast("/ip4/7.7.7.7/tcp/4242/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	// ai, err := peer.AddrInfoFromP2pAddr(ma)
	// if err != nil {
	// 	return fmt.Errorf("address info creation (%v)", err)
	// }
	log.Infof("serving find providers request for %v", key.String())
	go func() {
		ch <- client.FindProvidersAsyncResult{AddrInfo: []peer.AddrInfo{ /* *ai */ }}
		close(ch)
	}()
	return ch, nil
}

func (mockServer) GetIPNS(id []byte) (<-chan client.GetIPNSAsyncResult, error) {
	return nil, routing.ErrNotSupported
}

func (mockServer) PutIPNS(id []byte, record []byte) (<-chan client.PutIPNSAsyncResult, error) {
	return nil, routing.ErrNotSupported
}
