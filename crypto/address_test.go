package crypto

import (
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestVALDRAddress(t *testing.T) {
	priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	address, err := AddressFromPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(address, config.AddressPrefix) {
		t.Fatalf("address %q missing prefix", address)
	}
	if !ValidateAddress(address) {
		t.Fatalf("address %q did not validate", address)
	}

	last := address[len(address)-1]
	replacement := byte('A')
	if last == replacement {
		replacement = 'B'
	}
	bad := address[:len(address)-1] + string(replacement)
	if ValidateAddress(bad) {
		t.Fatalf("modified address %q validated", bad)
	}

	lowercaseBody := config.AddressPrefix + strings.ToLower(strings.TrimPrefix(address, config.AddressPrefix))
	if lowercaseBody != address && ValidateAddress(lowercaseBody) {
		t.Fatalf("non-canonical address %q validated", lowercaseBody)
	}
}
