#!/bin/bash
# Called by app.py to record ISS pass audio
# Usage: start_record.sh <duration_seconds> <output_filename>

DURATION=${1:-300}
OUTFILE=${2:-/opt/iss/captures/iss_$(date +%Y%m%d_%H%M%S).wav}

echo "Recording ISS pass for ${DURATION}s to ${OUTFILE}"

timeout "$DURATION" rtl_fm -f 145.800M -M fm -s 48000 -g 40 - | \
    sox -t raw -r 48000 -e signed -b 16 -c 1 - "$OUTFILE"

echo "Recording complete: $OUTFILE"
