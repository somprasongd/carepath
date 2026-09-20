SHELL := /bin/bash

.PHONY: up down logs api mock-his web fmt migrate-up migrate-up-local swag start stop fetch docs-erd

# `command -v migrate` covers a normal PATH
# install; non-interactive shells don't source ~/.bashrc so also check the
# common `go install` GOPATH location directly.
MIGRATE := $(shell command -v migrate 2>/dev/null || echo "$$HOME/go/bin/migrate")

up:
	docker compose up --build

down:
	docker compose down -v

logs:
	docker compose logs -f

api:
	cd apps/api && go run ./cmd/server

mock-his:
	cd apps/mock-his && PATIENT_APP_BASE_URL="$${PATIENT_APP_BASE_URL:-http://localhost:5173}" go run ./cmd/server

web:
	cd apps/web && npm install && npm run dev

fmt:
	cd apps/api && gofmt -w .
	cd apps/mock-his && gofmt -w .

migrate-up:
	docker compose run --rm migrate

# Regenerate docs/architecture/erd/ from the live schema (needs `make up` or
# `make migrate-up` first, and `tbls`: brew install k1LoW/tap/tbls). Docs use
# Mermaid (renders inline on GitHub); schema.png is a static image export for
# sharing outside the repo (e.g. slides, chat).
docs-erd:
	set -a; [ -f .env ] && . ./.env; set +a; \
	tbls doc --rm-dist -f; \
	tbls out -t png -o docs/architecture/erd/schema.png

swag:
	cd apps/api && go run github.com/swaggo/swag/cmd/swag init -g cmd/server/main.go -o docs --parseInternal

# Run api/mock-his/web as background processes; PIDs and output land in logs/.
start:
	@mkdir -p logs
	@-$(MAKE) stop
	@set -a; [ -f .env ] && . ./.env; set +a; \
	$(MIGRATE) -path infra/postgres/migrations \
		-database "postgres://$${POSTGRES_USER:-carepath}:$${POSTGRES_PASSWORD:-carepath}@$${POSTGRES_HOST:-localhost}:$${POSTGRES_PORT:-5432}/$${POSTGRES_DB:-carepath}?sslmode=disable" up; \
	$(MAKE) api > logs/api.log 2>&1 & echo $$! > logs/api.pid; \
	$(MAKE) mock-his > logs/mock-his.log 2>&1 & echo $$! > logs/mock-his.pid; \
	$(MAKE) web > logs/web.log 2>&1 & echo $$! > logs/web.pid 
	@echo "started: api=$$(cat logs/api.pid) mock-his=$$(cat logs/mock-his.pid) web=$$(cat logs/web.pid)"
	@echo "logs: logs/api.log logs/mock-his.log logs/web.log"
	@echo "stop with: make stop"

# Stop processes started by `make start`. For each service: kill by its
# recorded pid file first (report if that pid is stale/already gone); if
# then ALSO kill whoever holds the port (lsof) regardless of whether the
# pidfile kill "succeeded" — the recorded pid can be a wrapper/ancestor
# process (make/npm) that dies without taking its child (the actual
# node/vite process bound to the port) down with it, so pidfile success is
# not proof the port is free. Plain POSIX only, no Windows-specific
# commands, so this behaves the same on any contributor's machine.
stop:
	@set -a; [ -f .env ] && . ./.env; set +a; \
	for entry in "api:$${CAREPATH_API_PORT:-8080}" "mock-his:$${MOCK_HIS_PORT:-8090}" "web:5173"; do \
		svc=$${entry%%:*}; port=$${entry#*:}; \
		if [ -f logs/$$svc.pid ]; then \
			pid=$$(cat logs/$$svc.pid); \
			if kill -9 $$pid 2>/dev/null; then \
				echo "stopped $$svc: pidfile (pid $$pid)"; \
			else \
				echo "$$svc: kill pidfile pid $$pid failed (stale or already gone)"; \
			fi; \
			rm -f logs/$$svc.pid; \
		else \
			echo "$$svc: no pidfile"; \
		fi; \
		if command -v lsof >/dev/null 2>&1; then \
			pid=$$(lsof -ti tcp:$$port 2>/dev/null); \
			if [ -n "$$pid" ]; then \
				if kill -9 $$pid 2>/dev/null; then \
					echo "stopped $$svc: port $$port (pid $$pid)"; \
				else \
					echo "$$svc: kill port $$port pid $$pid failed"; \
				fi; \
			else \
				echo "$$svc: port $$port nothing listening"; \
			fi; \
		fi; \
	done

fetch:
	git fetch