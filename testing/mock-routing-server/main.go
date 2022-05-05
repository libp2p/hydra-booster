package main

import (
	"flag"
	"net/http"

	"github.com/ipfs/go-delegated-routing/server"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/hydra-booster/testing/reframe"
)

var log = logging.Logger("hydra/test-reframe-server")

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
	mx.HandleFunc(*httpAPIPath, server.DelegatedRoutingAsyncHandler(reframe.MockServer{}))

	s := &http.Server{
		Addr:    *httpAPIAddr,
		Handler: mx,
	}
	err := s.ListenAndServe()
	log.Errorf("server died with error (%v)", err)
}
