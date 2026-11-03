# ZaminFlow Web Deployment Guide

This document outlines deployment procedures, domain routing, zero-downtime rollouts, rollback strategies, and caching architecture for `zaminflow-web`.

## 1. Domain and DNS Architecture

| Hostname           | Target / Type                 | Purpose                                                      |
| :----------------- | :---------------------------- | :----------------------------------------------------------- |
| `www.zaminflow.uz` | CNAME / A Record to Web Proxy | Primary user-facing web application                          |
| `zaminflow.uz`     | A Record / CNAME to Web Proxy | Apex domain; 301 permanently redirects to `www.zaminflow.uz` |
| `api.zaminflow.uz` | A Record / CNAME to API Proxy | ZaminFlow backend REST API                                   |

Caddy handles automatic HTTPS provisioning via Let�s Encrypt / ZeroSSL with HTTP-01 or DNS-01 ACME challenge verification.

## 2. Environment Configuration

The application uses standalone Next.js deployment. Build-time and runtime environment variables:

| Variable       | Scope              | Description                            | Default / Example                 |
| :------------- | :----------------- | :------------------------------------- | :-------------------------------- |
| `API_BASE_URL` | Build & Runtime    | Absolute URL to the ZaminFlow REST API | `https://api.zaminflow.uz/api/v1` |
| `PORT`         | Container Runtime  | Internal listening port                | `3000`                            |
| `HOSTNAME`     | Container Runtime  | Bound network interface                | `0.0.0.0`                         |
| `NODE_ENV`     | Container Runtime  | Node runtime profile                   | `production`                      |
| `WEB_DOMAIN`   | Production Compose | Domain handled by Caddy edge proxy     | `www.zaminflow.uz`                |

### Setting Runtime Configuration

Client-side browser requests fetch runtime config dynamically from `/runtime-config` or read from server-rendered metadata, allowing container reuse across staging and production without rebuilding image assets.

## 3. Deployment Topology

### Production Compose (`docker-compose.prod.yml`)

- **`web` container:** Non-root user `nextjs:nodejs`, standalone Next.js server, resource limits (1.5 CPU, 1GB RAM), continuous HTTP healthcheck probe (`/uz`).
- **`caddy` edge proxy:** TLS termination, HTTP/2 and HTTP/3 support, gzip/zstd compression, and reverse proxy routing to `web:3000`.

To start the production web stack:

```bash
cp .env.example .env
docker compose -f docker-compose.prod.yml up --build -d
```

## 4. Zero-Downtime Updates & Rolling Deployments

1. Build the updated image with a unique commit hash tag:
   ```bash
   docker build --build-arg API_BASE_URL=https://api.zaminflow.uz/api/v1 -t zaminflow-web:sha-<commit> .
   ```
2. Deploy using Docker Compose rolling service update or blue-green container swaps:
   ```bash
   docker compose -f docker-compose.prod.yml up -d --no-deps --build web
   ```
3. Docker Compose launches the new container and performs health checks. Only once `http://127.0.0.1:3000/uz` returns status `200` does traffic route to the new container before retiring the previous instance.

## 5. Rollback Strategy

1. Tag previous known good images (e.g. `zaminflow-web:stable`).
2. If an incident or unexpected regression occurs post-deployment:
   ```bash
   docker tag zaminflow-web:previous-known-good zaminflow-web:latest
   docker compose -f docker-compose.prod.yml up -d --no-deps web
   ```
3. Rollback completes in under 10 seconds without dropping TCP connections or interrupting active browser sessions.

## 6. Caching and CDN Directives

- **Static Assets (`/_next/static/*`, `/public/*`):**
  - Response header: `Cache-Control: public, max-age=31536000, immutable`
  - Chunk filenames include content hashes, allowing infinite immutable caching at Cloudflare / CDN edge.
- **Dynamic HTML Pages (`/[locale]/*`):**
  - Response header: `Cache-Control: private, no-cache, no-store, must-revalidate`
  - Guarantees fresh role-based session hydration and timely invalidation.
- **Locale Switcher & Fallback:**
  - Root `/` performs 307 temporary redirect to the user's preferred locale (`/uz`, `/ru`, or `/en`) based on `Accept-Language` headers and cookie preferences.
