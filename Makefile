.PHONY: generate test test-unit test-int lint migrate-up docker-up docker-down docs-preview docs-deploy

generate: ## Generate code from OpenAPI spec
	@bash scripts/generate-openapi.sh

test: ## Run all tests
	cd pkg && go test ./...
	cd internal/orderintake && go test ./...

test-unit: ## Run unit tests only
	cd pkg && go test -short ./...
	cd internal/orderintake && go test -short ./...

test-int: ## Run integration tests only
	cd pkg && go test -run Integration ./...
	cd internal/orderintake && go test -run Integration ./...

lint: ## Run Go linters
	cd pkg && go vet ./...
	cd internal/orderintake && go vet ./...

migrate-up: ## Apply database migrations
	migrate -database "postgres://oms:oms_secret@localhost:5432/oms_orderintake?sslmode=disable" -path migrations/orderintake up

docker-up: ## Start infrastructure (Postgres, Kafka)
	docker-compose up -d

docker-down: ## Stop infrastructure
	docker-compose down

docs-preview: ## Preview documentation locally with Jekyll
	@echo "Starting Jekyll server..."
	@cd docs && bundle install && bundle exec jekyll serve --livereload
	@echo "Docs available at http://localhost:4000/oms/"

docs-deploy: ## Deploy documentation to GitHub Pages
	@echo "Deploying to GitHub Pages..."
	@git add docs/ README.md
	@git commit -m "docs: update documentation" || echo "No changes to commit"
	@git push origin main
	@echo "✅ Deployed to https://mcp-log.github.io/oms/"
