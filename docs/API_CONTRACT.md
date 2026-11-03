# ZaminFlow API Contract (Draft)

**Status: proposed contract for Step 0 approval.** OpenAPI is intended to be the source of truth after approval. No endpoint is implemented yet.

## Global conventions

- Base path `/api/v1`; JSON request/response; camelCase fields; UUID identifiers.
- Date-times are RFC 3339 UTC. Money is integer UZS; quantity is integer liters; enums are stable lowercase snake_case strings.
- Lists use `page` (default 1), `pageSize` (default 20, maximum 100), optional documented `filter` and `sort` query parameters, and `{items, page, pageSize, total}` response.
- Error body: `{"error":{"code":"...","message":"...","requestId":"...","fieldErrors":[{"field":"...","code":"...","message":"..."}]}}`.
- Status meanings: `400` malformed/unknown JSON or query; `401` missing/invalid authentication; `403` authenticated but forbidden/unapproved; `404` absent or inaccessible resource; `409` duplicate/conflicting state transition; `422` valid JSON but invalid field values; `429` rate limit; `5xx` unexpected server/dependency failure with request ID only.
- Requests reject unknown JSON fields and trailing JSON values. Success responses use stable typed shapes; errors never expose stack traces or secrets.
- Authenticated requests use `Authorization: Bearer <accessToken>`. Refresh uses the secure refresh cookie and CSRF header for browser clients. Admin sign-in requires TOTP/recovery code.
- Any marketplace route requires current DB status `approved`; DB role and status are authoritative. Ownership is checked for every resource.

## Common schemas

### User

`{id, role, accountStatus, email, emailVerified, fullName, phone, organizationName, regionCode, stir, rejectionReason?, createdAt, updatedAt}`.
`role` is `farmer | processor | admin`. `accountStatus` is `pending | approved | rejected | suspended`. `rejectionReason` is visible only to the applicant and admins.

### Listing

`{id, farmerId, regionCode, availableLiters, fatPercent, pricePerLiterUzs, availableFrom, availableUntil, status, createdAt, updatedAt}`.
Initial status is `active` or `draft` per approved workflow; only active, non-expired inventory can be ordered. Exact field limits are to be finalized before implementation.

### Order

`{id, listingId, farmerId, processorId, liters, pricePerLiterUzs, totalUzs, status, expectedPickupAt, note?, createdAt, updatedAt, history[]}`.
Unit price and total are immutable snapshots. `history[]` contains status, actorId, timestamp, and optional note.

### Dashboard

`{period:{from,to,bucket}, kpis:{...}, charts:{...}, tables:{...}}`. KPI values are calculated from stored rows; empty data returns zero or null where division is undefined, with empty arrays for series/tables.

## Endpoint contract inventory

The tables below enumerate the proposed operations and their access, inputs, success shape, and principal error cases. The OpenAPI draft provides machine-readable path/method/security declarations. More detailed field limits and examples are to be finalized after product approval.

### Public/authentication/profile

| Method/path | Auth and input | Success | Errors |
|---|---|---|---|
| `GET /meta/roles` | Public; no body | `{items:[{value,label}]}` containing only farmer/processor | `429` |
| `GET /meta/regions` | Public; optional `page,pageSize` | Paginated region references | `400,429,5xx` |
| `POST /auth/sign-up` | Public; `fullName,email,password,role,phone,organizationName,regionCode,stir`; role only farmer/processor | `201 {user, message}`; user status pending | `400,409,422,429,5xx` |
| `POST /auth/email-verification/request` | Public; `{email}` | `202 {message}` enumeration-safe | `400,422,429,5xx` |
| `POST /auth/email-verification/confirm` | Public; `{token}` | `200 {emailVerified:true}` | `400,409,422,429` |
| `POST /auth/sign-in` | Public; `{email,password,totpCode?}` | `200 {accessToken,expiresAt,user}`; refresh cookie set | `401` generic, `403 account_suspended`, `422,429` |
| `POST /auth/refresh` | Refresh cookie (or opt-in native mode); CSRF checks for browser | `200 {accessToken,expiresAt,user}`; rotated cookie | `401,403,409 refresh_reuse_detected,429` |
| `POST /auth/sign-out` | Authenticated; CSRF for cookie mode | `204`; current family revoked | `401,403` |
| `POST /auth/password-reset/request` | Public; `{email}` | `202 {message}` enumeration-safe | `400,422,429` |
| `POST /auth/password-reset/confirm` | Public; `{token,newPassword}` | `204`; sessions revoked | `400,409,422,429` |
| `GET /auth/sessions` | Authenticated; pagination | Paginated caller-owned sessions | `401,400` |
| `DELETE /auth/sessions/{sessionId}` | Authenticated; UUID path | `204` | `401,404` |
| `POST /auth/change-password` | Authenticated; `{currentPassword,newPassword}` | `204`; other sessions revoked | `401,403,422,429` |
| `GET /me` | Authenticated | `200 {user}` including live account status | `401` |
| `PATCH /me` | Authenticated; allowed profile fields only | `200 {user}` | `401,403,404,422,409` |
| `POST /me/resubmit` | Rejected farmer/processor; corrected profile already saved | `202 {user}` with pending status | `401,403,409,422` |
| `DELETE /me` | Authenticated; `{confirmation}` proposed | `202 {requestId,status}` | `401,403,409` |
| `POST /auth/totp/setup` | Admin authenticated; no body | `200 {secret,otpauthUri}` shown once | `401,403,409` |
| `POST /auth/totp/confirm` | Admin; `{code}` | `200 {recoveryCodes[]}` shown once | `401,403,409,422,429` |
| `POST /auth/totp/recovery-codes/rotate` | MFA-authenticated admin; `{code}` | `200 {recoveryCodes[]}` | `401,403,422,429` |

