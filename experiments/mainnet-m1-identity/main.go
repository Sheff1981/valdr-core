package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const (
	CandidateChainID         = "valdr-mainnet-1"
	CandidateP2PPort  uint16 = 27333
	CandidateRPCPort  uint16 = 27332
	CandidateProtocol uint16 = 2
	CandidateDataDir         = "mainnet"
)

type candidateIdentity struct {
	Status          string `json:"status"`
	ChainID         string `json:"chain_id"`
	P2PMagic        string `json:"p2p_magic"`
	P2PPort         uint16 `json:"p2p_port"`
	RPCPort         uint16 `json:"rpc_port"`
	ProtocolVersion uint16 `json:"protocol_version"`
	DataDir         string `json:"data_dir"`
	AddressPrefix   string `json:"address_prefix"`
	Bootstrap       string `json:"bootstrap"`
}

func candidateMagic() [4]byte {
	sum := sha256.Sum256([]byte(CandidateChainID))
	return [4]byte{sum[0], sum[1], sum[2], sum[3]}
}

func main() {
	magic := candidateMagic()
	out := candidateIdentity{
		Status:          "engineering-candidate-only",
		ChainID:         CandidateChainID,
		P2PMagic:        hex.EncodeToString(magic[:]),
		P2PPort:         CandidateP2PPort,
		RPCPort:         CandidateRPCPort,
		ProtocolVersion: CandidateProtocol,
		DataDir:         CandidateDataDir,
		AddressPrefix:   "UNRESOLVED",
		Bootstrap:       "DEFERRED_UNTIL_TESTNET2_P2P_VALIDATED",
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(raw))
}
