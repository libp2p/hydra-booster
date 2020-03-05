package main

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

// SetupMetrics sets up an endpoint to collect process metrics (e.g. pprof).
func SetupMetrics(port int) error {
	// setup Prometheus
	registry := prom.NewRegistry()
	goCollector := prom.NewGoCollector()
	procCollector := prom.NewProcessCollector(prom.ProcessCollectorOpts{})
	registry.MustRegister(goCollector, procCollector)
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "hydra-booster",
		Registry:  registry,
	})
	if err != nil {
		return err
	}

	_ = view.RegisterExporter
	/* Disabling opencensus for now, it allocates too much
	// register prometheus with opencensus
	view.RegisterExporter(pe)
	view.SetReportingPeriod(2)

	// register the metrics views of interest
	if err := view.Register(dhtmetrics.DefaultViews...); err != nil {
		return err
	}
	*/

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
