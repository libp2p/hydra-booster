package idgen

import "bytes"

// TrieKey is a vector of bits backed by a Go byte slice in big endian byte order and big-endian bit order.
type TrieKey []byte

func (bs TrieKey) BitAt(offset int) byte {
	if bs[offset/8]&(1<<(offset%8)) == 0 {
		return 0
	} else {
		return 1
	}
}

func (bs TrieKey) BitLen() int {
	return 8 * len(bs)
}

func TrieKeyEqual(x, y TrieKey) bool {
	return bytes.Equal(x, y)
}

// XorTrie is a trie for equal-length bit vectors, which stores values only in the leaves.
type XorTrie struct {
	branch [2]*XorTrie
	key    TrieKey
}

func NewXorTrie() *XorTrie {
	return &XorTrie{}
}

func (trie *XorTrie) Depth() int {
	return trie.depth(0)
}

func (trie *XorTrie) depth(depth int) int {
	if trie.branch[0] == nil && trie.branch[1] == nil {
		return depth
	} else {
		return max(trie.branch[0].depth(depth+1), trie.branch[1].depth(depth+1))
	}
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func (trie *XorTrie) Insert(q TrieKey) (insertedDepth int, insertedOK bool) {
	return trie.insert(0, q)
}

func (trie *XorTrie) insert(depth int, q TrieKey) (insertedDepth int, insertedOK bool) {
	if qb := trie.branch[q.BitAt(depth)]; qb != nil {
		return qb.insert(depth+1, q)
	} else {
		if trie.key == nil {
			trie.key = q
			return depth, true
		} else {
			if TrieKeyEqual(trie.key, q) {
				return depth, false
			} else {
				p := trie.key
				trie.key = nil
				// both branches are nil
				trie.branch[0], trie.branch[1] = &XorTrie{}, &XorTrie{}
				trie.branch[p.BitAt(depth)].insert(depth+1, p)
				return trie.branch[q.BitAt(depth)].insert(depth+1, q)
			}
		}
	}
}

func (trie *XorTrie) Remove(q TrieKey) (removedDepth int, removed bool) {
	return trie.remove(0, q)
}

func (trie *XorTrie) remove(depth int, q TrieKey) (reachedDepth int, removed bool) {
	if qb := trie.branch[q.BitAt(depth)]; qb != nil {
		if d, ok := qb.remove(depth+1, q); ok {
			trie.shrink()
			return d, true
		} else {
			return d, false
		}
	} else {
		if trie.key != nil && TrieKeyEqual(q, trie.key) {
			trie.key = nil
			return depth, true
		} else {
			return depth, false
		}
	}
}

func (trie *XorTrie) isEmptyLeaf() bool {
	return trie.key == nil && trie.branch[0] == nil && trie.branch[1] == nil
}

func (trie *XorTrie) isNonEmptyLeaf() bool {
	return trie.key != nil && trie.branch[0] == nil && trie.branch[1] == nil
}

func (trie *XorTrie) shrink() {
	b0, b1 := trie.branch[0], trie.branch[1]
	switch {
	case b0.isEmptyLeaf() && b1.isEmptyLeaf():
		trie.branch[0], trie.branch[1] = nil, nil
	case b0.isEmptyLeaf() && b1.isNonEmptyLeaf():
		trie.key = b1.key
		trie.branch[0], trie.branch[1] = nil, nil
	case b0.isNonEmptyLeaf() && b1.isEmptyLeaf():
		trie.key = b0.key
		trie.branch[0], trie.branch[1] = nil, nil
	}
}
