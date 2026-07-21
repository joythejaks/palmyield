include .env
export

.PHONY: run build migrate-up migrate-down sqlc test lint

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

sqlc:
	sqlc generate

test:
	go test ./... -v

lint:
	golangci-lint run ./...
