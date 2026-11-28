# Function connector: a long-running JetStream consumer that invokes a function over HTTP.
# Build context is the repository root (it needs packages/eventkit).
FROM drukleague-service-builder:local AS build
WORKDIR /src
COPY packages/eventkit/ /src/packages/eventkit/
COPY functions/connector/ /src/functions/connector/
WORKDIR /src/functions/connector
RUN mkdir -p /out && CGO_ENABLED=0 go build -trimpath -o /out/connector ./cmd/connector

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/connector /app/connector
ENTRYPOINT ["/app/connector"]
