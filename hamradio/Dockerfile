FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    rtl-sdr ffmpeg \
    icecast2 nginx supervisor \
    python3 python3-flask procps \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /var/log/supervisor /var/log/icecast2 /var/www/html && \
    chown -R icecast2:icecast /var/log/icecast2

COPY icecast.xml /etc/icecast2/icecast.xml
RUN chown -R icecast2:icecast /etc/icecast2
COPY nginx.conf /etc/nginx/nginx.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY start_ham.sh /usr/local/bin/start_ham.sh
COPY app.py /opt/app.py
COPY html/ /var/www/html/
RUN chmod +x /usr/local/bin/start_ham.sh

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
