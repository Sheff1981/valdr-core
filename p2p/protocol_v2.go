package p2p

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"unicode/utf8"

	"github.com/Sheff1981/valdr-core/config"
)

const (
	v2FrameHeaderSize = 16
	v2MaxFramePayload = 4 * 1024 * 1024
)

type V2MessageType uint16

const (
	V2MessageHello V2MessageType = iota + 1
	V2MessageHelloAck
	V2MessagePing
	V2MessagePong
	V2MessageInv
	V2MessageGetData
	V2MessageBlock
	V2MessageTx
	V2MessageGetHeaders
	V2MessageHeaders
	V2MessageGetBlocks
	V2MessageGetPeers
	V2MessagePeers
	V2MessageReject
)

var (
	ErrV2WrongNetwork       = errors.New("P2P v2 wrong network")
	ErrV2UnsupportedVersion = errors.New("P2P v2 unsupported protocol version")
	ErrV2Checksum           = errors.New("P2P v2 payload checksum mismatch")
	ErrV2InvalidUTF8        = errors.New("P2P v2 payload is not valid UTF-8")
	ErrV2UnknownMessageType = errors.New("P2P v2 unknown message type")
)

type V2Frame struct {
	ProtocolVersion uint16
	MessageType     V2MessageType
	Payload         []byte
}

type V2Hello struct {
	ChainID             string   `json:"chain_id"`
	ProtocolMin         uint16   `json:"protocol_min"`
	ProtocolMax         uint16   `json:"protocol_max"`
	NodeID              string   `json:"node_id"`
	Services            []string `json:"services"`
	ListenAddress       string   `json:"listen_address"`
	Height              uint64   `json:"height"`
	TipHash             string   `json:"tip_hash"`
	CumulativeChainwork string   `json:"cumulative_chainwork"`
	UserAgent           string   `json:"user_agent"`
	Timestamp           int64    `json:"timestamp"`
	Nonce               uint64   `json:"nonce"`
}

type V2HelloAck struct {
	ProtocolVersion uint16 `json:"protocol_version"`
}

func WriteV2Frame(
	w io.Writer,
	profile config.NetworkProfile,
	protocolVersion uint16,
	messageType V2MessageType,
	value any,
) error {
	if protocolVersion < profile.ProtocolMin || protocolVersion > profile.ProtocolMax {
		return fmt.Errorf(
			"%w: got %d supported %d..%d",
			ErrV2UnsupportedVersion,
			protocolVersion,
			profile.ProtocolMin,
			profile.ProtocolMax,
		)
	}
	if !validV2MessageType(messageType) {
		return fmt.Errorf("%w: %d", ErrV2UnknownMessageType, messageType)
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(payload) == 0 {
		return ErrInvalidFrame
	}
	if len(payload) > v2MaxFramePayload {
		return ErrFrameTooLarge
	}

	magic := profile.Magic()
	sum := sha256.Sum256(payload)
	header := make([]byte, v2FrameHeaderSize)
	copy(header[0:4], magic[:])
	binary.BigEndian.PutUint16(header[4:6], protocolVersion)
	binary.BigEndian.PutUint16(header[6:8], uint16(messageType))
	binary.BigEndian.PutUint32(header[8:12], uint32(len(payload)))
	copy(header[12:16], sum[:4])

	if err := writeAll(w, header); err != nil {
		return err
	}
	return writeAll(w, payload)
}

func ReadV2Frame(r io.Reader, profile config.NetworkProfile) (V2Frame, error) {
	header := make([]byte, v2FrameHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return V2Frame{}, err
	}

	magic := profile.Magic()
	if !bytes.Equal(header[0:4], magic[:]) {
		return V2Frame{}, ErrV2WrongNetwork
	}

	protocolVersion := binary.BigEndian.Uint16(header[4:6])
	if protocolVersion < profile.ProtocolMin || protocolVersion > profile.ProtocolMax {
		return V2Frame{}, fmt.Errorf(
			"%w: got %d supported %d..%d",
			ErrV2UnsupportedVersion,
			protocolVersion,
			profile.ProtocolMin,
			profile.ProtocolMax,
		)
	}

	messageType := V2MessageType(binary.BigEndian.Uint16(header[6:8]))
	if !validV2MessageType(messageType) {
		return V2Frame{}, fmt.Errorf("%w: %d", ErrV2UnknownMessageType, messageType)
	}

	size := binary.BigEndian.Uint32(header[8:12])
	if size == 0 {
		return V2Frame{}, ErrInvalidFrame
	}
	if size > v2MaxFramePayload {
		return V2Frame{}, ErrFrameTooLarge
	}

	payload := make([]byte, int(size))
	if _, err := io.ReadFull(r, payload); err != nil {
		return V2Frame{}, err
	}
	if !utf8.Valid(payload) {
		return V2Frame{}, ErrV2InvalidUTF8
	}

	sum := sha256.Sum256(payload)
	if !bytes.Equal(header[12:16], sum[:4]) {
		return V2Frame{}, ErrV2Checksum
	}

	return V2Frame{
		ProtocolVersion: protocolVersion,
		MessageType:     messageType,
		Payload:         payload,
	}, nil
}

func DecodeV2Payload(frame V2Frame, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(frame.Payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFrame, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ErrInvalidFrame
	}
	return nil
}

func NegotiateV2Hello(
	profile config.NetworkProfile,
	localMin uint16,
	localMax uint16,
	remote V2Hello,
) (uint16, error) {
	if remote.ChainID != profile.ChainID {
		return 0, fmt.Errorf(
			"%w: got %q want %q",
			ErrWrongChainID,
			remote.ChainID,
			profile.ChainID,
		)
	}
	if localMin == 0 || localMin > localMax ||
		remote.ProtocolMin == 0 || remote.ProtocolMin > remote.ProtocolMax {
		return 0, ErrV2UnsupportedVersion
	}

	minVersion := localMin
	if remote.ProtocolMin > minVersion {
		minVersion = remote.ProtocolMin
	}
	maxVersion := localMax
	if remote.ProtocolMax < maxVersion {
		maxVersion = remote.ProtocolMax
	}
	if minVersion > maxVersion {
		return 0, ErrV2UnsupportedVersion
	}

	if remote.NodeID == "" || len(remote.NodeID) > maxNodeIDLength {
		return 0, ErrInvalidHello
	}
	if _, _, err := net.SplitHostPort(remote.ListenAddress); err != nil {
		return 0, fmt.Errorf("%w: listen address: %v", ErrInvalidHello, err)
	}
	return maxVersion, nil
}

func validV2MessageType(messageType V2MessageType) bool {
	return messageType >= V2MessageHello && messageType <= V2MessageReject
}
