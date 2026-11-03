# Step 0 — Backend understanding

**Status:** Reviewed 3 November 2026. Backend endpoints and local integration tests are present. A farmer dashboard KPI defect found during review was corrected and verified. Step 1 design approval was received; Phase 2 frontend implementation may proceed.

## Evidence and boundaries

Reviewed the backend README, decision log, API conventions, OpenAPI 3.0.3 contract, API contract, frontend integration guide, security and deployment documentation, environment examples, Compose files, migrations, and relevant auth/marketplace implementation. `docs/DATA_MODEL.md` and a seed/demo-account command are absent. The local Compose stack includes PostgreSQL, API, migrations, Mailpit and optional Swagger UI; it does not include the web app.

Validation after the dashboard correction:

- Applied all three migrations to fresh disposable PostgreSQL databases.
- Ran `go test -race ./...` and `go vet ./...` using Go 1.27.3 in Linux against those databases; both passed.
- Previously in this session, built the file-secret API and PostgreSQL images, applied migrations, and observed API readiness with temporary test-only credentials.
- The complete Mailpit sign-up/verification/admin journey and seeded-account dashboards were not run: no seed command or demo accounts exist, and production SMTP/DNS are deferred.

This review uses the checked-in OpenAPI and source as the current implementation evidence. The API contract is authoritative for the client; source-derived response details below are recorded because some OpenAPI response schemas are too broad to generate useful types.

## API inventory

All versioned routes are below `/api/v1`. “Approved” means the current database account state must be `approved`; admin routes require role `admin` and completed MFA.

| Method and route                                                                                                                        | Access / state                                                             | Product surface                                                       |
| --------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| `GET /meta/roles`, `GET /meta/regions`                                                                                                  | Public                                                                     | Sign-up role and region choices                                       |
| `POST /auth/sign-up`                                                                                                                    | Public; farmer/processor only                                              | Registration                                                          |
| `POST /auth/sign-in`, `POST /auth/refresh`, `POST /auth/sign-out`                                                                       | Public / refresh cookie plus CSRF / bearer plus CSRF                       | Sign-in, session restore and sign-out                                 |
| `POST /auth/email-verification/request`, `POST /auth/email-verification/confirm`                                                        | Public                                                                     | Verification request/resend and confirmation                          |
| `POST /auth/password-reset/request`, `POST /auth/password-reset/confirm`                                                                | Public                                                                     | Forgot/reset password                                                 |
| `GET /auth/sessions`, `DELETE /auth/sessions/{sessionId}`, `POST /auth/change-password`                                                 | Authenticated                                                              | Security settings                                                     |
| `POST /auth/totp/setup`, `POST /auth/totp/confirm`, `POST /auth/totp/recovery-codes/rotate`                                             | Admin; enrollment-only token for initial setup, completed MFA for rotation | Admin MFA and recovery codes                                          |
| `GET /me`, `PATCH /me`, `POST /me/resubmit`, `DELETE /me`, `GET /me/deletion-request`                                                   | Authenticated; deletion request is non-admin; resubmit requires rejected   | Account state, profile, resubmission, deletion-request status         |
| `GET /admin/applications`, `GET /admin/applications/{userId}`                                                                           | Admin + MFA                                                                | Applicant queue and review detail                                     |
| `POST /admin/applications/{userId}/approve`, `POST /admin/applications/{userId}/reject`                                                 | Admin + MFA                                                                | Approval/rejection decisions                                          |
| `POST /admin/users/{userId}/suspend`, `POST /admin/users/{userId}/reinstate`                                                            | Admin + MFA                                                                | Account status operations                                             |
| `GET /admin/audit-events`                                                                                                               | Admin + MFA                                                                | Audit log                                                             |
| `GET /admin/deletion-requests`, `POST /admin/deletion-requests/{requestId}/approve`, `POST /admin/deletion-requests/{requestId}/reject` | Admin + MFA                                                                | Deletion request review; approval is not erasure or suspension        |
| `GET /listings`, `GET /listings/{listingId}`                                                                                            | Farmer or processor; approved                                              | Farmer's own listings or processor's currently available market       |
| `POST /listings`, `PATCH /listings/{listingId}`, `DELETE /listings/{listingId}`                                                         | Farmer; approved                                                           | Create/update/pause-or-publish/archive listing                        |
| `GET /orders`, `GET /orders/{orderId}`                                                                                                  | Farmer or processor; approved; party-scoped                                | Orders and timelines                                                  |
| `POST /orders`                                                                                                                          | Processor; approved                                                        | Place order and reserve stock                                         |
| `POST /orders/{orderId}/accept`, `/reject`                                                                                              | Farmer party; approved                                                     | Respond to placed order                                               |
| `POST /orders/{orderId}/cancel`                                                                                                         | Processor party; approved; before acceptance                               | Cancel order                                                          |
| `POST /orders/{orderId}/status`                                                                                                         | Farmer or processor party; approved; transition-specific actor             | Fulfillment milestones and receipt                                    |
| `POST /orders/{orderId}/disputes`, `/quality-reports`                                                                                   | Either order party; approved                                               | Record dispute or quality report                                      |
| `GET /dashboards/farmer`, `/dashboards/processor`                                                                                       | Matching role; approved                                                    | Farmer and processor dashboards                                       |
| `GET /dashboards/admin`                                                                                                                 | Admin + MFA                                                                | Admin dashboard                                                       |
| `GET /healthz`, `/readyz`, `/metrics`                                                                                                   | Operational; metrics requires bearer token                                 | Health/readiness and internal metrics; do not expose metrics publicly |

