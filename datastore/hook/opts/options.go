package opts

import (
	"fmt"

	"github.com/ipfs/go-datastore"
)

// Options are HookDatastore options
type Options struct {
	OnBeforeGet    func(datastore.Key) datastore.Key
	OnAfterGet     func(datastore.Key, []byte, error) ([]byte, error)
	OnBeforePut    func(datastore.Key, []byte) (datastore.Key, []byte)
	OnAfterPut     func(datastore.Key, []byte, error) error
	OnBeforeDelete func(datastore.Key) datastore.Key
	OnAfterDelete  func(datastore.Key, error) error
}

// Option is the Hydra option type.
type Option func(*Options) error

// Apply applies the given options to this Option.
func (o *Options) Apply(opts ...Option) error {
	for i, opt := range opts {
		if err := opt(o); err != nil {
			return fmt.Errorf("hook datastore option %d failed: %s", i, err)
		}
	}
	return nil
}

// Defaults are the default HookDatastore options. This option will be automatically
// prepended to any options you pass to the HookDatastore constructor.
var Defaults = func(o *Options) error {
	o.OnBeforeGet = func(k datastore.Key) datastore.Key { return k }
	o.OnAfterGet = func(k datastore.Key, b []byte, err error) ([]byte, error) { return b, err }
	return nil
}

// OnBeforeGet configures a hook that is called _before_ Get
// Defaults to noop.
func OnBeforeGet(f func(datastore.Key) datastore.Key) Option {
	return func(o *Options) error {
		o.OnBeforeGet = f
		return nil
	}
}

// OnAfterGet configures a hook that is called _after_ Get
// Defaults to noop.
func OnAfterGet(f func(datastore.Key, []byte, error) ([]byte, error)) Option {
	return func(o *Options) error {
		o.OnAfterGet = f
		return nil
	}
}
