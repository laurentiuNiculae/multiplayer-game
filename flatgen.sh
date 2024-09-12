#!/bin/bash

# Go
~/opt/flatbuffers/bin/flatc -o ./pkg/types --go ./pkg/types/flatbuffers/simple.fbs

# TS
~/opt/flatbuffers/bin/flatc -o ./client --filename-ext "mts" --ts ./pkg/types/flatbuffers/simple.fbs
sed -i "s/import \* as flatbuffers from 'flatbuffers';/import \* as flatbuffers from '..\/..\/flatbuffers\/flatbuffers.js';/" ./client/flatgen/game/*

tsc --project . 
