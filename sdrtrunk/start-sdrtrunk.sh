#!/bin/bash
# Find and run SDRTrunk
if [ -f /opt/sdrtrunk/bin/sdr-trunk ]; then
    exec /opt/sdrtrunk/bin/sdr-trunk
elif [ -f /opt/sdrtrunk/sdr-trunk ]; then
    exec /opt/sdrtrunk/sdr-trunk
else
    # Find any executable
    EXEC=$(find /opt/sdrtrunk -name "sdr-trunk" -o -name "SDRTrunk" 2>/dev/null | head -1)
    if [ -n "$EXEC" ]; then
        exec "$EXEC"
    else
        echo "SDRTrunk executable not found"
        sleep infinity
    fi
fi
