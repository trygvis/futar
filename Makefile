DATE_CMD := $(shell date -Iminutes)
BUILD_ROOT ?= .

all: futar

futar: $(wildcard *.go) gen
	go build \
		-o $(BUILD_ROOT)/futar \
		-ldflags "-X main.date=$(DATE_CMD) -X main.version=${TRINFRA_VERSION} -X main.commit=$(shell git rev-parse --short HEAD)"

.PHONY: gen
gen: server-api.gen.go

.PHONY: clean
clean:
	rm -f server-api.gen.go

server-api.gen.go: oapi-config.yaml demo-api.yaml
	bin/oapi-codegen --config oapi-config.yaml demo-api.yaml

.PHONY: docker-image
docker-image:
	DOCKER_BUILDKIT=1 docker build -t futar .
