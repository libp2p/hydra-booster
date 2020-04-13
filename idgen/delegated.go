package idgen

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/libp2p/go-libp2p-core/crypto"
)

// DelegatedIDGenerator is an identity generator whose work is delegated to
// another worker.
type DelegatedIDGenerator struct {
	addr string
}

// NewDelegatedIDGenerator creates a new delegated identity generator whose
// work is delegated to another worker. The delegate must be reachable on the
// passed HTTP address and respond to HTTP POST messages sent to the following
// endpoints:
// `/idgen/add` - returns a JSON string, a base64 encoded private key.
// `/idgen/remove` - accepts a JSON string, a base64 encoded private key.
func NewDelegatedIDGenerator(addr string) *DelegatedIDGenerator {
	return &DelegatedIDGenerator{addr: addr}
}

// AddBalanced generates a balanced random identity by sending a HTTP POST
// request to `/idgen/add`.
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

// Remove removes a previously generated identity by sending a HTTP POST request
// to `/idgen/remove`.
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
