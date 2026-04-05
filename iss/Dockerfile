FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    rtl-sdr ffmpeg sox \
    python3 python3-pip python3-flask \
    nginx supervisor procps ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 install --break-system-packages ephem flask-socketio simple-websocket

RUN mkdir -p /var/log/supervisor /var/www/html /opt/iss/captures

COPY nginx.conf /etc/nginx/nginx.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY app.py /opt/iss/app.py
COPY start_record.sh /opt/iss/start_record.sh
COPY html/ /var/www/html/
RUN chmod +x /opt/iss/start_record.sh

EXPOSE 8090
CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
