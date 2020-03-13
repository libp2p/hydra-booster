package idgen

import (
	"fmt"
	"math/bits"
	"sync"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
)

type BalancedIdentityGenerator struct {
	sync.Mutex
	xorTrie *XorTrie
}

func NewBalancedIdentityGenerator() *BalancedIdentityGenerator {
	return &BalancedIdentityGenerator{
		xorTrie: NewXorTrie(),
	}
}

func (bg *BalancedIdentityGenerator) Add() (crypto.PrivKey, error) {
	bg.Lock()
	defer bg.Unlock()
	p0, t0, d0, err0 := bg.genUniqueID()
	if err0 != nil {
		return nil, fmt.Errorf("generating first balanced ID candidate, %w", err0)
	}
	p1, t1, d1, err1 := bg.genUniqueID()
	if err1 != nil {
		return nil, fmt.Errorf("generating second balanced ID candidate, %w", err1)
	}
	if d0 < d1 {
		bg.xorTrie.Insert(t0)
		return p0, nil
	} else {
		bg.xorTrie.Insert(t1)
		return p1, nil
	}
}

func (bg *BalancedIdentityGenerator) genUniqueID() (privKey crypto.PrivKey, trieKey TrieKey, depth int, err error) {
	for {
		if privKey, trieKey, err = genID(); err != nil {
			return nil, nil, 0, err
		}
		if depth, ok := bg.xorTrie.Insert(trieKey); ok {
			bg.xorTrie.Remove(trieKey)
			return privKey, trieKey, depth, nil
		}
	}
}

func (bg *BalancedIdentityGenerator) Remove(privKey crypto.PrivKey) error {
	bg.Lock()
	defer bg.Unlock()
	if trieKey, err := privKeyToTrieKey(privKey); err != nil {
		return err
	} else {
		bg.xorTrie.Remove(trieKey)
		return nil
	}
}

func genID() (crypto.PrivKey, TrieKey, error) {
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("generating private key for trie, %w", err)
	}
	trieKey, err := privKeyToTrieKey(privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("converting private key to a trie key, %w", err)
	}
	return privKey, trieKey, nil
}

// PrivKey -> PeerID -> KadID -> TrieKey
func privKeyToTrieKey(privKey crypto.PrivKey) (TrieKey, error) {
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, err
	}
	kadID := kbucket.ConvertPeerID(peerID)
	trieKey := TrieKey(reversePerByteBits(kadID))
	return trieKey, nil
}

// reversePerByteBits reverses the bit-endianness of each byte in a slice.
func reversePerByteBits(blob []byte) []byte {
	for i := range blob {
		blob[i] = bits.Reverse8(blob[i])
	}
	return blob
}
