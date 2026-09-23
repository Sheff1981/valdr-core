#!/usr/bin/env python3
COIN=100_000_000; HALVING=210_000
def subsidy(height:int)->int:
    halvings=height//HALVING
    if halvings>=64: return 0
    return (50*COIN)>>halvings
def total_subsidy_atoms(include_genesis=True):
    total=0; h=0
    while True:
        s=subsidy(h)
        if not s: break
        total += s*HALVING
        h += HALVING
    if not include_genesis: total -= subsidy(0)
    return total
if __name__=='__main__':
    issued=total_subsidy_atoms(True); spendable=total_subsidy_atoms(False)
    print(f'theoretical subsidy sum: {issued/COIN:.8f} VLD')
    print(f'max spendable after unspendable genesis: {spendable/COIN:.8f} VLD')
    print(f'unspendable genesis difference: {(issued-spendable)/COIN:.8f} VLD')
