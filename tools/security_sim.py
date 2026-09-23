#!/usr/bin/env python3
"""Deterministic engineering models. Not a prediction of real attack probability."""
import math, importlib.util, pathlib
ROOT=pathlib.Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('asert',ROOT/'tools/asert_reference.py'); asert=importlib.util.module_from_spec(spec); spec.loader.exec_module(asert)

def attacker_catchup_probability(q,z):
    if not 0 <= q <= 1: raise ValueError
    if q >= 0.5: return 1.0
    p=1-q; lam=z*q/p; s=0.0
    for k in range(z+1):
        poisson=math.exp(-lam)*(lam**k)/math.factorial(k)
        s += poisson*(1-(q/p)**(z-k))
    return 1-s

def daa_shock(blocks,solve_time):
    ref=asert.POW_LIMIT>>8
    height_diff=blocks
    time_diff=600 + blocks*solve_time
    target=asert.asert_target(ref,time_diff,height_diff)
    return target/ref

def main():
    print('ASERT target ratio after deterministic hash-rate shocks (lower target = harder):')
    for n in (10,50,100,250,500):
        print(f' blocks={n:3d} 10x-hash(~60s): {daa_shock(n,60):.6f}x target   0.1x-hash(~6000s): {daa_shock(n,6000):.6f}x target')
    print('\nIllustrative Nakamoto catch-up model only (NOT sufficient for VALDR launch security):')
    for q in (0.10,0.20,0.30,0.40):
        vals=[attacker_catchup_probability(q,z) for z in (1,3,6,12,24)]
        print(f'q={q:.0%}: '+', '.join(f'z{z}={v:.3e}' for z,v in zip((1,3,6,12,24),vals)))
if __name__=='__main__': main()
