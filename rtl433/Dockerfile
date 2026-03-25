FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# Build rtl_433
RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake g++ make pkg-config \
    librtlsdr-dev libusb-1.0-0-dev \
    python3 python3-pip python3-venv \
    supervisor ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 https://github.com/merbanan/rtl_433.git /tmp/rtl_433 && \
    mkdir /tmp/rtl_433/build && cd /tmp/rtl_433/build && \
    cmake .. -DCMAKE_INSTALL_PREFIX=/usr/local && \
    make -j$(nproc) && make install && \
    rm -rf /tmp/rtl_433

# Web dashboard
RUN python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install flask

COPY web_dashboard.py /opt/web_dashboard.py
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
