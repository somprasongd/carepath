SHELL := /bin/bash

.PHONY: up down logs api mock-his web fmt migrate-up swag start stop fetch docs-erd

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
	@set -a; [ -f .env ] && . ./.env; set +a; \
	$(MAKE) api > logs/api.log 2>&1 & echo $$! > logs/api.pid; \
	$(MAKE) mock-his > logs/mock-his.log 2>&1 & echo $$! > logs/mock-his.pid; \
	$(MAKE) web > logs/web.log 2>&1 & echo $$! > logs/web.pid 
	@echo "started: api=$$(cat logs/api.pid) mock-his=$$(cat logs/mock-his.pid) web=$$(cat logs/web.pid)"
	@echo "logs: logs/api.log logs/mock-his.log logs/web.log"
	@echo "stop with: make stop"

# Stop processes started by `make start`. Kills by port rather than the pid
# files: on setups where SHELL resolves through a WSL/interop boundary, $!
# captures the wrapper's pid, not the actual server process, so pid-based
# kill silently misses it and ports stay held across restarts.
stop:
	@set -a; [ -f .env ] && . ./.env; set +a; \
	for port in "$${CAREPATH_API_PORT:-8080}" "$${MOCK_HIS_PORT:-8090}" 5173; do \
		if fuser -k $$port/tcp 2>/dev/null; then \
			echo "stopped port $$port"; \
		else \
			echo "port $$port: nothing listening"; \
		fi; \
	done
	@rm -f logs/*.pid

fetch:
	git fetch