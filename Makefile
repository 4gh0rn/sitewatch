.PHONY: help dev dev-build dev-up dev-down dev-logs test build clean docker-build docker-run prod-build prod-deploy

# Variables
APP_NAME := sitewatch
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DOCKER_IMAGE := $(APP_NAME):$(VERSION)
DOCKER_IMAGE_LATEST := $(APP_NAME):latest

# Default target
help: ## Show this help message
	@echo "$(APP_NAME) - Development & Production Commands"
	@echo ""
	@echo "Development Commands:"
	@echo "  dev          - Start development environment (docker-compose)"
	@echo "  dev-build    - Build development Docker image"
	@echo "  dev-up       - Start development containers"
	@echo "  dev-down     - Stop development containers"
	@echo "  dev-logs     - Show development logs"
	@echo "  dev-shell    - Open shell in development container"
	@echo ""
	@echo "Local Development:"
	@echo "  run          - Run locally (foreground)"
	@echo "  run-bg       - Run locally in background"
	@echo "  stop         - Stop background process"
	@echo "  restart      - Restart background process"
	@echo "  logs-local   - Show local application logs"
	@echo "  ps           - Show running processes"
	@echo "  kill-all     - Kill all SiteWatch processes (emergency)"
	@echo "  test         - Run tests"
	@echo "  build        - Build binary locally"
	@echo "  clean        - Clean build artifacts"
	@echo ""
	@echo "Production Commands:"
	@echo "  docker-build - Build production Docker image"
	@echo "  docker-run   - Run production Docker container"
	@echo "  prod-build   - Build production image with version tag"
	@echo "  prod-deploy  - Deploy to production (customize as needed)"
	@echo ""

# Development Commands
dev: dev-build dev-up ## Start complete development environment

dev-build: ## Build development Docker image
	@echo "🔨 Building development Docker image..."
	docker-compose -f deployments/docker/docker-compose.dev.yml build

dev-up: ## Start development containers
	@echo "🚀 Starting development environment..."
	docker-compose -f deployments/docker/docker-compose.dev.yml up -d
	@echo "✅ Development environment started!"
	@echo "📊 SiteWatch: http://localhost:8080"
	@echo "📈 Metrics (Prometheus format): http://localhost:8080/metrics"

dev-down: ## Stop development containers
	@echo "🛑 Stopping development environment..."
	docker-compose -f deployments/docker/docker-compose.dev.yml down

dev-logs: ## Show development logs
	docker-compose -f deployments/docker/docker-compose.dev.yml logs -f sitewatch

dev-shell: ## Open shell in development container
	docker-compose -f deployments/docker/docker-compose.dev.yml exec sitewatch sh

# Local Development (without Docker)
run: ## Run locally with Go
	@echo "🏃 Running locally..."
	@echo "📊 Development server starting on http://localhost:8080"
	@echo "🛑 Press Ctrl+C to stop"
	go run main.go

run-bg: ## Run locally in background
	@echo "🏃 Starting SiteWatch in background..."
	@nohup go run main.go > sitewatch.log 2>&1 & echo $$! > .sitewatch.pid
	@echo "✅ SiteWatch started in background (PID: $$(cat .sitewatch.pid))"
	@echo "📊 Available at: http://localhost:8080"
	@echo "📋 Logs: tail -f sitewatch.log"
	@echo "🛑 Stop with: make stop"

stop: ## Stop background process
	@if [ -f .sitewatch.pid ]; then \
		PID=$$(cat .sitewatch.pid); \
		if ps -p $$PID > /dev/null 2>&1; then \
			echo "🛑 Stopping SiteWatch (PID: $$PID)..."; \
			kill $$PID; \
			rm -f .sitewatch.pid; \
			echo "✅ SiteWatch stopped"; \
		else \
			echo "⚠️  Process $$PID not found, cleaning up..."; \
			rm -f .sitewatch.pid; \
		fi; \
	else \
		echo "❌ No background process found (.sitewatch.pid missing)"; \
	fi

restart: stop run-bg ## Restart background process

test: ## Run tests
	@echo "🧪 Running tests..."
	go test -v ./...

