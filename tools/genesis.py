#!/usr/bin/env python3
import hashlib, struct

TIMESTAMP = b"VALDR Genesis 23/Sep/2026 - Strength Builds Freedom"
OP_RETURN_DATA = b"VALDR: no premine; sovereign peer-to-peer money"
TIME = 1790121600
BITS = 0x207FFFFF
NONCE = 0
VERSION = 1
REWARD = 50 * 100_000_000
EXPECTED_MERKLE = "77d9d01946c415a41f52f6da59c63ca2e6c2e5466b59680b8fb60a60f8420737"
EXPECTED_HASH = "3ee8cf33862988e8575fb0fb1b6b207530f945b51c398c44be7c6c473aa4260b"

def dsha(b):
    return hashlib.sha256(hashlib.sha256(b).digest()).digest()

def vi(n):
    if n < 0xfd: return bytes([n])
    if n <= 0xffff: return b'\xfd' + struct.pack('<H', n)
    if n <= 0xffffffff: return b'\xfe' + struct.pack('<I', n)
    return b'\xff' + struct.pack('<Q', n)

def scriptnum(n):
    if n == 0: return b''
    neg = n < 0
    n = abs(n)
    out = bytearray()
    while n:
        out.append(n & 0xff); n >>= 8
    if out[-1] & 0x80: out.append(0x80 if neg else 0)
    elif neg: out[-1] |= 0x80
    return bytes(out)

def push(b):
    if len(b) >= 0x4c: raise ValueError("push too large for this genesis builder")
    return bytes([len(b)]) + b

def target_from_compact(bits):
    exp = bits >> 24
    mant = bits & 0x007fffff
    return (mant >> (8 * (3-exp))) if exp <= 3 else (mant << (8 * (exp-3)))

def build():
    script_sig = push(scriptnum(486604799)) + push(scriptnum(4)) + push(TIMESTAMP)
    script_pubkey = b'\x6a' + push(OP_RETURN_DATA)
    tx = (struct.pack('<I', 1) + vi(1) + b'\x00'*32 + struct.pack('<I', 0xffffffff)
          + vi(len(script_sig)) + script_sig + struct.pack('<I', 0xffffffff)
          + vi(1) + struct.pack('<Q', REWARD) + vi(len(script_pubkey)) + script_pubkey
          + struct.pack('<I', 0))
    merkle_raw = dsha(tx)
    header = (struct.pack('<I', VERSION) + b'\x00'*32 + merkle_raw
              + struct.pack('<III', TIME, BITS, NONCE))
    block_raw = dsha(header)
    merkle = merkle_raw[::-1].hex()
    block_hash = block_raw[::-1].hex()
    target = target_from_compact(BITS)
    valid_pow = int.from_bytes(block_raw, 'little') <= target
    return merkle, block_hash, valid_pow

if __name__ == '__main__':
    merkle, block_hash, valid_pow = build()
    print("VALDR DEVNET genesis")
    print("merkle    :", merkle)
    print("hash      :", block_hash)
    print("PoW valid :", valid_pow)
    if merkle != EXPECTED_MERKLE or block_hash != EXPECTED_HASH or not valid_pow:
        raise SystemExit("GENESIS VERIFICATION FAILED")
    print("status    : OK")
