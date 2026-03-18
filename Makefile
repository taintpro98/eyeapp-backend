.PHONY: build build-migrate run test clean docker-build docker-up docker-down help swagger migrate-up migrate-version migrate-create

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
BINARY_NAME=api
MIGRATE_BINARY=migrate

# Docker parameters
DOCKER_COMPOSE=docker compose

# Swagger parameters
SWAG=$(shell go env GOPATH)/bin/swag

## help: Show this help message
help:
	@echo "ALumiEye Backend API"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'

## build: Build the API application
build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/api

## build-migrate: Build the migrate tool
build-migrate:
	$(GOBUILD) -o $(MIGRATE_BINARY) ./cmd/migrate

## build-all: Build all binaries
build-all: build build-migrate

## run: Run the application locally
run:
	$(GOCMD) run ./cmd/api

## test: Run tests
test:
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(MIGRATE_BINARY)
	rm -f coverage.out coverage.html

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## swagger-install: Install swag CLI tool
swagger-install:
	$(GOCMD) install github.com/swaggo/swag/cmd/swag@latest

## swagger: Generate Swagger documentation
swagger:
	$(SWAG) init -g cmd/api/main.go -o docs --parseDependency --parseInternal

## swagger-fmt: Format Swagger comments
swagger-fmt:
	$(SWAG) fmt

## migrate-up: Apply all database migrations
migrate-up:
	$(GOCMD) run ./cmd/migrate up

## migrate-version: Show current migration version
migrate-version:
	$(GOCMD) run ./cmd/migrate version

## migrate-create: Create a new migration (usage: make migrate-create name=migration_name)
migrate-create:
	$(GOCMD) run ./cmd/migrate create $(name)

## migrate-force: Force migration version (usage: make migrate-force version=1)
migrate-force:
	$(GOCMD) run ./cmd/migrate force $(version)

## docker-build: Build Docker image
docker-build:
	$(DOCKER_COMPOSE) build

## docker-up: Start all services with Docker Compose
docker-up:
	$(DOCKER_COMPOSE) up -d

## docker-down: Stop all services
docker-down:
	$(DOCKER_COMPOSE) down

## docker-logs: View logs from all services
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## docker-restart: Restart all services
docker-restart: docker-down docker-up

## db-shell: Connect to PostgreSQL shell
db-shell:
	$(DOCKER_COMPOSE) exec postgres psql -U postgres -d alumieye

## lint: Run linter
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GOCMD) fmt ./...
