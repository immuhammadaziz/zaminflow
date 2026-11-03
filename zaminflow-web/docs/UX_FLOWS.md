# Approved UX flows — constrained by current API

## Farmer/processor registration and approval

```mermaid
flowchart TD
  A[Choose farmer or processor] --> B[Load roles and regions from API]
  B --> C[Submit sign-up]
  C -->|201| D[Verification instructions]
  C -->|422 / 409 / 429| E[Keep fields; show field or form errors]
  D --> F[Confirm one-time email token]
  F -->|verified| G[Sign in]
  G --> H[Fetch /me and route by live state]
  H -->|unverified| I[Verification screen; resend]
  H -->|pending| J[Awaiting approval; refresh on focus and interval]
  H -->|approved| K[Role dashboard]
  H -->|rejected| L[Show reason; edit /me]
  L --> M[POST /me/resubmit]
  M --> J
  H -->|suspended| N[Blocked screen; sign out]
```

## Session restore, password reset and suspension

```mermaid
flowchart TD
  A[App load] --> B[Try in-memory access token]
  B -->|missing / expired 401| C[Single-flight POST /auth/refresh with cookies and CSRF]
  C -->|success| D[Keep access token in memory; GET /me]
  C -->|reuse / invalid refresh| E[Clear memory; sign in with safe same-origin return path]
  D -->|account_not_approved / rejected / suspended| F[Refetch /me and route to authoritative state]
  G[Forgot password] --> H[Enumeration-safe reset request]
  H --> I[Reset token from email]
  I --> J[Confirm new password; revoke sessions]
```

## Listing to order/receipt

```mermaid
flowchart LR
  A[Approved farmer creates active listing] --> B[Processor filters/sorts supported listings]
  B --> C[Processor places order]
  C -->|stock conflict| B
  C --> D[placed: farmer decision]
  D -->|reject| E[Reservation released]
  D -->|accept| F[accepted]
  F --> G[ready_for_pickup]
  G --> H[picked_up]
  H --> I[delivered]
  I -->|processor confirms receipt| J[completed; stock consumed]
  F -->|either party records dispute| K[disputed; out-of-band handling]
```

## Admin TOTP and application decision

```mermaid
flowchart TD
  A[CLI-provisioned admin signs in] --> B[Enrollment-only access]
  B --> C[POST /auth/totp/setup]
  C --> D[Scan QR or enter manual key]
  D --> E[POST /auth/totp/confirm]
  E --> F[Show recovery codes once; confirm saved]
  F --> G[Sign in again with TOTP or recovery code]
  G --> H[GET /admin/applications]
  H --> I[Review detail/history]
  I -->|approve| J[Account approved]
  I -->|reject with reason| K[Account rejected; applicant may correct/resubmit]
```
