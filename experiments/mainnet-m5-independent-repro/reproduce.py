#!/usr/bin/env python3
from fractions import Fraction

TWO256 = 1 << 256

REFERENCE_RATE = Fraction(418723, 1000)
TARGET_SECONDS = 600
SAFETY = Fraction(1, 1)
POWLIMIT_FACTOR = 4

EXPECTED_INITIAL_WORK = 251233
EXPECTED_INITIAL_TARGET = "000042c78dcd2f09f6673e99417b7b7d0dbb0ddcf732f964dc73fce1c6b50768"
EXPECTED_POWLIMIT_WORK = 62808
EXPECTED_POWLIMIT_TARGET = "00010b1e7ce2d1034b183a2be3b26504779a86ab0ed6197fb90be6d3c07b200c"

def floor_fraction(x: Fraction) -> int:
    return x.numerator // x.denominator

def target_for_work(work: int) -> int:
    if work <= 0:
        raise ValueError("work must be positive")
    return TWO256 // work - 1

def work_for_target(target: int) -> int:
    if target <= 0 or target >= TWO256:
        raise ValueError("target outside uint256 positive range")
    return TWO256 // (target + 1)

def main() -> None:
    initial_work = floor_fraction(REFERENCE_RATE * TARGET_SECONDS * SAFETY)
    initial_target = target_for_work(initial_work)

    powlimit_work = initial_work // POWLIMIT_FACTOR
    if powlimit_work <= 0:
        raise AssertionError("PowLimit work became zero")
    powlimit_target = target_for_work(powlimit_work)

    got_initial_target = f"{initial_target:064x}"
    got_powlimit_target = f"{powlimit_target:064x}"

    assert initial_work == EXPECTED_INITIAL_WORK, (initial_work, EXPECTED_INITIAL_WORK)
    assert got_initial_target == EXPECTED_INITIAL_TARGET, got_initial_target
    assert powlimit_work == EXPECTED_POWLIMIT_WORK, (powlimit_work, EXPECTED_POWLIMIT_WORK)
    assert got_powlimit_target == EXPECTED_POWLIMIT_TARGET, got_powlimit_target

    # Round-trip work semantics independently check floor(2^256/(target+1)).
    assert work_for_target(initial_target) == initial_work
    assert work_for_target(powlimit_target) == powlimit_work

    print("VALDR_M5_INDEPENDENT_REPRO_V1")
    print(f"reference_hashrate_hs={float(REFERENCE_RATE):.3f}")
    print(f"initial_work={initial_work}")
    print(f"initial_target={got_initial_target}")
    print(f"powlimit_factor={POWLIMIT_FACTOR}")
    print(f"powlimit_work={powlimit_work}")
    print(f"powlimit_target={got_powlimit_target}")
    print("roundtrip_initial_work=PASS")
    print("roundtrip_powlimit_work=PASS")
    print("status=INDEPENDENT_REPRO_PASS")

if __name__ == "__main__":
    main()
