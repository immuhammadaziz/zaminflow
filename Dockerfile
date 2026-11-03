ARG GO_VERSION=1.27.3
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/admin ./cmd/admin

FROM alpine:3.22.7
RUN apk add --no-cache ca-certificates su-exec \
    && addgroup -S -g 65532 app \
    && adduser -S -D -H -u 65532 -G app app
WORKDIR /app
COPY --from=builder --chown=65532:65532 /out/api /app/api
COPY --from=builder --chown=65532:65532 /out/migrate /app/migrate
COPY --from=builder --chown=65532:65532 /out/admin /app/admin
COPY --chown=65532:65532 migrations /app/migrations
COPY scripts/app-entrypoint.sh /usr/local/bin/app-entrypoint.sh
RUN chmod 0755 /usr/local/bin/app-entrypoint.sh
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/app-entrypoint.sh"]
CMD ["/app/api"]
