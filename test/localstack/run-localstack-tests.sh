#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

LOCALSTACK_ENDPOINT="http://localhost:4566"

echo -e "${GREEN}=== LocalStack Integration Test Runner ===${NC}"
echo ""

# Set DOCKER_HOST for Podman on macOS/Linux
if [ -z "${DOCKER_HOST:-}" ]; then
    if [ -S "/var/run/docker.sock" ]; then
        export DOCKER_HOST=unix:///var/run/docker.sock
    elif [ -S "$HOME/.docker/run/docker.sock" ]; then
        export DOCKER_HOST=unix://$HOME/.docker/run/docker.sock
    fi
fi

# Start LocalStack
echo -e "${YELLOW}Starting LocalStack...${NC}"
cd "${PROJECT_ROOT}"
docker-compose -f docker-compose.localstack.yaml up -d

# Wait for LocalStack to be ready
echo -e "${YELLOW}Waiting for LocalStack to be ready...${NC}"
for i in {1..30}; do
    if curl -s "${LOCALSTACK_ENDPOINT}/_localstack/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ LocalStack is ready${NC}"
        break
    fi
    echo -n "."
    sleep 1
done
echo ""

# Build the binary
echo -e "${YELLOW}Building rosactl binary...${NC}"
go build -o rosactl ./cmd/rosactl
echo -e "${GREEN}✓ Binary built${NC}"
echo ""

# Set environment variables for tests
export LOCALSTACK_ENDPOINT="${LOCALSTACK_ENDPOINT}"
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export AWS_REGION="us-east-1"

# Run Ginkgo tests
echo -e "${YELLOW}Running Ginkgo tests against LocalStack...${NC}"
cd test/localstack
ginkgo -v

# Capture exit code
TEST_EXIT_CODE=$?

# Cleanup (optional - comment out if you want to inspect LocalStack after tests)
echo ""
read -p "Stop LocalStack? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Stopping LocalStack...${NC}"
    cd "${PROJECT_ROOT}"
    docker-compose -f docker-compose.localstack.yaml down -v
    echo -e "${GREEN}✓ LocalStack stopped${NC}"
fi

exit $TEST_EXIT_CODE
