.PHONY: build build-linux clean run help deploy-docker docker-local docker-restart docker-stop docker-logs remote-logs remote-restart remote-stop remote-status

# Binary name
BINARY_NAME=hubstack
OUTPUT_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GORUN=$(GOCMD) run

# Docker deployment parameters
HOST ?=
USER ?= $(shell whoami)
DEPLOY_PATH ?= /opt/hubstack

# Default target
all: build

# Build for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) -v ./cmd/hubstack

# Build for Linux (amd64)
build-linux:
	@echo "Building $(BINARY_NAME) for Linux (amd64)..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 -v ./cmd/hubstack

# Build for Linux (arm64) - for Raspberry Pi
build-linux-arm:
	@echo "Building $(BINARY_NAME) for Linux (arm64)..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-arm64 -v ./cmd/hubstack

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(OUTPUT_DIR)

# Run the application
run:
	$(GORUN) ./cmd/hubstack

# Run with custom port
run-port:
	$(GORUN) ./cmd/hubstack -port $(PORT)

# Docker: Build and test locally
docker-local: build-linux
	@echo "Checking for existing containers..."
	@if docker ps -a --filter "name=hubstack" --format "{{.Names}}" | grep -q "hubstack"; then \
		echo "Found existing container, stopping and removing..."; \
		docker-compose down; \
		echo "✓ Container stopped"; \
	fi
	@echo "Building and starting Docker container..."
	docker-compose up -d --build
	@echo "✓ Container started! Access at http://localhost:80"
	@echo "View logs with: docker-compose logs -f"

docker-restart:
	@echo "Restarting local Docker container..."
	docker-compose restart
	@echo "✓ Container restarted"
	@echo "View logs with: docker-compose logs -f"

# Docker: Stop local container
docker-stop:
	@echo "Stopping local Docker container..."
	docker-compose down
	@echo "✓ Container stopped"

# Docker: Deploy to remote host
deploy-docker: build-linux
	@if [ -z "$(HOST)" ]; then \
		echo "Error: HOST is required. Usage: make deploy-docker HOST=192.168.1.50"; \
		echo "Optional: USER=username DEPLOY_PATH=/custom/path"; \
		exit 1; \
	fi
	@echo "Deploying to $(USER)@$(HOST):$(DEPLOY_PATH)"
	./deploy.sh $(HOST) $(USER) $(DEPLOY_PATH)

# Docker: View logs on remote host
remote-logs:
	@if [ -z "$(HOST)" ]; then \
		echo "Error: HOST is required. Usage: make remote-logs HOST=192.168.1.50"; \
		exit 1; \
	fi
	ssh -t $(USER)@$(HOST) "cd $(DEPLOY_PATH) && docker-compose logs -f"

# Docker: Restart container on remote host
remote-restart:
	@if [ -z "$(HOST)" ]; then \
		echo "Error: HOST is required. Usage: make remote-restart HOST=192.168.1.50"; \
		exit 1; \
	fi
	@echo "Restarting container on $(USER)@$(HOST)..."
	ssh -t $(USER)@$(HOST) "cd $(DEPLOY_PATH) && docker-compose restart"

# Docker: Stop container on remote host
remote-stop:
	@if [ -z "$(HOST)" ]; then \
		echo "Error: HOST is required. Usage: make remote-stop HOST=192.168.1.50"; \
		exit 1; \
	fi
	@echo "Stopping container on $(USER)@$(HOST)..."
	ssh -t $(USER)@$(HOST) "cd $(DEPLOY_PATH) && docker-compose down"

# Docker: Check status of container on remote host
remote-status:
	@if [ -z "$(HOST)" ]; then \
		echo "Error: HOST is required. Usage: make remote-status HOST=192.168.1.50"; \
		exit 1; \
	fi
	ssh -t $(USER)@$(HOST) "cd $(DEPLOY_PATH) && docker-compose ps -a --filter 'name=hubstack'"

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Building:"
	@echo "  make build               - Build for current platform (output: $(OUTPUT_DIR)/$(BINARY_NAME))"
	@echo "  make build-linux         - Build for Linux amd64 (output: $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64)"
	@echo "  make build-linux-arm     - Build for Linux arm64 (output: $(OUTPUT_DIR)/$(BINARY_NAME)-linux-arm64)"
	@echo "  make clean               - Remove build artifacts"
	@echo ""
	@echo "Running locally:"
	@echo "  make run                 - Run the application with default settings"
	@echo "  make run-port PORT=3000  - Run with custom port"
	@echo ""
	@echo "Docker (local testing):"
	@echo "  make docker-local        - Build and run in Docker locally on port 80 (auto-restarts if running)"
	@echo "  make docker-restart      - Restart local Docker container"
	@echo "  make docker-stop         - Stop and remove local Docker container"
	@echo ""
	@echo "Docker (remote deployment):"
	@echo "  make deploy-docker HOST=192.168.1.50 [USER=user] [DEPLOY_PATH=/opt/hubstack]"
	@echo "                           - Build, copy files, and deploy to remote host"
	@echo "  make remote-logs HOST=192.168.1.50    - View logs from remote container"
	@echo "  make remote-restart HOST=192.168.1.50 - Restart remote container"
	@echo "  make remote-stop HOST=192.168.1.50    - Stop remote container"
	@echo "  make remote-status HOST=192.168.1.50  - Check status of remote container"
	@echo ""
	@echo "Other:"
	@echo "  make help                - Show this help message"

