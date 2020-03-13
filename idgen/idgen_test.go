package idgen

import (
	"testing"
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
