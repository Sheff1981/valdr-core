package wallet

import "testing"

func FuzzWalletV2StrictDecoder(f *testing.F) {
	f.Add([]byte(`{
		"version":2,
		"name":"wallet",
		"address":"VDR1invalid-seed-only",
		"public_key":"00",
		"created_at":"2026-09-26T00:00:00Z",
		"kdf":{"name":"scrypt","n":32768,"r":8,"p":1,"key_len":32,"salt":"00000000000000000000000000000000"},
		"cipher":{"name":"aes-256-gcm","nonce":"000000000000000000000000","ciphertext":"00"}
	}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"version":2,"unknown":true}`))
	f.Add([]byte("not-json"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, raw []byte) {
		var file WalletFileV2
		_ = decodeWalletV2Strict(raw, &file)
	})
}
