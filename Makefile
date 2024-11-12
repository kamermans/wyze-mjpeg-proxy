.PHONY: docker
docker:
	docker buildx build --pull --push \
		--platform linux/amd64,linux/arm64 \
		-t kamermans/wyze-mjpeg-proxy .
