FROM drukleague-service-builder:local AS build
WORKDIR /src
COPY packages/servicekit/ /src/packages/servicekit/
COPY packages/contracts/ /src/packages/contracts/
COPY services/bff/ /src/services/bff/
WORKDIR /src/services/bff
RUN mkdir -p /out && CGO_ENABLED=0 go build -trimpath -o /out/bff ./cmd/bff
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/bff /app/bff
ENV BFF_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/bff"]
