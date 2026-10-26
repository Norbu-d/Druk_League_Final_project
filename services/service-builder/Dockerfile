FROM golang:1.26-alpine
WORKDIR /src
# Pre-download the shared modules' dependencies so service builds mostly hit the cache.
COPY packages/servicekit/go.mod packages/servicekit/go.sum ./packages/servicekit/
COPY packages/contracts/go.mod packages/contracts/go.sum ./packages/contracts/
COPY packages/eventkit/go.mod packages/eventkit/go.sum ./packages/eventkit/
RUN cd /src/packages/servicekit && go mod download && cd /src/packages/contracts && go mod download && cd /src/packages/eventkit && go mod download
