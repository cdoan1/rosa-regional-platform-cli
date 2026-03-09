# System Architecture

## Overview

`rosactl` is a command-line tool for the ROSA Regional HCP platform.

* The cli manages AWS Lambda functions and S3-backed OIDC (OpenID Connect) identity providers. 
* It simplifies the creation and management of ZIP-based Lambda functions with specialized handlers (default, OIDC issuer, OIDC deletion).

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Interface                            │
│                   (Cobra Command Router)                        │
└────────────┬────────────────────────────┬──────────────────────┘
             │                            │
   ┌─────────▼────────┐        ┌─────────▼────────┐
   │  Lambda Commands │        │  OIDC Commands   │
   │  - create        │        │  - create        │
   │  - delete        │        │  - delete        │
   │  - invoke        │        │  - list          │
   │  - list          │        └─────────┬────────┘
   │  - versions      │                  │
   └─────────┬────────┘                  │
             │                           │
   ┌─────────▼───────────────────────────▼─-───────┐
   │         AWS Service Clients                   │
   │  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
   │  │  Lambda  │  │    S3    │  │   IAM    │     │
   │  └──────────┘  └──────────┘  └──────────┘     │
   └───────────────────────┬───────────────────────┘
                           │
   ┌───────────────────────▼───────────────────────┐
   │            AWS Services                       │
   │  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
   │  │ Lambda   │  │ S3 Bucket│  │ IAM OIDC │     │
   │  │ Functions│  │          │  │ Provider │     │
   │  └──────────┘  └──────────┘  └──────────┘     │
   └───────────────────────────────────────────────┘
