build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wyze-mjpeg-proxy ./cmd/wyze-mjpeg-proxy

docker: build
	docker build -t wyze-mjpeg-proxy .

start:
	docker rm -vf wyze-mjpeg-proxy || echo "No container to remove"
	docker run -d \
		--name wyze-mjpeg-proxy \
		-p 8190:8190 \
		-v $(PWD)/config.yaml:/config.yaml \
		wyze-mjpeg-proxy

stop:
	docker stop wyze-mjpeg-proxy
