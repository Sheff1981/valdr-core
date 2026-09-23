#!/usr/bin/env python3
import hashlib, struct
TIMESTAMP=b"VALDR TESTNET Genesis 23/Sep/2026 - Verify before trust"
OP_RETURN_DATA=b"VALDR TESTNET: no premine; public verification network"
TIME=1790125200; BITS=0x1e0fffff; NONCE=147425; VERSION=1; REWARD=50*100_000_000
EXPECTED_MERKLE="c7742c1db54c04b717e2e48b4c68e50a156dd98e59d685f97a6b06dfd4fd819a"
EXPECTED_HASH="00000cae1bd0c4bf8439e00052222799079075f09b3d53647a2fc43ac22aa97c"
def dsha(b): return hashlib.sha256(hashlib.sha256(b).digest()).digest()
def vi(n):
    if n < 0xfd: return bytes([n])
    if n <= 0xffff: return b'\xfd'+struct.pack('<H',n)
    if n <= 0xffffffff: return b'\xfe'+struct.pack('<I',n)
    return b'\xff'+struct.pack('<Q',n)
def scriptnum(n):
    if n == 0: return b''
    neg=n<0; n=abs(n); out=bytearray()
    while n: out.append(n&255); n >>= 8
    if out[-1]&0x80: out.append(0x80 if neg else 0)
    elif neg: out[-1]|=0x80
    return bytes(out)
def push(b):
    if len(b)>=0x4c: raise ValueError('push too large')
    return bytes([len(b)])+b
def target(bits):
    exp=bits>>24; mant=bits&0x007fffff
    return mant>>(8*(3-exp)) if exp<=3 else mant<<(8*(exp-3))
def build():
    ss=push(scriptnum(486604799))+push(scriptnum(4))+push(TIMESTAMP)
    spk=b'\x6a'+push(OP_RETURN_DATA)
    tx=struct.pack('<I',1)+vi(1)+b'\0'*32+struct.pack('<I',0xffffffff)+vi(len(ss))+ss+struct.pack('<I',0xffffffff)+vi(1)+struct.pack('<Q',REWARD)+vi(len(spk))+spk+struct.pack('<I',0)
    mr=dsha(tx)
    hdr=struct.pack('<I',VERSION)+b'\0'*32+mr+struct.pack('<III',TIME,BITS,NONCE)
    bh=dsha(hdr)
    return mr[::-1].hex(),bh[::-1].hex(),int.from_bytes(bh,'little')<=target(BITS)
if __name__=='__main__':
    m,h,p=build(); print('VALDR TESTNET genesis'); print('merkle:',m); print('hash:',h); print('PoW valid:',p)
    if m!=EXPECTED_MERKLE or h!=EXPECTED_HASH or not p: raise SystemExit('GENESIS VERIFICATION FAILED')
    print('status: OK')
