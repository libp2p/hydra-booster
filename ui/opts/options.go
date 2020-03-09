package opts

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Options are the UI options
type Options struct {
	Start  time.Time
	Writer io.Writer
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
	o.Start = time.Now()
	o.Writer = os.Stderr
	return nil
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
