FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# Build multimon-ng + web terminal
RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake g++ make pkg-config \
    libpulse-dev libx11-dev \
    rtl-sdr librtlsdr-dev \
    python3 python3-pip python3-venv \
    supervisor sox \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build multimon-ng
RUN git clone --depth 1 https://github.com/EliasOenal/multimon-ng.git /tmp/multimon-ng && \
    mkdir /tmp/multimon-ng/build && cd /tmp/multimon-ng/build && \
    cmake .. && make -j$(nproc) && make install && \
    rm -rf /tmp/multimon-ng

# Web dashboard
RUN python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install flask

COPY web_dashboard.py /opt/web_dashboard.py
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
