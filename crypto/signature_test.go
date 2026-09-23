package crypto

import "testing"

func TestSignatureRoundTrip(t *testing.T) {
	priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	privHex, err := EncodePrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	decodedPriv, err := DecodePrivateKey(privHex)
	if err != nil {
		t.Fatal(err)
	}

	pubHex, err := EncodePublicKey(&decodedPriv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	decodedPub, err := DecodePublicKey(pubHex)
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("VALDR signature test")
	sig, err := Sign(decodedPriv, message)
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(decodedPub, message, sig) {
		t.Fatal("valid signature rejected")
	}
	if Verify(decodedPub, []byte("tampered"), sig) {
		t.Fatal("tampered message accepted")
	}
}
