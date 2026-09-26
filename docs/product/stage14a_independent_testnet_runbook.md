# VALDR Stage 14A independent Testnet runbook

**Master baseline:** `docs/VALDR_Master_TZ_v0.2.11.md`  
**Network:** `testnet2` / `valdr-testnet-2`  
**Status:** operational procedure only; Stage 14A is not complete until real independent-machine evidence exists.

## Purpose

Run the first real distributed VALDR Testnet2 soak on at least three independently launched computers/clients for at least 24 hours.

This procedure does not change consensus or network constants. It turns the automated/local preflight evidence into a reproducible real-network test.

## Required topology

Use at least three independently launched systems:

- **A — bootstrap/public node:** reachable from the Internet on TCP/17333.
- **B — independent node/miner:** connects initially to A.
- **C — independent node/client:** connects initially to A and must learn B through peer exchange.

The systems may be ordinary user PCs or volunteer public nodes. They must not be three containers on one CI host.

RPC stays localhost-only on TCP/17332.

## Before starting

On all three systems use the exact same accepted source/build commit.

Verify:

```bash
valdrd version
valdr-miner version
valdr-cli version
```

Use separate data directories and wallets on every machine. Never copy a wallet/private key between operators merely to run the soak.

## Start node A

Node A must have a real externally reachable address. Replace `PUBLIC_A:17333` with its actual routable address.

```bash
valdrd start \
  --network testnet2 \
  --data ./valdr-testnet2-a \
  --node-id stage14a-node-a \
  --p2p-host 0.0.0.0 \
  --advertise-address PUBLIC_A:17333 \
  --rpc-host 127.0.0.1
```

The operator/router/firewall must allow inbound TCP/17333.

Do not expose RPC/17332 to the Internet.

## Start node B

```bash
valdrd start \
  --network testnet2 \
  --data ./valdr-testnet2-b \
  --node-id stage14a-node-b \
  --p2p-host 0.0.0.0 \
  --advertise-address PUBLIC_B:17333 \
  --rpc-host 127.0.0.1 \
  --seed PUBLIC_A:17333
```

If B cannot accept inbound traffic, run it outbound-only and omit the advertised public address. At least one real initial reachable bootstrap route is still mandatory.

## Start node C

```bash
valdrd start \
  --network testnet2 \
  --data ./valdr-testnet2-c \
  --node-id stage14a-node-c \
  --p2p-host 0.0.0.0 \
  --advertise-address PUBLIC_C:17333 \
  --rpc-host 127.0.0.1 \
  --seed PUBLIC_A:17333
```

## Confirm the network identity

On every machine:

```bash
valdrd status --node http://127.0.0.1:17332
valdr-cli peers --node http://127.0.0.1:17332
valdr-cli mining info --node http://127.0.0.1:17332
```

Required identity:

- network: `testnet2`
- Chain ID: `valdr-testnet-2`
- P2P port: 17333
- RPC remains localhost-only
- all nodes eventually agree on height, tip hash and cumulative chainwork.

## Start evidence collection

On each system run:

```bash
bash scripts/stage14a-soak-observe.sh \
  --node http://127.0.0.1:17332 \
  --data ./valdr-testnet2-a \
  --duration-seconds 86400 \
  --interval-seconds 60 \
  --output stage14a-node-a.jsonl
```

Use the matching local data directory and a unique output filename on B and C.

The observer records Testnet2 identity, height, tip, cumulative chainwork, target/retarget information, peers, mempool, mining information and local data-directory size.

## Mining during the soak

At least two independent operators should mine Testnet blocks during the window using their own reward addresses and their own local node RPC.

Example:

```bash
valdr-miner start \
  --node http://127.0.0.1:17332 \
  --reward-address VDR1... \
  --blocks 1
```

Do not share private keys. The miner receives only a reward address and talks to the local node over RPC.

## Required restart/bootstrap-loss exercise

After peer exchange is confirmed:

1. Verify C has learned B in `valdr-cli peers`.
2. Stop A, the original bootstrap route.
3. Keep B running.
4. Restart C.
5. C must reconnect from its persisted learned-peer cache without A.
6. Mine a block on B.
7. C must converge to B's height/tip/chainwork.
8. Restart A.
9. A must catch up without deleting or repairing its database.

Record the approximate UTC times of each action.

## What must be watched for >=24 hours

Record any:

- height/tip/chainwork disagreement;
- unexpected reorg or orphan pattern;
- difficulty/target transition problem;
- peer-count collapse or inability to reconnect;
- bootstrap failure;
- stuck mempool transaction;
- database error or need for manual repair;
- node/miner crash;
- Desktop synchronization problem;
- installer/package failure.

No issue should be hidden by deleting node data.

## Stage 14A exit evidence

Stage 14A can be marked complete only when all of the following are true:

- at least three independently launched nodes/clients participated;
- at least one real cold-client bootstrap route was reachable;
- peer exchange learned additional peers;
- peer cache survived restart;
- losing the original bootstrap route did not break connected consensus;
- the soak ran for at least 24 hours;
- no consensus split occurred;
- no manual database repair was required;
- any blockers found were fixed;
- full CI was rerun green after the last blocker fix.

Store the three observer JSONL files and a short UTC event log with the accepted source commit as Stage 14A evidence.

Local Docker/CI tests are preflight evidence only and must never be substituted for this independent-machine soak.
