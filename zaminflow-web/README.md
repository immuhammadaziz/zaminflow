# ZaminFlow Web

The locale-aware web application for ZaminFlow's farmer, processor, and administrator workflows. Built with Next.js App Router, TypeScript, Tailwind CSS, TanStack Query, and next-intl.

## Five-Command Quick Start

```bash
git clone https://github.com/zaminflow/zaminflow-web.git
cd zaminflow-web
cp .env.example .env
pnpm install
pnpm dev
```

The application will be running at [http://localhost:3000](http://localhost:3000). The default locale is Uzbek (`/uz`), with Russian (`/ru`) and English (`/en`) supported.

Or start the full stack via Docker:

```bash
cp .env.example .env
docker compose up --build
```

## Requirements

- Node.js `26.13.0`
- pnpm `12.16.1`
- A running ZaminFlow API for authenticated workflows

Configure the API URL before starting or building the app. It must be an absolute HTTP(S) URL ending in `/api/v1`; production deployments must use HTTPS.

```powershell
$env:API_BASE_URL = "http://localhost:8080/api/v1"
pnpm install --frozen-lockfile
pnpm dev
```

## Verification & Quality Gates

Run all quality checks locally:

```bash
pnpm lint
pnpm typecheck
pnpm check:translations
pnpm check-client-drift
pnpm test
pnpm build
pnpm build-storybook
pnpm e2e
```

Or run the consolidated check:

```bash
pnpm check
```

Playwright starts the server and runs public-page, auth-form validation, localized not-found, axe-core accessibility checks, and security-header smoke checks.

## API Contract & Drift Check

`contracts/openapi.yaml` is the frontend's API contract snapshot. Generate and verify its TypeScript client with:

```bash
pnpm generate-client
pnpm check-client-drift
```

Do not simulate unsupported backend behavior. Current known limitations and requested API improvements are tracked in [docs/BACKEND_REQUESTS.md](./docs/BACKEND_REQUESTS.md); supported screen inventory is documented in [docs/SCREEN_INVENTORY.md](./docs/SCREEN_INVENTORY.md).

## Production and Docker

```bash
pnpm build
pnpm start
```

Build the standalone production image:

```bash
docker build --build-arg API_BASE_URL=https://api.zaminflow.uz/api/v1 -t zaminflow-web .
docker run --rm -p 3000:3000 zaminflow-web
```

To run with production Caddy reverse proxy:

```bash
docker compose -f docker-compose.prod.yml up --build -d
```

## Documentation

- [Backend Understanding](docs/BACKEND_UNDERSTANDING.md)
- [Backend Requests & Gaps](docs/BACKEND_REQUESTS.md)
- [Site Review & Identity](docs/SITE_REVIEW.md)
- [Screen Inventory](docs/SCREEN_INVENTORY.md)
- [UX User Flows](docs/UX_FLOWS.md)
- [Design System & Tokens](docs/DESIGN_SYSTEM.md)
- [Architectural Decisions](docs/DECISIONS.md)
- [Security & OWASP Review](docs/SECURITY.md)
- [Deployment Guide](docs/DEPLOY.md)
- [Testing Strategy](docs/TESTING.md)
