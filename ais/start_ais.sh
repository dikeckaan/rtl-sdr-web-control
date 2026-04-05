#!/bin/bash
sleep 2
echo "Starting AIS receiver on 161.975/162.025 MHz"
exec rtl_ais -n -p 0 -g 40 -S 12 2>&1
