// Copyright (c) 2020 The Bitcoin developers
// VALDR integration modifications Copyright (c) 2026 The VALDR developers
// Distributed under the MIT software license.
#ifndef VALDR_ASERTI32D_H
#define VALDR_ASERTI32D_H
#include <cstdint>
class arith_uint256; class CBlockHeader; class CBlockIndex;
namespace Consensus { struct Params; }
arith_uint256 CalculateVALDRASERT(const arith_uint256&, int64_t, int64_t, int64_t, const arith_uint256&, int64_t) noexcept;
uint32_t GetNextVALDRASERTWorkRequired(const CBlockIndex*, const CBlockHeader*, const Consensus::Params&) noexcept;
#endif
