# ZaminFlow Web Security Architecture & OWASP Review

This document specifies the client-side security architecture, defense-in-depth measures, and operational constraints for `zaminflow-web`.

## 1. CORS Allowlist & Cookie Domain Assumptions

### CORS Allowed Origins

The ZaminFlow backend API must configure its CORS allowlist (`Access-Control-Allow-Origin`) to strictly include:

- **Local development:** `http://localhost:3000` and `http://127.0.0.1:3000`
- **Production web origin:** `https://www.zaminflow.uz` (and optionally `https://zaminflow.uz` if apex serves traffic directly)
- `Access-Control-Allow-Credentials: true` must be enabled on all authenticated endpoints.
- Preflight `OPTIONS` requests must allow `Content-Type`, `Authorization`, `X-CSRF-Token`, and `X-Request-ID`.

### Cookie Domain Assumptions

- The refresh token (`zf_refresh`) and CSRF token (`zf_csrf`) are issued as HttpOnly, Secure cookies by the API.
- For same-site apex domain setups where API is on `api.zaminflow.uz` and frontend on `www.zaminflow.uz`:
  - `Domain=.zaminflow.uz; Path=/api/v1/auth; SameSite=Lax; Secure; HttpOnly` (for `zf_refresh`)
  - `Domain=.zaminflow.uz; Path=/; SameSite=Lax; Secure` (for readable `zf_csrf` cookie)
- In development environments (`localhost`), cookies must omit the `Domain` attribute so the browser attributes them to the API port, with `SameSite=Lax`.

---

## 2. Threat Modeling & OWASP Client-Side Assessment

### A01: Broken Access Control & Role Enforcement

- **Client vs Server Gating:** Role-based gating in the UI (`RoleGate`, `AccountStatusGate`) is strictly treated as user experience (navigation, conditional buttons, form visibility), **never** as a security barrier.
- **Server Enforcement:** Every API request validates authorization headers and active session status.
- **State Invalidation:** Global interceptors watch for `account_not_approved`, `account_rejected`, and `account_suspended` error codes. If received, the client immediately refetches `/me` and switches the UI into the restricted account state.

### A02: Cryptographic Failures & Token Storage

- **In-Memory Only Access Tokens:** Short-lived access tokens (~15 minutes) are held strictly in memory in `src/lib/auth/token-store.ts`. They are **never** persisted to `localStorage`, `sessionStorage`, IndexedDB, or URL query parameters.
- **Refresh Token Isolation:** Long-lived refresh tokens are stored exclusively in browser-managed `HttpOnly`, `SameSite=Lax`, `Secure` cookies inaccessible to JavaScript.
- **Single-Flight Silent Refresh:** Token refresh uses a mutex lock to prevent concurrent redundant refresh requests and races.
- **Token Rotation & Revocation:** When a refresh token is rotated or invalid, the session context terminates immediately and flushes memory.

### A03: Injection & Cross-Site Scripting (XSS)

- **Zero `dangerouslySetInnerHTML`:** Untrusted user input and API payloads are rendered strictly as React text nodes.
- **Strict Content Security Policy (CSP):** Delivered via HTTP response headers and Next.js middleware with dynamic per-request nonces:
  ```
  default-src 'self';
  script-src 'self' 'nonce-{nonce}' 'strict-dynamic';
  style-src 'self' 'nonce-{nonce}';
  img-src 'self' blob: data: https:;
  font-src 'self' data:;
  connect-src 'self' ${API_BASE_URL};
  frame-ancestors 'none';
  base-uri 'self';
  form-action 'self';
  ```
- **X-Content-Type-Options:** `nosniff` enforced.
- **Referrer-Policy:** `strict-origin-when-cross-origin`.

### A04: Insecure Design & Cross-Site Request Forgery (CSRF)

- **CSRF Token Synchronization:** State-changing requests (`POST`, `PUT`, `PATCH`, `DELETE`) extract the `zf_csrf` cookie value and submit it in the `X-CSRF-Token` HTTP header on credentials-mode requests.
- **Double-Submit Prevention:** All submission buttons disable themselves and render loading indicators during pending network requests to eliminate duplicate submissions.

### A05: Security Misconfiguration & Headers

The application enforces HTTP security headers in production via reverse proxy (Caddy) and Next.js config:

- `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload`
- `X-Frame-Options: DENY`
- `Permissions-Policy: camera=(), microphone=(), geolocation=(), payment=()`

### A06: Vulnerable and Outdated Components

- Pinned package versions in `package.json` with deterministic `pnpm-lock.yaml`.
- Automated vulnerability scanning via `pnpm audit --prod --audit-level high`.
- Regular license compliance validation with `pnpm run licenses`.

### A07: Identification and Authentication Failures

- **Password Policy:** Enforces client-side validation mirroring backend rules: minimum length, uppercase, lowercase, numbers, and symbols.
- **Admin Multi-Factor Authentication (TOTP):** Enforced enrollment with QR code, manual base32 secret, OTP code verification, and backup recovery codes.
- **Open Redirect Prevention:** Post-login and recovery redirects are strictly validated against a same-origin allowlist (`/uz`, `/ru`, `/en`). External URLs in `returnTo` parameters are rejected.

### A08: Software and Data Integrity Failures

- **OpenAPI Drift Detection:** Generated API client and schema types (`pnpm check-client-drift`) run in CI to prevent contract drift and desynchronized validation schemas.

### A09: Security Logging and Monitoring Failures

- **PII Scrubbing:** Phone numbers (`+998...`), STIR tax IDs, emails, and passwords are never logged to browser console or telemetry.
- **Traceability:** Unhandled errors surface request tracking IDs (`X-Request-ID` or error digest) to assist users and support without exposing internal stack traces.
