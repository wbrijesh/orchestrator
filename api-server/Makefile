ifneq (,$(wildcard .env))
  include .env
  export
endif

COMPOSE            := docker compose
COMPOSE_FILES_DEV  := -f docker-compose.yml -f docker-compose.dev.yml
COMPOSE_FILES_PROD := -f docker-compose.yml

# shorthand for whichever compose file list we need
ifeq ($(APP_ENV),prod)
  COMPOSE_FILES := $(COMPOSE_FILES_PROD)
else
  COMPOSE_FILES := $(COMPOSE_FILES_DEV)
endif


# Build the application
all: build test

build:
	@echo "Building..."
	@go build -o main cmd/api/main.go

# Run the application
run:
	@go run cmd/api/main.go

# start app and db
docker-run:
	@echo ">>> Starting in '$(APP_ENV)' APP_ENV..."
	@$(COMPOSE) $(COMPOSE_FILES) up --build

# stop app and db
docker-down:
	@echo ">>> Stopping all services..."
	@$(COMPOSE) $(COMPOSE_FILES) down

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Integrations Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch:
	@if command -v air > /dev/null; then \
        air; \
        echo "Watching...";\
    else \
        read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
        if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
            go install github.com/air-verse/air@latest; \
            air; \
            echo "Watching...";\
        else \
            echo "You chose not to install air. Exiting..."; \
            exit 1; \
        fi; \
    fi

migrate-up:
	@migrate -path ./migrations -database "$(DB_URL)" up

migrate-down:
	@migrate -path ./migrations -database "$(DB_URL)" down

.PHONY: all build run test clean watch docker-run docker-down itest migrate-up migrate-down
