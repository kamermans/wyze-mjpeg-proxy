FROM jrottenberg/ffmpeg:7.1-scratch

COPY wyze-mjpeg-proxy /usr/local/bin/wyze-mjpeg-proxy

ENTRYPOINT [ "/usr/local/bin/wyze-mjpeg-proxy", "--config", "/config.yaml" ]
