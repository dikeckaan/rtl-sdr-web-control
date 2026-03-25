FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# OP25 dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake g++ make pkg-config \
    gnuradio gnuradio-dev gr-osmosdr \
    librtlsdr-dev libusb-1.0-0-dev rtl-sdr \
    python3 python3-pip python3-venv python3-dev \
    python3-numpy python3-scipy python3-gi \
    libboost-all-dev liblog4cpp5-dev \
    supervisor sox \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build OP25 - use boatbod fork with cmake fix
RUN git clone --depth 1 https://github.com/boatbod/op25.git /opt/op25 && \
    cd /opt/op25/op25/gr-op25_repeater && \
    mkdir -p cmake && \
    echo 'foreach(_variableName ${CMAKE_INSTALL_MANIFEST_FILES})\n  message(STATUS "Uninstalling: ${_variableName}")\n  file(REMOVE "${_variableName}")\nendforeach()' > cmake/cmake_uninstall.cmake.in && \
    mkdir build && cd build && \
    cmake .. && make -j$(nproc) && make install && ldconfig

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY start-op25.sh /opt/start-op25.sh
RUN chmod +x /opt/start-op25.sh

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
