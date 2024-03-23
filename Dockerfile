# syntax=docker/dockerfile:1.7-labs

FROM golang:1.22.0 as builder

COPY bin/* ./bin/
RUN bin/oapi-codegen --version

COPY --parents * ./
RUN make

FROM debian:12.5-slim

COPY --from=builder /go/futar /futar

EXPOSE 8080
CMD [ "/futar" ]
