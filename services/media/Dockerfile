FROM drukleague-service-builder:local AS build
WORKDIR /src
COPY packages/servicekit/ /src/packages/servicekit/
COPY packages/contracts/ /src/packages/contracts/
COPY packages/eventkit/ /src/packages/eventkit/
COPY services/media/ /src/services/media/
WORKDIR /src/services/media
RUN mkdir -p /out/data && CGO_ENABLED=0 go build -trimpath -o /out/service ./cmd/service
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/service /app/service
COPY --chown=nonroot:nonroot --from=build /out/data /data
COPY services/media/migrations /app/migrations
ENV SERVICE_NAME=media-service
ENV MIGRATIONS_DIR=/app/migrations
EXPOSE 8080
ENTRYPOINT ["/app/service"]
