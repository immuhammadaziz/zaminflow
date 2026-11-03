# Testing and CI

## Fast local checks

From the repository root:

```powershell
gofmt -w (Get-ChildItem -Recurse -Filter *.go | ForEach-Object FullName)
go test ./...
go vet ./...
go build ./cmd/api ./cmd/migrate ./cmd/admin
```

Tests that exercise auth/approval and marketplace flows skip when `DATABASE_URL` is unset. Set it to a disposable PostgreSQL database with migrations applied to run them. The outbox integration test uses a separate `MAIL_TEST_DATABASE_URL` so it does not race with auth tests that touch the same outbox table.

Example setup with Docker Compose:

```powershell
docker compose up -d postgres
$env:DATABASE_URL = "postgres://zaminflow:change-me-local-only@localhost:5432/zaminflow?sslmode=disable"
go run ./cmd/migrate
$postgres = docker compose ps -q postgres
docker exec $postgres createdb -U zaminflow zaminflow_mail_test
$env:MAIL_TEST_DATABASE_URL = "postgres://zaminflow:change-me-local-only@localhost:5432/zaminflow_mail_test?sslmode=disable"
$env:DATABASE_URL = $env:MAIL_TEST_DATABASE_URL
go run ./cmd/migrate
$env:DATABASE_URL = "postgres://zaminflow:change-me-local-only@localhost:5432/zaminflow?sslmode=disable"
go test ./...
```

Use an isolated database for integration tests; test cleanup deletes rows it creates. Never point tests at a production or user-data database. To validate migrations from scratch, create a disposable database and run `go run ./cmd/migrate`; test down/up only against that database.

## Coverage of critical behavior

- Unit tests cover configuration validation, access/CSRF tokens, HTTP middleware, metrics authentication/counters, and SMTP behavior.
- PostgreSQL integration tests exercise email verification, admin enrollment/review/resubmission/suspension/reinstatement, audited account-deletion request review, session and rate-limit behavior, concurrent order reservations that assert inventory cannot be oversold, order transitions, dashboard aggregates, and outbox delivery/retry/scrubbing.
- CI builds and runs the custom PostgreSQL image, then runs formatting, vet, migration up/down/up, race-enabled tests, token fuzzing, and reachable-vulnerability checks. It builds the API and Caddy images and scans all production images, failing on fixable HIGH or CRITICAL findings. Mail tests use a separate database to avoid cross-package mutation races.

On Windows, the Go race detector may require a working CGO C compiler. CI runs race tests on Ubuntu; use Linux or WSL when the local Windows toolchain cannot provide CGO.

## Deployment validation checklist

- Validate `deploy/docker-compose.prod.yml` with `docker compose config` using a local non-production environment file.
- Scan every production image, not just the API image. PostgreSQL and Caddy use custom images derived from digest-pinned bases, and CI scans each production image.
- Verify that the API and PostgreSQL have no host-published ports in production Compose; confirm Caddy obtains a valid TLS certificate.
- Confirm production config rejects missing HTTPS origins, SMTP TLS, application secret, or metrics token.
- Exercise sign-up/email verification, admin TOTP, account review, listing, order reservation/release/completion, and metrics protection in a disposable deployment.
- Generate a backup, verify its SHA-256/archive listing, and run `scripts/restore-test.ps1` against an isolated database before relying on recovery.
- Run dependency, container-image, load, and external infrastructure scans in the release pipeline appropriate to the deployment environment; results depend on the pinned base images and production network configuration.
