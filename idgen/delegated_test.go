package idgen

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"testing"

	"github.com/libp2p/go-libp2p-core/crypto"
)

func TestDelegatedAddBalanced(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	bidg := NewBalancedIdentityGenerator()

	mux := http.NewServeMux()
	mux.HandleFunc("/idgen/add", func(w http.ResponseWriter, r *http.Request) {
		pk, _ := bidg.AddBalanced()
		b, _ := crypto.MarshalPrivateKey(pk)
		json.NewEncoder(w).Encode(base64.StdEncoding.EncodeToString(b))
	})

	go http.Serve(listener, mux)
	defer listener.Close()

	count := bidg.Count()
	if count != 0 {
		t.Fatal("unexpected count")
	}

	didg := NewDelegatedIDGenerator("http://" + listener.Addr().String())
	_, err = didg.AddBalanced()
	if err != nil {
		t.Fatal(err)
	}

	count = bidg.Count()
	if count != 1 {
		t.Fatal("unexpected count")
	}
}

func TestDelegatedRemove(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	bidg := NewBalancedIdentityGenerator()

	mux := http.NewServeMux()
	mux.HandleFunc("/idgen/remove", func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		var b64 string
		dec.Decode(&b64)
		bytes, _ := base64.StdEncoding.DecodeString(b64)
		pk, _ := crypto.UnmarshalPrivateKey(bytes)
		err = bidg.Remove(pk)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	go http.Serve(listener, mux)
	defer listener.Close()

	pk, err := bidg.AddBalanced()
	if err != nil {
		t.Fatal(err)
	}

	count := bidg.Count()
	if count != 1 {
		t.Fatal("unexpected count")
	}

	didg := NewDelegatedIDGenerator("http://" + listener.Addr().String())
	err = didg.Remove(pk)
	if err != nil {
		t.Fatal(err)
	}

	count = bidg.Count()
	if count != 0 {
		t.Fatal("unexpected count")
	}
}
