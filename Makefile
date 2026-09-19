SHELL := /bin/bash

.PHONY: up down logs api mock-his web fmt migrate-up swag start stop fetch

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

# Run api/mock-his/web as background processes; PIDs and output land in logs/.
start:
	@mkdir -p logs
	nohup $(MAKE) api > logs/api.log 2>&1 & echo $$! > logs/api.pid
	nohup $(MAKE) mock-his > logs/mock-his.log 2>&1 & echo $$! > logs/mock-his.pid
	nohup $(MAKE) web > logs/web.log 2>&1 & echo $$! > logs/web.pid
	@echo "started: api=$$(cat logs/api.pid) mock-his=$$(cat logs/mock-his.pid) web=$$(cat logs/web.pid)"
	@echo "logs: logs/api.log logs/mock-his.log logs/web.log"
	@echo "stop with: make stop"

# Stop processes started by `make deploy`.
stop:
	@for svc in api mock-his web; do \
		if [ -f logs/$$svc.pid ]; then \
			pid=$$(cat logs/$$svc.pid); \
			if kill -0 $$pid 2>/dev/null; then \
				pkill -P $$pid 2>/dev/null; \
				kill $$pid 2>/dev/null; \
				echo "stopped $$svc (pid $$pid)"; \
			else \
				echo "$$svc not running (stale pid $$pid)"; \
			fi; \
			rm -f logs/$$svc.pid; \
		else \
			echo "$$svc: no pid file"; \
		fi; \
	done

fetch:
	git fetch