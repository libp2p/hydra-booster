package opts

import (
	"fmt"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/multiformats/go-multiaddr"
)

// Options are Hydra Head options
type Options struct {
	Datastore        ds.Batching
	Peerstore        peerstore.Peerstore
	RoutingTable     *kbucket.RoutingTable
	EnableRelay      bool
	Addrs            []multiaddr.Multiaddr
	ProtocolPrefix   protocol.ID
	BucketSize       int
	Limiter          chan struct{}
	BootstrapPeers   []multiaddr.Multiaddr
	ID               crypto.PrivKey
	DisableProvGC    bool
	DisableProviders bool
	DisableValues    bool
}

// Option is the Hydra Head option type.
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

// Defaults are the default Hydra Head options. This option will be automatically
// prepended to any options you pass to the Hydra Head constructor.
var Defaults = func(o *Options) error {
	o.Datastore = dssync.MutexWrap(ds.NewMapDatastore())
	tcpAddr, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	quicAddr, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/udp/0/quic")
	o.Addrs = []multiaddr.Multiaddr{tcpAddr, quicAddr}
	o.ProtocolPrefix = dht.DefaultPrefix
	o.BucketSize = 20
	o.BootstrapPeers = dht.DefaultBootstrapPeers
	return nil
}

// Datastore configures the Hydra Head to use the specified datastore.
// Defaults to an in-memory (temporary) map.
func Datastore(ds ds.Batching) Option {
	return func(o *Options) error {
		o.Datastore = ds
		return nil
	}
}

// Peerstore configures the Hydra Head to use the specified peerstore.
// Defaults to an in-memory (temporary) map.
func Peerstore(ps peerstore.Peerstore) Option {
	return func(o *Options) error {
		o.Peerstore = ps
		return nil
	}
}

// RoutingTable configures the Hydra Head to use the specified routing table.
// Defaults to the routing table provided by IpfsDHT.
func RoutingTable(rt *kbucket.RoutingTable) Option {
	return func(o *Options) error {
		o.RoutingTable = rt
		return nil
	}
}

// EnableRelay configures whether this node acts as a relay node.
// The default value is false.
func EnableRelay() Option {
	return func(o *Options) error {
		o.EnableRelay = true
		return nil
	}
}

// Addrs configures the swarm addresses for this Hydra node.
// The default value is /ip4/0.0.0.0/tcp/0 and /ip4/0.0.0.0/udp/0/quic.
func Addrs(addrs []multiaddr.Multiaddr) Option {
	return func(o *Options) error {
		o.Addrs = addrs
		return nil
	}
}

// ProtocolPrefix configures the application specific prefix attached to all DHT protocols by default.
// The default value is "/ipfs".
func ProtocolPrefix(pfx protocol.ID) Option {
	return func(o *Options) error {
		if pfx != "" {
			o.ProtocolPrefix = pfx
		}
		return nil
	}
}

// BucketSize configures the bucket size of the routing table.
// The default value is 20.
func BucketSize(bucketSize int) Option {
	return func(o *Options) error {
		if bucketSize != 0 {
			o.BucketSize = bucketSize
		}
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
		if len(addrs) > 0 {
			o.BootstrapPeers = addrs
		}
		return nil
	}
}

// ID for the head
func ID(id crypto.PrivKey) Option {
	return func(o *Options) error {
		if id != nil {
			o.ID = id
		}
		return nil
	}
}

// DisableProvGC disables garbage collections of provider records from the shared datastore.
// The default value is false.
func DisableProvGC() Option {
	return func(o *Options) error {
		o.DisableProvGC = true
		return nil
	}
}

// DisableProviders disables storing and retrieving provider records.
// The default value is false.
func DisableProviders() Option {
	return func(o *Options) error {
		o.DisableProviders = true
		return nil
	}
}

// DisableValues disables storing and retrieving value records (including public keys).
// The default value is false.
func DisableValues() Option {
	return func(o *Options) error {
		o.DisableValues = true
		return nil
	}
}
