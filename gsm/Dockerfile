FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

# rtl_power based GSM scanner - no gnuradio needed
RUN apt-get update && apt-get install -y --no-install-recommends \
    rtl-sdr \
    python3 python3-flask \
    nginx supervisor procps \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /var/log/supervisor /var/www/html /opt/gsm

COPY nginx.conf /etc/nginx/nginx.conf
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY app.py /opt/gsm/app.py
COPY html/ /var/www/html/

EXPOSE 8090
CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
