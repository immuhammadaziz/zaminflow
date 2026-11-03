# ZaminFlow API

Contract-first Go API for the ZaminFlow dairy marketplace. The product and API design is awaiting approval before authentication and marketplace endpoints are implemented.

## Local quick start

Install Docker Desktop and Go 1.27.3 first. Copy the example environment file and replace the local-only database password before starting the stack:

```powershell
Copy-Item .env.example .env
docker compose up --build -d
Invoke-RestMethod http://localhost:8080/healthz
Invoke-RestMethod http://localhost:8080/readyz
docker compose --profile dev-tools up -d swagger-ui
```

The current Foundation phase provides PostgreSQL, migration execution, liveness/readiness endpoints, and local Mailpit/Swagger UI containers. Authentication, approval, listings, orders, and dashboards are not implemented yet.

## Design and API draft

- [Step 0 design report](docs/DESIGN_REPORT.md)
- [API conventions](docs/API_CONVENTIONS.md)
- [API contract](docs/API_CONTRACT.md)
- [OpenAPI draft](docs/openapi.yaml)
- [Proposed decisions](docs/DECISIONS.md)

Do not treat the draft marketplace routes as implemented endpoints. The only HTTP routes currently served are `GET /healthz` and `GET /readyz`.
