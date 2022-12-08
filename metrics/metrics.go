package metrics

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/go-kit/log"
	"github.com/ipfs/go-libipfs/routing/http/client"
	"github.com/ncabatoff/process-exporter/collector"
	"github.com/ncabatoff/process-exporter/config"
	prom "github.com/prometheus/client_golang/prometheus"
	necoll "github.com/prometheus/node_exporter/collector"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/zpages"
)

// PrometheusNamespace is the unique prefix for metrics exported from the app
var PrometheusNamespace = "hydrabooster"

func buildProcCollector() (*collector.NamedProcessCollector, error) {
	rules := config.MatcherRules{{
		ExeRules: []string{os.Args[0]},
	}}
	config, err := rules.ToConfig()
	if err != nil {
		return nil, fmt.Errorf("building process collector config: %w", err)
	}
	proc1Collector, err := collector.NewProcessCollector(collector.ProcessCollectorOption{
		ProcFSPath:  "/proc",
		Children:    true,
		Threads:     true,
		GatherSMaps: false,
		Namer:       config.MatchNamers,
	})
	if err != nil {
		return nil, fmt.Errorf("creating process collector: %w", err)
	}
	return proc1Collector, nil
}

// ListenAndServe sets up an endpoint to collect process metrics (e.g. pprof).
func ListenAndServe(address string) error {
	// setup Prometheus
	registry := prom.NewRegistry()
	goCollector := prom.NewGoCollector()
	procCollector := prom.NewProcessCollector(prom.ProcessCollectorOpts{})

	nodeCollector, err := necoll.NewNodeCollector(log.NewNopLogger())
	if err != nil {
		return err
	}

	proc1Collector, err := buildProcCollector()
	if err != nil {
		return err
	}

	registry.MustRegister(goCollector, procCollector, nodeCollector, proc1Collector)
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: PrometheusNamespace,
		Registry:  registry,
	})
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	view.RegisterExporter(pe)

	views := DefaultViews
	for _, view := range client.OpenCensusViews {
		// add name tag to each view so we can distinguish hydra instances
		view.TagKeys = append(view.TagKeys, tag.MustNewKey("name"))
		views = append(views, view)
	}

	if err := view.Register(views...); err != nil {
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
