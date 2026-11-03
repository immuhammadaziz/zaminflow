# API Conventions (Draft)

**Status: proposed for Step 0 approval.** These conventions apply to every `/api/v1` endpoint and should be treated as contract rules after approval.

## Naming, identifiers, and values

- JSON property names use `camelCase`.
- Resource identifiers use UUIDs.
- Date-times are RFC 3339 strings in UTC, for example `2026-11-03T07:00:00Z`.
- Money is an integer amount in UZS; never use floating-point money.
- Quantities use integer liters unless the product approves fractional quantities.
- Enum values use stable lowercase `snake_case` strings. Labels and translations are separate display fields.
- Phone numbers are normalized Uzbek international numbers (`+998` followed by nine digits).

## Pagination, filters, and sorting

Every list endpoint uses the same page-based response:

```json
{
  "items": [],
  "page": 1,
  "pageSize": 20,
  "total": 0
}
```

`page` is 1-based and defaults to `1`; `pageSize` defaults to `20` and is limited to `100`. List endpoints document their allowlisted filters and sort fields; clients cannot request arbitrary database column names. Sorting is explicit (for example `sort=createdAt:desc`) and deterministic with a stable ID tie-breaker. Cursor pagination is not proposed for the initial release.

## Errors and request IDs

Every error uses the following envelope, including validation failures:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "One or more fields are invalid.",
    "requestId": "req_...",
    "fieldErrors": [
      {
        "field": "phone",
        "code": "invalid_format",
        "message": "Use +998 followed by nine digits."
      }
    ]
  }
}
```

`code` is stable, machine-readable, and suitable for client-side localization. `message` is safe, concise fallback text; clients should branch on `code`, not message wording. `fieldErrors` is omitted when there are no field-specific issues. A request ID is returned in both the error body and `X-Request-ID` response header and is included in structured server logs.

## HTTP status meanings

| Status | Meaning |
|---|---|
| `200` | Successful read or mutation with a response body |
| `201` | Resource created |
| `202` | Request accepted for asynchronous processing, such as email or deletion request |
| `204` | Successful operation with no response body |
| `400` | Malformed JSON/query, unknown JSON field, or trailing JSON value |
| `401` | Missing, invalid, expired, or unauthenticated credentials |
| `403` | Authenticated but wrong role, insufficient permission, or account trading gate |
| `404` | Resource absent or not visible to caller; avoid disclosing another user's resource |
| `405` | Method is not supported for the requested resource |
| `409` | Duplicate/conflicting state or invalid lifecycle transition |
| `422` | Well-formed request with invalid field values |
| `429` | Rate limit or temporary authentication lockout |
| `5xx` | Unexpected server or dependency failure; response contains request ID, not internals |

Unapproved trading users receive `403` with `account_not_approved`, `account_rejected`, or `account_suspended` as applicable. Invalid/missing authentication is `401`; wrong-role authorization is `403`.

## Authentication transport and CORS

- Access tokens are short-lived JWTs returned in JSON and sent as `Authorization: Bearer`.
- Refresh tokens are rotating, HttpOnly cookies for browser clients, scoped to the auth path with `Secure` in production and `SameSite=Lax`; browser refresh/sign-out requests require an allowed `Origin` and a custom CSRF header.
- A non-browser refresh-token body transport is disabled unless explicitly enabled by deployment configuration.
- CORS uses an explicit environment-configured origin allowlist; wildcard origins are never used with credentials. Proposed production origins are `https://www.zaminflow.uz` and `https://zaminflow.uz`; local origins are `http://localhost:3000` and `http://localhost:5173`.
- Exact cookie Domain and production API hostname must follow the DNS and deployment decision.

## Request and response behavior

- JSON bodies are size-limited, decoded strictly, and reject unknown fields and trailing values.
- Validation is applied consistently to length, format, enum membership, normalization, and cross-field constraints.
- Error responses never include stack traces, SQL details, credentials, token values, or other sensitive internals.
- Every resource read and mutation enforces role and ownership server-side. Account status is read live from the database for every trading request, not trusted from token contents.
- Dashboard endpoints aggregate the requested data server-side; clients should not need per-row follow-up calls.

Detailed schemas, per-operation constraints, and error cases are maintained in [API_CONTRACT.md](./API_CONTRACT.md) and [openapi.yaml](./openapi.yaml).
