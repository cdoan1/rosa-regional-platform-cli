# LocalStack Integration Tests

This directory contains integration tests that run `rosactl` commands against LocalStack.

## Overview

These tests verify that the CLI correctly creates AWS resources (VPC, IAM, CloudFormation stacks) by running against a local LocalStack instance instead of real AWS.

## Prerequisites

- **Docker or Podman** and Docker Compose
  - Podman users (Fedora/RHEL): The compose file automatically uses your Podman socket at `/run/user/$(id -u)/podman/podman.sock`
  - Docker users: Set `export DOCKER_SOCK=/var/run/docker.sock` before running
- Go 1.24+
- Ginkgo CLI (install if not present):
  ```bash
  go install github.com/onsi/ginkgo/v2/ginkgo@latest
  ```
- AWS CLI v2 (optional, for manual inspection)

## Running Tests

### Quick Start

```bash
# From the project root
./test/localstack/run-localstack-tests.sh
```

This script will:
1. Start LocalStack via docker-compose
2. Build the `rosactl` binary
3. Run Ginkgo tests against LocalStack
4. Optionally stop LocalStack when done

### Manual Execution

```bash
# 0. (Docker users only) Set Docker socket location
export DOCKER_SOCK=/var/run/docker.sock

# 1. Start LocalStack
docker-compose -f docker-compose.localstack.yaml up -d

# 2. Build binary
go build -o rosactl ./cmd/rosactl

# 3. Set environment variables
export LOCALSTACK_ENDPOINT="http://localhost:4566"
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export AWS_REGION="us-east-1"

# 4. Run tests
cd test/localstack
ginkgo -v

# 5. Cleanup
docker-compose -f docker-compose.localstack.yaml down -v
```

## What's Tested

### VPC Management (`cluster-vpc`)
- CloudFormation template validation
- VPC resource creation
- Subnet creation across availability zones
- Security group creation
- Route53 private hosted zone creation

### IAM Management (`cluster-iam`)
- CloudFormation template validation
- IAM OIDC provider creation
- Control plane IAM roles (7 roles)
- Worker node IAM role and instance profile

### CloudFormation Operations
- Stack creation
- Stack listing
- Stack status checking
- Stack deletion

## Test Structure

```
test/localstack/
├── README.md                      # This file
├── run-localstack-tests.sh        # Test runner script
├── localstack_suite_test.go       # Ginkgo suite setup
└── localstack_test.go             # Integration tests
```

## LocalStack Configuration

The `docker-compose.localstack.yaml` file configures LocalStack with:
- CloudFormation
- IAM
- EC2
- Route53
- Lambda (for future Lambda execution tests)
- ECR (for container image testing)
- S3 (if needed)

## Current Limitations

1. **Lambda Execution Not Tested**: Tests currently create CloudFormation stacks directly rather than invoking the Lambda function. This will be added in a future iteration.

2. **LocalStack Pro Features**: Some AWS features may require LocalStack Pro (e.g., full IAM simulation, VPC endpoints).

3. **Resource Validation**: Tests verify that resources are created but don't deeply validate all properties (e.g., IAM policy attachments, security group rules).

## Future Enhancements

- [ ] Test actual Lambda invocation (requires Lambda function deployment to LocalStack)
- [ ] Add tests for `bootstrap` command
- [ ] Validate CloudFormation stack outputs
- [ ] Test stack update operations
- [ ] Test error handling and rollback scenarios
- [ ] Add integration with CI/CD (Prow)

## Debugging

### View LocalStack logs
```bash
docker-compose -f docker-compose.localstack.yaml logs -f
```

### Check LocalStack health
```bash
curl http://localhost:4566/_localstack/health
```

### List resources in LocalStack
```bash
# CloudFormation stacks
aws cloudformation list-stacks --endpoint-url http://localhost:4566

# VPCs
aws ec2 describe-vpcs --endpoint-url http://localhost:4566

# IAM roles
aws iam list-roles --endpoint-url http://localhost:4566
```

### Keep LocalStack running after tests
Edit `run-localstack-tests.sh` and comment out the cleanup section, or answer 'N' when prompted to stop LocalStack.

## Troubleshooting

**Problem**: LocalStack doesn't start
- **Podman users**: Make sure Podman socket is running: `systemctl --user status podman.socket`
  - Start if needed: `systemctl --user start podman.socket`
- **Docker users**: Set `export DOCKER_SOCK=/var/run/docker.sock` before starting
- Check Docker/Podman is running: `docker ps` or `podman ps`
- Check logs: `docker-compose -f docker-compose.localstack.yaml logs`

**Problem**: Tests fail with connection refused
- Verify LocalStack is healthy: `curl http://localhost:4566/_localstack/health`
- Check `LOCALSTACK_ENDPOINT` environment variable is set correctly

**Problem**: CloudFormation stack creation hangs
- LocalStack may not support all CloudFormation resource types
- Check LocalStack logs for unsupported features
- Consider using LocalStack Pro for advanced features
