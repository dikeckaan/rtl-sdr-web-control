FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    rtl-sdr \
    multimon-ng \
    python3 \
    python3-pip \
    python3-flask \
    nginx \
    supervisor \
    procps \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 install --break-system-packages flask-socketio simple-websocket

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf
RUN rm -f /etc/nginx/sites-enabled/default

COPY start_pager.sh /opt/pager/start_pager.sh
RUN chmod +x /opt/pager/start_pager.sh

COPY app.py /opt/pager/app.py
COPY html/ /opt/pager/html/

RUN mkdir -p /var/log/pager && touch /tmp/pager_live.log

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