Pagination is 1-based (`page=1`, `pageSize=20`, max 100), returning `{items,page,pageSize,total}`. Invalid query values are `400`; malformed/unknown JSON fields are `400`; valid JSON with invalid field values is `422`. Lists accept documented filters only; unrecognized query keys are ignored.

## Session, account, and authorization behavior

- Access JWT lifetime is 15 minutes; rotating refresh cookie lifetime is 30 days. Access tokens belong in memory only. Refresh tokens are HttpOnly cookies and are never returned in JSON.
- Browser refresh and sign-out require `credentials: include`, exact allowlisted `Origin`, the `zf_csrf` cookie, and matching `X-CSRF-Token`. The refresh cookie is scoped to `/api/v1/auth`; CSRF cookie is readable by the browser and scoped to `/`. Refresh rotates both cookies; reuse revokes the family and returns `refresh_token_reuse_detected`.
- Local origins configured by `.env.example`: `http://localhost:3000` and `http://localhost:5173`, plus apex and `www` production origins. Production example config uses `https://zaminflow.uz` and `https://www.zaminflow.uz`. API is `https://api.zaminflow.uz`; shared cookie domain is `.zaminflow.uz` only when the web and API are on that trusted site.
- After sign-in/refresh, call `GET /me` and route by `role`, `accountStatus`, and `emailVerified`. Pending trading requests are denied; rejected and suspended statuses are authoritative on server requests. Rejected users may `PATCH /me` and `POST /me/resubmit`. Resend verification is enumeration-safe.
- Admin is provisioned out-of-band. Initial sign-in yields an enrollment-only token; call TOTP setup/confirm, display one-time recovery codes, then sign in again with TOTP or a recovery code.

## Validation and enum rules

| Field             | Backend rule                                                                                                                                   |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Role              | `farmer` or `processor` for public sign-up; never offer `admin`                                                                                |
| Full name         | 2–160 Unicode characters                                                                                                                       |
| Organization name | 2–200 Unicode characters                                                                                                                       |
| Email             | Valid address, maximum 254 bytes; normalized lowercase                                                                                         |
| Phone             | Exactly `+998` followed by nine ASCII digits                                                                                                   |
| STIR              | Exactly nine ASCII digits                                                                                                                      |
| Region            | Existing region code, 1–32 characters; codes and Uzbek/Russian/English labels are migration data                                               |
| Password          | 12–128 characters; four known common values rejected; Argon2id                                                                                 |
| Listing           | Whole liters > 0, fat percentage 0–100, integer UZS price > 0, `availableUntil > availableFrom`; edits allow only active/paused status changes |
| Order             | Whole liters > 0; quantity reservation and price/total snapshot are atomic; pickup time is RFC 3339                                            |

