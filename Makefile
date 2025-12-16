.PHONY: help install dev build migrate-up migrate-down migrate-redo migrate-status migrate-create migrate-delete migrate-lock migrate-unlock templ-generate tailwind-watch tailwind-build clean dbup dbdown

help:
	@echo "Available commands:"
	@echo "  make install         - Install dependencies"
	@echo "  make dev             - Run development server with hot reload"
	@echo "  make dbup            - Bring up dockerized database"
	@echo "  make dbdown          - Bring down dockerized database"
	@echo "  make build           - Build production binary"
	@echo "  make migrate-up      - Apply all pending migrations"
	@echo "  make migrate-down    - Rollback last migration"
	@echo "  make migrate-redo    - Rollback and re-apply last migration"
	@echo "  make migrate-status  - Show migration status"
	@echo "  make migrate-create  - Create a new migration (usage: make migrate-create name=migration_name)"
	@echo "  make migrate-delete  - Delete unapplied migration (usage: make migrate-delete name=20241124000001_migration_name)"
	@echo "  make migrate-lock    - Show migration lock status"
	@echo "  make migrate-unlock  - Force release migration lock"
	@echo "  make templ-generate  - Generate templ templates"
	@echo "  make tailwind-watch  - Watch and build Tailwind CSS"
	@echo "  make tailwind-build  - Build Tailwind CSS for production"
	@echo "  make clean           - Clean build artifacts"

install:
	go mod download
	go install github.com/air-verse/air@latest
	go install github.com/a-h/templ/cmd/templ@latest
	npm install -D tailwindcss

dev:
	air

templ-watch:
	templ generate --watch

templ-generate:
	templ generate

tailwind-watch:
	npx tailwindcss -i ./static/css/input.css -o ./static/css/output.css --watch

tailwind-build:
	npx tailwindcss -i ./static/css/input.css -o ./static/css/output.css --minify

migrate-up:
	@go run cmd/migrate/main.go up

migrate-down:
	@go run cmd/migrate/main.go down

migrate-redo:
	@go run cmd/migrate/main.go redo

migrate-status:
	@go run cmd/migrate/main.go status

migrate-create:
	@go run cmd/migrate/main.go create $(name)

migrate-delete:
	@go run cmd/migrate/main.go delete $(name)

migrate-lock:
	@go run cmd/migrate/main.go lock

migrate-unlock:
	@go run cmd/migrate/main.go unlock

build: templ-generate tailwind-build
	go build -o bin/server cmd/server/main.go

dbup:
	@docker compose up -d

dbdown:
	@docker compose down

clean:
	rm -rf bin/
	rm -f static/css/output.css
	find . -name "*_templ.go" -type f -delete
	docker compose down -v
