# OpenFaaS function image ("dockerfile" language): the official of-watchdog in HTTP mode
# starts the Go handler and forwards every invocation to it.
FROM ghcr.io/openfaas/of-watchdog:0.10.7 AS watchdog

FROM golang:1.26.11-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/handler .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=watchdog /fwatchdog /usr/bin/fwatchdog
COPY --from=build /out/handler /usr/bin/handler
ENV fprocess="/usr/bin/handler" \
    mode="http" \
    upstream_url="http://127.0.0.1:8082" \
    exec_timeout="15s" \
    read_timeout="15s" \
    write_timeout="15s" \
    prefix_logs="false"
EXPOSE 8080
ENTRYPOINT ["/usr/bin/fwatchdog"]
