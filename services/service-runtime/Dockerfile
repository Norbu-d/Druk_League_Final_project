FROM golang:1.26-alpine AS build
WORKDIR /src
COPY packages/servicekit/go.mod packages/servicekit/go.sum ./
RUN go mod download
COPY packages/servicekit/ ./
RUN mkdir -p /out && CGO_ENABLED=0 go build -trimpath -o /out/service ./cmd/runner
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/service /app/service
ENTRYPOINT ["/app/service"]
