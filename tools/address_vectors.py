#!/usr/bin/env python3
import hashlib
ALPH='123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz'
def b58(b):
    n=int.from_bytes(b,'big'); s=''
    while n: n,r=divmod(n,58); s=ALPH[r]+s
    pad=len(b)-len(b.lstrip(b'\0')); return '1'*pad+(s or '')
def b58check(ver,payload):
    raw=bytes([ver])+payload; chk=hashlib.sha256(hashlib.sha256(raw).digest()).digest()[:4]; return b58(raw+chk)
if __name__=='__main__':
    payload=bytes(20)
    print('DEVNET P2PKH vector:',b58check(70,payload))
    print('TESTNET P2PKH vector:',b58check(65,payload))
    assert b58check(70,payload).startswith('V')
    assert b58check(65,payload).startswith('T')
    print('status: OK')
