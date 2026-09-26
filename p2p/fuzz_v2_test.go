package p2p

import (
	"bytes"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func FuzzReadV2Frame(f *testing.F) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		f.Fatal(err)
	}

	var valid bytes.Buffer
	if err := WriteV2Frame(
		&valid,
		profile,
		profile.ProtocolMax,
		V2MessagePing,
		map[string]uint64{"nonce": 1},
	); err != nil {
		f.Fatal(err)
	}

	f.Add(valid.Bytes())
	f.Add([]byte{})
	f.Add([]byte("not-a-frame"))
	f.Add(make([]byte, v2FrameHeaderSize))

	f.Fuzz(func(t *testing.T, data []byte) {
		frame, err := ReadV2Frame(bytes.NewReader(data), profile)
		if err != nil {
			return
		}
		if len(frame.Payload) == 0 || len(frame.Payload) > v2MaxFramePayload {
			t.Fatalf("accepted invalid payload length %d", len(frame.Payload))
		}
		if !validV2MessageType(frame.MessageType) {
			t.Fatalf("accepted invalid message type %d", frame.MessageType)
		}

		var payload any
		_ = DecodeV2Payload(frame, &payload)
	})
}

func FuzzDecodeV2HelloPayload(f *testing.F) {
	f.Add([]byte(`{"chain_id":"valdr-testnet-2","protocol_min":2,"protocol_max":2,"node_id":"node","services":[],"listen_address":"127.0.0.1:17333","height":0,"tip_hash":"","cumulative_chainwork":"","user_agent":"fuzz","timestamp":0,"nonce":1}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"unknown":true}`))

	f.Fuzz(func(t *testing.T, payload []byte) {
		frame := V2Frame{
			ProtocolVersion: 2,
			MessageType:     V2MessageHello,
			Payload:         payload,
		}
		var hello V2Hello
		_ = DecodeV2Payload(frame, &hello)
	})
}
