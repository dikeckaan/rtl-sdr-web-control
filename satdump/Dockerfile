FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake make g++ pkg-config ca-certificates \
    librtlsdr-dev libusb-1.0-0-dev libfftw3-dev \
    libpng-dev libjpeg-dev zlib1g-dev \
    libnng-dev libcurl4-openssl-dev \
    libzstd-dev libtiff-dev \
    libvolk2-dev libarmadillo-dev \
    libsqlite3-dev libjemalloc-dev \
    && rm -rf /var/lib/apt/lists/*

# Build SatDump from source (headless server mode)
RUN git clone --depth 1 https://github.com/SatDump/SatDump.git /tmp/satdump && \
    cd /tmp/satdump && mkdir build && cd build && \
    cmake .. \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_GUI=OFF \
        -DCMAKE_INSTALL_PREFIX=/usr \
        -DPLUGIN_SCRIPTING=OFF && \
    make -j$(nproc) && \
    make install && \
    rm -rf /tmp/satdump

RUN mkdir -p /output /root/.config/satdump

EXPOSE 8080
CMD ["satdump", "server", "--port", "8080", "--bind", "0.0.0.0", "--output_folder", "/output"]
