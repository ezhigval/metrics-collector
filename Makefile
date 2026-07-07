.PHONY: run test lint docker-up docker-down build migrate-up ingest

DATABASE_URL ?= postgres://metrics:metrics@localhost:5438/metrics?sslmode=disable

run:
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/server

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

ingest:
	curl -s -X POST localhost:8089/api/v1/ingest -H 'Content-Type: application/json' \
	  -d '{"metrics":[{"name":"cpu_usage","value":42.5,"labels":{"service":"api","host":"node-1"}}]}' | jq
