#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR/deploy"

echo "Starting k3s..."
docker-compose up -d

echo "Waiting for k3s to be ready..."
sleep 10

until [ -f ./kubeconfig/kubeconfig.yaml ]; do
    echo "Waiting for kubeconfig..."
    sleep 2
done

export KUBECONFIG="$PROJECT_DIR/deploy/kubeconfig/kubeconfig.yaml"

# Fix server address for local access
sed -i 's/127.0.0.1/host.docker.internal/g' "$KUBECONFIG" 2>/dev/null || \
sed -i '' 's/127.0.0.1/host.docker.internal/g' "$KUBECONFIG" 2>/dev/null || true

echo "Waiting for k3s API to be ready..."
until kubectl get nodes &>/dev/null; do
    echo "Waiting for k3s API..."
    sleep 2
done

echo "k3s is ready!"
kubectl get nodes

echo ""
echo "Environment setup complete!"
echo "Run the following commands to start development:"
echo ""
echo "  export KUBECONFIG=$KUBECONFIG"
echo "  cd $PROJECT_DIR/backend && go run ./cmd/server"
echo ""
