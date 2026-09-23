#!/usr/bin/env python3
from pathlib import Path
import re, sys, shutil, subprocess
if len(sys.argv)!=2: raise SystemExit('usage: apply_valdr_testnet.py /path/to/bitcoin-31.1')
root=Path(sys.argv[1]).resolve(); here=Path(__file__).resolve().parents[1]
if not (root/'src/kernel/chainparams.cpp').exists(): raise SystemExit('Bitcoin Core 31.1 source tree not found')
subprocess.run([sys.executable,str(here/'tools/apply_valdr.py'),str(root)],check=True)
def sub(path,pat,repl,flags=0):
    p=root/path; s=p.read_text(encoding='utf-8'); out,n=re.subn(pat,repl,s,count=1,flags=flags)
    if n!=1: raise SystemExit(f'patch failed {path}: count={n}')
    p.write_text(out,encoding='utf-8'); print('patched',path)
sub('src/chainparams.cpp',r'case ChainType::TESTNET:\n\s*return CChainParams::TestNet\(\);','case ChainType::TESTNET:\n        throw std::runtime_error("VALDR Core: inherited Bitcoin testnet3 is disabled; use -testnet4 for VALDR TESTNET");')
sub('src/chainparams.cpp',r'case ChainType::SIGNET: \{.*?\n\s*\}','case ChainType::SIGNET:\n        throw std::runtime_error("VALDR Core: inherited Bitcoin signet is disabled");',re.S)
p=root/'src/consensus/params.h'; s=p.read_text(encoding='utf-8'); needle='    int64_t nPowTargetTimespan;\n'
if needle not in s: raise SystemExit('params.h anchor missing')
s=s.replace(needle,needle+'    bool fPowUseASERT{false};\n    int64_t nDAAHalfLife{172800};\n    int nASERTAnchorHeight{1};\n',1); p.write_text(s,encoding='utf-8')
(root/'src/pow').mkdir(exist_ok=True)
shutil.copy2(here/'overlay/src/pow/aserti32d.h',root/'src/pow/aserti32d.h')
shutil.copy2(here/'overlay/src/pow/aserti32d.cpp',root/'src/pow/aserti32d.cpp')
for cm in ['src/kernel/CMakeLists.txt','src/CMakeLists.txt']:
    p=root/cm; s=p.read_text(encoding='utf-8')
    anchor='  ../pow.cpp\n' if cm.startswith('src/kernel') else '  pow.cpp\n'
    add='  ../pow/aserti32d.cpp\n' if cm.startswith('src/kernel') else '  pow/aserti32d.cpp\n'
    if anchor not in s: raise SystemExit(f'{cm} pow anchor missing')
    p.write_text(s.replace(anchor,anchor+add,1),encoding='utf-8')
