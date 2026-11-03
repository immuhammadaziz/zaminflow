# Approved screen inventory — constrained by current API

All screens are mobile-first and must implement loading/skeleton, empty, error/retry with request ID, forbidden, offline/slow-network, and success states appropriate to their data. No table or metric is populated with fabricated data. The list below is constrained to current routes; entries marked **backend gap** are not implemented until a matching contract is approved.

## Public and account access

| Screen                               | Purpose / API                                                                           | Key states                                                                                                    |
| ------------------------------------ | --------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| Landing                              | Product story, SEO, links; no business data                                             | Static marketing content; locale                                                                              |
| Sign up                              | `GET /meta/roles`, `/meta/regions`; `POST /auth/sign-up`                                | Field-level 422, duplicate/conflict, 429, success/verify-next                                                 |
| Sign in                              | `POST /auth/sign-in`; then `/me` via auth response/refresh                              | Invalid credentials, admin MFA challenge, enrollment-only                                                     |
| Verify email                         | `POST /auth/email-verification/confirm`; `POST .../request` resend                      | Invalid/expired token, rate-limited, confirmed                                                                |
| Forgot/reset password                | `POST /auth/password-reset/request`, `POST .../confirm`                                 | Enumeration-safe accepted, invalid/expired token, field errors                                                |
| Awaiting approval / email unverified | `GET /me`, resend request; refetch on focus and modest interval                         | Separate unverified and verified-pending guidance                                                             |
| Rejected / resubmit                  | `GET /me`, `PATCH /me`, `POST /me/resubmit`                                             | Rejection reason, editable profile, submit/error/pending                                                      |
| Suspended                            | `GET /me`; global handling of `account_suspended`                                       | Blocked explanation and sign-out                                                                              |
| Legal placeholders, 404, 500         | Static legal copy requires owner/legal approval; generic error and request ID           | Never imply approved legal policy without supplied copy                                                       |
| Shared profile                       | `GET/PATCH /me`                                                                         | View/edit/save/validation                                                                                     |
| Security / sessions                  | `GET /auth/sessions`, `DELETE /auth/sessions/{sessionId}`, `POST /auth/change-password` | Paginated sessions, revoke confirm, password change/errors; API does not identify the current browser session |
| Account deletion request             | `DELETE /me`, `GET /me/deletion-request`                                                | Explain request is not erasure; requested/approved/rejected/no request                                        |

## Farmer (approved)

| Screen                   | API                                                                          | Notes                                                                                  |
| ------------------------ | ---------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Dashboard                | `GET /dashboards/farmer?period=...`                                          | Only API KPI/chart/table values; zero/null empty states                                |
| Listings                 | `GET /listings`; `POST /listings`; `GET/PATCH/DELETE /listings/{id}`         | Create and edit offer, active/pause, archive confirmation; deletion conflicts surfaced |
| Incoming orders / detail | `GET /orders`, `GET /orders/{id}`, accept/reject, `POST /orders/{id}/status` | Role-safe transitions, reason confirmation, timeline                                   |
| Quality reports          | `POST /orders/{id}/quality-reports`; order `qualityOutcome`                  | **Backend gap:** no report-list/history route                                          |
| Farms                    | None                                                                         | **Backend gap:** one profile organization field only, no farm resource                 |

## Processor (approved)

| Screen                       | API                                                                     | Notes                                                                          |
| ---------------------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Dashboard                    | `GET /dashboards/processor?period=...`                                  | Real KPIs/charts/upcoming orders only                                          |
| Market browse                | `GET /listings` with supported `regionCode`, `minLiters`, `sort`        | **Backend gap:** no fat/price range filters; no seller profile details         |
| Listing detail / place order | `GET /listings/{id}`, `POST /orders`                                    | Whole liters and available inventory; no blind retry on timeout                |
| Orders / tracking / receipt  | `GET /orders`, `GET /orders/{id}`, cancel, status `completed`, disputes | Party-scoped; explicit confirmation and conflict refresh                       |
| Suppliers                    | No dedicated route                                                      | **Backend gap:** no directory/profile; listing exposes seller UUID only        |
| Quality                      | `POST /orders/{id}/quality-reports`                                     | **Backend gap:** no history/read route; only order quality outcome is returned |

## Admin (MFA required)

| Screen                                     | API                                                           | Notes                                                                                                                            |
| ------------------------------------------ | ------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Initial MFA enrollment / sign-in challenge | `POST /auth/totp/setup`, `/confirm`, `POST /auth/sign-in`     | Show otpauth QR and manual key, one-time recovery codes and save acknowledgment                                                  |
| Dashboard                                  | `GET /dashboards/admin?period=...`                            | Operational counts, oldest applicant, chart and tables                                                                           |
| Application queue/detail                   | `GET /admin/applications`, `GET /admin/applications/{userId}` | Role/region filters, API order and age highlight; server-side ordering must be verified                                          |
| Approve/reject                             | `POST /admin/applications/{userId}/approve                    | reject`                                                                                                                          | Reject requires reason; confirm decisions           |
| User directory                             | No list endpoint                                              | **Backend gap:** cannot build without a documented list/search API                                                               |
| Suspend/reinstate                          | `POST /admin/users/{userId}/suspend                           | reinstate`                                                                                                                       | Required reason for suspension; reason/confirmation |
| Audit log                                  | `GET /admin/audit-events`                                     | Supported filters and pagination; redacted fields only                                                                           |
| Deletion requests                          | `GET /admin/deletion-requests`, approve/reject                | Clearly state approval does not delete or restrict account                                                                       |
| Admin security                             | sessions, change password                                     | MFA-secure actions; recovery-code rotation is deferred because the OpenAPI 200 response omits the one-time recovery-code payload |
