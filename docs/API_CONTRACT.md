# ZaminFlow API Contract

**Status: implemented.** The versioned API is served below `/api/v1`; liveness, readiness, and metrics are root-level operational endpoints. [openapi.yaml](./openapi.yaml) describes the request and response schemas.

## Conventions and security

- JSON uses camelCase. IDs are UUIDs; timestamps are RFC 3339; money is integer UZS; listing and order quantities are whole liters.
- Lists use `page` (default 1), `pageSize` (default 20, maximum 100), and return `{items,page,pageSize,total}`. Invalid query values return `400`.
- Malformed JSON, unknown JSON fields, and trailing JSON values return `400`. Well-formed bodies with invalid values return `422`. Error responses have the shape `{"error":{"code":"...","message":"...","requestId":"...","fieldErrors":[...]}}`.
- Authenticated APIs use `Authorization: Bearer <access-token>`. Access tokens expire; clients must use the refresh cookie to obtain a new one. Refresh tokens are not returned in JSON.
- Browser clients must send credentials. `POST /auth/refresh` and `POST /auth/sign-out` require an allowlisted `Origin`, the `zf_csrf` cookie, and an identical `X-CSRF-Token` header. The CSRF cookie is HMAC-signed and is not HttpOnly so browser code can copy it to the header. See the [frontend integration guide](./FRONTEND_INTEGRATION.md).
- Public sign-up permits only `farmer` and `processor`. Administrators are provisioned out of band and must enroll TOTP before normal admin access; recovery codes are shown once.
- Marketplace and role-dashboard requests check current database status and role. Farmers manage their own listings; processors browse active listings; orders are visible only to their two parties.

## Account and authentication routes

| Method and path | Access and behavior | Success |
|---|---|---|
| `GET /meta/roles` | Public sign-up roles; never includes admin | `{items:[{value,label}]}` |
| `GET /meta/regions` | Public; optional `page,pageSize` | Paginated region references with Uzbek, Russian, and English labels |
| `POST /auth/sign-up` | Public; creates a pending farmer/processor account and queues email verification | `201 {user,message}` |
| `POST /auth/email-verification/request` | Public, enumeration-safe | `202 {message}` |
| `POST /auth/email-verification/confirm` | Public, one-time token | `200 {emailVerified:true}` |
| `POST /auth/sign-in` | Public; admin must supply TOTP or recovery code | `200 {accessToken,expiresAt,user}`; sets refresh and CSRF cookies |
| `POST /auth/refresh` | Refresh cookie + CSRF checks | `200 {accessToken,expiresAt,user}`; rotates both cookies |
| `POST /auth/sign-out` | Bearer token + CSRF checks | `204`; revokes the current refresh-token family |
| `POST /auth/password-reset/request` | Public, enumeration-safe | `202 {message}` |
| `POST /auth/password-reset/confirm` | Public, one-time token; revokes sessions | `204` |
| `GET /auth/sessions` | Authenticated; caller's active sessions | Paginated session records |
| `DELETE /auth/sessions/{sessionId}` | Authenticated; revokes a caller-owned session | `204` |
| `POST /auth/change-password` | Authenticated; changes password and revokes other sessions | `204` |
| `POST /auth/totp/setup` | Admin enrollment-only or MFA-authenticated admin | `200 {secret,otpauthUri}` |
| `POST /auth/totp/confirm` | Admin enrollment-only; verifies first code | `200 {recoveryCodes}`; enrollment sessions are revoked, so sign in again |
| `POST /auth/totp/recovery-codes/rotate` | MFA-authenticated admin, verifies current factor | `200 {recoveryCodes}` |
| `GET /me` | Authenticated | `200 {user}` |
| `PATCH /me` | Authenticated; profile fields only, role/status are immutable here | `200 {user}` |
| `POST /me/resubmit` | Rejected applicant after correcting profile | `202 {user}` and returns account to pending |
| `DELETE /me` | Authenticated non-admin; records a deletion request, not an immediate delete | `202 {status:"requested"}`; active orders block the request |
| `GET /me/deletion-request` | Farmer or processor; returns the caller's latest deletion request and review decision | `200 {request}`; `404` if no request exists |

Email verification and password-reset links are delivered through the transactional outbox. Deletion requests are reviewed by administrators, but approval only authorizes separate manual retention/legal review. The API does not erase, anonymize, or suspend accounts through this workflow.

## Administrator routes

All routes below require admin role and completed MFA. Mutations are audit-logged.

