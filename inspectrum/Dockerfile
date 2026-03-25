FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99

RUN apt-get update && apt-get install -y --no-install-recommends \
    inspectrum \
    rtl-sdr librtlsdr-dev \
    xvfb x11vnc websockify novnc \
    supervisor dbus-x11 \
    libpulse0 pulseaudio \
    libgl1-mesa-dri libgl1-mesa-glx libegl1 \
    && rm -rf /var/lib/apt/lists/*

# PulseAudio config (copied from sdrpp)
RUN mkdir -p /root/.config/pulse /etc/pulse/default.pa.d && \
    echo "load-module module-null-sink sink_name=virtual_sink sink_properties=device.description=SDR_Audio rate=48000" \
    > /etc/pulse/default.pa.d/virtual-sink.pa && \
    echo "default-sink = virtual_sink" >> /etc/pulse/client.conf && \
    sed -i 's/; default-sample-rate = 44100/default-sample-rate = 48000/' /etc/pulse/daemon.conf

RUN ln -sf /usr/share/novnc/vnc.html /usr/share/novnc/index.html
RUN mkdir -p /root/recordings

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
