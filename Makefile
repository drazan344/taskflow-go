.PHONY: build run test clean docker-build docker-run migrate-up migrate-down swagger

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

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

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