.PHONY: build
build:
	rm -rf ./build/*
	mkdir -p ./build/amd64 ./build/arm64

	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-o build/amd64/wyze-mjpeg-proxy \
		./cmd/wyze-mjpeg-proxy

	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
		-o build/arm64/wyze-mjpeg-proxy \
		./cmd/wyze-mjpeg-proxy

.PHONY: docker
docker: build
	docker buildx build --pull --push \
		--platform linux/amd64,linux/arm64 \
		-t kamermans/wyze-mjpeg-proxy .