build: ## Build binary locally
	@echo "🔨 Building binary..."
	@mkdir -p bin
	go build -o bin/$(APP_NAME) .

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	rm -rf bin/$(APP_NAME)
	rm -f .sitewatch.pid sitewatch.log
	docker system prune -f

# Production Commands
docker-build: ## Build production Docker image
	@echo "🔨 Building production Docker image..."
	docker build -f deployments/docker/Dockerfile -t $(DOCKER_IMAGE_LATEST) .
	@echo "✅ Built: $(DOCKER_IMAGE_LATEST)"

docker-run: docker-build ## Run production Docker container
	@echo "🚀 Running production container..."
	docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		-v $(PWD)/configs/config.yaml:/app/configs/config.yaml:ro \
		-v $(PWD)/configs/sites.yaml:/app/configs/sites.yaml:ro \
		$(DOCKER_IMAGE_LATEST)
	@echo "✅ Production container started on http://localhost:8080"

prod-build: ## Build production image with version tag
	@echo "🔨 Building production image $(VERSION)..."
	docker build -f deployments/docker/Dockerfile -t $(DOCKER_IMAGE) -t $(DOCKER_IMAGE_LATEST) .
	@echo "✅ Built: $(DOCKER_IMAGE)"

prod-deploy: prod-build ## Deploy to production
	@echo "🚀 Deploying to production..."
	@echo "⚠️  Customize this target for your deployment environment"
	@echo "📦 Image ready: $(DOCKER_IMAGE)"
	# Add your deployment commands here:
	# docker push $(DOCKER_IMAGE)
	# kubectl apply -f k8s/
	# docker-compose -f docker-compose.prod.yml up -d

# Utility Commands
setup: ## Setup development environment
	@echo "🔧 Setting up development environment..."
	@echo "📝 Creating config files from examples..."
	@cp configs/config.example.yaml configs/config.yaml 2>/dev/null || echo "configs/config.yaml already exists"
	@cp configs/sites.example.yaml configs/sites.yaml 2>/dev/null || echo "configs/sites.yaml already exists"
	@echo "✅ Setup complete! Use 'make help' to see available commands"

logs: ## Show application logs (production)
	docker logs -f $(APP_NAME)

logs-local: ## Show local application logs
	@if [ -f sitewatch.log ]; then \
		tail -f sitewatch.log; \
	else \
		echo "❌ No local log file found. Use 'make run-bg' to start in background mode."; \
	fi

ps: ## Show running processes
	@echo "🔍 SiteWatch Processes:"
	@ps aux | grep -E "(sitewatch|go run main.go)" | grep -v grep || echo "❌ No SiteWatch processes found"

kill-all: ## Kill all SiteWatch processes (emergency stop)
	@echo "🔥 Killing all SiteWatch processes..."
	@pkill -f sitewatch 2>/dev/null && echo "✅ Binary processes killed" || echo "ℹ️  No binary processes found"
	@pkill -f "go run main.go" 2>/dev/null && echo "✅ Go run processes killed" || echo "ℹ️  No go run processes found"
	@rm -f .sitewatch.pid
	@echo "🧹 Cleaned up PID file"
	@echo "✅ All processes stopped"

status: ## Show container status
	@echo "📊 Container Status:"
	@docker ps --filter "name=$(APP_NAME)" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

health: ## Check application health
	@echo "🔍 Health Check:"
	@curl -s http://localhost:8080/health || echo "❌ Service not reachable"

metrics: ## Show current metrics
	@echo "📈 Current Metrics:"
	@curl -s http://localhost:8080/metrics | head -20

sites: ## Show all sites status
	@echo "🌐 Sites Status:"
	@curl -s http://localhost:8080/api/sites | jq '.' 2>/dev/null || curl -s http://localhost:8080/api/sites

# Authentication Commands
token-generate: ## Generate a new API token (usage: make token-generate name="Token Name" permissions="read,test")
	@go run tools/token-gen/main.go generate --name="$(name)" --permissions="$(permissions)"

token-list: ## List all configured API tokens
	@go run tools/token-gen/main.go list

ui-secret-generate: ## Generate a new UI secret
	@go run tools/token-gen/main.go ui-secret

auth-example: ## Show authentication configuration example
	@go run tools/token-gen/main.go example
