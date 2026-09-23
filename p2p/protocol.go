package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

const (
	messageTypeHello       = "hello"
	messageTypeBlock       = "block"
	messageTypeTransaction = "transaction"
	messageTypeGetBlock    = "get_block"
	messageTypeGetPeers    = "get_peers"
	messageTypePeers       = "peers"

	maxFramePayload = 4 * 1024 * 1024
	maxNodeIDLength = 128
)

var (
	ErrFrameTooLarge   = errors.New("P2P frame too large")
	ErrInvalidFrame    = errors.New("invalid P2P frame")
	ErrInvalidHello    = errors.New("invalid P2P hello")
	ErrUnknownMessage  = errors.New("unknown P2P message")
	ErrWrongChainID    = errors.New("P2P chain id mismatch")
	ErrProtocolVersion = errors.New("P2P protocol version mismatch")
	ErrSelfConnection  = errors.New("P2P self connection")
	ErrDuplicatePeer   = errors.New("P2P peer already connected")
)

type helloMessage struct {
	Type            string `json:"type"`
	ProtocolVersion uint32 `json:"protocol_version"`
	ChainID         string `json:"chain_id"`
	NodeID          string `json:"node_id"`
	ListenAddress   string `json:"listen_address"`
	Height          uint64 `json:"height"`
}

type messageHeader struct {
	Type string `json:"type"`
}

type blockMessage struct {
	Type  string       `json:"type"`
	Block *block.Block `json:"block"`
}

type transactionMessage struct {
	Type        string                   `json:"type"`
	Transaction *transaction.Transaction `json:"transaction"`
}

type getBlockMessage struct {
	Type   string `json:"type"`
	Height uint64 `json:"height"`
}

type getPeersMessage struct {
	Type string `json:"type"`
}

type peerAdvertisement struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

type peersMessage struct {
	Type  string              `json:"type"`
	Peers []peerAdvertisement `json:"peers"`
}

func writeFrame(w io.Writer, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(payload) == 0 || len(payload) > maxFramePayload {
		return ErrFrameTooLarge
	}

	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if err := writeAll(w, header[:]); err != nil {
		return err
	}
	return writeAll(w, payload)
}

func readFrame(r io.Reader, value any) error {
	payload, err := readFramePayload(r)
	if err != nil {
		return err
	}
	return decodePayload(payload, value)
}

func readFramePayload(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	size := binary.BigEndian.Uint32(header[:])
	if size == 0 {
		return nil, ErrInvalidFrame
	}
	if size > maxFramePayload {
		return nil, ErrFrameTooLarge
	}

	payload := make([]byte, int(size))
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func decodeMessageHeader(payload []byte) (messageHeader, error) {
	var header messageHeader
	if err := json.Unmarshal(payload, &header); err != nil {
		return messageHeader{}, fmt.Errorf("%w: %v", ErrInvalidFrame, err)
	}
	if header.Type == "" {
		return messageHeader{}, ErrInvalidFrame
	}
	return header, nil
}

func decodePayload(payload []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFrame, err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ErrInvalidFrame
	}
	return nil
}

func writeAll(w io.Writer, payload []byte) error {
	for len(payload) > 0 {
		n, err := w.Write(payload)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		payload = payload[n:]
	}
	return nil
}
