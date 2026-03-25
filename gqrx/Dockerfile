FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99

# Install GQRX + noVNC + audio stack
RUN apt-get update && apt-get install -y --no-install-recommends \
    gqrx-sdr rtl-sdr librtlsdr-dev \
    xvfb x11vnc websockify \
    novnc nginx \
    supervisor dbus-x11 \
    libgl1-mesa-dri libgl1-mesa-glx libegl1 \
    pulseaudio pulseaudio-utils \
    python3 python3-pip python3-websockets \
    && rm -rf /var/lib/apt/lists/*

# PulseAudio config (copied from sdrpp)
RUN mkdir -p /root/.config/pulse /etc/pulse/default.pa.d && \
    echo "load-module module-null-sink sink_name=virtual_sink sink_properties=device.description=SDR_Audio rate=48000" \
    > /etc/pulse/default.pa.d/virtual-sink.pa && \
    echo "default-sink = virtual_sink" >> /etc/pulse/client.conf && \
    sed -i 's/; default-sample-rate = 44100/default-sample-rate = 48000/' /etc/pulse/daemon.conf

# noVNC + audio widget injection
RUN ln -sf /usr/share/novnc/vnc.html /usr/share/novnc/index.html
COPY inject_audio.py /tmp/inject_audio.py
RUN python3 /tmp/inject_audio.py

# Audio bridge
COPY audio_bridge.py /opt/audio_bridge.py

# nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Supervisord config
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# GQRX config
RUN mkdir -p /root/.config/gqrx
COPY default.conf /root/.config/gqrx/default.conf

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
