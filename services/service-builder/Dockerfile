FROM golang:1.24-alpine
WORKDIR /src
COPY packages/servicekit/go.mod packages/servicekit/go.sum ./packages/servicekit/
COPY packages/contracts/go.mod packages/contracts/go.sum ./packages/contracts/
RUN cd /src/packages/servicekit && go mod download && cd /src/packages/contracts && go mod download
