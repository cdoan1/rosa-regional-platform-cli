# rosactl

A command-line tool for ROSA Regional Platform

* Used to managing AWS Lambda functions and S3-backed OIDC (OpenID Connect) identity providers.

![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)

## Features

### Lambda Functions
- **ZIP-based deployment**: Python Lambda functions with ZIP packaging
- **Specialized handlers**: Default Python handler, OIDC issuer management, OIDC deletion
- **Full lifecycle management**: Create, invoke, list, update, delete, and version Lambda functions
- **Automatic IAM role management**: Creates execution roles with appropriate permissions

### OIDC Providers (OpenID Connect)
- **S3-backed OIDC issuers**: Create OIDC providers hosted on S3 buckets
- **RSA key pair generation**: Automatic key generation with secure private key storage
- **IAM integration**: Automatically registers OIDC providers with AWS IAM
- **Complete lifecycle**: Create, list, and delete OIDC issuers

### Developer Experience
- **Semantic versioning**: Automated version management with conventional commits
- **Comprehensive testing**: End-to-end tests against real AWS services
- **Clear error messages**: User-friendly error reporting
- **Extensive documentation**: Architecture docs, guides, and examples

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/openshift-online/rosa-regional-platform-cli.git
cd rosa-regional-platform-cli

# Build
make build

# Install globally (optional)
make install
```

### Basic Usage

```bash
# Create a basic Lambda function
rosactl lambda create my-function

# Create an OIDC issuer management Lambda
rosactl lambda create my-oidc --handler oidc

# Create an OIDC issuer
rosactl oidc create my-cluster --function my-oidc

# List OIDC issuers
rosactl oidc list

# Check version
rosactl version
```

## Prerequisites

- **Go 1.24+** (for building from source)
- **AWS credentials** configured via:
  - `~/.aws/credentials` file, or
  - Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`), or
  - IAM role (when running on EC2/ECS)
- **AWS IAM permissions**:
  - Lambda: `CreateFunction`, `DeleteFunction`, `InvokeFunction`, `GetFunction`, `ListFunctions`, `UpdateFunctionCode`, `PublishVersion`
  - IAM: `CreateRole`, `GetRole`, `AttachRolePolicy`, `PutRolePolicy`
  - S3 (for OIDC): `CreateBucket`, `DeleteBucket`, `PutObject`, `DeleteObject`, `ListBucket`, `PutBucketPolicy`, `PutPublicAccessBlock`
  - IAM OIDC: `CreateOpenIDConnectProvider`, `DeleteOpenIDConnectProvider`, `ListOpenIDConnectProviders`, `GetOpenIDConnectProvider`

### Optional Tools

- **go-semver-release** - For semantic versioning (install: `go install github.com/s0ders/go-semver-release@latest`)

## AWS Configuration

rosactl uses the AWS default credential chain:

```bash
# Option 1: AWS credentials file
cat ~/.aws/credentials
[default]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
region = us-east-1

# Option 2: Environment variables
export AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY
export AWS_SECRET_ACCESS_KEY=YOUR_SECRET_KEY
export AWS_REGION=us-east-1

# Option 3: AWS profile
export AWS_PROFILE=your-profile-name
```

## Usage

### Lambda Functions

#### Create Lambda Functions

```bash
# Lambda with default handler (hello world)
rosactl lambda create my-function

# OIDC issuer management Lambda
rosactl lambda create my-oidc-issuer --handler oidc

# OIDC deletion Lambda
rosactl lambda create my-oidc-cleanup --handler oidc-delete
```

**Handler types:**
- `default` - Basic Python handler that returns "hello world" with timestamp
- `oidc` - Creates S3-backed OIDC issuers with IAM providers
- `oidc-delete` - Deletes S3 buckets and IAM OIDC providers

#### Invoke Lambda Functions

```bash
# Invoke a function
rosactl lambda invoke my-function

# Invoke with specific version
rosactl lambda invoke my-function --version 2
```

#### List Lambda Functions

```bash
# Table format (default)
rosactl lambda list

# JSON format
rosactl lambda list --output json
```

#### Delete Lambda Functions

```bash
rosactl lambda delete my-function
```

#### List Function Versions

```bash
rosactl lambda versions my-function
```

### OIDC Providers

#### Create OIDC Issuer

```bash
# First, create the OIDC management Lambda (if not already created)
rosactl lambda create my-oidc-issuer --handler oidc

# Create an OIDC issuer
rosactl oidc create my-cluster --region us-east-1 --function my-oidc-issuer
```