Roles: `farmer`, `processor`, `admin`. Account states: `pending`, `approved`, `rejected`, `suspended`. Applicants must verify email before admin review. Rejected applicants can resubmit to `pending`; admin can approve/reject verified applicants and suspend/reinstate accounts.

Listing states: `draft`, `active`, `paused`, `sold_out`, `expired`, `archived`. Public creation is active; owners can switch active/paused and archive via DELETE. Only active and currently available processor-visible listings can be ordered.

Order states: `placed -> accepted | rejected | cancelled`; accepted orders progress `ready_for_pickup -> picked_up -> delivered -> completed`; eligible in-progress orders can become `disputed`. Farmer accepts/rejects placed orders and records pickup/delivery milestones; processor cancels before acceptance and records completion after receipt. Rejection/cancellation releases reservation; completion consumes it. Disputed orders have no automated resolution.

Deletion request states: `requested`, `approved`, `completed`, `rejected`. Admin approval only records an audited decision for separate manual retention review; it does not erase, anonymize, suspend, or restrict the account.

## Dashboard payloads

All dashboards return `{period:{from,to,bucket},kpis,charts,tables}`. Periods are `7d`, `30d` (default), `90d`, `12m`, or paired RFC 3339 `from`/`to`, maximum 366 days. Buckets are daily or monthly.

- Farmer KPIs: `milkSoldLiters`, `revenueUzs`, `activeListings`, `ordersAwaitingResponse`, `ownAveragePricePerLiter`, `platformAveragePricePerLiter`. Charts: `completedVolume`, `qualityTrend`. Tables: `upcomingOrders`, `recentActivity`.
- Processor KPIs: `volumeSourcedLiters`, `spendUzs`, `activeSuppliers`, `averagePricePerLiter`, `openOrders`, `qualityAcceptanceRate`. Charts: `completedVolume`, `supplyByRegion`. Tables: `upcomingOrders`.
- Admin KPI: `accountsAwaitingApproval` with `oldestApplicant`; charts `signups`, `completedVolume`; tables `accountsByRoleStatus`, `activeListingsByRegion`, `ordersByStatus`, `recentAuditEvents`.
- Empty dashboard values use zero/null and empty arrays. All metrics are database-derived. The platform average is a volume-weighted average of completed platform orders, not an external benchmark.

**Correction made during Step 0:** Farmer `ordersAwaitingResponse` previously counted `placed` inside a query filtered to `completed`, making it permanently zero. It now separately counts the farmer's placed orders. The zero-completed-orders case also now returns a valid false boolean rather than SQL NULL. A PostgreSQL race integration assertion covers five concurrently accepted reservations plus one additional placed order and verifies the KPI is six.

The OpenAPI `Dashboard` schema still declares KPI/chart/table objects with `additionalProperties: true`; precise generated TypeScript response types cannot be guaranteed until backend schemas are made explicit.

## Error catalog and client behavior

The backend returns English messages, stable codes, optional `fieldErrors`, `requestId`, and `X-Request-ID`. These Uzbek/Russian/English strings are proposed frontend translations, not backend translations; native-speaker review is required before release.

