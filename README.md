# LiteBoxd

A lightweight K8s sandbox system inspired by e2b, designed to run on k3s.

## Features

- Sandbox lifecycle management (create, list, get, delete)
- Command execution in sandboxes
- File upload/download
- Automatic TTL-based cleanup
- Web UI for easy management

## Quick Start

### 1. Start k3s (Development Environment)

```bash
cd deploy
docker-compose up -d
```

Wait for k3s to be ready and generate kubeconfig:

```bash
# Wait for kubeconfig to be generated
until [ -f ./kubeconfig/kubeconfig.yaml ]; do sleep 2; done

# Set KUBECONFIG environment variable
export KUBECONFIG=$(pwd)/kubeconfig/kubeconfig.yaml

# Verify k3s is ready
kubectl get nodes
```

### 2. Start Backend

```bash
cd backend
go mod tidy
go run ./cmd/server
```

The API server will start on `http://localhost:8080`.

### 3. Start Frontend

```bash
cd web
npm install
npm run dev
```

The web UI will be available at `http://localhost:3000`.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/sandboxes | Create a new sandbox |
| GET | /api/v1/sandboxes | List all sandboxes |
| GET | /api/v1/sandboxes/:id | Get sandbox details |
| DELETE | /api/v1/sandboxes/:id | Delete a sandbox |
| POST | /api/v1/sandboxes/:id/exec | Execute command in sandbox |
| POST | /api/v1/sandboxes/:id/files | Upload file to sandbox |
| GET | /api/v1/sandboxes/:id/files | Download file from sandbox |

### Create Sandbox

```bash
curl -X POST http://localhost:8080/api/v1/sandboxes \
  -H "Content-Type: application/json" \
  -d '{
    "image": "python:3.11-slim",
    "cpu": "500m",
    "memory": "512Mi",
    "ttl": 3600
  }'
```

### Execute Command

```bash
curl -X POST http://localhost:8080/api/v1/sandboxes/{id}/exec \
  -H "Content-Type: application/json" \
  -d '{
    "command": ["python", "-c", "print(\"hello\")"],
    "timeout": 30
  }'
```

### Upload File

```bash
curl -X POST http://localhost:8080/api/v1/sandboxes/{id}/files \
  -F "file=@main.py" \
  -F "path=/workspace/main.py"
```

### Download File

```bash
curl -X GET "http://localhost:8080/api/v1/sandboxes/{id}/files?path=/workspace/output.txt" \
  -o output.txt
```

## Project Structure

```
liteboxd/
├── backend/                # Go backend
│   ├── cmd/server/         # Entry point
│   ├── internal/
│   │   ├── handler/        # HTTP handlers
│   │   ├── service/        # Business logic
│   │   ├── k8s/            # K8s client wrapper
│   │   └── model/          # Data models
│   └── Dockerfile
├── web/                    # Vue3 frontend
│   ├── src/
│   │   ├── views/          # Page components
│   │   ├── api/            # API client
│   │   └── router/         # Vue router
│   └── Dockerfile
├── deploy/
│   └── docker-compose.yml  # k3s deployment
└── scripts/
    └── dev-env.sh          # Development setup script
```

## Security

- Pods run as non-root user (UID 1000)
- Privilege escalation is disabled
- Resource limits prevent resource exhaustion
- All sandboxes run in dedicated namespace
- Seccomp profile enabled
