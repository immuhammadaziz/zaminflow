# Decisions and Assumptions

**Status: the baseline design was approved by the user.** This log records implemented choices and the issues that still require product, legal, or deployment input.

| Topic | Implemented decision | Follow-up |
|---|---|---|
| Product scope | Dairy listing, order, and fulfillment status in Uzbekistan; no payments or logistics service | None in backend scope |
| Roles | `farmer`, `processor`, and `admin`; only farmer/processor self-register | None |
| Account approval | Email verification precedes admin review; marketplace access requires database status `approved` | None |
| Review history | Approval, rejection, suspension, reinstatement, and resubmission are retained and audited | None |
| Admin security | CLI provisioning, required TOTP, one-time hashed recovery codes, rate limits, audit records | Operate TOTP/recovery through an approved secrets process |
| Order lifecycle | Processor places; farmer accepts/rejects and records fulfillment; processor completes after receipt | Operational teams should confirm who records pickup and delivery |
| Disputes | Parties can record a dispute; no automated resolution endpoint | Define the manual adjudication process |
| Measurements and money | Whole liters, bounded decimal fat percentage, integer UZS | Quality measurements currently accept numeric keys; standardize taxonomy/units before external reporting |
| Regions and identity | Uzbek phone format `+998` plus nine digits; region codes and translated labels are migration-seeded; STIR is stored as supplied | Choose an authoritative region source and confirm STIR format/validation |
| Auth transport | Short-lived bearer JWT, rotating hashed refresh cookie, signed CSRF double-submit token for browser refresh/sign-out | No native refresh-token body transport |
| Passwords | 12–128 characters, common-password screening, Argon2id | Tune Argon2 parameters using production capacity |
| Pagination | 1-based `page`, `pageSize` default 20 and maximum 100, `total`, `items` | None |
| Dashboard period | Default trailing 30 days; 7d/30d/90d/12m or custom RFC 3339 range, UTC buckets | No custom range over 366 days |
| Market average | Volume-weighted completed platform orders in the same period; not an external benchmark | Communicate platform-only meaning in UI |
| Account deletion | `DELETE /me` records a request and blocks it when active orders exist; MFA-protected admins review and audit approve/reject decisions; no data is erased | Legal/product must define retention/anonymization and the separate manual process before any erasure |
| Deployment | Caddy terminates HTTPS; API/PostgreSQL are not host-published; SMTP STARTTLS and metrics token are required in production | Supply DNS, SMTP, secret management, monitoring, and off-site backup infrastructure |
| Out of scope | Payments, tax computation, logistics, chat, ratings, document uploads, external benchmarks, automated dispute resolution, and frontend | Separate product initiatives |

See [DESIGN_REPORT.md](./DESIGN_REPORT.md), [API_CONTRACT.md](./API_CONTRACT.md), and [DEPLOYMENT.md](./DEPLOYMENT.md).
