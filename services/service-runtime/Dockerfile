FROM golang:1.26.10-alpine AS build
WORKDIR /src
COPY packages/servicekit/go.mod packages/servicekit/go.sum ./
RUN set -eu; \
    attempt=1; \
    until go mod download; do \
        if [ "${attempt}" -ge 5 ]; then echo "Dependency download failed after ${attempt} attempts" >&2; exit 1; fi; \
        delay=$((attempt * 5)); \
        echo "Dependency download failed; retrying in ${delay}s (${attempt}/5)" >&2; \
        sleep "${delay}"; \
        attempt=$((attempt + 1)); \
    done
COPY packages/servicekit/ ./
RUN mkdir -p /out && CGO_ENABLED=0 go build -trimpath -o /out/service ./cmd/runner
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/service /app/service
ENTRYPOINT ["/app/service"]
