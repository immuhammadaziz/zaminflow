# Production deployment

This guide covers preparing the production host and launching the Compose stack. DNS and production SMTP are deliberately deferred; do not expose the service until those are configured and validated.

## Prepare configuration and secrets

1. On a Linux host with Docker Engine and the Compose plugin, copy `deploy/.env.production.example` to the repository-root `.env.production`. Set non-secret values including `API_DOMAIN`, `DB_NAME`, exact HTTPS `CORS_ALLOWED_ORIGINS`, `WEB_BASE_URL`, and `SECRETS_DIR`. DNS and SMTP settings can be finalized later, but production traffic must not be enabled until they are operational.
2. Create the secret files described in [SECURITY.md](./SECURITY.md) under `SECRETS_DIR`. For example, with `SECRETS_DIR=/opt/zaminflow/secrets`, filenames must exactly match the documented identifiers. Ensure the files are owned by root with mode `0600`, and the directory is root-owned mode `0700`.
3. Keep secret values out of `.env.production`. That file is for non-secret configuration and may be passed to Compose. Use separate secret directories and independent values for staging and production.
4. Validate the merged Compose configuration:

   ```sh
   docker compose --env-file .env.production -f deploy/docker-compose.prod.yml config
   ```

## Deploy and verify

```sh
docker compose --env-file .env.production -f deploy/docker-compose.prod.yml up --build -d
docker compose --env-file .env.production -f deploy/docker-compose.prod.yml ps
docker compose --env-file .env.production -f deploy/docker-compose.prod.yml logs --tail=100 postgres migrate api
```

Migrations run before the API starts. PostgreSQL is not published to the host, and Caddy is the only service with public ports. Before enabling public traffic, configure DNS, SMTP, private monitoring, encrypted off-host backups, and the approved backup-retention and legal data-retention policies. Confirm `/healthz` and `/readyz` over the intended HTTPS hostname and validate delivery of verification/reset email.

## Bootstrap the first administrator

Temporarily add `ADMIN_EMAIL` and `ADMIN_FULL_NAME` to `.env.production`, and create the one-time `zaminflow_admin_bootstrap_secret` file in the secrets directory. Run the separate admin overlay:

```sh
docker compose --env-file .env.production \
  -f deploy/docker-compose.prod.yml \
  -f deploy/docker-compose.admin.yml run --rm admin
```

After the command succeeds, remove the bootstrap secret file from the host and its encrypted backup, then remove the temporary administrator variables from `.env.production`. Sign in, enroll TOTP, save recovery codes in the approved secrets vault, and sign in again. The admin tool is not part of the regular production stack.

## Backup, restore, and updates

See [DEPLOYMENT.md](./DEPLOYMENT.md) for the PowerShell backup, restore-test, and restore procedures. Backups contain personal and business data; encrypt them, store them off-host, restrict access, and follow the retention policy approved by the operator. Do not run restore tests against the production database.

For updates, keep a verified pre-deploy backup and inspect migration/application logs. Use `docker compose down` to stop without deleting persistent volumes. Never use `down -v` on production.

## Secret rotation

Follow the per-secret rotation steps in [SECURITY.md](./SECURITY.md). In particular, changing the PostgreSQL password file does not change the credential of an initialized database; rotate the database role and both runtime files as one coordinated operation.
