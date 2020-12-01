package utils

import (
	"testing"
)

func TestPortSelector(t *testing.T) {
	begin := 5
	getPort := PortSelector(begin)
	for i := begin; i < begin*1000; i++ {
		port := getPort()
		if port != i {
			t.Fatalf("expected next port to be %v but got %v", i+1, port)
		}
	}
}

func TestPortSelectorBeginZero(t *testing.T) {
	begin := 0
	getPort := PortSelector(begin)
	for i := begin; i < 1000; i++ {
		port := getPort()
		if port != 0 {
			t.Fatalf("expected next port to be 0 but got %v", port)
		}
	}
}
