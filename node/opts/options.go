package opts

import (
	"fmt"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/multiformats/go-multiaddr"
)

// Options are Hydra Node options
type Options struct {
	Datastore      ds.Batching
	RoutingTable   *kbucket.RoutingTable
	Relay          bool
	Addr           multiaddr.Multiaddr
	BucketSize     int
	Limiter        chan struct{}
	BootstrapPeers []multiaddr.Multiaddr
}

// Option is the Hydra option type.
type Option func(*Options) error

// Apply applies the given options to this Option.
func (o *Options) Apply(opts ...Option) error {
	for i, opt := range opts {
		if err := opt(o); err != nil {
			return fmt.Errorf("hydra node option %d failed: %s", i, err)
		}
	}
	return nil
}

// Defaults are the default Hydra options. This option will be automatically
// prepended to any options you pass to the Hydra constructor.
var Defaults = func(o *Options) error {
	o.Datastore = dssync.MutexWrap(ds.NewMapDatastore())
	o.Relay = false
	o.Addr, _ = multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	o.BucketSize = 20
	o.BootstrapPeers = dht.DefaultBootstrapPeers
	return nil
}

// Datastore configures the Hydra Node to use the specified datastore.
// Defaults to an in-memory (temporary) map.
func Datastore(ds ds.Batching) Option {
	return func(o *Options) error {
		o.Datastore = ds
		return nil
	}
}

// RoutingTable configures the Hydra Node to use the specified routing table.
// Defaults to the routing table provided by IpfsDHT.
func RoutingTable(rt *kbucket.RoutingTable) Option {
	return func(o *Options) error {
		o.RoutingTable = rt
		return nil
	}
}

// Relay configures whether this node acts as a relay node.
// The default value is false.
func Relay(relay bool) Option {
	return func(o *Options) error {
		o.Relay = relay
		return nil
	}
}

// Addr configures the swarm address for this Hydra node.
// The default value is /ip4/0.0.0.0/tcp/0.
func Addr(addr multiaddr.Multiaddr) Option {
	return func(o *Options) error {
		o.Addr = addr
		return nil
	}
}

// BucketSize configures the bucket size of the routing table.
// The default value is 20.
func BucketSize(bucketSize int) Option {
	return func(o *Options) error {
		o.BucketSize = bucketSize
		return nil
	}
}

// Limiter configures ???.
// The default value is nil.
func Limiter(l chan struct{}) Option {
	return func(o *Options) error {
		o.Limiter = l
		return nil
	}
}

// BootstrapPeers configures the set of bootstrap peers that should be randomly selected from.
// The default value is `dht.DefaultBootstrapPeers`.
func BootstrapPeers(addrs []multiaddr.Multiaddr) Option {
	return func(o *Options) error {
		o.BootstrapPeers = addrs
		return nil
	}
}
