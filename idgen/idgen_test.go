package idgen

import (
	"fmt"
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

	fmt.Printf("balanced depth %d, unbalanced depth %d\n", genBalanced.Depth(), genUnbalanced.Depth())
}
