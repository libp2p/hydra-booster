package idgen

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
)

func TestBalancedGeneration(t *testing.T) {
	const N = 10000

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
	seed := RandomSeed()
	bg1 := NewBalancedIdentityGeneratorFromSeed(seed, 0)
	bg2 := NewBalancedIdentityGeneratorFromSeed(seed, 0)
	const N = 10000
	for i := 0; i < N; i++ {
		bg1_id, err := bg1.AddBalanced()
		if err != nil {
			t.Error("bg1_id creation error")
		}
		bg2_id, err := bg2.AddBalanced()
		if err != nil {
			t.Error("bg2_id creation error")
		}
		if !crypto.KeyEqual(bg1_id, bg2_id) {
			t.Error("IDs not same with same seed")
		}
	}
	// To generate N IDs, we should have tried 2*N candidates
	if bg1.idgenCount != 2*N {
		t.Errorf("bg1 should have %d items but it has %d", 2*N, bg1.idgenCount)
	}
	if bg2.idgenCount != 2*N {
		t.Errorf("bg2 should have %d items but it has %d", 2*N, bg2.idgenCount)
	}
}

func TestWithOffSet(t *testing.T) {
	seed := RandomSeed()
	bg1 := NewBalancedIdentityGeneratorFromSeed(seed, 0)
	bg2 := NewBalancedIdentityGeneratorFromSeed(seed, 100)

	// After using up 100 of bg1
	for i := 0; i < 100; i++ {
		bg1.AddBalanced()
	}
	// It should start to be the same as bg2
	for i := 0; i < 100; i++ {
		bg1_id, err := bg1.AddBalanced()
		if err != nil {
			t.Error("bg1_id creation error")
		}
		bg2_id, err := bg2.AddBalanced()
		if err != nil {
			t.Error("bg2_id creation error")
		}
		if !crypto.KeyEqual(bg1_id, bg2_id) {
			t.Error("IDs not same with same seed")
		}
	}
}
