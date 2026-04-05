#!/bin/bash
if [ -f /tmp/current_freq ]; then
    FREQ=$(cat /tmp/current_freq)
else
    FREQ="${HAM_FREQ:-145.500M}"
    echo "$FREQ" > /tmp/current_freq
fi

if [ -f /tmp/current_mode ]; then
    MODE=$(cat /tmp/current_mode)
else
    MODE="${HAM_MODE:-fm}"
    echo "$MODE" > /tmp/current_mode
fi

echo "Starting HAM Radio receiver: freq=$FREQ mode=$MODE"

trap 'kill 0' EXIT

# Narrowband FM: -s 24000 -r 24000 (matching rates)
# AM/SSB: -s 12000 -r 12000
if [ "$MODE" = "fm" ]; then
    SR=24000
else
    SR=12000
fi

rtl_fm -M "$MODE" -f "$FREQ" -s $SR -r $SR -l 10 -E deemp - | \
ffmpeg -hide_banner -loglevel warning \
    -f s16le -ar $SR -ac 1 -i pipe:0 \
    -af "highpass=f=200,lowpass=f=5000,volume=1.3" \
    -ar 44100 \
    -codec:a libmp3lame -b:a 96k \
    -f mp3 -content_type audio/mpeg \
    icecast://source:hackme@127.0.0.1:8000/hamradio &

wait
