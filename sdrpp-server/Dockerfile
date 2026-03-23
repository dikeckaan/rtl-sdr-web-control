FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb x11vnc novnc websockify supervisor nginx \
    librtlsdr-dev libusb-1.0-0-dev libfftw3-dev \
    libglfw3 libglfw3-dev libvolk2-dev libzstd-dev \
    libhackrf-dev \
    git cmake make g++ pkg-config ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build SDR++
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
        -DOPT_BUILD_AUDIO_SINK=OFF \
        -DOPT_BUILD_PORTAUDIO_SINK=OFF \
        -DOPT_BUILD_NEW_PORTAUDIO_SINK=OFF \
        -DOPT_BUILD_M17_DECODER=OFF \
        -DOPT_BUILD_USRP_SOURCE=OFF \
        -DOPT_BUILD_SOAPY_SOURCE=OFF \
        -DOPT_BUILD_NETWORK_SINK=ON \
    && make -j$(nproc) && make install && \
    rm -rf /tmp/sdrpp

# noVNC setup
RUN ln -sf /usr/share/novnc/vnc.html /usr/share/novnc/index.html
COPY inject_audio.py /tmp/inject_audio.py
RUN python3 /tmp/inject_audio.py

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