| Code                           | Client behavior                                                       | Uzbek (Latin)                                                      | Russian                                                               | English                                                                   |
| ------------------------------ | --------------------------------------------------------------------- | ------------------------------------------------------------------ | --------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `invalid_request`              | Preserve input; explain malformed request/query                       | So‘rov noto‘g‘ri. Maydonlarni tekshirib, qayta urinib ko‘ring.     | Некорректный запрос. Проверьте данные и повторите попытку.            | The request is invalid. Check the details and try again.                  |
| `validation_failed`            | Map `fieldErrors` to controls; keep all values                        | Kiritilgan ma’lumotlarni tekshiring.                               | Проверьте введённые данные.                                           | Check the entered information.                                            |
| `unauthenticated`              | Single-flight refresh once; otherwise sign-in                         | Sessiya tugadi. Qayta kiring.                                      | Сеанс завершён. Войдите снова.                                        | Your session ended. Sign in again.                                        |
| `invalid_credentials`          | Generic sign-in error; do not reveal which field                      | Email yoki tasdiqlash kodi noto‘g‘ri.                              | Неверный email или код подтверждения.                                 | Email or verification code is incorrect.                                  |
| `forbidden`                    | Show access denied; preserve safe navigation                          | Bu amal uchun ruxsat yo‘q.                                         | У вас нет разрешения на это действие.                                 | You do not have permission to do this.                                    |
| `account_not_approved`         | Refetch `/me`; route to pending/verification                          | Hisobingiz tasdiqlanishini kutmoqda.                               | Ожидается подтверждение аккаунта.                                     | Your account is awaiting approval.                                        |
| `account_rejected`             | Refetch `/me`; show reason and correction/resubmit                    | Arizangiz rad etildi. Sababni ko‘rib, ma’lumotlarni tuzating.      | Заявка отклонена. Ознакомьтесь с причиной и исправьте данные.         | Your application was rejected. Review the reason and update your details. |
| `account_suspended`            | Refetch `/me`; show blocked screen                                    | Hisobingiz vaqtincha to‘xtatilgan.                                 | Аккаунт временно заблокирован.                                        | Your account is suspended.                                                |
| `not_found`                    | Render not-found state; do not leak ownership                         | Ma’lumot topilmadi.                                                | Данные не найдены.                                                    | The requested information was not found.                                  |
| `conflict`                     | Refresh resource and explain current-state conflict                   | Ma’lumotlar holati o‘zgargan. Yangilab ko‘ring.                    | Состояние данных изменилось. Обновите страницу.                       | The resource changed. Refresh and try again.                              |
| `rate_limited`                 | Do not retry early; show countdown only if API supplies `Retry-After` | So‘rovlar ko‘payib ketdi. Biroz kutib qayta urinib ko‘ring.        | Слишком много запросов. Подождите и повторите попытку.                | Too many requests. Wait before trying again.                              |
| `refresh_token_reuse_detected` | Clear in-memory session and force sign-in                             | Xavfsizlik sabab sessiya yakunlandi. Qayta kiring.                 | Сеанс завершён по соображениям безопасности. Войдите снова.           | Your session was revoked for security. Sign in again.                     |
| `admin_mfa_required`           | Route to enrollment or MFA challenge                                  | Administrator uchun ikki bosqichli tasdiqlashni yakunlang.         | Завершите настройку двухфакторной аутентификации администратора.      | Complete administrator two-factor authentication.                         |
| `csrf_validation_failed`       | Do not retry mutation; refresh/sign in and report if repeated         | Xavfsizlik tekshiruvi o‘tmadi. Sahifani yangilang va qayta kiring. | Проверка безопасности не пройдена. Обновите страницу и войдите снова. | Security validation failed. Refresh and sign in again.                    |
| `internal_error`               | Safe generic error, retry, show request ID                            | Kutilmagan xatolik. Qayta urinib ko‘ring.                          | Произошла ошибка. Повторите попытку.                                  | An unexpected error occurred. Try again.                                  |
| `dependency_unavailable`       | Offline/maintenance state; bounded retry                              | Xizmat vaqtincha ishlamayapti.                                     | Сервис временно недоступен.                                           | The service is temporarily unavailable.                                   |
| `method_not_allowed`           | Treat as client/route defect; do not retry                            | Bu amal bu yerda mavjud emas.                                      | Это действие здесь недоступно.                                        | This action is not available here.                                        |
| `cors_origin_denied`           | Configuration failure; no retry workaround                            | Sayt manzili serverda ruxsat etilmagan.                            | Источник сайта не разрешён сервером.                                  | This website origin is not allowed by the server.                         |
| `cors_origin_required`         | Configuration/request failure; no retry workaround                    | Xavfsiz manba manzili talab qilinadi.                              | Требуется разрешённый адрес источника.                                | An allowed request origin is required.                                    |

Every unexpected error displays a request ID when present. Do not log email, STIR, phone, tokens, error bodies with personal data, or form values. `Retry-After` is not currently emitted by the backend, so a client countdown cannot yet use an authoritative delay.

## Verification not yet completed

No sample seed command/accounts are present. Therefore do not invent seeded dashboard numbers or claim the requested full Mailpit/admin approval journeys were verified. The API integration/race suite is evidence for backend behavior, not an end-to-end browser proof. The frontend plan must remain limited to contract-backed capabilities until outstanding API requests are resolved or explicitly accepted as deferred.
