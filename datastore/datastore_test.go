package datastore

import (
	"testing"

	"github.com/ipfs/go-datastore"
)

func TestProviderKeyToCIDErrorForInvalidKey(t *testing.T) {
	_, err := providerKeyToCID(datastore.NewKey("invalid"))
	if err == nil {
		t.Fatal("expected error")
	}
}
