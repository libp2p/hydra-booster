package idgen

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/libp2p/go-libp2p-core/crypto"
)

type DelegatedIdentityGenerator struct {
	addr string
}

func NewDelegatedIdentityGenerator(addr string) *DelegatedIdentityGenerator {
	return &DelegatedIdentityGenerator{addr: addr}
}

func (g *DelegatedIdentityGenerator) AddBalanced() (crypto.PrivKey, error) {
	res, err := http.Post(fmt.Sprintf("%s/idgen/add", g.addr), "application/json", nil)
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

	priv, err := crypto.UnmarshalPrivateKey(bytes)
	if err != nil {
		return nil, err
	}

	return priv, nil
}
