.PHONY: dev api web test lint build migrate

dev:
	@echo "Run 'make api' and 'make web' in separate terminals"

api:
	cd api && DEV_AUTH=true go run ./cmd/api

web:
	cd web && npm run dev

test:
	cd api && go test ./...
	cd web && npm test -- --run

lint:
	cd api && go vet ./...
	cd web && npm run lint

build:
	cd api && go build ./cmd/...
	cd web && npm run build

migrate:
	cd api && go run ./cmd/migrate