```

## Components

### Component 1: CLI Layer
**Responsibility**: Parse user commands, validate arguments, coordinate execution flow

**Interfaces**:
- Input: Command-line arguments from user
- Output: Formatted output to stdout/stderr, exit codes

**Dependencies**:
- `github.com/spf13/cobra` - Command routing and parsing
- Internal command packages

**Key Modules**:
- `internal/commands/root.go` - Root command setup and global flags
- `internal/commands/lambda/` - Lambda command group (create, delete, invoke, list, versions)
- `internal/commands/oidc/` - OIDC command group (create, delete, list)
- `internal/commands/version/` - Version command

---

### Component 2: AWS Service Clients
**Responsibility**: Interact with AWS services via SDK, encapsulate AWS-specific logic

**Interfaces**:
- Input: Context, operation parameters
- Output: AWS resource details, errors

**Dependencies**:
- `github.com/aws/aws-sdk-go-v2` - AWS SDK for Go v2
- AWS credentials (from environment, profile, or IAM role)

**Key Modules**:
- `internal/aws/lambda/client.go` - Lambda service wrapper
- `internal/aws/lambda/create.go` - Lambda creation functions
- `internal/aws/lambda/role.go` - IAM role management
- `internal/aws/s3/client.go` - S3 service wrapper
- `internal/aws/s3/bucket.go` - S3 bucket operations
- `internal/aws/oidc/client.go` - IAM OIDC provider wrapper
- `internal/aws/oidc/list.go` - OIDC provider listing

---

### Component 3: Crypto Layer
**Responsibility**: Generate RSA key pairs for OIDC issuers, convert keys to various formats

**Interfaces**:
- Input: None (generates new keys)
- Output: RSA key pair with JWKS-compatible fields

**Dependencies**:
- `crypto/rsa` - Go standard library RSA
- `crypto/x509` - PEM encoding

**Key Modules**:
- `internal/crypto/rsa.go` - RSA key generation and conversion
  - `GenerateRSAKeyPair()` - Creates 2048-bit RSA key pair
  - `PrivateKeyToPEM()` - Converts private key to PEM format

**Key Design**:
- Generates RSA-2048 keys
- Extracts public key components (N, E) as base64url-encoded strings
- Generates Key ID (KID) as SHA256 hash of modulus
- Saves private key to `/tmp/oidc-private-key-{KID}.pem` with `0600` permissions

---

### Component 4: Python Handler Layer
**Responsibility**: Provide Python Lambda handler code for different function types

**Interfaces**:
- Input: Handler type (default, OIDC, OIDC delete)
- Output: Python source code as string constant

**Dependencies**:
- `internal/python/package.go` - ZIP packaging utilities

**Key Modules**:
- `internal/python/handler.go` - Handler source code constants
  - `DefaultHandler` - Basic hello world handler with timestamp
  - `OIDCHandler` - Creates S3-backed OIDC issuer with IAM provider
  - `OIDCDeleteHandler` - Deletes S3 bucket and IAM OIDC provider

---

## Data Flow

### Flow 1: Create OIDC Lambda Function

1. User runs: `rosactl lambda create my-oidc-issuer --handler oidc`
2. CLI parses command and dispatches to Lambda create handler
3. Lambda client generates RSA key pair via crypto layer
4. **Private key saved to `/tmp/oidc-private-key-{KID}.pem`** (0600 permissions)
5. Public key components (N, E, KID) extracted for Lambda environment variables
6. Lambda client ensures OIDC execution IAM role exists (with S3 + IAM permissions)
7. Python OIDC handler code packaged into ZIP
8. Lambda function created with:
   - Runtime: Python 3.12
   - Environment variables: `JWK_N`, `JWK_E`, `JWK_KID`, `CREATED_AT`
   - Role: `rosactl-lambda-oidc-execution-role`
   - Handler: `lambda_function.lambda_handler`
9. Function published as version 1
10. User sees success message with private key location

```
User → CLI → Lambda Client → Crypto Layer → IAM Role → Package Handler → Create Function → User
                    ↓
              Save Private Key
              to /tmp/*.pem
```

---

### Flow 2: Create OIDC Issuer (via Lambda Invocation)

1. User runs: `rosactl oidc create my-cluster --region us-east-1 --function my-oidc-issuer`
2. CLI auto-prefixes bucket name: `oidc-issuer-my-cluster`
3. CLI constructs payload: `{"bucket_name": "oidc-issuer-my-cluster", "region": "us-east-1"}`
4. Lambda client invokes OIDC Lambda function with payload
5. **Lambda function executes (Python):**
   - Reads RSA public key from environment variables (`JWK_N`, `JWK_E`, `JWK_KID`)
   - Creates S3 bucket with public read policy
   - Generates OIDC discovery document (`.well-known/openid-configuration`)
   - Generates JWKS document (`keys.json`) with public key
   - Uploads both documents to S3
   - Creates IAM OIDC provider pointing to S3 bucket URL
6. Lambda returns success with provider ARN and URLs
7. User sees formatted JSON output

```
User → CLI → Lambda Client → Invoke Function → Python Handler
                                                      ↓
                                    S3 Bucket + OIDC Docs + IAM Provider
```

---

### Flow 3: Using the Private Key

The private key saved in `/tmp/oidc-private-key-{KID}.pem` can be used to:

1. **Sign JWTs** that will be validated against the published public key
2. **Create service account tokens** for Kubernetes/OpenShift
3. **Issue custom claims** for applications

**Example JWT Signing Flow:**
```
Application → Read Private Key → Sign JWT with Claims → JWT Token
                                                            ↓
                                              Token sent to verifier
                                                            ↓
                    Verifier → Fetch JWKS from OIDC Issuer → Validate JWT
```

The JWKS at `https://{bucket}.s3.{region}.amazonaws.com/keys.json` contains the public key matching the private key.

---

## Design Patterns

### Pattern 1: Command Pattern
**Where Used**: CLI command structure (`internal/commands/`)
**Why**: Provides extensible command structure with clear separation of concerns
**Implementation**: Each command group (lambda, oidc) is a separate package with subcommands

### Pattern 2: Client Wrapper Pattern
**Where Used**: AWS service clients (`internal/aws/*/client.go`)
**Why**: Encapsulates AWS SDK complexity, provides consistent error handling
**Implementation**: Each AWS service has a client struct with methods for operations

### Pattern 3: Strategy Pattern
**Where Used**: Lambda function creation (Default vs OIDC handler)
**Why**: Allows different creation strategies based on `--handler` flag
**Implementation**: Switch statement dispatches to different creation methods

---

## External Integrations

### Integration 1: AWS Lambda
**Purpose**: Deploy and execute serverless functions
**Protocol**: AWS SDK (HTTPS/REST)
**Authentication**: AWS credentials (profile, environment, IAM role)
**Operations**:
- `CreateFunction` - Deploy Lambda function
- `InvokeFunction` - Execute Lambda function
- `DeleteFunction` - Remove Lambda function
- `ListFunctions` - List all functions
- `PublishVersion` - Create immutable version

### Integration 2: AWS S3
**Purpose**: Host OIDC discovery documents for S3-backed issuers
**Protocol**: AWS SDK (HTTPS/REST)
**Authentication**: AWS credentials
**Operations**:
- `CreateBucket` - Create S3 bucket
- `PutObject` - Upload OIDC documents
- `PutBucketPolicy` - Enable public read access
- `PutPublicAccessBlock` - Configure public access
- `DeleteBucket` - Remove bucket

### Integration 3: AWS IAM
**Purpose**: Create OIDC identity providers and manage Lambda execution roles
**Protocol**: AWS SDK (HTTPS/REST)
**Authentication**: AWS credentials
**Operations**:
- `CreateRole` - Create Lambda execution role
- `AttachRolePolicy` - Attach managed policies
- `PutRolePolicy` - Attach inline policies
- `CreateOpenIDConnectProvider` - Register OIDC issuer
- `DeleteOpenIDConnectProvider` - Remove OIDC provider

---

## Security Architecture

### RSA Private Key Management

**Key Generation:**
- RSA-2048 keys generated using Go's `crypto/rsa`
- Cryptographically secure random number generator
- Keys generated on-demand during OIDC Lambda creation

**Key Storage:**
- Private key saved to: `/tmp/oidc-private-key-{KID}.pem`
- File permissions: `0600` (owner read/write only)
- Format: PEM-encoded RSA PRIVATE KEY
- **User responsibility**: Move to secure location or AWS Secrets Manager for production

**Key Usage:**
- Public key (N, E, KID) stored in Lambda environment variables
- Public key published in OIDC issuer's JWKS endpoint
- Private key used offline to sign JWTs

**Security Warnings:**
- User warned via CLI output about private key location
- User advised to keep file secure
- Temp directory not encrypted by default (OS-dependent)

### AWS IAM Permissions

**OIDC Lambda Execution Role:**
- S3 permissions scoped to `oidc-issuer-*` buckets only
- IAM permissions for OIDC provider management only
- CloudWatch Logs for observability

**User Permissions Required:**
- Lambda: CreateFunction, DeleteFunction, InvokeFunction, etc.
- IAM: CreateRole, AttachRolePolicy, CreateOpenIDConnectProvider
- S3: CreateBucket, PutObject, PutBucketPolicy (for OIDC issuers)

---

## Trade-offs & Constraints

### Private Key Storage Trade-off

**Decision**: Save private key to `/tmp` instead of AWS Secrets Manager

**Rationale**:
- Simplicity: No additional AWS service dependencies
- User control: User decides where to store key long-term
- Transparency: User sees exactly what's happening

**Trade-offs**:
- ✅ Simple implementation, no secrets management complexity
- ✅ User has full control over key lifecycle
- ⚠️ Key stored in plaintext on disk (OS encryption may apply)
- ⚠️ Temp directory may be cleared on reboot
- ⚠️ User must manually move key to secure location

**Future Enhancement**: Add `--save-key-to` flag to specify custom location or Secrets Manager ARN

### S3-Backed OIDC Pattern

**Decision**: Use S3 buckets to host OIDC discovery documents instead of API Gateway or CloudFront

**Rationale**:
- Cost-effective: S3 storage + bandwidth cheaper than API Gateway
- Simple: No Lambda@Edge or API Gateway setup required
- Proven pattern: Used by ROSA and other AWS-based Kubernetes distributions

**Trade-offs**:
- ✅ Low cost, high availability
- ✅ Simple architecture
- ⚠️ S3 URLs not user-friendly (long bucket URLs)
- ⚠️ No custom domain support (would require CloudFront)
- ⚠️ S3 eventual consistency can delay deletions

### Lambda-Based OIDC Management

**Decision**: Use Lambda functions to create/delete OIDC issuers instead of direct SDK calls from CLI

**Rationale**:
- Consistency: Lambda has same permissions model as other AWS services
- Reusability: Lambda can be invoked from other tools (Terraform, CloudFormation)
- Auditability: CloudWatch logs capture all OIDC operations

**Trade-offs**:
- ✅ Lambda invocations logged in CloudWatch
- ✅ Can be used in automation pipelines
- ✅ Permissions scoped to Lambda execution role
- ⚠️ Additional step (create Lambda before creating issuers)
- ⚠️ Lambda cold starts add latency

---

## Monitoring & Observability

**Logging:**
- CLI: stdout/stderr for user-facing messages
- Lambda: CloudWatch Logs for function execution
- AWS SDK: Debug logging via `AWS_SDK_DEBUG` environment variable

**Metrics:**
- Lambda invocation count, duration, errors (CloudWatch Metrics)
- S3 bucket operations (S3 metrics)

**Tracing:**
- Not currently implemented
- Future: AWS X-Ray integration for Lambda tracing

---

## Scalability Considerations

**CLI Scalability:**
- Stateless CLI tool, scales horizontally (multiple users)
- No persistent connections or state

**Lambda Scalability:**
- AWS Lambda auto-scales based on invocation rate
- OIDC Lambda: Low invocation frequency (create/delete operations)

**S3 Bucket Limits:**
- S3 bucket names globally unique (collision risk with common names)
- Auto-prefixing with `oidc-issuer-` reduces collisions
- No practical limit on number of buckets per account

---

## Future Enhancements

1. **AWS Secrets Manager Integration**: Store private keys in Secrets Manager instead of temp directory
2. **Custom Domains**: Support CloudFront + custom domains for OIDC issuers
3. **Key Rotation**: Automate RSA key rotation with multiple keys in JWKS
4. **JWT Signing Command**: Add `rosactl jwt sign` command using saved private keys
5. **Backup/Restore**: Export/import OIDC issuer configurations
6. **Multi-Region**: Replicate OIDC issuers across regions for high availability
