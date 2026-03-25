FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip python3-venv python3-dev \
    g++ cmake make pkg-config \
    librtlsdr-dev libusb-1.0-0-dev rtl-sdr \
    python3-pyqt5 python3-numpy python3-psutil \
    cython3 \
    xvfb x11vnc websockify novnc \
    supervisor dbus-x11 \
    libpulse0 pulseaudio \
    libgl1-mesa-dri libegl1 \
    ca-certificates git \
    && rm -rf /var/lib/apt/lists/*

# Build URH from git with regenerated Cython extensions
RUN python3 -m venv --system-site-packages /opt/venv && \
    /opt/venv/bin/pip install cython setuptools pyqtgraph && \
    git clone --depth 1 https://github.com/jopohl/urh.git /tmp/urh && \
    cd /tmp/urh && \
    /opt/venv/bin/python3 setup.py build_ext --inplace 2>/dev/null; \
    /opt/venv/bin/pip install . --no-build-isolation 2>/dev/null || \
    /opt/venv/bin/pip install /tmp/urh 2>/dev/null || \
    echo "URH install attempted" && \
    rm -rf /tmp/urh

# PulseAudio config (copied from sdrpp)
RUN mkdir -p /root/.config/pulse /etc/pulse/default.pa.d && \
    echo "load-module module-null-sink sink_name=virtual_sink sink_properties=device.description=SDR_Audio rate=48000" \
    > /etc/pulse/default.pa.d/virtual-sink.pa && \
    echo "default-sink = virtual_sink" >> /etc/pulse/client.conf && \
    sed -i 's/; default-sample-rate = 44100/default-sample-rate = 48000/' /etc/pulse/daemon.conf

RUN ln -sf /usr/share/novnc/vnc.html /usr/share/novnc/index.html

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
