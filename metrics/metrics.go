package metrics

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"

	"contrib.go.opencensus.io/exporter/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
)

// PrometheusNamespace is the unique prefix for metrics exported from the app
var PrometheusNamespace = "hydrabooster"

// ListenAndServe sets up an endpoint to collect process metrics (e.g. pprof).
func ListenAndServe(address string) error {
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
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	view.RegisterExporter(pe)
	if err := view.Register(DefaultViews...); err != nil {
		return fmt.Errorf("failed to register hydra views: %w", err)
	}

	mux := http.NewServeMux()
	zpages.Handle(mux, "/debug")
	mux.Handle("/metrics", pe)
	mux.Handle("/debug/vars", expvar.Handler())

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return http.ListenAndServe(address, mux)
}
