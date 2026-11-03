# Production secrets and security operations

## Secret storage

Production Compose reads secret files from `SECRETS_DIR`, which defaults to `/opt/zaminflow/secrets`. Create a separate directory and independent secret set for each environment. Do not copy production secrets into a developer machine, commit them, put them in `.env.production`, or pass them as Compose interpolation values.

On the production host, create the directory as root with restrictive permissions. Secret files must be owned by `root:root` and mode `0600`; the directory should be mode `0700`. The expected filenames are:

| Compose secret | File |
|---|---|
| `zaminflow_app_secret` | `zaminflow_app_secret` |
| `zaminflow_db_user` | `zaminflow_db_user` |
| `zaminflow_db_password` | `zaminflow_db_password` |
| `zaminflow_metrics_token` | `zaminflow_metrics_token` |
| `zaminflow_admin_bootstrap_secret` | `zaminflow_admin_bootstrap_secret` (only during initial provisioning) |

Generate independent random values with an approved password manager or secret manager. `APP_SECRET` and `METRICS_TOKEN` must each contain at least 32 characters. Keep the database username and password in their respective files. The app builds its PostgreSQL connection string at startup; production must not set `DATABASE_URL` or plaintext secret environment variables.

The Compose file-backed secrets are host files, not Swarm-managed encrypted secrets. The API/migration/admin container entrypoint copies mounted secrets to a small in-memory filesystem, assigns them to UID 65532, and drops privileges before starting the Go process. PostgreSQL's root entrypoint reads its credential files, places its password file on a small in-memory filesystem, then starts PostgreSQL as its unprivileged `postgres` user. Do not change these containers to run their application processes as root.

Keep the secrets directory on encrypted host storage where available and restrict host access. Protect host backups separately. For encrypted secret backups, use SOPS with age and environment-specific encrypted files. Keep the age decryption identity offline or in a separately controlled secrets vault; never store the age private key beside the encrypted backup. Verify that authorized recovery personnel can decrypt a backup before relying on it.

## Rotation

Coordinate rotations during a maintenance window, preserve a verified rollback copy in the approved encrypted vault, and update the host secret file atomically with root ownership and mode `0600`. Restart affected containers after each rotation and verify health/readiness and logs without printing secret contents.

- **Database user/password:** PostgreSQL only reads its initialization credential files when creating a new data directory; changing those files alone does not change an existing database role. Change the database role password through a privileged PostgreSQL session, update both host secret files, then recreate the API and migration containers. Verify connectivity before ending the maintenance window. For username changes, create and validate a replacement role before switching `zaminflow_db_user`; do not rename or remove the active role until all services have moved.
- **Application secret:** Replacing this key invalidates signed access tokens and makes existing encrypted MFA secrets and recovery-code verifiers unusable. Plan to require users to sign in again and re-enroll MFA; verify recovery/admin access before retiring the old key. The current implementation does not support a dual-key transition.
- **Metrics token:** Update `zaminflow_metrics_token` and the monitoring client configuration together, recreate the API container, then verify authorized metrics access from the private monitoring network and denial from public ingress.
- **SMTP credentials:** SMTP is intentionally deferred from the initial deployment setup. When production SMTP is configured, add its credentials as mounted file secrets and wire those files into Compose; do not put SMTP passwords in `.env.production`.
- **Administrator bootstrap secret:** This is a one-time credential. After successful initial admin provisioning, remove the host file and its encrypted backup copy. Do not keep it in the regular production secret set.

Review host access, secret file permissions, Compose service configuration, and the encrypted recovery copy after every rotation.
