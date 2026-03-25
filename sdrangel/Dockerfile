FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV DISPLAY=:99

# SDRAngel build dependencies - full Qt5 set
RUN apt-get update && apt-get install -y --no-install-recommends \
    git cmake g++ make pkg-config \
    librtlsdr-dev libusb-1.0-0-dev \
    libfftw3-dev liblog4cpp5-dev libboost-all-dev \
    qtbase5-dev qtmultimedia5-dev libqt5opengl5-dev libqt5websockets5-dev qttools5-dev \
    libqt5serialport5-dev qtpositioning5-dev \
    libqt5charts5-dev libqt5texttospeech5-dev libqt5gamepad5-dev \
    libqt5location5 qml-module-qtlocation qml-module-qtpositioning \
    libgl1-mesa-dev libglu1-mesa-dev \
    libavcodec-dev libavformat-dev libavutil-dev libswscale-dev \
    libopus-dev libcodec2-dev libsamplerate0-dev \
    liblimesuite-dev libsoapysdr-dev soapysdr-module-rtlsdr \
    xvfb x11vnc websockify novnc \
    supervisor dbus-x11 \
    libgl1-mesa-dri libegl1 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build SDRAngel - server mode (web API, no GUI needed for heavy Qt)
RUN git clone --depth 1 https://github.com/f4exb/sdrangel.git /tmp/sdrangel && \
    mkdir /tmp/sdrangel/build && cd /tmp/sdrangel/build && \
    cmake .. \
      -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_INSTALL_PREFIX=/opt/sdrangel \
      -DDEBUG_OUTPUT=OFF \
      -DBUILD_SERVER=ON \
      -DBUILD_GUI=OFF && \
    make -j1 && make install && \
    rm -rf /tmp/sdrangel

ENV PATH="/opt/sdrangel/bin:${PATH}"
ENV LD_LIBRARY_PATH="/opt/sdrangel/lib:${LD_LIBRARY_PATH}"

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 8090
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
