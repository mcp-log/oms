.PHONY: generate test test-unit test-int lint migrate-up docker-up docker-down

generate:
	@bash scripts/generate-openapi.sh

test:
	cd pkg && go test ./...
	cd internal/orderintake && go test ./...

test-unit:
	cd pkg && go test -short ./...
	cd internal/orderintake && go test -short ./...

test-int:
	cd pkg && go test -run Integration ./...
	cd internal/orderintake && go test -run Integration ./...

lint:
	cd pkg && go vet ./...
	cd internal/orderintake && go vet ./...

migrate-up:
	migrate -database "postgres://oms:oms_secret@localhost:5432/oms_orderintake?sslmode=disable" -path migrations/orderintake up

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