**What this creates:**
1. S3 bucket: `oidc-issuer-my-cluster`
2. OIDC discovery documents: `.well-known/openid-configuration` and `keys.json`
3. IAM OIDC provider pointing to the S3 bucket

**RSA Private Key:**
When creating an OIDC Lambda with `--handler oidc`, the RSA private key is saved to:
```
/tmp/oidc-private-key-{KEY_ID}.pem
```

⚠️ **Keep this file secure!** It can be used to sign JWTs for the OIDC issuer.

#### List OIDC Issuers

```bash
# Table format
rosactl oidc list

# JSON format
rosactl oidc list --output json
```

#### Delete OIDC Issuer

```bash
# First, create the deletion Lambda (if not already created)
rosactl lambda create my-oidc-cleanup --handler oidc-delete

# Delete an OIDC issuer
rosactl oidc delete my-cluster --region us-east-1 --function my-oidc-cleanup
```

### Version Management

```bash
# Check current version
rosactl version

# Check what next version would be (based on commits)
make release-dry-run

# Create a semantic version release
make release
```

See [docs/guides/VERSIONING.md](docs/guides/VERSIONING.md) for details on semantic versioning.

## Examples

### Complete Lambda Workflow

```bash
# Create a function
rosactl lambda create hello-world
# Successfully created Lambda function: hello-world
# ARN: arn:aws:lambda:us-east-1:123456789012:function:hello-world (version: 1)

# Invoke it
rosactl lambda invoke hello-world
# {
#   "statusCode": 200,
#   "body": "{\"current_time\": \"2026-02-22T21:30:00+00:00\", \"message\": \"hello world\", \"ago\": \"5 seconds ago\"}"
# }

# List all functions
rosactl lambda list
# NAME          RUNTIME      CREATED
# hello-world   python3.12   2026-02-22T21:29:55+00:00

# Delete it
rosactl lambda delete hello-world
# Successfully deleted Lambda function: hello-world
```

### Complete OIDC Workflow

```bash
# Step 1: Create OIDC management Lambdas
rosactl lambda create oidc-issuer --handler oidc
# ⚠️  Private key saved to: /tmp/oidc-private-key-1a2b3c4d5e6f7890.pem
# ⚠️  Keep this file secure! It can be used to sign JWTs for this OIDC issuer.

rosactl lambda create oidc-cleanup --handler oidc-delete

# Step 2: Create an OIDC issuer
rosactl oidc create production-cluster --region us-east-1 --function oidc-issuer
# Using bucket name: oidc-issuer-production-cluster
# Invoking OIDC Lambda function 'oidc-issuer'...
#
# OIDC Issuer created successfully:
# {
#   "bucket_name": "oidc-issuer-production-cluster",
#   "issuer_url": "https://oidc-issuer-production-cluster.s3.us-east-1.amazonaws.com",
#   "provider_arn": "arn:aws:iam::123456789012:oidc-provider/oidc-issuer-production-cluster.s3.us-east-1.amazonaws.com",
#   "discovery_url": "https://oidc-issuer-production-cluster.s3.us-east-1.amazonaws.com/.well-known/openid-configuration",
#   "jwks_url": "https://oidc-issuer-production-cluster.s3.us-east-1.amazonaws.com/keys.json"
# }

# Step 3: Verify it was created
rosactl oidc list
# BUCKET NAME                       ISSUER URL                                                      STATUS   PROVIDER ARN
# oidc-issuer-production-cluster    https://oidc-issuer-production-cluster.s3.us-east-1...         Active   arn:aws:iam::123...

# Step 4: Delete when done
rosactl oidc delete production-cluster --region us-east-1 --function oidc-cleanup
# OIDC issuer deleted successfully
```

## Development

### Build

```bash
make build
# Output: ./bin/rosactl
```

### Run Tests

```bash
# E2E tests (requires AWS_PROFILE)
export AWS_PROFILE=your-profile-name
make test-e2e

# Install test dependencies
make test-deps
```

### Clean

```bash
make clean
```

### Release a New Version

```bash
# Make changes with conventional commits
git commit -m "feat: add new feature"
git commit -m "fix: bug fix"

# Check what next version would be
make release-dry-run

# Create release tag
make release

# Push tag to GitHub
git push origin v0.2.0
```

## Project Structure

