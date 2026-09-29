# VALDR M5 independent reproduction

This is an intentionally independent implementation of the M5 exact arithmetic.

It does **not** import VALDR Go consensus or calibration code. It uses Python standard-library arbitrary-precision integers and `fractions.Fraction`.

Inputs:
- reference all-thread RandomX rate: `418.723 H/s`;
- target interval: `600 s`;
- safety multiplier: `1/1`;
- PowLimit policy candidate: `4x easier in work terms`.

Expected exact values:
- InitialWork: `251233`;
- InitialTarget: `000042c78dcd2f09f6673e99417b7b7d0dbb0ddcf732f964dc73fce1c6b50768`;
- PowLimitWork: `62808`;
- PowLimitTarget: `00010b1e7ce2d1034b183a2be3b26504779a86ab0ed6197fb90be6d3c07b200c`.

The script also performs an independent work/target round trip using:
`work = floor(2^256 / (target + 1))`.

Passing this gate independently reproduces the exact 256-bit M5 candidate values. It does not by itself freeze Mainnet consensus; the values still require formal recording in a new Master-TZ revision after all pre-freeze alignment checks are green.
