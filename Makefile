.PHONY: dev api web db db-stop migrate test-scan clean

# Run everything
dev:
	@echo "Starting PostgreSQL, API, and frontend..."
	@make db &
	@make api &
	@make web
	@wait

# Go API with hot reload
api:
	cd api && air

# SvelteKit dev server
web:
	cd web && bun dev

# PostgreSQL via Docker
db:
	docker run --name securescan-pg \
		-e POSTGRES_USER=securescan \
		-e POSTGRES_PASSWORD=securescan \
		-e POSTGRES_DB=securescan \
		-p 5432:5432 \
		-v $(PWD)/pgdata:/var/lib/postgresql/data \
		--rm postgres:18

db-stop:
	docker stop securescan-pg

# Run migrations
migrate:
	cd api && go run main.go --migrate

# Test scan against Juice Shop
test-scan:
	curl -s -X POST http://localhost:3000/api/projects \
		-H "Content-Type: application/json" \
		-d '{"name":"juice-shop","source_type":"git","source_url":"https://github.com/juice-shop/juice-shop"}' | jq .

# Clean up
clean:
	rm -rf /tmp/securescan pgdata/
	docker stop securescan-pg 2>/dev/null || true
