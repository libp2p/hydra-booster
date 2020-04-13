package idgen

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/libp2p/go-libp2p-core/crypto"
)

// DelegatedIDGenerator is an identity generator whose work is delegated to
// another worker.
type DelegatedIDGenerator struct {
	addr string
}

// NewDelegatedIDGenerator creates a new delegated identiy generator whose
// work is delegated to another worker. The delegate must be reachable on the
// passed HTTP address and respond to HTTP POST messages to the following
// endpoints:
// `/idgen/add` - returns a JSON string, a base64 encoded private key.
// `/idgen/remove` - accepts a JSON string, a base64 encoded private key.
func NewDelegatedIDGenerator(addr string) *DelegatedIDGenerator {
	return &DelegatedIDGenerator{addr: addr}
}

// AddBalanced generates a random identity, which
// is balanced with respect to the existing identities in the generator.
// The generated identity is stored in the generator's memory.
func (g *DelegatedIDGenerator) AddBalanced() (crypto.PrivKey, error) {
	res, err := http.Post(fmt.Sprintf("%s/idgen/add", g.addr), "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected HTTP status %d", res.StatusCode)
	}

	dec := json.NewDecoder(res.Body)
	var b64 string
	if err := dec.Decode(&b64); err != nil {
		return nil, err
	}

	bytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}

	pk, err := crypto.UnmarshalPrivateKey(bytes)
	if err != nil {
		return nil, err
	}

	return pk, nil
}

// Remove removes a previously generated identity from the generator's memory.
func (g *DelegatedIDGenerator) Remove(privKey crypto.PrivKey) error {
	b, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return err
	}

	data, err := json.Marshal(base64.StdEncoding.EncodeToString(b))
	if err != nil {
		return err
	}

	res, err := http.Post(fmt.Sprintf("%s/idgen/remove", g.addr), "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		return fmt.Errorf("unexpected HTTP status %d", res.StatusCode)
	}

	return nil
}

// DelegatedIDGeneratorCleaner is a delegated identity generator that
// provides an extra method to remove all previously generated identities without
// passing any arguments.
type DelegatedIDGeneratorCleaner struct {
	DelegatedIDGenerator
	keys   []crypto.PrivKey
	locker sync.Mutex
}

// NewDelegatedIDGeneratorCleaner creates a new delegated identity
// generator that provides an extra method to remove all previously generated
// identities without passing any arguments.
func NewDelegatedIDGeneratorCleaner(addr string) *DelegatedIDGeneratorCleaner {
	return &DelegatedIDGeneratorCleaner{
		DelegatedIDGenerator: DelegatedIDGenerator{addr: addr},
	}
}

// AddBalanced generates a random identity, which
// is balanced with respect to the existing identities in the generator.
// The generated identity is stored in the generator's memory.
func (g *DelegatedIDGeneratorCleaner) AddBalanced() (crypto.PrivKey, error) {
	pk, err := g.DelegatedIDGenerator.AddBalanced()
	if err != nil {
		return nil, err
	}
	g.locker.Lock()
	defer g.locker.Unlock()
	g.keys = append(g.keys, pk)
	return pk, nil
}

// Remove removes a previously generated identity from the generator's memory.
func (g *DelegatedIDGeneratorCleaner) Remove(privKey crypto.PrivKey) error {
	err := g.DelegatedIDGenerator.Remove(privKey)
	if err != nil {
		return err
	}
	g.locker.Lock()
	defer g.locker.Unlock()
	var keys []crypto.PrivKey
	for _, pk := range g.keys {
		if !pk.Equals(privKey) {
			keys = append(keys, pk)
		}
	}
	g.keys = keys
	return nil
}

// Clean removes ALL previously generated keys from the generator's memory.
func (g *DelegatedIDGeneratorCleaner) Clean() error {
	var errs error
	g.locker.Lock()
	defer g.locker.Unlock()
	for _, pk := range g.keys {
		err := g.DelegatedIDGenerator.Remove(pk)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	g.keys = []crypto.PrivKey{}
	return errs
}
