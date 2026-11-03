FROM caddy:2.11.7-builder-alpine@sha256:95a5e7e2692b5eb9545aadc5b90a39f6f509bee59c9508a493f83ee902eaef63 AS builder

RUN xcaddy build v2.11.7 \
    --output=/out/caddy \
    --replace=github.com/go-jose/go-jose/v4@v4.1.5=github.com/go-jose/go-jose/v4@v4.1.6 \
    --replace=golang.org/x/net@v0.59.0=golang.org/x/net@v0.60.0

FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates

ENV XDG_CONFIG_HOME=/config \
    XDG_DATA_HOME=/data

WORKDIR /srv

COPY --from=builder /out/caddy /usr/bin/caddy

ENTRYPOINT ["/usr/bin/caddy"]
CMD ["run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
