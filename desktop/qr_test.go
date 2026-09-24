package desktop

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

func TestDesktopAddressQRCodeMatchesVersion3QVector(t *testing.T) {
	privateKey, err := valdrcrypto.DecodePrivateKey(
		strings.Repeat("0", 63) + "1",
	)
	if err != nil {
		t.Fatal(err)
	}
	address, err := valdrcrypto.AddressFromPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	const expectedAddress = "VDR1NGF6UY64ISRUIZR76FBJV2QQQQW7E63LY2C6SUA"
	if address != expectedAddress {
		t.Fatalf("address=%q want=%q", address, expectedAddress)
	}

	matrix, err := addressQRMatrix(address)
	if err != nil {
		t.Fatal(err)
	}
	if len(matrix) != qrSize || len(matrix[0]) != qrSize {
		t.Fatalf("matrix size=%dx%d want=%dx%d", len(matrix), len(matrix[0]), qrSize, qrSize)
	}

	var bits strings.Builder
	bits.Grow(qrSize * qrSize)
	for _, row := range matrix {
		for _, dark := range row {
			if dark {
				bits.WriteByte('1')
			} else {
				bits.WriteByte('0')
			}
		}
	}
	sum := sha256.Sum256([]byte(bits.String()))
	const expectedMatrixSHA256 = "4298cdd4a242aadf3a411055030959fa2704301b1580076276e8a888cac0d87a"
	if got := hex.EncodeToString(sum[:]); got != expectedMatrixSHA256 {
		t.Fatalf("matrix sha256=%s want=%s", got, expectedMatrixSHA256)
	}
}

func TestDesktopAddressQRCodeIsLocalSVGDataURI(t *testing.T) {
	privateKey, err := valdrcrypto.DecodePrivateKey(
		strings.Repeat("0", 63) + "1",
	)
	if err != nil {
		t.Fatal(err)
	}
	address, err := valdrcrypto.AddressFromPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	uri, err := AddressQRCodeDataURI(address)
	if err != nil {
		t.Fatal(err)
	}
	const prefix = "data:image/svg+xml;base64,"
	if !strings.HasPrefix(uri, prefix) {
		t.Fatalf("unexpected QR URI prefix: %q", uri)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, prefix))
	if err != nil {
		t.Fatal(err)
	}
	svg := string(raw)
	if !strings.Contains(svg, "<svg ") ||
		!strings.Contains(svg, `viewBox="0 0 37 37"`) ||
		strings.Contains(svg, "http://") && !strings.Contains(svg, `xmlns="http://www.w3.org/2000/svg"`) ||
		strings.Contains(svg, "https://") {
		t.Fatalf("unexpected QR SVG: %s", svg)
	}
}

func TestDesktopAddressQRCodeRejectsInvalidAddress(t *testing.T) {
	if _, err := AddressQRCodeDataURI("VDR1INVALID"); err == nil {
		t.Fatal("invalid address produced QR code")
	}
}
