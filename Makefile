IMG ?= gcs-proxy
SHA := $(shell git rev-parse --short HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
TAG := ${BRANCH}-${SHA}

# Local builds only; CI publishes the images.
docker-build:
	docker build --build-arg VERSION=${TAG} -t ${IMG}:${TAG} .

test:
	go test -race ./...

.PHONY: docker-build test
