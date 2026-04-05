#!/bin/bash

# Read frequency from override file or environment
if [ -f /tmp/current_freq ]; then
    FREQ=$(cat /tmp/current_freq)
else
    FREQ="${PAGER_FREQ:-153.350M}"
fi

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting pager decoder on frequency: $FREQ" | tee -a /var/log/pager/messages.log

# Ensure live log exists
touch /tmp/pager_live.log

rtl_fm -M fm -f "$FREQ" -s 22050 -g 40 - | \
    multimon-ng -t raw -a POCSAG512 -a POCSAG1200 -a POCSAG2400 -a FLEX -f alpha /dev/stdin 2>/dev/null | \
    while IFS= read -r line; do
        TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
        ENTRY="[$TIMESTAMP] $line"
        echo "$ENTRY" >> /var/log/pager/messages.log
        echo "$ENTRY" >> /tmp/pager_live.log
    done
