#!/usr/bin/env python3
"""VALDR ASERT reference/vector generator. Consensus port still requires independent review."""
POW_LIMIT=(1<<236)-1
SPACING=600
HALF_LIFE=172800
def trunc_div(n,d): return (abs(n)//d) * (-1 if n<0 else 1)
def asert_target(ref_target:int,time_diff:int,height_diff:int,pow_limit:int=POW_LIMIT,spacing:int=SPACING,half_life:int=HALF_LIFE)->int:
    if not (0 < ref_target <= pow_limit): raise ValueError('ref target')
    if height_diff < 0: raise ValueError('height diff')
    exponent=trunc_div((time_diff-spacing*(height_diff+1))*65536,half_life)
    shifts=exponent>>16; frac=exponent & 0xffff
    factor=65536+((195766423245049*frac+971821376*frac*frac+5127*frac*frac*frac+(1<<47))>>48)
    nxt=ref_target*factor; shifts-=16
    if shifts<=0: nxt >>= -shifts
    else: nxt <<= shifts
    return max(1,min(nxt,pow_limit))
def compact(target:int)->int:
    size=(target.bit_length()+7)//8
    word=(target << (8*(3-size))) if size<=3 else (target >> (8*(size-3)))
    if word & 0x00800000: word >>= 8; size += 1
    return (size<<24)|(word&0x007fffff)
def run_vectors():
    ref=POW_LIMIT>>8
    steady=asert_target(ref,600,0); slow=asert_target(ref,600+HALF_LIFE,0); fast=asert_target(ref,600-HALF_LIFE,0)
    assert abs(steady-ref)/ref < 0.001
    assert 1.99 < slow/ref < 2.01
    assert 0.49 < fast/ref < 0.51
    return {'ref':hex(ref),'steady':hex(steady),'slow_2d':hex(slow),'fast_2d':hex(fast),'bits_ref':hex(compact(ref))}
if __name__=='__main__':
    import json; print(json.dumps(run_vectors(),indent=2)); print('status: OK')
