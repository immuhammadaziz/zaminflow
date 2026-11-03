.PHONY: up down dev test e2e lint build generate-client check-client-drift check audit licenses format format-check

up:
	docker compose up -d --build

down:
	docker compose down -v

dev:
	pnpm dev

test:
	pnpm test

e2e:
	pnpm e2e

lint:
	pnpm lint

build:
	pnpm build

generate-client:
	pnpm generate-client

check-client-drift:
	pnpm check-client-drift

check:
	pnpm check

audit:
	pnpm audit --prod --audit-level high

licenses:
	pnpm run licenses

format:
	pnpm format

format-check:
	pnpm format:check