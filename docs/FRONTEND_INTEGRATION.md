# Frontend integration

This is a backend integration reference; the repository does not include a frontend.

## Base URL and browser requests

Use `https://api.zaminflow.uz/api/v1` in production and `http://localhost:8080/api/v1` for local Compose. Configure the exact web origins in `CORS_ALLOWED_ORIGINS`. Browser calls that need cookies must use `credentials: "include"`; the API allows credentials only for configured origins.

```js
const response = await fetch(`${API_BASE}/auth/sign-in`, {
  method: "POST",
  credentials: "include",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email, password }),
});
```

Keep the short-lived `accessToken` in memory rather than persistent browser storage. Send it as `Authorization: Bearer <accessToken>` for authenticated calls. The API returns a new access token from refresh; the opaque refresh token itself is only in an HttpOnly cookie.

## Refresh and sign-out CSRF

Sign-in and refresh set:

- `zf_refresh`: HttpOnly, SameSite=Lax, Secure in production, scoped to `/api/v1/auth`.
- `zf_csrf`: signed double-submit token, SameSite=Lax, Secure in production, scoped to `/`.

The web app reads the CSRF cookie and sends the same value in `X-CSRF-Token`. Both operations also require an exact allowed `Origin`. Do not cache the CSRF value independently from its cookie. A successful refresh rotates both cookies; use the replacement cookie on the next request.

```js
function csrfCookie() {
  return document.cookie
    .split("; ")
    .find((part) => part.startsWith("zf_csrf="))
    ?.slice("zf_csrf=".length);
}

async function refreshAccessToken() {
  const csrf = csrfCookie();
  const response = await fetch(`${API_BASE}/auth/refresh`, {
    method: "POST",
    credentials: "include",
    headers: { "X-CSRF-Token": csrf ?? "" },
  });
  if (!response.ok) throw new Error(`Refresh failed: ${response.status}`);
  return (await response.json()).accessToken;
}
```

`POST /auth/sign-out` additionally needs the current bearer access token and the same CSRF header/cookie/origin checks. A CSRF failure returns `403 csrf_validation_failed`; an expired or missing refresh cookie returns `401`.

The production example uses `REFRESH_COOKIE_DOMAIN=.zaminflow.uz` so a site at the apex or `www` can use the API subdomain. Set this only when the production web and API hosts are under the same trusted registrable domain. Otherwise leave it empty and host the web app as an API subdomain sibling compatible with cookie policy.

## Sign-up, verification, and approval

1. Fetch `/meta/roles` and `/meta/regions` for reference values. `admin` is intentionally not selectable.
2. Submit `POST /auth/sign-up`. New accounts are `pending` and cannot use listings, orders, or role dashboards.
3. Ask the user to open the verification link delivered by email. The link targets `{WEB_BASE_URL}/verify-email?token=...`; the web app should submit that token to `/auth/email-verification/confirm`.
4. An administrator reviews only verified applications. Applicants can inspect current status with `/me`; rejected applicants can edit permitted fields with `PATCH /me`, then call `/me/resubmit`.
5. Account state is checked against the database on authenticated requests. Clients must treat `403 account_not_approved`, `account_rejected`, and `account_suspended` as authoritative even if a cached profile suggests otherwise.

Password reset links target `{WEB_BASE_URL}/reset-password?token=...`; submit the token and new password to `/auth/password-reset/confirm`. Do not log or persist the token.

## Administrator MFA

The initially provisioned admin can sign in without a factor only to enroll. The returned enrollment token is limited to `/auth/totp/setup` and `/auth/totp/confirm`. Confirm a TOTP code from an authenticator app, store the returned recovery codes securely for the administrator, then sign in again. Enrollment revokes the limited session. All subsequent admin operations require completed MFA; supply either `totpCode` or `recoveryCode` at sign-in.

## Roles, state, and error handling

- `farmer`: manages their own listings and responds to orders as the seller.
- `processor`: searches active listings, places orders, and confirms completion as buyer.
- `admin`: reviews applications, manages account status, reads audit events, and sees admin aggregates. Admins do not trade.

Order creation reserves stock atomically. Rejection or eligible cancellation releases it; completion consumes it. A disputed order is not automatically resolved. Render prices as integer UZS values and quantities as whole liters.

Handle the shared error envelope and `X-Request-ID`. `400` means malformed input or invalid query parameters; `422` means valid JSON with invalid field values; `401` requires sign-in/refresh; `403` means a role/account/MFA/CSRF restriction; `404` includes resources the caller cannot access; `409` means a state conflict; `429` means rate-limited. Never retry a non-idempotent order submission blindly after a network timeout; first query the processor's order list.

The `DELETE /me` route only records a deletion request and returns `{status:"requested"}`; use `GET /me/deletion-request` to display the latest request and its review decision. Administrators review requests through `/admin/deletion-requests`; approval does not erase, anonymize, or suspend the account or prevent further marketplace activity. Before any separate manual retention action, operators must recheck open transactions and applicable legal obligations. Do not promise immediate erasure.
