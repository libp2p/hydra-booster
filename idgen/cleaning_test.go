package idgen

import (
	"testing"
)

func TestCleaningIDGenerator(t *testing.T) {
	bidg := NewBalancedIdentityGenerator()

	count := bidg.Count()
	if count != 0 {
		t.Fatal("unexpected count")
	}

	didg := NewCleaningIDGenerator(bidg)
	_, err := didg.AddBalanced()
	if err != nil {
		t.Fatal(err)
	}

	pk, err := didg.AddBalanced()
	if err != nil {
		t.Fatal(err)
	}

	count = bidg.Count()
	if count != 2 {
		t.Fatal("unexpected count")
	}

	err = didg.Remove(pk)
	if err != nil {
		t.Fatal(err)
	}

	count = bidg.Count()
	if count != 1 {
		t.Fatal("unexpected count")
	}

	err = didg.Clean()
	if err != nil {
		t.Fatal(err)
	}

	count = bidg.Count()
	if count != 0 {
		t.Fatal("unexpected count")
	}
}
