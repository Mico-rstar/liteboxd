# LiteBoxd

A lightweight K8s sandbox system inspired by e2b, designed to run on k3s.

## Features

- Sandbox lifecycle management (create, list, get, delete)
- Command execution in sandboxes
- File upload/download
- Automatic TTL-based cleanup
- Web UI for easy management

## Quick Start

### Prerequisites

```bash
cd backend && go mod tidy
```

### 1. Start k3s (Development Environment)

```bash
make start-k3s
```

Wait for k3s to be ready (kubeconfig will be auto-generated).

### 2. Start Backend

```bash
make run-backend
```

The API server will start on `http://localhost:8080`.

### 3. Start Frontend

```bash
make run-frontend
```

The web UI will be available at `http://localhost:3000`.

### All-in-One

Run both backend and frontend:

```bash
make run-all
```

> **Tip**: Run `make help` to see all available commands.

## Security

- Pods run as non-root user (UID 1000)
- Privilege escalation is disabled
- Resource limits prevent resource exhaustion
- All sandboxes run in dedicated namespace
- Seccomp profile enabled

## License

[GPL-3.0](LICENSE)
