# ZaminFlow Web Verification & Testing Guide

This document records the testing strategy, test suites, coverage expectations, and quality assurance gates for `zaminflow-web`.

## 1. Testing Pyramid Overview

- **Unit & Component Tests (Vitest + React Testing Library):**
  - Tests auth routing logic, token store isolation, API error transformations, and form validations.
  - Run via: `pnpm test` or `pnpm test:coverage`.
- **End-to-End Tests (Playwright):**
  - Executes real browser journeys against the server.
  - Smoke checks: public landing pages, localized 404 boundaries, form accessibility, and CSP nonces.
  - Run via: `pnpm e2e`.
- **Accessibility Audits (axe-core):**
  - Automated WCAG 2.2 AA audits embedded directly into Playwright tests (`@axe-core/playwright`).
  - Strict zero serious or critical violations requirement.
- **Contract Drift Verification (openapi-typescript):**
  - Validates that the generated TypeScript API client (`src/lib/api/schema.d.ts`) is strictly synchronized with `contracts/openapi.yaml`.
  - Run via: `pnpm check-client-drift`.
- **Storybook Component Library:**
  - Isolated component development and visual verification with accessibility addon (`@storybook/addon-a11y`).
  - Run via: `pnpm storybook` or build via `pnpm build-storybook`.

## 2. Running Local Verification Suites

Execute the full verification pipeline before pushing changes:

```bash
# 1. Formatting and Code Quality
pnpm format:check
pnpm lint

# 2. Type Checking
pnpm typecheck

# 3. I18n Translation Completeness
pnpm check:translations

# 4. OpenAPI Contract Drift
pnpm check-client-drift

# 5. Unit and Component Tests
pnpm test

# 6. Production App Build
pnpm build

# 7. Storybook Component Build
pnpm build-storybook

# 8. Playwright E2E & Accessibility Tests
pnpm e2e

# Or run the consolidated check script:
pnpm check
```

## 3. Dependency and Security Audit

```bash
pnpm audit --prod --audit-level high
pnpm run licenses
```

## 4. Continuous Integration Pipeline (GitHub Actions)

Defined in `.github/workflows/ci.yml`:

1. Checkout repository.
2. Setup pnpm (v12.16.1) and Node.js (v26.13.0).
3. Dependency installation with `--frozen-lockfile`.
4. Linters, TypeScript compiler, translation validator, client drift check, Vitest run.
5. Production Next.js build and Storybook static build.
6. Playwright browser installation and headless E2E run.
