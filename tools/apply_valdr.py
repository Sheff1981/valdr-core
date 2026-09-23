#!/usr/bin/env python3
"""Apply the VALDR 0.1.0 DEVNET overlay to an unpacked Bitcoin Core 31.1 source tree.

This deliberately changes only what is needed for an independent dev network.
It does NOT claim mainnet readiness.
"""
from pathlib import Path
import re, sys

if len(sys.argv) != 2:
    raise SystemExit("usage: apply_valdr.py /path/to/bitcoin-31.1")
root = Path(sys.argv[1]).resolve()
if not (root / "src/kernel/chainparams.cpp").exists():
    raise SystemExit("Bitcoin Core 31.1 source tree not found")

MAIN_CLASS = r'''class CMainParams : public CChainParams {
public:
    CMainParams() {
        m_chain_type = ChainType::MAIN;
        consensus.signet_blocks = false;
        consensus.signet_challenge.clear();

        consensus.nSubsidyHalvingInterval = 210000;

        consensus.BIP34Height = 1;
        consensus.BIP34Hash = uint256{};
        consensus.BIP65Height = 1;
        consensus.BIP66Height = 1;
        consensus.CSVHeight = 1;
        consensus.SegwitHeight = 0;
        consensus.MinBIP9WarningHeight = 0;

        consensus.powLimit = uint256{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"};
        consensus.nPowTargetTimespan = 14 * 24 * 60 * 60;
        consensus.nPowTargetSpacing = 10 * 60;
        consensus.fPowAllowMinDifficultyBlocks = true;
        consensus.enforce_BIP94 = false;
        consensus.fPowNoRetargeting = false;

        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].bit = 28;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].nStartTime = Consensus::BIP9Deployment::NEVER_ACTIVE;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].nTimeout = Consensus::BIP9Deployment::NO_TIMEOUT;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].min_activation_height = 0;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].threshold = 1815;
        consensus.vDeployments[Consensus::DEPLOYMENT_TESTDUMMY].period = 2016;

        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].bit = 2;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].nStartTime = Consensus::BIP9Deployment::ALWAYS_ACTIVE;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].nTimeout = Consensus::BIP9Deployment::NO_TIMEOUT;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].min_activation_height = 0;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].threshold = 1815;
        consensus.vDeployments[Consensus::DEPLOYMENT_TAPROOT].period = 2016;

        consensus.nMinimumChainWork = uint256{};
        consensus.defaultAssumeValid = uint256{};

        pchMessageStart[0] = 0xf6;
        pchMessageStart[1] = 0xc1;
        pchMessageStart[2] = 0xb7;
        pchMessageStart[3] = 0xd2;
        nDefaultPort = 27333;
        nPruneAfterHeight = 1000;
        m_assumed_blockchain_size = 0;
        m_assumed_chain_state_size = 0;

        const char* valdrTimestamp = "VALDR Genesis 23/Sep/2026 - Strength Builds Freedom";
        const CScript valdrGenesisScript = CScript() << OP_RETURN
            << "56414c44523a206e6f207072656d696e653b20736f7665726569676e20706565722d746f2d70656572206d6f6e6579"_hex;
        genesis = CreateGenesisBlock(valdrTimestamp, valdrGenesisScript,
                                     1790121600, 0, 0x207fffff, 1, 50 * COIN);
        consensus.hashGenesisBlock = genesis.GetHash();
        assert(consensus.hashGenesisBlock == uint256{"3ee8cf33862988e8575fb0fb1b6b207530f945b51c398c44be7c6c473aa4260b"});
        assert(genesis.hashMerkleRoot == uint256{"77d9d01946c415a41f52f6da59c63ca2e6c2e5466b59680b8fb60a60f8420737"});

        vSeeds.clear();
        vFixedSeeds.clear();

        base58Prefixes[PUBKEY_ADDRESS] = std::vector<unsigned char>(1, 70);
        base58Prefixes[SCRIPT_ADDRESS] = std::vector<unsigned char>(1, 23);
        base58Prefixes[SECRET_KEY] = std::vector<unsigned char>(1, 198);
        base58Prefixes[EXT_PUBLIC_KEY] = {0x04, 0x5f, 0x1c, 0xf6};
        base58Prefixes[EXT_SECRET_KEY] = {0x04, 0x5f, 0x18, 0xbc};
        bech32_hrp = "vld";

        fDefaultConsistencyChecks = true;
        m_is_mockable_chain = true;
        m_assumeutxo_data = {};
        chainTxData = ChainTxData{.nTime = 0, .tx_count = 0, .dTxRate = 0.001};

        m_headers_sync_params = HeadersSyncParams{
            .commitment_period = 275,
            .redownload_buffer_size = 7017,
        };
    }
};'''

def replace_once(path, pattern, replacement, flags=0):
    p = root / path
    s = p.read_text(encoding='utf-8')
    out, n = re.subn(pattern, replacement, s, count=1, flags=flags)
    if n != 1:
        raise SystemExit(f"patch failed: {path}: pattern count={n}")
    p.write_text(out, encoding='utf-8')
    print("patched", path)

replace_once(
    "src/kernel/chainparams.cpp",
    r'class CMainParams : public CChainParams \{.*?\n\};\n\n/\*\*\n \* Testnet \(v3\):',
    MAIN_CLASS + '\n\n/**\n * Testnet (v3):',
    re.S,
)

