all: futar

futar: server-api.gen.go
	go build

clean:
	rm -f server-api.gen.go

server-api.gen.go: oapi-config.yaml demo-api.yaml
	bin/oapi-codegen --config oapi-config.yaml demo-api.yaml
