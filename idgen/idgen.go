package idgen

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/bits"
	"sync"
	"sync/atomic"

	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/hkdf"
)

// HydraIdentityGenerator is a shared balanced ID generator.
var HydraIdentityGenerator = NewBalancedIdentityGenerator()

// IdentityGenerator describes a facility that can generate IPFS private keys.
type IdentityGenerator interface {
	AddBalanced() (crypto.PrivKey, error)
	Remove(privKey crypto.PrivKey) error
}

// BalancedIdentityGenerator is a facility for generating IPFS identities (i.e. IPFS private keys),
// whose corresponding DHT keys are highly balanced, compared to just generating random keys.
// Balancing is accomplished using "the power of two choices" paradigm:
// https://www.eecs.harvard.edu/~michaelm/postscripts/mythesis.pdf
//
// New identities are generated by calling AddBalanced. BalancedIdentityGenerator remembers
// generated identities, in order to ensure balance for future identities.
// Generated identities can be removed using Remove.
//
// BalancedIdentityGenerator maintains the invariant that all identities, presently in its memory,
// form an almost-perfectly balanced set.
type BalancedIdentityGenerator struct {
	sync.Mutex
	xorTrie    *XorTrie
	count      int
	idgenCount uint32
	seed       []byte
}

func RandomSeed() (blk []byte) {
	blk = make([]byte, 32)
	rand.Read(blk)
	return blk
}

// NewBalancedIdentityGenerator creates a new balanced identity generator.
func NewBalancedIdentityGenerator() *BalancedIdentityGenerator {
	seed := RandomSeed()
	return NewBalancedIdentityGeneratorFromSeed(seed, 0)
}

func NewBalancedIdentityGeneratorFromSeed(seed []byte, idOffset int) *BalancedIdentityGenerator {
	idGenerator := &BalancedIdentityGenerator{
		xorTrie: NewXorTrie(),
		seed:    seed,
	}
	for i := 0; i < idOffset; i++ {
		idGenerator.AddBalanced()
	}
	return idGenerator
}

// AddUnbalanced is used for testing purposes. It generates a purely random identity,
// which is not balanced with respect to the existing identities in the generator.
// The generated identity is stored in the generator's memory.
func (bg *BalancedIdentityGenerator) AddUnbalanced() (crypto.PrivKey, error) {
	bg.Lock()
	defer bg.Unlock()
	p0, t0, _, err0 := bg.genUniqueID()
	if err0 != nil {
		return nil, fmt.Errorf("generating unbalanced ID candidate, %w", err0)
	}
	bg.xorTrie.Insert(t0)
	bg.count++
	return p0, nil
}

// AddBalanced generates a random identity, which
// is balanced with respect to the existing identities in the generator.
// The generated identity is stored in the generator's memory.
func (bg *BalancedIdentityGenerator) AddBalanced() (crypto.PrivKey, error) {
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
		bg.count++
		return p0, nil
	} else {
		bg.xorTrie.Insert(t1)
		bg.count++
		return p1, nil
	}
}

func (bg *BalancedIdentityGenerator) genUniqueID() (privKey crypto.PrivKey, trieKey TrieKey, depth int, err error) {
	for {
		if privKey, trieKey, err = bg.genID(); err != nil {
			return nil, nil, 0, err
		}
		if depth, ok := bg.xorTrie.Insert(trieKey); ok {
			bg.xorTrie.Remove(trieKey)
			return privKey, trieKey, depth, nil
		}
	}
}

// Remove removes a previously generated identity from the generator's memory.
func (bg *BalancedIdentityGenerator) Remove(privKey crypto.PrivKey) error {
	bg.Lock()
	defer bg.Unlock()
	if trieKey, err := privKeyToTrieKey(privKey); err != nil {
		return err
	} else {
		if _, ok := bg.xorTrie.Remove(trieKey); ok {
			bg.count--
		}
		return nil
	}
}

func (bg *BalancedIdentityGenerator) Count() int {
	bg.Lock()
	defer bg.Unlock()
	return bg.count
}

func (bg *BalancedIdentityGenerator) Depth() int {
	bg.Lock()
	defer bg.Unlock()
	return bg.xorTrie.Depth()
}

func (bg *BalancedIdentityGenerator) genID() (crypto.PrivKey, TrieKey, error) {
	hash := sha256.New
	info := []byte("hydra keys")
	seed := bg.seed
	salt := atomic.AddUint32(&bg.idgenCount, 1)
	salt_bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(salt_bytes, salt)
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, hkdf.New(hash, seed, salt_bytes, info))
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
