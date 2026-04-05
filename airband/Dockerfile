FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# Install build dependencies and runtime packages
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    librtlsdr-dev \
    libmp3lame-dev \
    libshout3-dev \
    libconfig++-dev \
    libfftw3-dev \
    libsoapysdr-dev \
    pkg-config \
    icecast2 \
    nginx \
    supervisor \
    && rm -rf /var/lib/apt/lists/*

# Build rtl_airband from source
RUN git clone https://github.com/charlie-foxtrot/RTLSDR-Airband.git /tmp/rtl_airband && \
    cd /tmp/rtl_airband && \
    mkdir build && cd build && \
    cmake .. -DPLATFORM=generic -DNFM=ON && \
    make -j$(nproc) && \
    make install && \
    rm -rf /tmp/rtl_airband

# Copy configuration files
COPY rtl_airband.conf /usr/local/etc/rtl_airband.conf
COPY icecast.xml /etc/icecast2/icecast.xml
COPY nginx.conf /etc/nginx/nginx.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY html/ /var/www/html/

# Fix icecast permissions
RUN chown -R icecast2:icecast /etc/icecast2

EXPOSE 8090

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
