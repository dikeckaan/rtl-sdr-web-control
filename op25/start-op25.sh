#!/bin/bash
cd /opt/op25/op25/gr-op25_repeater/apps
python3 rx.py \
  --args "rtl=0" \
  --gains "lna:36" \
  -S 960000 \
  -X \
  -w \
  -W 8090 \
  -q 0 \
  -v 1 \
  2>&1