replace_once(
    "src/chainparamsbase.cpp",
    r'case ChainType::MAIN:\n\s*return std::make_unique<CBaseChainParams>\("", 8332\);',
    'case ChainType::MAIN:\n        return std::make_unique<CBaseChainParams>("", 27332);',
)

cmake = root / "CMakeLists.txt"
s = cmake.read_text(encoding='utf-8')
repls = {
    'set(CLIENT_NAME "Bitcoin Core")': 'set(CLIENT_NAME "VALDR Core")',
    'set(CLIENT_VERSION_MAJOR 31)': 'set(CLIENT_VERSION_MAJOR 0)',
    'set(CLIENT_VERSION_MINOR 1)': 'set(CLIENT_VERSION_MINOR 1)',
    'set(CLIENT_VERSION_IS_RELEASE "true")': 'set(CLIENT_VERSION_IS_RELEASE "false")',
    'DESCRIPTION "Bitcoin client software"': 'DESCRIPTION "VALDR peer-to-peer electronic cash node"',
    'HOMEPAGE_URL "https://bitcoincore.org/"': 'HOMEPAGE_URL "https://valdr.example/"',
    'set(CLIENT_BUGREPORT "https://github.com/bitcoin/bitcoin/issues")': 'set(CLIENT_BUGREPORT "https://valdr.example/issues")',
}
for old,new in repls.items():
    if old not in s: raise SystemExit(f"patch failed CMakeLists.txt: {old}")
    s=s.replace(old,new,1)
cmake.write_text(s, encoding='utf-8')
print("patched CMakeLists.txt")

p = root / "src/policy/feerate.h"
s = p.read_text(encoding='utf-8')
if 'const std::string CURRENCY_UNIT = "BTC";' not in s:
    raise SystemExit("patch failed src/policy/feerate.h")
s = s.replace('const std::string CURRENCY_UNIT = "BTC";', 'const std::string CURRENCY_UNIT = "VLD";', 1)
p.write_text(s, encoding='utf-8')
print("patched src/policy/feerate.h")

p = root / "src/qt/guiconstants.h"
s = p.read_text(encoding='utf-8')
for old,new in {
    '#define QAPP_ORG_NAME "Bitcoin"':'#define QAPP_ORG_NAME "VALDR"',
    '#define QAPP_ORG_DOMAIN "bitcoin.org"':'#define QAPP_ORG_DOMAIN "valdr.example"',
    '#define QAPP_APP_NAME_DEFAULT "Bitcoin-Qt"':'#define QAPP_APP_NAME_DEFAULT "VALDR-Qt"',
}.items():
    if old not in s: raise SystemExit(f"patch failed qt/guiconstants.h: {old}")
    s=s.replace(old,new,1)
p.write_text(s, encoding='utf-8')
print("patched src/qt/guiconstants.h")

p = root / "src/qt/bitcoinunits.cpp"
s = p.read_text(encoding='utf-8')
for old,new in {
    'case Unit::BTC: return QString("BTC");':'case Unit::BTC: return QString("VLD");',
    'case Unit::mBTC: return QString("mBTC");':'case Unit::mBTC: return QString("mVLD");',
    'case Unit::uBTC: return QString::fromUtf8("µBTC (bits)");':'case Unit::uBTC: return QString::fromUtf8("µVLD");',
    'case Unit::uBTC: return QString("bits");':'case Unit::uBTC: return QString::fromUtf8("µVLD");',
    'case Unit::SAT: return QString("Satoshi (sat)");':'case Unit::SAT: return QString("VALDR atom");',
    'case Unit::SAT: return QString("sat");':'case Unit::SAT: return QString("atom");',
    'case Unit::BTC: return QString("Bitcoins");':'case Unit::BTC: return QString("VALDR");',
    'case Unit::mBTC: return QString("Milli-Bitcoins (1 / 1" THIN_SP_UTF8 "000)");':'case Unit::mBTC: return QString("Milli-VALDR (1 / 1" THIN_SP_UTF8 "000)");',
    'case Unit::uBTC: return QString("Micro-Bitcoins (bits) (1 / 1" THIN_SP_UTF8 "000" THIN_SP_UTF8 "000)");':'case Unit::uBTC: return QString("Micro-VALDR (1 / 1" THIN_SP_UTF8 "000" THIN_SP_UTF8 "000)");',
    'case Unit::SAT: return QString("Satoshi (sat) (1 / 100" THIN_SP_UTF8 "000" THIN_SP_UTF8 "000)");':'case Unit::SAT: return QString("VALDR atom (1 / 100" THIN_SP_UTF8 "000" THIN_SP_UTF8 "000)");',
}.items():
    if old not in s: raise SystemExit(f"patch failed qt/bitcoinunits.cpp: {old}")
    s=s.replace(old,new,1)
p.write_text(s, encoding='utf-8')
print("patched src/qt/bitcoinunits.cpp")

(root / "VALDR-DEVNET-MARKER.txt").write_text(
    "VALDR Core 0.1.0 DEVNET\nBase: Bitcoin Core 31.1\nNot mainnet-ready.\n", encoding='utf-8')
print("VALDR overlay applied successfully")
