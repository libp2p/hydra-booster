package idgen

import (
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/libp2p/go-libp2p-core/crypto"
)

// CleaningIDGenerator is an identity generator that provides an extra method to
// remove all previously generated identities without passing any arguments.
type CleaningIDGenerator struct {
	idgen  IdentityGenerator
	keys   []crypto.PrivKey
	locker sync.Mutex
}

// NewCleaningIDGenerator creates a new delegated identity
// generator that provides an extra method to remove all previously generated
// identities without passing any arguments.
func NewCleaningIDGenerator(idgen IdentityGenerator) *CleaningIDGenerator {
	return &CleaningIDGenerator{idgen: idgen}
}

// AddBalanced stores the result of calling AddBalanced on the underlying
// identify generator and then returns it.
func (c *CleaningIDGenerator) AddBalanced() (crypto.PrivKey, error) {
	pk, err := c.idgen.AddBalanced()
	if err != nil {
		return nil, err
	}
	c.locker.Lock()
	defer c.locker.Unlock()
	c.keys = append(c.keys, pk)
	return pk, nil
}

// Remove calls Remove on the underlying identity generator and also removes the
// passed key from it's memory of keys generated.
func (c *CleaningIDGenerator) Remove(privKey crypto.PrivKey) error {
	err := c.idgen.Remove(privKey)
	if err != nil {
		return err
	}
	c.locker.Lock()
	defer c.locker.Unlock()
	var keys []crypto.PrivKey
	for _, pk := range c.keys {
		if !pk.Equals(privKey) {
			keys = append(keys, pk)
		}
	}
	c.keys = keys
	return nil
}

// Clean removes ALL previously generated keys by calling Remove on the
// underlying identity generator for each key in it's memory.
func (c *CleaningIDGenerator) Clean() error {
	var errs error
	c.locker.Lock()
	defer c.locker.Unlock()
	for _, pk := range c.keys {
		err := c.idgen.Remove(pk)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	c.keys = []crypto.PrivKey{}
	return errs
}
