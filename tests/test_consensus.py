#!/usr/bin/env python3
import pathlib, importlib.util
ROOT=pathlib.Path(__file__).resolve().parents[1]
def load(name,path):
    s=importlib.util.spec_from_file_location(name,ROOT/path); m=importlib.util.module_from_spec(s); s.loader.exec_module(m); return m
gen=load('gen','tools/genesis_testnet.py'); sup=load('sup','tools/supply.py'); addr=load('addr','tools/address_vectors.py'); asert=load('asert','tools/asert_reference.py'); sec=load('sec','tools/security_sim.py')
def test_genesis():
    m,h,p=gen.build(); assert m==gen.EXPECTED_MERKLE and h==gen.EXPECTED_HASH and p
def test_supply_boundaries():
    assert sup.subsidy(0)==50*sup.COIN
    assert sup.subsidy(209999)==50*sup.COIN
    assert sup.subsidy(210000)==25*sup.COIN
    assert sup.subsidy(420000)==1_250_000_000
    assert sup.subsidy(64*210000)==0
    issued=sup.total_subsidy_atoms(True); spendable=sup.total_subsidy_atoms(False)
    assert issued<=21_000_000*sup.COIN
    assert issued-spendable==50*sup.COIN
def test_addresses():
    p=bytes(20); assert addr.b58check(70,p).startswith('V'); assert addr.b58check(65,p).startswith('T')
def test_asert(): asert.run_vectors()
def test_daa_shocks():
    assert sec.daa_shock(100,60) < 1
    assert sec.daa_shock(100,6000) > 1
if __name__=='__main__':
    for f in [test_genesis,test_supply_boundaries,test_addresses,test_asert,test_daa_shocks]: f(); print(f.__name__+': OK')