| Method and path | Behavior and success |
|---|---|
| `GET /admin/deletion-requests?page&pageSize&status` | Lists requests with account profiles; defaults to `requested`; `status=all` includes history |
| `POST /admin/deletion-requests/{requestId}/approve` | Records approval for separate manual retention review; requests with active orders cannot be approved at review time; account data/status remain unchanged |
| `POST /admin/deletion-requests/{requestId}/reject` | Requires `{reason}`; records reviewer, timestamp, reason, and audit event |
| `GET /admin/applications?page&pageSize&role&regionCode` | Verified pending applicants; paginated result |
| `GET /admin/applications/{userId}` | `{user,reviews}` with applicant record and review history |
| `POST /admin/applications/{userId}/approve` | Optional `{reason}`; returns the updated user |
| `POST /admin/applications/{userId}/reject` | Required `{reason}`; returns the updated user |
| `POST /admin/users/{userId}/suspend` | Required `{reason}`; revokes user sessions; returns the updated user |
| `POST /admin/users/{userId}/reinstate` | Optional `{reason}`; returns the updated user |
| `GET /admin/audit-events?page&pageSize&actorId&action&targetId&targetType&from&to` | Filtered, paginated redacted audit events |
| `GET /dashboards/admin` | Admin aggregates and operational counts; see dashboard routes below |

## Marketplace routes

All routes require an approved user. Invalid role or ownership returns `403`; missing or inaccessible objects return `404`.

| Method and path | Access and behavior |
|---|---|
| `GET /listings?page&pageSize&regionCode&minLiters&sort` | Farmer sees own listings; processor searches active listings |
| `POST /listings` | Farmer; creates an active listing from `regionCode,availableLiters,fatPercent,pricePerLiterUzs,availableFrom,availableUntil`; returns `201 {listing}` |
| `GET /listings/{listingId}` | Listing owner or processor; returns a listing |
| `PATCH /listings/{listingId}` | Farmer owner; updates allowed listing fields; returns a listing |
| `DELETE /listings/{listingId}` | Farmer owner; archives listing when active reservations permit; returns `204` |
| `GET /orders?page&pageSize&status` | Lists only the caller's orders, newest first |
| `POST /orders` | Processor; `{listingId,liters,expectedPickupAt,note?}`; atomically reserves stock; returns `201 {order}` |
| `GET /orders/{orderId}` | Order party; includes status history |
| `POST /orders/{orderId}/accept` | Farmer party; optional `{reason}` recorded in history; returns `{order}` |
| `POST /orders/{orderId}/reject` | Farmer party; optional reason; releases reserved stock; returns `{order}` |
| `POST /orders/{orderId}/cancel` | Processor party before acceptance; optional `{reason}`; releases reserved stock; returns `{order}` |
| `POST /orders/{orderId}/status` | Party permitted by the transition; `{status,note?}`; returns `{order}` |
| `POST /orders/{orderId}/disputes` | Order party before completion; `{reason}`; returns `201 {status:"disputed"}` |
| `POST /orders/{orderId}/quality-reports` | Order party; `{measurements,outcome,note?}`; returns `201 {status:"recorded"}` |

Order creation locks and reserves available inventory in a database transaction. Rejection and eligible cancellation release the reservation; completion consumes it. Unit price and order total are snapshots. Dispute resolution is not automated; a disputed order remains for out-of-band review.

## Dashboards and operations

`GET /dashboards/farmer` and `/dashboards/processor` require the matching approved role. `GET /dashboards/admin` requires MFA-authenticated admin. All accept `period=7d|30d|90d|12m` (default `30d`) or an RFC 3339 `from` and `to` pair with a maximum span of 366 days. Custom ranges over 90 days use monthly buckets; shorter ranges use daily buckets. Responses contain KPI, time-series, and table aggregates; empty sets return zero/null values and empty arrays.

- `GET /healthz`: process liveness, root path.
- `GET /readyz`: database readiness probe, root path.
- `GET /metrics`: Prometheus request counters by route template, method, and status; root path. Requires `Authorization: Bearer <METRICS_TOKEN>` when configured. Production Caddy denies public access to this path.

## Error codes and status

Common error codes include `invalid_request`, `validation_failed`, `unauthenticated`, `forbidden`, `not_found`, `conflict`, `rate_limited`, `account_not_approved`, `account_rejected`, `account_suspended`, `invalid_credentials`, `refresh_token_reuse_detected`, `admin_mfa_required`, `csrf_validation_failed`, and `internal_error`. Operational middleware may also return `method_not_allowed`, `cors_origin_denied`, `cors_origin_required`, or `dependency_unavailable`. See each OpenAPI operation for its expected status codes. Unexpected server errors include a request ID and do not expose internal details.
