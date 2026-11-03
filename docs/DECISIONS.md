# Decisions and Assumptions

**Status: proposed; awaiting product approval.** These are Step 0 design decisions, not implemented behavior.

| Topic | Proposal | Approval needed |
|---|---|---|
| Product scope | Dairy listing, ordering, and fulfillment status in Uzbekistan; no payment or logistics service | Confirm |
| Roles | Exactly `farmer`, `processor`, `admin`; only the first two can self-register | Confirm |
| Sign-up status | New farmer/processor starts `pending`; email verification precedes admin review | Confirm |
| Account gate | All marketplace and role dashboard requests check current DB role/status; only approved accounts trade | Confirm |
| Review history | Every approval/rejection/resubmission is append-only in account review history | Confirm |
| Rejection | Reason required; applicant may correct profile and resubmit | Confirm |
| Suspension | Reason required; sign-in blocked and sessions revoked immediately | Confirm |
| Reinstatement | Suspended account returns to `approved`; rejected applicant uses resubmission | Confirm |
| Last admin | Prevent self-status change and suspension/deletion of last active admin | Confirm |
| Admin security | Mandatory TOTP, hashed one-time recovery codes, strict rate limits, all admin actions audit-logged; initial CLI-provisioned sign-in is restricted to MFA enrollment until confirmed | Confirm |
| Transaction flow | Place order -> farmer accepts/rejects -> farmer records fulfillment milestones -> processor confirms receipt or opens dispute | Confirm who records pickup/delivery |
| Disputes | Record dispute; initial release has no automated resolution | Confirm |
| Measurements | Whole liters; fat percentage as bounded decimal; integer UZS money | Confirm units and fractional liters need |
| Regions and STIR | Uzbek phone `+998` + 9 digits; region source and STIR validation to be chosen | Confirm accepted formats/source |
| Auth transport | 15-minute bearer access JWT; rotating hashed refresh cookie with CSRF checks; native body transport disabled by default | Confirm |
| Passwords | 12-128 characters, common-password rejection, Argon2id | Confirm |
| Pagination | `page`, `pageSize`, `total`, `items`; page size max 100 | Confirm |
| Dashboard period | Default trailing 30 days; selectable 7d/30d/90d/12m/custom, UTC buckets | Confirm |
| Market average | Volume-weighted completed platform orders over same period | Confirm this is not an external market quote |
| Account deletion | Request/anonymize subject to order, review, audit retention rather than immediate cascade delete | Legal/product retention policy needed |
| Out of scope | Payments, tax, logistics providers/GPS, chat, ratings, document uploads, external benchmarks, automated dispute resolution, frontend | Confirm |
| CORS | Production ZaminFlow web origins and localhost development origins; exact API hostname/cookie Domain depend on DNS | Confirm deployment domain |

See [DESIGN_REPORT.md](./DESIGN_REPORT.md) for the full approval workflow, endpoint inventory, KPI definitions, ER diagram, and phase plan.
