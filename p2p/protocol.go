package p2p

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	messageTypeHello  = "hello"
	maxFramePayload   = 16 * 1024
	maxNodeIDLength   = 128
)

var (
	ErrFrameTooLarge       = errors.New("P2P frame too large")
	ErrInvalidFrame        = errors.New("invalid P2P frame")
	ErrInvalidHello        = errors.New("invalid P2P hello")
	ErrWrongChainID        = errors.New("P2P chain id mismatch")
	ErrProtocolVersion     = errors.New("P2P protocol version mismatch")
	ErrSelfConnection      = errors.New("P2P self connection")
	ErrDuplicatePeer       = errors.New("P2P peer already connected")
)

type helloMessage struct {
	Type            string `json:"type"`
	ProtocolVersion uint32 `json:"protocol_version"`
	ChainID         string `json:"chain_id"`
	NodeID          string `json:"node_id"`
	ListenAddress   string `json:"listen_address"`
	Height          uint64 `json:"height"`
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
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return err
	}

	size := binary.BigEndian.Uint32(header[:])
	if size == 0 {
		return ErrInvalidFrame
	}
	if size > maxFramePayload {
		return ErrFrameTooLarge
	}

	payload := make([]byte, int(size))
	if _, err := io.ReadFull(r, payload); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFrame, err)
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
