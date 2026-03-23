FROM python:3.11-slim

RUN pip install --no-cache-dir flask

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl && \
    curl -fsSL https://get.docker.com | sh && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY app.py .
COPY templates/ templates/

EXPOSE 8090
CMD ["python", "app.py"]