```
rosa-regional-platform-cli/
├── cmd/rosactl/              # Entry point
├── internal/
│   ├── commands/             # CLI commands
│   │   ├── lambda/           # Lambda subcommands
│   │   ├── oidc/             # OIDC subcommands
│   │   └── version/          # Version command
│   ├── aws/                  # AWS service clients
│   │   ├── lambda/           # Lambda client and operations
│   │   ├── s3/               # S3 bucket operations
│   │   └── oidc/             # IAM OIDC provider operations
│   ├── crypto/               # RSA key generation
│   └── python/               # Python handler code
├── test/
│   └── e2e/                  # End-to-end tests
│       └── fixtures/         # Test helpers
├── docs/
│   ├── architecture/         # Architecture documentation
│   ├── guides/               # User guides
│   └── specs/                # Feature specifications
├── .semver.yaml              # Semantic versioning config
├── Makefile                  # Build and test targets
├── go.mod
└── README.md
```

## Architecture

For detailed architecture documentation, see [docs/architecture/ARCHITECTURE.md](docs/architecture/ARCHITECTURE.md).

**High-level overview:**

```
┌─────────────────────────────────────────┐
│          rosactl CLI                    │
│       (Cobra Framework)                 │
└────────────┬────────────────────────────┘
             │
   ┌─────────┼─────────┐
   │         │         │
┌──▼───┐ ┌──▼───┐ ┌──▼─────┐
│Lambda│ │OIDC  │ │Version │
│Cmds  │ │Cmds  │ │Cmd     │
└──┬───┘ └──┬───┘ └────────┘
   │         │
┌──▼─────────▼───────────┐
│   AWS Service Clients  │
│  Lambda | S3 | IAM     │
└────────┬────────────────┘
         │
┌────────▼────────────────┐
│    AWS Services         │
│  Lambda, S3, IAM OIDC   │
└─────────────────────────┘
```

## IAM Execution Roles

rosactl automatically creates and manages IAM execution roles:

1. **`rosactl-lambda-execution-role`** - Basic Lambda execution role
   - Policy: `AWSLambdaBasicExecutionRole` (CloudWatch Logs)

2. **`rosactl-lambda-oidc-execution-role`** - OIDC Lambda execution role
   - Policy: `AWSLambdaBasicExecutionRole` (CloudWatch Logs)
   - Inline policy: S3 bucket management (`oidc-issuer-*` buckets)
   - Inline policy: IAM OIDC provider management

**Note:** These roles are NOT deleted when Lambda functions are removed, ensuring they remain available for future function creations.

## Security Considerations

### RSA Private Keys

When creating OIDC Lambdas (`--handler oidc`), the RSA private key is saved to:
```
/tmp/oidc-private-key-{KEY_ID}.pem
```

**Security best practices:**
- File permissions are set to `0600` (owner read/write only)
- Move the key to a secure location (e.g., AWS Secrets Manager) for production use
- Delete from `/tmp` when no longer needed
- **Never commit private keys to version control**

### S3 Bucket Public Access

OIDC issuers require **publicly readable** S3 buckets to serve OIDC discovery documents:
- Bucket policy allows `s3:GetObject` for all principals
- No public write access is granted
- Buckets are prefixed with `oidc-issuer-` for easy identification

## Troubleshooting

### Common Issues

**"AWS_PROFILE environment variable must be set"**
```bash
export AWS_PROFILE=your-profile-name
```

**"Lambda function already exists"**
```bash
# Delete the existing function first
rosactl lambda delete function-name
```

**"Bucket still exists after deletion" (E2E tests)**
- S3's eventual consistency can take up to 5 minutes
- The bucket will be deleted, just wait for propagation

**"go-semver-release not found"**
```bash
go install github.com/s0ders/go-semver-release@latest
```

## Documentation

- [Architecture](docs/architecture/ARCHITECTURE.md) - System architecture and design decisions
- [Versioning Guide](docs/guides/VERSIONING.md) - Semantic versioning with conventional commits
- [Development Guide](docs/guides/DEVELOPMENT.md) - Development setup and guidelines
- [OIDC Feature Spec](docs/specs/feature-oidc.md) - OIDC implementation details
- [E2E Testing Guide](test/e2e/README.md) - Running end-to-end tests

## Contributing

Contributions are welcome! Please follow the conventional commit format for commit messages:

```bash
# Features (minor version bump)
git commit -m "feat: add custom domain support"

# Bug fixes (patch version bump)
git commit -m "fix: handle timeout errors"

# Other changes (patch version bump)
git commit -m "docs: update README"
git commit -m "chore: update dependencies"
git commit -m "refactor: simplify OIDC creation"
```

See [docs/guides/VERSIONING.md](docs/guides/VERSIONING.md) for details.

## License

Apache License 2.0

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) - AWS integration
- [Ginkgo](https://github.com/onsi/ginkgo) - Testing framework
- [go-semver-release](https://github.com/s0ders/go-semver-release) - Semantic versioning
