# VALDR Mainnet mining blob / RandomX seed v1 candidate

Status: **prototype / CI candidate / not frozen**

## Mining blob

To minimize consensus change, Mainnet does **not** introduce another header serialization merely for RandomX.

Candidate:

```
block_id = SHA256(existing canonical v2 header)
pow_blob = "VALDR/RANDOMX/POW/V1\\x00" || existing canonical v2 header
pow_hash = RandomX(seed_key, pow_blob)
valid iff uint256(pow_hash) <= target
```

This preserves:

- existing canonical header layout;
- existing exact 256-bit target field;
- existing SHA-256 block identifiers;
- existing chainwork formula derived from target.

RandomX is only the expensive PoW function.

## Seed schedule candidate

- epoch: 2,048 blocks;
- lag: 64 blocks;
- heights 0..63 use fixed epoch-0 seed;
- otherwise:
  `seed_height = floor((height - 64) / 2048) * 2048`;
- `seed_key = block_id(seed_height)`;
- epoch-0 fixed seed:
  `SHA256("VALDR/RandomX/Mainnet/v1/epoch-0")`
  = `0fca8879c33460a767246a02b156794ec9958113261e04db0b3a9e66fe6febb9`.

At 10-minute blocks:

- epoch ≈ 14.22 days;
- lag ≈ 10.67 hours.

## Golden prototype vector

Sample header:

- version 2;
- height 262800;
- previous hash: 32 zero bytes encoded by existing v2 header rules;
- Merkle root: 32 bytes of 0x20 represented by existing v2 header rules;
- timestamp 1800000000;
- target: 32 bytes of 0x40;
- nonce: 0x0102030405060708;
- Chain ID: valdr-mainnet-1;
- ExtraData: empty.

Expected:

- canonical header bytes: 235;
- mining blob bytes: 256;
- SHA256(header): `9d7838b61a5e10c252d690149eec1a2fe80e32617fd29ebacaaf6d9903144a8a`;
- SHA256(mining blob): `474a5ca610d1573c3f2b0cd0d634448ee0a9a909af40ea0773b9b780ccbf29bf`.

The SHA-256 mining-blob digest is only a serialization test oracle, not the Mainnet PoW hash.

## Why this changes the previous draft

The earlier draft described a new fixed binary mining blob with its own field list. This prototype instead reuses the already canonical v2 header serialization and adds a domain prefix.

Benefits:

- smaller consensus surface;
- fewer duplicate encoders;
- easier golden-vector testing;
- block ID and PoW bind exactly the same header fields.

Risk:

- the current v2 header serializes previous/Merkle hashes as length-prefixed hexadecimal strings rather than raw 32-byte values; this is less compact but already consensus-tested. Changing that purely for compactness is not justified.

If accepted, the Mainnet Master-TZ must replace the earlier custom-field mining-blob proposal with this rule.


## Cross-platform RandomX golden hash

Workflow `VALDR RandomX Prototype #11` passed on Windows, Linux and macOS.

All three platforms produced the same candidate Mainnet RandomX hash:

`85c60db39d4d13caeeb2e3841da2ef26d2ce64c9659a48f843811ba6ea8560f2`

This is now the prototype golden RandomX vector for the current candidate mining blob + epoch-0 seed.

A pure-Go target-comparison validator is included in this experiment so RandomX execution and 256-bit consensus target comparison are tested as separate failure domains. Production Mainnet consensus is still disabled.
