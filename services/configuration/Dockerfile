FROM drukleague-service-builder:local AS build
WORKDIR /src
COPY packages/servicekit/ /src/packages/servicekit/
COPY packages/contracts/ /src/packages/contracts/
COPY services/configuration/ /src/services/configuration/
WORKDIR /src/services/configuration
RUN mkdir -p /out && CGO_ENABLED=0 go build -trimpath -o /out/service ./cmd/service
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/service /app/service
COPY services/configuration/migrations /app/migrations
ENV SERVICE_NAME=configuration-service
ENV MIGRATIONS_DIR=/app/migrations
EXPOSE 50051
ENTRYPOINT ["/app/service"]
