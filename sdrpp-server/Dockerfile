FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies (build + runtime)
RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb x11vnc novnc websockify supervisor nginx \
    pulseaudio pulseaudio-utils libpulse-dev librtaudio-dev \
    python3 python3-pip python3-websockets \
    librtlsdr-dev libusb-1.0-0-dev libfftw3-dev \
    libglfw3 libglfw3-dev libvolk2-dev libzstd-dev \
    libhackrf-dev \
    git cmake make g++ pkg-config ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build SDR++ with PulseAudio audio sink
RUN git clone --depth 1 https://github.com/AlexandreRouma/SDRPlusPlus.git /tmp/sdrpp && \
    cd /tmp/sdrpp && mkdir build && cd build && \
    cmake .. \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX=/usr \
        -DOPT_BUILD_RTL_SDR_SOURCE=ON \
        -DOPT_BUILD_HACKRF_SOURCE=ON \
        -DOPT_BUILD_AIRSPY_SOURCE=OFF \
        -DOPT_BUILD_AIRSPYHF_SOURCE=OFF \
        -DOPT_BUILD_BLADERF_SOURCE=OFF \
        -DOPT_BUILD_SDRPLAY_SOURCE=OFF \
        -DOPT_BUILD_PLUTOSDR_SOURCE=OFF \
        -DOPT_BUILD_LIMESDR_SOURCE=OFF \
        -DOPT_BUILD_SPYSERVER_SOURCE=OFF \
        -DOPT_BUILD_RFSPACE_SOURCE=OFF \
        -DOPT_BUILD_AUDIO_SOURCE=OFF \
        -DOPT_BUILD_AUDIO_SINK=ON \
        -DOPT_BUILD_PORTAUDIO_SINK=OFF \
        -DOPT_BUILD_NEW_PORTAUDIO_SINK=OFF \
        -DOPT_BUILD_M17_DECODER=OFF \
        -DOPT_BUILD_USRP_SOURCE=OFF \
        -DOPT_BUILD_SOAPY_SOURCE=OFF \
        -DOPT_BUILD_NETWORK_SINK=OFF \
    && make -j$(nproc) && make install && \
    rm -rf /tmp/sdrpp

# PulseAudio: virtual sink config
RUN mkdir -p /root/.config/pulse /etc/pulse/default.pa.d && \
    echo "load-module module-null-sink sink_name=virtual_sink sink_properties=device.description=SDR_Audio rate=48000" \
    > /etc/pulse/default.pa.d/virtual-sink.pa && \
    echo "default-sink = virtual_sink" >> /etc/pulse/client.conf && \
    sed -i 's/; default-sample-rate = 44100/default-sample-rate = 48000/' /etc/pulse/daemon.conf

# noVNC setup + audio injection
RUN ln -sf /usr/share/novnc/vnc.html /usr/share/novnc/index.html
COPY inject_audio.py /tmp/inject_audio.py
RUN python3 /tmp/inject_audio.py

# Audio bridge
COPY audio_bridge.py /opt/audio_bridge.py

# Nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Supervisord config
RUN mkdir -p /var/log/supervisor
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Default SDR++ config
RUN mkdir -p /root/.config/sdrpp
COPY config.json /root/.config/sdrpp/config.json

# Display settings
ENV DISPLAY=:99
ENV SCREEN_WIDTH=1280
ENV SCREEN_HEIGHT=720
EXPOSE 8090

CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
