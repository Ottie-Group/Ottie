# Build React
FROM --platform=$BUILDPLATFORM oven/bun:1-alpine AS frontend-builder

WORKDIR /src/frontend
COPY frontend/package.json frontend/bun.lock* ./
RUN bun install --frozen-lockfile || bun install

COPY frontend/ ./
RUN bun run build

# Build Go
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
# Copy Compiled React
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

ARG TARGETOS TARGETARCH
ARG VERSION=""
RUN if [ -n "$VERSION" ]; then \
        LDFLAGS="-s -w -X main.Version=${VERSION}"; \
    else \
        LDFLAGS="-s -w"; \
    fi && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="${LDFLAGS}" -o /out/ottie .

RUN adduser -D -u 10001 -g "" appuser && \
    mkdir -p /out/data && \
    chown -R 10001:10001 /out/data

# Alpine Deploy Stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -u 10001 -g "" appuser

WORKDIR /app

COPY --from=builder /out/ottie /app/ottie
COPY --from=builder --chown=10001:10001 /out/data /app/data

VOLUME ["/app/data"]
ENV OTTIE_DB_PATH=/app/data/ottie.db
ENV OTTIE_LISTEN_ADDR=0.0.0.0:8080

EXPOSE 8080
USER appuser

ENTRYPOINT ["/app/ottie"]
