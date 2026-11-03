# Production deployment and recovery

## Prerequisites

- A Linux host with Docker Engine and the Compose plugin.
- Production DNS and SMTP setup are deferred. Complete and validate both before exposing the service to production users.
- A secure process for generating, storing, rotating, and backing up production secrets. Follow [SECURITY.md](./SECURITY.md); do not commit `.env.production` or secrets.
- PostgreSQL and Caddy use custom images based on digest-pinned upstream images. Their Dockerfiles update Alpine security packages; PostgreSQL runs directly as its non-root `postgres` user without the vulnerable upstream `gosu` binary, and Caddy is rebuilt with patched Go dependencies. CI builds and scans all three production images and exercises migrations and integration tests against the custom PostgreSQL image.
- Review the pinned upstream digests and vulnerability scan results before updating either base image. The CI threshold fails builds for fixable HIGH/CRITICAL findings.

## Configure and deploy

Follow [DEPLOY.md](./DEPLOY.md) to prepare the root `.env.production` and root-owned file secrets before running Compose. The environment file contains non-secret settings only. Separate environment secret directories and secret values are required.

Validate interpolation and Compose configuration:

   ```sh
   docker compose --env-file .env.production -f deploy/docker-compose.prod.yml config
   ```

Start the stack:

   ```sh
   docker compose --env-file .env.production -f deploy/docker-compose.prod.yml up --build -d
   docker compose --env-file .env.production -f deploy/docker-compose.prod.yml ps
   ```

Compose applies database migrations before starting the API. PostgreSQL has no published host port and is on an internal network. The API has no published host port; Caddy is the only public service and supplies the client IP header used for IP-based throttling. Caddy denies `/metrics` externally; configure internal monitoring to access the API container directly with `Authorization: Bearer $METRICS_TOKEN`.

Verify `https://$API_DOMAIN/healthz` and `/readyz` after DNS and SMTP have been configured. Confirm a sign-up and test email delivery before enabling normal production traffic.

## Bootstrap the first administrator

Use the separate overlay and temporary bootstrap secret as described in [DEPLOY.md](./DEPLOY.md):

```sh
docker compose --env-file .env.production \
  -f deploy/docker-compose.prod.yml \
  -f deploy/docker-compose.admin.yml run --rm admin
```

Remove the bootstrap secret from the server and its encrypted backup after provisioning. Sign in, enroll TOTP, save recovery codes in an approved secrets vault, then sign in again. Provision future admin accounts only through an audited operational process; do not make the admin tool publicly accessible.

## Backups

Run from the repository root on the host with Docker access:

```powershell
.\scripts\backup.ps1
```

The script uses `pg_dump` custom format, checks that the archive can be listed, copies it to `backups\`, and writes a SHA-256 sidecar. The archive contains personal/business and trading data: restrict filesystem permissions, encrypt it at rest and in transit, store it separately from the host, and apply an approved retention schedule. The script does not implement off-site replication or encryption.

## Restore testing and recovery

Test a backup against a uniquely named temporary database inside the running PostgreSQL container:

```powershell
.\scripts\restore-test.ps1 -BackupPath .\backups\<backup-name>.dump
```

This verifies the checksum when present, restores the archive, checks the expected schema, and drops only its own temporary database.

For an actual restore, first stop API writes and take a separate copy of the current database. Then restore to the intended existing database name:

```powershell
.\scripts\restore.ps1 -BackupPath .\backups\<backup-name>.dump -TargetDatabase zaminflow
```

The restore script verifies the sidecar checksum when present and uses `pg_restore --clean --if-exists`; it overwrites objects in the selected database. It prompts for the target identifier unless `-Force` is supplied. Validate readiness and application behavior before reopening traffic. Never test a recovery procedure against production data or the production target.

## Updates and shutdown

Deploy code changes during a maintenance window when migrations are not backward compatible. Keep a verified pre-deploy backup and inspect Compose logs:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.prod.yml logs --tail=200 api migrate
```

To stop services without deleting persistent data:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.prod.yml down
```

Do not use `down -v` on production; it removes persistent database and Caddy volumes.
