# ZaminFlow API

ZaminFlow is a contract-first Go/PostgreSQL backend for a dairy marketplace in Uzbekistan. The API supports farmer and processor registration, email verification, administrator approval, listings, reserved-inventory orders, quality/dispute records, and role dashboards. It does not process payments, arrange transportation, or include a frontend.

## Local development

Requirements: Docker Compose and Go 1.27.3.

1. Copy `.env.example` to `.env`. Replace `POSTGRES_PASSWORD` and `APP_SECRET`; keep development-only values out of production.
2. Start the API, PostgreSQL, migrations, and local SMTP inbox:

   ```powershell
   docker compose up --build -d
   Invoke-RestMethod http://localhost:8080/healthz
   Invoke-RestMethod http://localhost:8080/readyz
   ```

3. Provision the first administrator by setting `ADMIN_EMAIL`, `ADMIN_FULL_NAME`, and `ADMIN_PASSWORD` in `.env`, then run:

   ```powershell
   docker compose --profile admin-tools run --rm admin
   ```

   Sign in as this administrator, enroll TOTP, confirm the code, save the one-time recovery codes, then sign in again for a full administrator session.
4. Open Mailpit at <http://localhost:8025> to retrieve sign-up verification and password-reset links. Swagger UI is available at <http://localhost:8081> after `docker compose --profile dev-tools up -d swagger-ui`.

The local Compose stack exposes PostgreSQL, API, Mailpit, and optional Swagger UI for development. Do not expose these development ports or credentials to the public internet.

## API and client integration

- [OpenAPI specification](docs/openapi.yaml)
- [Implemented API contract and workflow](docs/API_CONTRACT.md)
- [Frontend integration guide](docs/FRONTEND_INTEGRATION.md)
- [Design and domain decisions](docs/DESIGN_REPORT.md) and [decision log](docs/DECISIONS.md)
- [Production deployment](docs/DEPLOY.md) and [secret security/rotation](docs/SECURITY.md)
- [Production deployment and recovery](docs/DEPLOYMENT.md)
- [Testing and CI](docs/TESTING.md)

The API base path is `/api/v1`. Bearer access tokens are short-lived; refresh tokens are rotated in secure, HttpOnly cookies. Browser refresh and sign-out requests require an allowed `Origin`, the `zf_csrf` cookie, and a matching `X-CSRF-Token` header. All marketplace operations require an approved account and the relevant role.

## Production operations

Review [docs/DEPLOY.md](docs/DEPLOY.md) and [docs/SECURITY.md](docs/SECURITY.md) before deployment. Production Compose expects a root `.env.production` with non-secret configuration and root-owned file secrets under `SECRETS_DIR`. DNS and SMTP setup are deferred; configure them before enabling production traffic. Caddy terminates HTTPS; the API and database are not published as host ports. Bootstrap the initial administrator only through the separate one-shot Compose overlay.

## Verification

```powershell
go test ./...
go vet ./...
go build ./cmd/api ./cmd/migrate ./cmd/admin
```

Database-backed integration tests run when `DATABASE_URL` is set and skip otherwise. See [docs/TESTING.md](docs/TESTING.md) for isolated PostgreSQL setup, race tests, migration checks, and CI behavior.
