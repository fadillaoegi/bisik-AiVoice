.PHONY: help db db-down db-reset migrate dev build test tidy psql auth-secret

# Muat konfigurasi lokal tanpa menaruh secret di source code atau command line.
# File .env diabaikan Git.
-include .env
export

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

db: ## Jalankan Postgres di Docker (migrasi dijalankan backend saat startup)
	docker compose up -d postgres

db-down: ## Hentikan Postgres
	docker compose down

db-reset: ## Hapus volume database lalu buat Postgres kosong
	docker compose down -v && docker compose up -d postgres

psql: ## Masuk ke shell psql
	docker compose exec postgres psql -U saksi -d saksi

migrate: ## Jalankan migrasi database tanpa menyalakan API
	go run ./cmd/migrate

dev: ## Jalankan API di host (butuh `make db` dulu)
	go run ./cmd/api

build: ## Build binary ke bin/bisik
	go build -o bin/bisik ./cmd/api

test: ## Jalankan test
	go test ./...

tidy: ## Rapikan dependensi
	go mod tidy

auth-secret: ## Buat nilai acak untuk AUTH_SECRET
	@head -c 32 /dev/urandom | base64