### Admin

All admin endpoints require admin role and completed MFA; all mutations are audited.

| Method/path | Input | Success | Errors |
|---|---|---|---|
| `GET /admin/applications` | `page,pageSize,status,role,regionCode,sort`; verified pending only by default | Paginated applicant summaries including oldest wait metadata | `401,403,400` |
| `GET /admin/applications/{userId}` | UUID path | Applicant, status and review history | `401,403,404` |
| `POST /admin/applications/{userId}/approve` | Optional `{reason}` | `200 {user,review}` | `401,403,404,409` |
| `POST /admin/applications/{userId}/reject` | `{reason}` required | `200 {user,review}` | `401,403,404,409,422` |
| `POST /admin/users/{userId}/suspend` | `{reason}` required | `200 {user,review}` and sessions revoked | `401,403,404,409,422` |
| `POST /admin/users/{userId}/reinstate` | Optional `{reason}` | `200 {user,review}`; approved status | `401,403,404,409` |
| `GET /admin/audit-events` | `page,pageSize,actorId,action,targetId,from,to,sort` | Paginated redacted events | `401,403,400` |
| `GET /dashboards/admin` | Period query | Admin dashboard | `401,403,400` |

### Marketplace and role dashboards

All require authenticated approved account. Listings: processor can search active listings; farmer can read/manage own listings only. Orders are visible only to their farmer and processor parties. Mutation transitions are atomic.

| Method/path | Role/input | Success | Errors |
|---|---|---|---|
| `GET /listings` | Processor; filters region, min/max price, min liters, availability; pagination/sort | Paginated active listings | `401,403 account_not_approved/account_rejected/account_suspended,400` |
| `POST /listings` | Farmer; quantity, fat %, integer price, region, availability | `201 {listing}` | `401,403,409,422` |
| `GET /listings/{listingId}` | Approved farmer owner or processor | `{listing}` | `401,403,404` |
| `PATCH /listings/{listingId}` | Farmer owner; mutable listing fields | `{listing}` | `401,403,404,409,422` |
| `DELETE /listings/{listingId}` | Farmer owner | `204` archived | `401,403,404,409` |
| `GET /orders` | Farmer or processor; status/date filters; pagination/sort | Paginated own-party orders | `401,403,400` |
| `POST /orders` | Processor; `{listingId,liters,expectedPickupAt,note?}` | `201 {order}` | `401,403,404,409,422` |
| `GET /orders/{orderId}` | Farmer or processor party | `{order}` with history | `401,403,404` |
| `POST /orders/{orderId}/accept` | Farmer party; optional note | `{order}` | `401,403,404,409` |
| `POST /orders/{orderId}/reject` | Farmer party; `{reason?}` | `{order}` | `401,403,404,409,422` |
| `POST /orders/{orderId}/cancel` | Processor party; optional reason; only before acceptance | `{order}` | `401,403,404,409,422` |
| `POST /orders/{orderId}/status` | Correct party; `{status,note?}` next allowed fulfillment state only | `{order}` | `401,403,404,409,422` |
| `POST /orders/{orderId}/disputes` | Either party; `{reason}` | `201 {dispute}` | `401,403,404,409,422` |
| `POST /orders/{orderId}/quality-reports` | Either party; proposed `{measurements,outcome,note?}` | `201 {qualityReport}` | `401,403,404,409,422` |
| `GET /dashboards/farmer` | Farmer; period query | Farmer dashboard aggregate | `401,403,400` |
| `GET /dashboards/processor` | Processor; period query | Processor dashboard aggregate | `401,403,400` |

### Platform operations

| Method/path | Access | Success | Errors |
|---|---|---|---|
| `GET /healthz` | Public liveness | `200 {status:"ok"}` | `503` if process unhealthy |
| `GET /readyz` | Public or internal; DB probe | `200 {status:"ready"}` | `503` if dependency unavailable |
| `GET /metrics` | Internal/protected only | Prometheus metrics | `401/403` or network denied |

## Stable proposed error codes

`invalid_request`, `validation_failed`, `unauthenticated`, `forbidden`, `not_found`, `conflict`, `rate_limited`, `account_not_approved`, `account_rejected`, `account_suspended`, `email_not_verified`, `invalid_credentials`, `refresh_token_reuse_detected`, `invalid_state_transition`, `ownership_required`, `last_active_admin`, `admin_mfa_required`, `internal_error`.

The exact field validations, sorting allowlists, and additional endpoint-specific error cases must be completed in OpenAPI before implementation. This draft intentionally awaits approval of product decisions rather than treating proposed details as final.
