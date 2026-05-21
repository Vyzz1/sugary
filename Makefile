APP_NAME=sugary-api
MIGRATE_CMD?=up
MIGRATE_ARGS?=

.PHONY: run test fmt vet lint sqlc migrate migrate-up migrate-down migrate-create

run:
	go run ./cmd/api

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

sqlc:
	sqlc generate

migrate:
	go run ./cmd/migrate $(MIGRATE_CMD) $(MIGRATE_ARGS)

migrate-up:
	go run ./cmd/migrate up $(MIGRATE_ARGS)

migrate-down:
	go run ./cmd/migrate down $(MIGRATE_ARGS)

migrate-create:
	go run ./cmd/migrate create $(MIGRATE_ARGS)
