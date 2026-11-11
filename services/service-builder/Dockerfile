FROM golang:1.26.10-alpine
WORKDIR /src
# Pre-download the shared modules' dependencies so service builds mostly hit the cache.
COPY packages/servicekit/go.mod packages/servicekit/go.sum ./packages/servicekit/
COPY packages/contracts/go.mod packages/contracts/go.sum ./packages/contracts/
COPY packages/eventkit/go.mod packages/eventkit/go.sum ./packages/eventkit/
RUN set -eu; \
    for module in packages/servicekit packages/contracts packages/eventkit; do \
        attempt=1; \
        until (cd "/src/${module}" && go mod download); do \
            if [ "${attempt}" -ge 5 ]; then \
                echo "Dependency download failed after ${attempt} attempts: ${module}" >&2; \
                exit 1; \
            fi; \
            delay=$((attempt * 5)); \
            echo "Dependency download failed for ${module}; retrying in ${delay}s (${attempt}/5)" >&2; \
            sleep "${delay}"; \
            attempt=$((attempt + 1)); \
        done; \
    done
