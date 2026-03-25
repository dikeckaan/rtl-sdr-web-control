FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99

# Build dependencies + runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake g++ make pkg-config \
    librtlsdr-dev libsoapysdr-dev soapysdr-module-rtlsdr \
    libwxgtk3.2-dev libglew-dev libpulse-dev libliquid-dev \
    freeglut3-dev \
    xvfb x11vnc websockify novnc nginx \
    supervisor dbus-x11 \
    pulseaudio pulseaudio-utils \
    python3 python3-websockets \
    libgl1-mesa-dri libgl1-mesa-glx libegl1 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build CubicSDR
RUN git clone --depth 1 https://github.com/cjcliffe/CubicSDR.git /tmp/CubicSDR && \
    mkdir /tmp/CubicSDR/build && cd /tmp/CubicSDR/build && \
    cmake .. -DCMAKE_BUILD_TYPE=Release -DUSE_HAMLIB=OFF && \
    make -j$(nproc) && make install && \
    rm -rf /tmp/CubicSDR

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
