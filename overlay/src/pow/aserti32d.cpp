// Derived from Bitcoin ABC ASERTI3-2d implementation (MIT).
// VALDR-specific names/integration Copyright (c) 2026 The VALDR developers.
#include <pow/aserti32d.h>
#include <arith_uint256.h>
#include <chain.h>
#include <consensus/params.h>
#include <primitives/block.h>
#include <uint256.h>
#include <cassert>
#include <cstdlib>

arith_uint256 CalculateVALDRASERT(const arith_uint256 &refTarget, int64_t spacing,
    int64_t timeDiff, int64_t heightDiff, const arith_uint256 &powLimit, int64_t halfLife) noexcept {
    assert(refTarget > 0 && refTarget <= powLimit);
    assert((powLimit >> 236) == 0);
    assert(heightDiff >= 0);
    assert(llabs(timeDiff - spacing * heightDiff) < (1ll << (63-16)));
    const int64_t exponent=((timeDiff-spacing*(heightDiff+1))*65536)/halfLife;
    static_assert(int64_t(-1)>>1==int64_t(-1));
    int64_t shifts=exponent>>16; const auto frac=uint16_t(exponent);
    const uint32_t factor=65536+((195766423245049ull*frac+971821376ull*frac*frac+5127ull*frac*frac*frac+(1ull<<47))>>48);
    arith_uint256 nextTarget=refTarget*factor; shifts-=16;
    if(shifts<=0) nextTarget >>= -shifts;
    else { const auto shifted=nextTarget<<shifts; nextTarget=((shifted>>shifts)!=nextTarget)?powLimit:shifted; }
    if(nextTarget==0) nextTarget=arith_uint256(1); else if(nextTarget>powLimit) nextTarget=powLimit;
    return nextTarget;
}

uint32_t GetNextVALDRASERTWorkRequired(const CBlockIndex *prev, const CBlockHeader *block, const Consensus::Params &params) noexcept {
    assert(prev); const arith_uint256 powLimit=UintToArith256(params.powLimit);
    if(params.fPowAllowMinDifficultyBlocks && block->GetBlockTime()>prev->GetBlockTime()+2*params.nPowTargetSpacing) return powLimit.GetCompact();
    const CBlockIndex *anchor=prev->GetAncestor(params.nASERTAnchorHeight); assert(anchor);
    const int64_t anchorTime=anchor->pprev?anchor->pprev->GetBlockTime():anchor->GetBlockTime();
    const int64_t timeDiff=prev->GetBlockTime()-anchorTime;
    const int64_t heightDiff=prev->nHeight-anchor->nHeight;
    arith_uint256 ref; ref.SetCompact(anchor->nBits);
    return CalculateVALDRASERT(ref,params.nPowTargetSpacing,timeDiff,heightDiff,powLimit,params.nDAAHalfLife).GetCompact();
}
