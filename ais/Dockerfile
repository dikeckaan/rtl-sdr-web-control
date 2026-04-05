FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake git pkg-config \
    librtlsdr-dev libusb-1.0-0-dev \
    rtl-sdr \
    python3 python3-pip python3-flask \
    nginx supervisor procps ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build rtl-ais from source
RUN git clone --depth 1 https://github.com/dgiardini/rtl-ais.git /tmp/rtl-ais && \
    cd /tmp/rtl-ais && \
    make && \
    cp rtl_ais /usr/local/bin/ && \
    rm -rf /tmp/rtl-ais

RUN pip3 install --break-system-packages flask-socketio simple-websocket pyais

RUN mkdir -p /var/log/supervisor /var/www/html

COPY nginx.conf /etc/nginx/nginx.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY start_ais.sh /opt/start_ais.sh
COPY app.py /opt/app.py
COPY html/ /var/www/html/
RUN chmod +x /opt/start_ais.sh

EXPOSE 8090
CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