p=root/'src/pow.cpp'; s=p.read_text(encoding='utf-8')
if '#include <pow.h>\n' not in s: raise SystemExit('pow include anchor missing')
s=s.replace('#include <pow.h>\n','#include <pow.h>\n#include <pow/aserti32d.h>\n',1)
needle='    unsigned int nProofOfWorkLimit = UintToArith256(params.powLimit).GetCompact();\n'
if needle not in s: raise SystemExit('GetNextWorkRequired anchor missing')
s=s.replace(needle,needle+'\n    if (params.fPowUseASERT && pindexLast->nHeight >= params.nASERTAnchorHeight)\n        return GetNextVALDRASERTWorkRequired(pindexLast, pblock, params);\n',1)
needle='    if (params.fPowAllowMinDifficultyBlocks) return true;\n'
if needle not in s: raise SystemExit('PermittedDifficultyTransition anchor missing')
s=s.replace(needle,'    if (params.fPowAllowMinDifficultyBlocks || params.fPowUseASERT) return true;\n',1)
p.write_text(s,encoding='utf-8')
TEST = r'''class CTestNet4Params : public CChainParams {
public:
    CTestNet4Params() {
        m_chain_type = ChainType::TESTNET4;
        consensus.signet_blocks = false;
        consensus.signet_challenge.clear();
        consensus.nSubsidyHalvingInterval = 210000;
        consensus.BIP34Height = 1; consensus.BIP34Hash = uint256{};
        consensus.BIP65Height = 1; consensus.BIP66Height = 1; consensus.CSVHeight = 1; consensus.SegwitHeight = 1;
        consensus.MinBIP9WarningHeight = 0;
        consensus.powLimit = uint256{"00000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"};
        consensus.nPowTargetTimespan = 14 * 24 * 60 * 60;
        consensus.nPowTargetSpacing = 10 * 60;
        consensus.fPowAllowMinDifficultyBlocks = true;
        consensus.enforce_BIP94 = false;
        consensus.fPowNoRetargeting = false;
        consensus.fPowUseASERT = true;
        consensus.nDAAHalfLife = 2 * 24 * 60 * 60;
        consensus.nASERTAnchorHeight = 1;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].bit = 28;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].nStartTime = Consensus::BIP9Deployment::NEVER_ACTIVE;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].nTimeout = Consensus::BIP9Deployment::NO_TIMEOUT;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].min_activation_height = 0;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].threshold = 1512;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].period = 2016;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].bit = 2;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].nStartTime = Consensus::BIP9Deployment::ALWAYS_ACTIVE;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].nTimeout = Consensus::BIP9Deployment::NO_TIMEOUT;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].min_activation_height = 0;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].threshold = 1512;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].period = 2016;
        consensus.nMinimumChainWork = uint256{};
        consensus.defaultAssumeValid = uint256{};
        pchMessageStart[0] = 0x8e; pchMessageStart[1] = 0xa4; pchMessageStart[2] = 0x25; pchMessageStart[3] = 0x6a;
        nDefaultPort = 37333; nPruneAfterHeight = 1000; m_assumed_blockchain_size = 0; m_assumed_chain_state_size = 0;
        const char* msg = "VALDR TESTNET Genesis 23/Sep/2026 - Verify before trust";
        const CScript out = CScript() << OP_RETURN << "56414c445220544553544e45543a206e6f207072656d696e653b207075626c696320766572696669636174696f6e206e6574776f726b"_hex;
        genesis = CreateGenesisBlock(msg, out, 1790125200, 147425, 0x1e0fffff, 1, 50 * COIN);
        consensus.hashGenesisBlock = genesis.GetHash();
        assert(consensus.hashGenesisBlock == uint256{"00000cae1bd0c4bf8439e00052222799079075f09b3d53647a2fc43ac22aa97c"});
        assert(genesis.hashMerkleRoot == uint256{"c7742c1db54c04b717e2e48b4c68e50a156dd98e59d685f97a6b06dfd4fd819a"});
        vFixedSeeds.clear(); vSeeds.clear();
        base58Prefixes[PUBKEY_ADDRESS] = std::vector<unsigned char>(1,65);
        base58Prefixes[SCRIPT_ADDRESS] = std::vector<unsigned char>(1,127);
        base58Prefixes[SECRET_KEY] = std::vector<unsigned char>(1,193);
        base58Prefixes[EXT_PUBLIC_KEY] = {0x04,0x88,0xB2,0x1F};
        base58Prefixes[EXT_SECRET_KEY] = {0x04,0x88,0xAD,0xE5};
        bech32_hrp = "tvld";
        fDefaultConsistencyChecks = true;
        m_is_mockable_chain = false;
        m_assumeutxo_data = {};
        chainTxData = ChainTxData{.nTime = 0, .tx_count = 0, .dTxRate = 0.001};
        m_headers_sync_params = HeadersSyncParams{.commitment_period = 275, .redownload_buffer_size = 7017};
    }
};'''
sub('src/kernel/chainparams.cpp',r'class CTestNet4Params : public CChainParams \{.*?\n\};\n\n/\*\*\n \* Signet:',TEST+'\n\n/**\n * Signet:',re.S)
sub('src/chainparamsbase.cpp',r'case ChainType::TESTNET4:\n\s*return std::make_unique<CBaseChainParams>\("testnet4", 48332\);','case ChainType::TESTNET4:\n        return std::make_unique<CBaseChainParams>("testnet4", 37332);')
p=root/'CMakeLists.txt'; s=p.read_text(encoding='utf-8')
if 'set(CLIENT_VERSION_MINOR 1)' not in s: raise SystemExit('version anchor missing')
p.write_text(s.replace('set(CLIENT_VERSION_MINOR 1)','set(CLIENT_VERSION_MINOR 5)',1),encoding='utf-8')
(root/'VALDR-TESTNET-MARKER.txt').write_text('VALDR Core 0.5.0 PUBLIC TESTNET overlay\nBase: Bitcoin Core 31.1\nMainnet not frozen.\n',encoding='utf-8')
print('VALDR 0.5.0 PUBLIC TESTNET overlay applied')
