//go:build randomx_native && cgo

package randomxnative

import (
	"encoding/hex"
	"testing"
)

func TestOfficialRandomXVector(t *testing.T) {
	h, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	got, err := h.Hash([]byte("test key 000"), []byte("This is a test"))
	if err != nil {
		t.Fatal(err)
	}
	const want = "639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f"
	if hex.EncodeToString(got) != want {
		t.Fatalf("hash=%x want=%s", got, want)
	}
}

func TestSeedCacheReinitialization(t *testing.T) {
	h, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	a, err := h.Hash([]byte("seed-a"), []byte("same input"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.Hash([]byte("seed-b"), []byte("same input"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := h.Hash([]byte("seed-a"), []byte("same input"))
	if err != nil {
		t.Fatal(err)
	}
	if string(a) == string(b) {
		t.Fatal("different seed keys produced identical hash")
	}
	if string(a) != string(c) {
		t.Fatal("reinitialized seed key did not reproduce original hash")
	}
}
