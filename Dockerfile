# syntax=docker/dockerfile:1.7-labs

FROM golang:1.22.0

COPY --parents bin/oapi-codegen bin/
RUN bin/oapi-codegen || true

COPY --parents * ./
RUN make

EXPOSE 8080
CMD [ "futar" ]
