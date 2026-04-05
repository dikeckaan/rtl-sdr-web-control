#!/bin/bash
FREQ="${FM_FREQ:-93.0M}"
if [ -f /tmp/current_freq ]; then
    FREQ=$(cat /tmp/current_freq)
fi

echo "Starting FM Radio on frequency: $FREQ"

trap 'kill 0' EXIT

(while true; do
    FREQ=$(cat /tmp/current_freq 2>/dev/null || echo "${FM_FREQ:-93.0M}")
    echo "Tuning to: $FREQ" >&2
    rtl_fm -M wbfm -f "$FREQ" -s 200000 - &
    echo $! > /tmp/rtl_fm.pid
    wait $!
    sleep 0.2
done) | ffmpeg -hide_banner -loglevel warning \
    -f s16le -ar 200000 -ac 1 -i pipe:0 \
    -ar 48000 \
    -codec:a libmp3lame -b:a 192k \
    -f mp3 -content_type audio/mpeg \
    icecast://source:hackme@127.0.0.1:8000/fmradio &

wait
