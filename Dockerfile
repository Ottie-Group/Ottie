# --- Build Stage ---
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/ottie .

RUN adduser -D -u 10001 -g "" appuser && \
    mkdir -p /out/data && \
    chown -R 10001:10001 /out/data

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
