package metrics

import (
	"expvar"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"

	"contrib.go.opencensus.io/exporter/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
)

// PrometheusNamespace is the unique prefix for metrics exported from the app
var PrometheusNamespace = "hydrabooster"

// SetupMetrics sets up an endpoint to collect process metrics (e.g. pprof).
func SetupMetrics(port int) error {
	// setup Prometheus
	registry := prom.NewRegistry()
	goCollector := prom.NewGoCollector()
	procCollector := prom.NewProcessCollector(prom.ProcessCollectorOpts{})
	registry.MustRegister(goCollector, procCollector)
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: PrometheusNamespace,
		Registry:  registry,
	})
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	view.RegisterExporter(pe)
	if err := view.Register(DefaultViews...); err != nil {
		log.Fatalf("Failed to register hydra views: %v", err)
	}

	go func() {
		mux := http.NewServeMux()
		zpages.Handle(mux, "/debug")
		mux.Handle("/metrics", pe)
		mux.Handle("/debug/vars", expvar.Handler())

		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux); err != nil {
			log.Fatalf("Failed to run Prometheus /metrics endpoint: %v", err)
		}
	}()
	return nil
}
