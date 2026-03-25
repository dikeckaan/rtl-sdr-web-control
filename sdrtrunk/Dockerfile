FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99

# Java + SDRTrunk dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    openjdk-17-jre wget unzip \
    librtlsdr-dev libusb-1.0-0-dev rtl-sdr \
    xvfb x11vnc websockify novnc \
    supervisor dbus-x11 \
    libpulse0 pulseaudio \
    libgl1-mesa-dri libgl1-mesa-glx libegl1 \
    libfontconfig1 libxrender1 libxtst6 libxi6 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Download SDRTrunk
RUN wget -q "https://github.com/DSheirer/sdrtrunk/releases/download/v0.6.1/sdr-trunk-linux-aarch64-v0.6.1.zip" \
      -O /tmp/sdrtrunk.zip 2>/dev/null || \
    wget -q "https://github.com/DSheirer/sdrtrunk/releases/download/v0.6.0/sdr-trunk-linux-aarch64-v0.6.0.zip" \
      -O /tmp/sdrtrunk.zip 2>/dev/null || \
    echo "SDRTrunk ARM64 release not found - will need manual install" && \
    ([ -f /tmp/sdrtrunk.zip ] && unzip -o /tmp/sdrtrunk.zip -d /opt/sdrtrunk && rm /tmp/sdrtrunk.zip || true)

# PulseAudio config (copied from sdrpp)
RUN mkdir -p /root/.config/pulse /etc/pulse/default.pa.d && \
    echo "load-module module-null-sink sink_name=virtual_sink sink_properties=device.description=SDR_Audio rate=48000" \
    > /etc/pulse/default.pa.d/virtual-sink.pa && \
    echo "default-sink = virtual_sink" >> /etc/pulse/client.conf && \
    sed -i 's/; default-sample-rate = 44100/default-sample-rate = 48000/' /etc/pulse/daemon.conf

RUN ln -sf /usr/share/novnc/vnc.html /usr/share/novnc/index.html

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY start-sdrtrunk.sh /opt/start-sdrtrunk.sh
RUN chmod +x /opt/start-sdrtrunk.sh

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
