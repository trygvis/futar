# syntax=docker/dockerfile:1.7-labs

FROM golang:1.22.0

COPY bin/* ./bin/
RUN bin/oapi-codegen --version

COPY --parents * ./
RUN make

EXPOSE 8080
CMD [ "futar" ]
