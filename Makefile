SHELL := /bin/bash

.PHONY: up down logs api mock-his web fmt migrate-up swag

up:
	docker compose up --build

down:
	docker compose down -v

logs:
	docker compose logs -f

api:
	cd apps/api && go run ./cmd/server

mock-his:
	cd apps/mock-his && go run ./cmd/server

web:
	cd apps/web && npm install && npm run dev

fmt:
	cd apps/api && gofmt -w .
	cd apps/mock-his && gofmt -w .

migrate-up:
	docker compose run --rm migrate

swag:
	cd apps/api && go run github.com/swaggo/swag/cmd/swag init -g cmd/server/main.go -o docs --parseInternal
fetch:
	git fetch