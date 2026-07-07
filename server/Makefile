.PHONY: dev build test migrate-up migrate-down migrate-create sync-events

# Run the API server in development mode (with hot reload via air if available)
dev:
	@which air > /dev/null 2>&1 && air || go run ./cmd/api

# Build the API server binary
build:
	go build -o bin/api ./cmd/api

# Run database migrations up
migrate-up:
	migrate -path migrations -database "$$DATABASE_URL" up

# Run database migrations down
migrate-down:
	migrate -path migrations -database "$$DATABASE_URL" down

# Create a new migration file
# Usage: make migrate-create NAME=add_users_table
migrate-create:
	migrate create -ext sql -dir migrations -seq $(NAME)

# Sync comic events from allcpp.cn
sync-events:
	go run ./cmd/sync

# Run all tests
test:
	go test ./... -v
