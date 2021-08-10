package idgen

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
)

func TestBalancedGeneration(t *testing.T) {
	const N = 100

	genBalanced := NewBalancedIdentityGenerator()
	for i := 0; i < N; i++ {
		if _, err := genBalanced.AddBalanced(); err != nil {
			t.Errorf("adding balanced ID, %w", err)
		}
	}

	genUnbalanced := NewBalancedIdentityGenerator()
	for i := 0; i < N; i++ {
		if _, err := genUnbalanced.AddUnbalanced(); err != nil {
			t.Errorf("adding unbalanced ID, %w", err)
		}
	}

	if dBal, dUnbal := genBalanced.Depth(), genUnbalanced.Depth(); dBal > dUnbal {
		t.Errorf("balanced depth %d is bigger than unbalanced depth %d\n", dBal, dUnbal)
	}
}

func TestGenFromSeed(t *testing.T) {
	seed := GenRandomBytes(256)
	bg1 := NewBalancedIdentityGeneratorFromSeed(seed)
	bg2 := NewBalancedIdentityGeneratorFromSeed(seed)
	const N = 100
	for i := 0; i < N; i++ {
		bg1_id, _, _ := bg1.genID()
		bg2_id, _, _ := bg2.genID()
		if !crypto.KeyEqual(bg1_id, bg2_id) {
			t.Error("IDs not same with same seed")
		}
	}
}
