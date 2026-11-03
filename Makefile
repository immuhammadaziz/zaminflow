.PHONY: up down test lint vet build migrate migrate-down

up:
	docker compose up --build

down:
	docker compose down

test:
	go test -race ./...

lint:
	test -z "$$(gofmt -l $$(git ls-files '*.go'))"

vet:
	go vet ./...

build:
	go build ./cmd/api ./cmd/migrate

migrate:
	go run ./cmd/migrate

migrate-down:
	go run ./cmd/migrate --direction down
