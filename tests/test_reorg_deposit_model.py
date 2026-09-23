#!/usr/bin/env python3
from pathlib import Path
import importlib.util
p=Path(__file__).resolve().parents[1]/'tools'/'reorg_deposit_model.py'
spec=importlib.util.spec_from_file_location('rdm',p); m=importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
d=m.Deposit('00'*32,1)
d.include(500,'aa'*32,500,12); assert (d.confirmations,d.credited)==(1,False)
d.advance(511,12); assert (d.confirmations,d.credited)==(12,True)
d.reorg_out(); assert d.confirmations==0 and d.credited is False
d.include(520,'bb'*32,531,12); assert d.credited is True
print('test_reorg_deposit_model: PASS')
