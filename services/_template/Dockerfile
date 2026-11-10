FROM golang:1.26.10-alpine AS build
WORKDIR /src/services/service
COPY packages/servicekit /src/packages/servicekit
COPY services/service-template /src/services/service
RUN set -eu; \
    attempt=1; \
    until go mod download; do \
        if [ "${attempt}" -ge 5 ]; then echo "Dependency download failed after ${attempt} attempts" >&2; exit 1; fi; \
        delay=$((attempt * 5)); \
        echo "Dependency download failed; retrying in ${delay}s (${attempt}/5)" >&2; \
        sleep "${delay}"; \
        attempt=$((attempt + 1)); \
    done
RUN CGO_ENABLED=0 go build -trimpath -o /out/service ./cmd/service
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/service /app/service
COPY --from=build /src/services/service/migrations /app/migrations
ENV MIGRATIONS_DIR=/app/migrations
EXPOSE 50051
ENTRYPOINT ["/app/service"]
