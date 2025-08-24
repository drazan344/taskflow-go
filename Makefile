.PHONY: build run test test-unit test-integration test-e2e test-coverage test-benchmark clean docker-build docker-run migrate-up migrate-down swagger fmt lint deps init

# Build the application
build:
	@echo "Building taskflow-go..."
	@go build -o bin/api cmd/api/main.go
	@go build -o bin/worker cmd/worker/main.go

# Run the application
run:
	@echo "Running taskflow-go..."
	@go run cmd/api/main.go

# Run the worker
run-worker:
	@echo "Running worker..."
	@go run cmd/worker/main.go

# Run all tests
test:
	@echo "Running all tests..."
	@go test -v -race ./tests/unit/... ./tests/integration/...

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@go test -v ./tests/unit/...

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	@go test -v ./tests/integration/...

# Run E2E tests (requires running server)
test-e2e:
	@echo "Running E2E tests..."
	@echo "Make sure the server is running on localhost:8080"
	@go test -v ./tests/api/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./tests/unit/... ./tests/integration/...
	@go tool cover -html=coverage.out -o coverage.html

# Run benchmark tests
test-benchmark:
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./tests/...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Generate swagger docs
swagger:
	@echo "Generating swagger docs..."
	@swag init -g cmd/api/main.go -o docs

# Database migrations
migrate-up:
	@echo "Running database migrations up..."
	@migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

migrate-down:
	@echo "Running database migrations down..."
	@migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down

# Docker commands
docker-build:
	@echo "Building Docker image..."
	@docker build -t taskflow-go .

docker-run:
	@echo "Running Docker container..."
	@docker-compose up -d

docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Initialize project (first time setup)
init:
	@echo "Initializing project..."
	@cp .env.example .env
	@echo "Please edit .env file with your configuration"