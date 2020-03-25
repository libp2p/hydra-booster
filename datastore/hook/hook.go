package hook

import (
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/libp2p/hydra-booster/datastore/hook/opts"
	hookopts "github.com/libp2p/hydra-booster/datastore/hook/opts"
)

// Datastore is a datastore that has optional before and after hooks into it's methods
type Datastore struct {
	ds      datastore.Datastore
	options opts.Options
}

// NewDatastore creates a new datastore that has optional before and after hooks into it's methods
func NewDatastore(ds datastore.Datastore, options ...hookopts.Option) *Datastore {
	opts := hookopts.Options{}
	opts.Apply(append([]hookopts.Option{hookopts.Defaults}, options...)...)
	return &Datastore{ds: ds, options: opts}
}

// Put stores the object `value` named by `key`, it calls OnBeforePut and OnAfterPut hooks.
func (hds *Datastore) Put(key datastore.Key, value []byte) error {
	key, value = hds.options.OnBeforePut(key, value)
	err := hds.ds.Put(key, value)
	return hds.options.OnAfterPut(key, value, err)
}

// Delete removes the value for given `key`, it calls OnBeforeDelete and OnAfterDelete hooks.
func (hds *Datastore) Delete(key datastore.Key) error {
	// TODO
	return hds.ds.Delete(key)
}

// Get retrieves the object `value` named by `key`, it calls OnBeforeGet and OnAfterGet hooks.
func (hds *Datastore) Get(key datastore.Key) ([]byte, error) {
	// TODO
	return hds.ds.Get(key)
}

// Has returns whether the `key` is mapped to a `value`.
func (hds *Datastore) Has(key datastore.Key) (bool, error) {
	return hds.ds.Has(key)
}

// GetSize returns the size of the `value` named by `key`.
// In some contexts, it may be much cheaper to only get the size of the
// value rather than retrieving the value itself.
func (hds *Datastore) GetSize(key datastore.Key) (int, error) {
	return hds.ds.GetSize(key)
}

// Query searches the datastore and returns a query result.
func (hds *Datastore) Query(q query.Query) (query.Results, error) {
	return hds.Query(q)
}

// Batching is a datastore with hooks that also supports batching
type Batching struct {
	hds *Datastore
	ds  datastore.Batching
}

// TODO
