FROM golang:1.24-alpine AS build
WORKDIR /src/services/service
COPY packages/servicekit /src/packages/servicekit
COPY services/service-template /src/services/service
RUN go mod download
RUN CGO_ENABLED=0 go build -trimpath -o /out/service ./cmd/service
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/service /app/service
COPY --from=build /src/services/service/migrations /app/migrations
ENV MIGRATIONS_DIR=/app/migrations
EXPOSE 50051
ENTRYPOINT ["/app/service"]