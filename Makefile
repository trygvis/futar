all: futar

futar: $(wildcard *.go) gen
	go build

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
