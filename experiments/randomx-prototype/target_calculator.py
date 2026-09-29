#!/usr/bin/env python3
"""VALDR Mainnet initial target calculator.

Consensus note:
    expected_work ~= network_hashrate_hps * target_block_seconds
    target = floor(2^256 / expected_work) - 1

This is an engineering calibration helper, not consensus code.
"""

from __future__ import annotations

import argparse

TWO256 = 1 << 256


def target_for(hashrate_hps: int, block_seconds: int) -> tuple[int, int]:
    if hashrate_hps <= 0:
        raise ValueError("hashrate_hps must be > 0")
    if block_seconds <= 0:
        raise ValueError("block_seconds must be > 0")
    expected_work = hashrate_hps * block_seconds
    target = TWO256 // expected_work - 1
    return expected_work, target


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--block-seconds", type=int, default=600)
    parser.add_argument(
        "--hashrates",
        type=int,
        nargs="+",
        default=[50, 100, 250, 500, 1000, 5000, 10000, 50000, 100000],
    )
    args = parser.parse_args()

    print("hashrate_hps,expected_work,target_hex")
    for h in args.hashrates:
        work, target = target_for(h, args.block_seconds)
        print(f"{h},{work},{target:064x}")


if __name__ == "__main__":
    main()
