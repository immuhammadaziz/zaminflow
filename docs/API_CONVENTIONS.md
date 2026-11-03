# API Conventions

**Status: implemented.** These conventions are reflected in [API_CONTRACT.md](./API_CONTRACT.md) and [openapi.yaml](./openapi.yaml).

## Values and serialization

- JSON properties use `camelCase`; resource IDs are UUIDs.
- Date-times use RFC 3339. Database values and dashboard periods use UTC.
- Money uses integer UZS; milk quantities use whole liters; fat percentage is a bounded decimal.
- Role, account, listing, order, and report values use stable lowercase strings. Labels and translations are separate.
- Uzbek phone numbers are validated as `+998` followed by nine digits.

## Pagination

Paginated endpoints use 1-based `page` (default 1), `pageSize` (default 20, maximum 100), and return:

```json
{"items":[],"page":1,"pageSize":20,"total":0}
```

Only documented filters and sort values are accepted. Invalid query values return `400`; unknown query keys are currently ignored.

## Errors and request IDs

Errors use:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "One or more input values are invalid.",
    "requestId": "..."
  }
}
```

Field-level validation failures may include `fieldErrors`. The API returns `X-Request-ID` and logs the ID with server-side failures. Error messages are safe display fallbacks; clients should branch on codes. SQL details, stack traces, credentials, and token values are not returned.

## Authentication and browser protections

- Access JWTs are short-lived and sent as `Authorization: Bearer`.
- Rotating refresh tokens are stored in HttpOnly cookies; they are never returned in JSON.
- Browser refresh and sign-out require exact allowed `Origin`, a signed `zf_csrf` cookie, and matching `X-CSRF-Token`.
- CORS allows only explicit origins and credentials. Production configuration requires HTTPS origins.
- Account status and role are checked from PostgreSQL for marketplace routes; token claims do not replace these checks.

See the [frontend guide](./FRONTEND_INTEGRATION.md) for request examples and the [production guide](./DEPLOYMENT.md) for deployment settings.
