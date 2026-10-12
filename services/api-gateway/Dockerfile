FROM golang:1.24-alpine AS build
WORKDIR /src
COPY services/api-gateway/go.mod services/api-gateway/go.sum ./
RUN go mod download
COPY services/api-gateway/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/drukleague-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/drukleague-api /app/drukleague-api
COPY --from=build /src/migrations /app/migrations
ENV API_ADDR=:8080 MIGRATION_FILE=/app/migrations/001_init.sql APP_ENV=production
EXPOSE 8080
ENTRYPOINT ["/app/drukleague-api"]
