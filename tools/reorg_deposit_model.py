#!/usr/bin/env python3
"""Deterministic exchange deposit/reorg state model for VALDR QA."""
from dataclasses import dataclass

@dataclass
class Deposit:
    txid: str
    vout: int
    included_height: int | None = None
    included_hash: str | None = None
    confirmations: int = 0
    credited: bool = False

    def include(self, height: int, block_hash: str, tip_height: int, threshold: int) -> None:
        self.included_height = height
        self.included_hash = block_hash
        self.confirmations = max(0, tip_height - height + 1)
        self.credited = self.confirmations >= threshold

    def advance(self, tip_height: int, threshold: int) -> None:
        if self.included_height is None:
            self.confirmations = 0
            self.credited = False
            return
        self.confirmations = max(0, tip_height - self.included_height + 1)
        self.credited = self.confirmations >= threshold

    def reorg_out(self) -> None:
        self.included_height = None
        self.included_hash = None
        self.confirmations = 0
        self.credited = False

def self_test() -> None:
    d = Deposit("aa" * 32, 0)
    d.include(100, "11" * 32, 100, 6)
    assert d.confirmations == 1 and not d.credited
    d.advance(105, 6)
    assert d.confirmations == 6 and d.credited
    d.reorg_out()
    assert d.confirmations == 0 and not d.credited and d.included_hash is None
    d.include(108, "22" * 32, 108, 6)
    d.advance(113, 6)
    assert d.credited
    print("deposit/reorg model: PASS")

if __name__ == "__main__":
    self_test()
