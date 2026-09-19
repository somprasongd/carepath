SHELL := /bin/bash

.PHONY: up down logs api mock-his web fmt

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
fetch:
	git fetch