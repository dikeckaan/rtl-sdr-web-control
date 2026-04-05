#!/bin/bash
# Run grgsm_scanner and output results
# Usage: start_scan.sh [band]
# band: GSM900, DCS1800, or ALL

BAND="${1:-ALL}"
OUTFILE="/tmp/gsm_scan_$(date +%Y%m%d_%H%M%S).json"

echo "Starting GSM scan for band: $BAND"

if [ "$BAND" = "ALL" ]; then
    grgsm_scanner -v 2>&1 | tee /tmp/gsm_raw.txt
elif [ "$BAND" = "GSM900" ]; then
    grgsm_scanner -b GSM900 -v 2>&1 | tee /tmp/gsm_raw.txt
elif [ "$BAND" = "DCS1800" ]; then
    grgsm_scanner -b DCS1800 -v 2>&1 | tee /tmp/gsm_raw.txt
fi

echo "Scan complete"
