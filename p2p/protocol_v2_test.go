package p2p

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestV2FrameGoldenVector(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := WriteV2Frame(
		&buf,
		profile,
		2,
		V2MessagePing,
		struct {
			Nonce uint64 `json:"nonce"`
		}{Nonce: 42},
	); err != nil {
		t.Fatal(err)
	}

	const wantHex = "4b6638e3000200030000000ca728374c7b226e6f6e6365223a34327d"
	if got := hex.EncodeToString(buf.Bytes()); got != wantHex {
		t.Fatalf("frame=%s want=%s", got, wantHex)
	}

	frame, err := ReadV2Frame(bytes.NewReader(buf.Bytes()), profile)
	if err != nil {
		t.Fatal(err)
	}
	if frame.ProtocolVersion != 2 || frame.MessageType != V2MessagePing {
		t.Fatalf("header=%+v", frame)
	}

	var ping struct {
		Nonce uint64 `json:"nonce"`
	}
	if err := DecodeV2Payload(frame, &ping); err != nil {
		t.Fatal(err)
	}
	if ping.Nonce != 42 {
		t.Fatalf("nonce=%d want=42", ping.Nonce)
	}
}

func TestV2FrameRejectsWrongNetworkBeforePayloadDecode(t *testing.T) {
	devnet, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	testnet, _ := config.ResolveNetworkProfile(config.NetworkTestnetV029)

	var buf bytes.Buffer
	if err := WriteV2Frame(&buf, devnet, 2, V2MessagePing, map[string]any{"nonce": 1}); err != nil {
		t.Fatal(err)
	}

	_, err := ReadV2Frame(bytes.NewReader(buf.Bytes()), testnet)
	if !errors.Is(err, ErrV2WrongNetwork) {
		t.Fatalf("error=%v want ErrV2WrongNetwork", err)
	}
}

func TestV2FrameRejectsChecksumMismatch(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)

	var buf bytes.Buffer
	if err := WriteV2Frame(&buf, profile, 2, V2MessagePing, map[string]any{"nonce": 42}); err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), buf.Bytes()...)
	raw[len(raw)-2] = '3'

	_, err := ReadV2Frame(bytes.NewReader(raw), profile)
	if !errors.Is(err, ErrV2Checksum) {
		t.Fatalf("error=%v want ErrV2Checksum", err)
	}
}

func TestV2StrictJSONRejectsUnknownFields(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)

	var buf bytes.Buffer
	if err := WriteV2Frame(
		&buf,
		profile,
		2,
		V2MessagePing,
		map[string]any{"nonce": 42, "unexpected": true},
	); err != nil {
		t.Fatal(err)
	}
	frame, err := ReadV2Frame(bytes.NewReader(buf.Bytes()), profile)
	if err != nil {
		t.Fatal(err)
	}

	var target struct {
		Nonce uint64 `json:"nonce"`
	}
	if err := DecodeV2Payload(frame, &target); !errors.Is(err, ErrInvalidFrame) {
		t.Fatalf("error=%v want ErrInvalidFrame", err)
	}
}

func TestV2HelloNegotiatesHighestMutualVersion(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	hello := V2Hello{
		ChainID:       profile.ChainID,
		ProtocolMin:   1,
		ProtocolMax:   2,
		NodeID:        "node-b",
		ListenAddress: "127.0.0.1:7333",
	}

	selected, err := NegotiateV2Hello(profile, 1, 2, hello)
	if err != nil {
		t.Fatal(err)
	}
	if selected != 2 {
		t.Fatalf("selected=%d want=2", selected)
	}

	hello.ProtocolMin = 3
	hello.ProtocolMax = 3
	if _, err := NegotiateV2Hello(profile, 1, 2, hello); !errors.Is(err, ErrV2UnsupportedVersion) {
		t.Fatalf("error=%v want ErrV2UnsupportedVersion", err)
	}

	hello.ProtocolMin = 2
	hello.ProtocolMax = 2
	hello.ChainID = "valdr-testnet-2"
	if _, err := NegotiateV2Hello(profile, 2, 2, hello); !errors.Is(err, ErrWrongChainID) {
		t.Fatalf("error=%v want ErrWrongChainID", err)
	}
}


func TestV2PerMessagePayloadLimits(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oversizedPing := struct {
		Nonce uint64 `json:"nonce"`
		Pad   string `json:"pad"`
	}{
		Nonce: 1,
		Pad:   string(bytes.Repeat([]byte("x"), v2MessagePayloadLimit(V2MessagePing))),
	}
	if err := WriteV2Frame(
		&buf,
		profile,
		2,
		V2MessagePing,
		oversizedPing,
	); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("WriteV2Frame error=%v want ErrFrameTooLarge", err)
	}

	payload := bytes.Repeat([]byte("x"), v2MessagePayloadLimit(V2MessagePing)+1)
	magic := profile.Magic()
	sum := sha256.Sum256(payload)
	header := make([]byte, v2FrameHeaderSize)
	copy(header[0:4], magic[:])
	binary.BigEndian.PutUint16(header[4:6], 2)
	binary.BigEndian.PutUint16(header[6:8], uint16(V2MessagePing))
	binary.BigEndian.PutUint32(header[8:12], uint32(len(payload)))
	copy(header[12:16], sum[:4])

	raw := append(header, payload...)
	if _, err := ReadV2Frame(bytes.NewReader(raw), profile); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("ReadV2Frame error=%v want ErrFrameTooLarge", err)
	}
}
