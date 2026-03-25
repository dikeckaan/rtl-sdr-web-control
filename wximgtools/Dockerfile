FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
# NOAA APT decoder tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake g++ pkg-config \
    rtl-sdr librtlsdr-dev libusb-1.0-0-dev \
    python3 python3-pip python3-venv python3-dev \
    python3-numpy python3-scipy python3-pillow \
    sox libsndfile1-dev \
    supervisor \
    ca-certificates wget \
    && rm -rf /var/lib/apt/lists/*

# Install noaa-apt decoder (Rust-based NOAA APT decoder)
RUN python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install flask matplotlib

COPY web_dashboard.py /opt/web_dashboard.py
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

RUN mkdir -p /opt/recordings /opt/images

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
