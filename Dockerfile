FROM linuxserver/ffmpeg:version-7.1-cli

COPY build/ /tmp/
RUN cp /tmp/$(dpkg --print-architecture)/wyze-mjpeg-proxy /usr/local/bin/wyze-mjpeg-proxy

ENTRYPOINT [ "/usr/local/bin/wyze-mjpeg-proxy", "--config", "/config.yaml" ]
