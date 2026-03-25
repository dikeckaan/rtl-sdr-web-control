FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# Build DSD (Digital Speech Decoder)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake g++ make pkg-config \
    librtlsdr-dev libusb-1.0-0-dev rtl-sdr \
    libsndfile1-dev libportaudio2 portaudio19-dev \
    libitpp-dev \
    python3 python3-pip python3-venv \
    supervisor sox \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build mbelib first
RUN git clone --depth 1 https://github.com/szechyjs/mbelib.git /tmp/mbelib && \
    mkdir /tmp/mbelib/build && cd /tmp/mbelib/build && \
    cmake .. && make -j$(nproc) && make install && ldconfig && \
    rm -rf /tmp/mbelib

# Build DSD
RUN git clone --depth 1 https://github.com/szechyjs/dsd.git /tmp/dsd && \
    mkdir /tmp/dsd/build && cd /tmp/dsd/build && \
    cmake .. && make -j$(nproc) && make install && \
    rm -rf /tmp/dsd

# Web dashboard
RUN python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install flask

COPY web_dashboard.py /opt/web_dashboard.py
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
