.PHONY: help db db-down db-reset db-prod db-prod-down db-prod-reset migrate scorer-probe add-user dev build test tidy psql psql-prod auth-secret

# Muat konfigurasi lokal tanpa menaruh secret di source code atau command line.
# File .env diabaikan Git.
-include .env
export

# Compose file shorthand
DC_DEV  = docker compose -f docker-compose.yml -f docker-compose.dev.yml
DC_PROD = docker compose -f docker-compose.yml -f docker-compose.prod.yml

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# ── Development (port 5432 / 8080) ──────────────────────────────────────────
db: ## [dev]  Jalankan Postgres di Docker
	$(DC_DEV) up -d postgres

db-down: ## [dev]  Hentikan semua service
	$(DC_DEV) down

db-reset: ## [dev]  Hapus volume lalu buat Postgres kosong
	$(DC_DEV) down -v && $(DC_DEV) up -d postgres

psql: ## [dev]  Masuk ke shell psql
	$(DC_DEV) exec postgres psql -U bisik -d bisik

# ── Production (port 8100 / 8101) ────────────────────────────────────────────
db-prod: ## [prod] Jalankan Postgres di Docker
	$(DC_PROD) up -d postgres

db-prod-down: ## [prod] Hentikan semua service
	$(DC_PROD) down

db-prod-reset: ## [prod] Hapus volume lalu buat Postgres kosong
	$(DC_PROD) down -v && $(DC_PROD) up -d postgres

psql-prod: ## [prod] Masuk ke shell psql
	$(DC_PROD) exec postgres psql -U bisik -d bisik

migrate: ## Jalankan migrasi database tanpa menyalakan API
	go run ./cmd/migrate

scorer-probe: ## Uji kualitas tiap penyedia penilai (LLM_PROVIDERS) dengan kalimat yang sudah diketahui jawabannya
	go run ./cmd/scorerprobe

add-user: ## Tambah akun: make add-user USERNAME=handoko NAME="Handoko" [ROLE=supervisor]
	go run ./cmd/adduser -username "$(USERNAME)" -name "$(NAME)" -role "$(or $(ROLE),officer)"

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
