FROM golang:1.23-bookworm AS builder

ENV CGO_ENABLED=0

COPY . /app
WORKDIR /app
RUN go build \
    -o /tmp/wyze-mjpeg-proxy \
    ./cmd/wyze-mjpeg-proxy

FROM linuxserver/ffmpeg:version-7.1-cli

COPY --from=builder /tmp/wyze-mjpeg-proxy /usr/local/bin/wyze-mjpeg-proxy

ENTRYPOINT [ "/usr/local/bin/wyze-mjpeg-proxy", "--config", "/config.yaml" ]
