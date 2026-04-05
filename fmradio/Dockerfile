FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    rtl-sdr librtlsdr-dev ffmpeg \
    icecast2 nginx supervisor \
    python3 python3-flask procps \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /var/log/supervisor /var/log/icecast2 /var/www/html && \
    chown -R icecast2:icecast /var/log/icecast2 /etc/icecast2

COPY icecast.xml /etc/icecast2/icecast.xml
COPY nginx.conf /etc/nginx/nginx.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY start_fm.sh /opt/start_fm.sh
COPY app.py /opt/app.py
COPY html/ /var/www/html/
RUN chmod +x /opt/start_fm.sh

EXPOSE 8090
CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
