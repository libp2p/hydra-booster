package opts

import (
	"fmt"
	"net/http"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	hproviders "github.com/libp2p/hydra-booster/providers"
	"github.com/multiformats/go-multiaddr"
)

type ProviderStoreBuilderFunc func(opts Options, host host.Host) (providers.ProviderStore, error)

// Options are Hydra Head options
type Options struct {
	Datastore                 ds.Batching
	Peerstore                 peerstore.Peerstore
	ProviderStoreBuilder      ProviderStoreBuilderFunc
	DelegateHTTPClient        *http.Client
	RoutingTable              *kbucket.RoutingTable
	EnableRelay               bool
	Addrs                     []multiaddr.Multiaddr
	ProtocolPrefix            protocol.ID
	BucketSize                int
	Limiter                   chan struct{}
	BootstrapPeers            []multiaddr.Multiaddr
	ID                        crypto.PrivKey
	DisableProvGC             bool
	DisableProvCounts         bool
	DisableProviders          bool
	DisableValues             bool
	ProvidersFinder           hproviders.ProvidersFinder
	DisableResourceManager    bool
	ResourceManagerLimitsFile string
	ConnMgrHighWater          int
	ConnMgrLowWater           int
	ConnMgrGracePeriod        int
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
	o.ConnMgrHighWater = 1800
	o.ConnMgrLowWater = 1200
	o.ConnMgrGracePeriod = 60000
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

func ProviderStoreBuilder(builder func(Options, host.Host) (providers.ProviderStore, error)) Option {
	return func(o *Options) error {
		o.ProviderStoreBuilder = builder
		return nil
	}
}

func DelegateHTTPClient(c *http.Client) Option {
	return func(o *Options) error {
		o.DelegateHTTPClient = c
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

// DisableProvCounts disables counting the number of providers in the provider store.
func DisableProvCounts() Option {
	return func(o *Options) error {
		o.DisableProvCounts = true
		return nil
	}
}

func ProvidersFinder(f hproviders.ProvidersFinder) Option {
	return func(o *Options) error {
		o.ProvidersFinder = f
		return nil
	}
}

func DisableResourceManager(b bool) Option {
	return func(o *Options) error {
		o.DisableResourceManager = b
		return nil
	}
}

func ResourceManagerLimitsFile(f string) Option {
	return func(o *Options) error {
		o.ResourceManagerLimitsFile = f
		return nil
	}
}

func ConnMgrHighWater(n int) Option {
	return func(o *Options) error {
		o.ConnMgrHighWater = n
		return nil
	}
}

func ConnMgrLowWater(n int) Option {
	return func(o *Options) error {
		o.ConnMgrLowWater = n
		return nil
	}
}

func ConnMgrGracePeriod(n int) Option {
	return func(o *Options) error {
		o.ConnMgrGracePeriod = n
		return nil
	}
}
