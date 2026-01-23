.PHONY: help run-backend run-frontend run-all stop-k3s start-k3s build-backend build-frontend clean fmt

help:
	@echo "LiteBoxd Development Commands"
	@echo ""
	@echo "  make start-k3s      - Start k3s in Docker"
	@echo "  make stop-k3s       - Stop k3s"
	@echo "  make run-backend    - Run backend server"
	@echo "  make run-frontend   - Run frontend dev server"
	@echo "  make run-all        - Run both backend and frontend"
	@echo "  make build-backend  - Build backend binary"
	@echo "  make build-frontend - Build frontend for production"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make fmt            - Format backend and frontend code"

# K3s management
start-k3s:
	cd deploy && docker-compose up -d
	@echo "Waiting for kubeconfig..."
	@until [ -f deploy/kubeconfig/kubeconfig.yaml ]; do sleep 2; done
	@echo "K3s is ready!"

stop-k3s:
	cd deploy && docker-compose down

# Development
run-backend:
	@export KUBECONFIG=$(PWD)/deploy/kubeconfig/kubeconfig.yaml && \
	cd backend && go run ./cmd/server

run-frontend:
	cd web && npm run dev

run-all:
	@echo "Starting backend and frontend..."
	@make run-backend &
	@make run-frontend

# Build
build-backend:
	cd backend && go build -o bin/server ./cmd/server

build-frontend:
	cd web && npm run build

# Clean
clean:
	rm -rf backend/bin
	rm -rf web/dist

# Format
fmt:
	@echo "Formatting backend..."
	cd backend && go fmt ./...
	@echo "Formatting frontend..."
	cd web && npm run format
	@echo "Done!"
