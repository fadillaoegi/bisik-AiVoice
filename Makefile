.PHONY: deploy-build deploy-image

deploy-build: ## Build frontend memakai Node/pnpm host
	cd bisik_frontend && pnpm install --frozen-lockfile && pnpm build

deploy-image: deploy-build ## Build image app setelah frontend selesai
	docker build -t bisik:deployment-test .
