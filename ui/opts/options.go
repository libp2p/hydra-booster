package opts

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Options are the UI options
type Options struct {
	MetricsURL    string
	Start         time.Time
	Writer        io.Writer
	RefreshPeriod time.Duration
}

// Option is the UI option type.
type Option func(*Options) error

// Apply applies the given options to this Option.
func (o *Options) Apply(opts ...Option) error {
	for i, opt := range opts {
		if err := opt(o); err != nil {
			return fmt.Errorf("UI option %d failed: %s", i, err)
		}
	}
	return nil
}

// Defaults are the default UI options. This option will be automatically
// prepended to any options you pass to the NewUI constructor.
var Defaults = func(o *Options) error {
	o.MetricsURL = "http://127.0.0.1:8888/metrics"
	o.Start = time.Now()
	o.Writer = os.Stderr
	o.RefreshPeriod = time.Second * 5
	return nil
}

// MetricsURL configures the URL the Prometheus /metrics endpoint is at
// Defaults to http://127.0.0.1:8888/metrics.
func MetricsURL(url string) Option {
	return func(o *Options) error {
		o.MetricsURL = url
		return nil
	}
}

// Start configures the start time for the UI to calculate the uptime vaue from.
// Defaults to an time.Now().
func Start(t time.Time) Option {
	return func(o *Options) error {
		o.Start = t
		return nil
	}
}

// Writer configures where the output should be written to.
// The default value is os.Stderr.
func Writer(w io.Writer) Option {
	return func(o *Options) error {
		o.Writer = w
		return nil
	}
}

// RefreshPeriod configures the period beiween UI refeshes.
// Defaults to 5s.
func RefreshPeriod(rp time.Duration) Option {
	return func(o *Options) error {
		o.RefreshPeriod = rp
		return nil
	}
}
