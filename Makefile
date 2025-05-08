ifneq (,$(wildcard .env))
  include .env
  export
endif

COMPOSE := docker compose

# Default environment is development
APP_ENV ?= dev

# Choose compose files based on environment
ifeq ($(APP_ENV),prod)
  COMPOSE_FILES := -f docker-compose.yml -f docker-compose.prod.yml
else
  COMPOSE_FILES := -f docker-compose.yml -f docker-compose.dev.yml
endif

.PHONY: up down build logs ps clean clean-all help migrate-up migrate-down test itest

help:
	@echo "Available commands:"
	@echo "  make up           Start all services in $(APP_ENV) mode"
	@echo "  make down         Stop all services"
	@echo "  make build        Rebuild all services"
	@echo "  make logs         Show logs from all services"
	@echo "  make ps           Show running services"
	@echo "  make clean        Stop and remove all containers, networks, and volumes"
	@echo "  make clean-all    Stop and remove all containers (including orphaned ones), networks, and volumes"
	@echo "  make migrate-up   Run database migrations up"
	@echo "  make migrate-down Run database migrations down"
	@echo "  make test         Run unit tests for the API"
	@echo "  make itest        Run integration tests for the API database"
	@echo ""
	@echo "Environment options:"
	@echo "  make APP_ENV=dev up    Start in development mode (default)"
	@echo "  make APP_ENV=prod up   Start in production mode"

up: build
	@echo ">>> Starting services in $(APP_ENV) mode..."
	@$(COMPOSE) $(COMPOSE_FILES) up -d

down:
	@echo ">>> Stopping services..."
	@$(COMPOSE) $(COMPOSE_FILES) down

build:
	@echo ">>> Building services in $(APP_ENV) mode..."
	@$(COMPOSE) $(COMPOSE_FILES) build

logs:
	@$(COMPOSE) $(COMPOSE_FILES) logs -f

ps:
	@$(COMPOSE) $(COMPOSE_FILES) ps

clean:
	@echo ">>> Cleaning up..."
	@$(COMPOSE) $(COMPOSE_FILES) down -v

clean-all:
	@echo ">>> Cleaning up including orphaned containers..."
	@$(COMPOSE) $(COMPOSE_FILES) down -v --remove-orphans

# Migration commands - run migrations on the database container
migrate-up:
	@echo ">>> Running migrations up..."
	@echo "Using DB connection for migrations"
	@migrate -path ./api-server/migrations -database "postgres://${DB_USERNAME}:${DB_PASSWORD}@localhost:5432/${DB_DATABASE}?sslmode=disable" up

migrate-down:
	@echo ">>> Running migrations down..."
	@echo "Using DB connection for migrations"
	@migrate -path ./api-server/migrations -database "postgres://${DB_USERNAME}:${DB_PASSWORD}@localhost:5432/${DB_DATABASE}?sslmode=disable" down

test:
	@echo ">>> Running unit tests..."
	@$(COMPOSE) $(COMPOSE_FILES) exec api go test ./... -v

itest:
	@echo ">>> Running integration tests..."
	@$(COMPOSE) $(COMPOSE_FILES) exec api go test ./internal/database -v
