# OIDC Feature Specification

## Overview

This feature implements S3-backed OIDC (OpenID Connect) issuer management for `rosactl`. The implementation creates complete OIDC issuers using S3 buckets to serve OIDC discovery documents, following the ROSA/OpenShift pattern for Kubernetes service account authentication and workload identity federation.

## Requirements

* Introduce a new Lambda function `oidc`
* Support `rosactl lambda create oidc` to create the OIDC Lambda function
* Lambda function implemented in Python
* Support `rosactl oidc create <bucket-name>` to create S3-backed OIDC issuers
* When the OIDC Lambda is created (`rosactl lambda create oidc`):
  - Generate RSA key pair for JWT signature verification (in Go CLI)
  - Embed the public key components in Lambda environment variables
  - All OIDC issuers created by this Lambda will use the same RSA key pair
* When an OIDC issuer is created (`rosactl oidc create <bucket-name>`):
  - Create an S3 bucket to host OIDC discovery documents
  - Upload `.well-known/openid-configuration` and `keys.json` (JWKS)
  - Create an IAM OIDC provider pointing to the S3 bucket URL
* Support `rosactl oidc list` to list all OIDC issuers in the current AWS account

## Architecture

### S3-Backed OIDC Pattern

```
S3 Bucket: s3://oidc-issuer-{unique-id}/
├── .well-known/openid-configuration  (OIDC metadata)
└── keys.json                         (JWKS - public keys)

IAM OIDC Provider
└── Issuer URL: https://{bucket}.s3.{region}.amazonaws.com
```

### Components

1. **Go CLI (`rosactl`)**
   - Generates RSA key pairs locally using Go `crypto/rsa`
   - Manages OIDC issuer lifecycle via `oidc` command group
   - No external dependencies required

2. **Python Lambda (`oidc` function)**
   - Creates S3 buckets with public read access
   - Uploads OIDC discovery documents
   - Registers IAM OIDC providers
   - **No cryptography dependencies** (keys provided by CLI)

3. **S3 Service Layer** (`internal/aws/s3/`)
   - Bucket operations: create, upload, list, delete
   - Public access configuration for OIDC discovery

4. **OIDC Service Layer** (`internal/aws/oidc/`)
   - IAM OIDC provider operations
   - Lists and correlates providers with S3 buckets

5. **Crypto Package** (`internal/crypto/`)
   - RSA key pair generation
   - JWKS format conversion
   - PEM export for private keys

## Architecture Flow

### One-Time Lambda Setup

```
User runs: rosactl lambda create oidc
    ↓
1. Go generates RSA key pair locally (crypto/rsa)
    ↓
2. Extracts public key components (n, e) in base64url format
    ↓
3. Generates key ID (kid) from SHA256 hash of modulus
    ↓
4. Creates Lambda function with environment variables:
   {
     "JWK_N": "base64url-encoded-modulus",
     "JWK_E": "base64url-encoded-exponent",
     "JWK_KID": "key-id"
   }
    ↓
5. Lambda function is ready to create multiple OIDC issuers
   All issuers will use the same RSA public key
```

### Creating an OIDC Issuer (Can be done multiple times)

```
User runs: rosactl oidc create my-cluster
    ↓
1. Auto-prefixes bucket name to "oidc-issuer-my-cluster"
    ↓
2. Constructs JSON payload:
   {
     "bucket_name": "oidc-issuer-my-cluster",
     "region": "us-east-1"
   }
    ↓
3. Invokes OIDC Lambda function with payload
    ↓
4. Lambda reads RSA key from environment variables
    ↓
5. Lambda creates S3 bucket with public read policy
    ↓
6. Lambda generates OIDC discovery document:
   {
     "issuer": "https://bucket.s3.region.amazonaws.com",
     "jwks_uri": "https://bucket.s3.region.amazonaws.com/keys.json",
     "response_types_supported": ["id_token"],
     "subject_types_supported": ["public"],
     "id_token_signing_alg_values_supported": ["RS256"]
   }
    ↓
7. Lambda generates JWKS document from embedded public key:
   {
     "keys": [{
       "kty": "RSA",
       "use": "sig",
       "kid": "embedded-key-id",
       "n": "embedded-modulus",
       "e": "embedded-exponent",
       "alg": "RS256"
     }]
   }
    ↓
8. Lambda uploads both documents to S3
    ↓
9. Lambda creates IAM OIDC provider with issuer URL
    ↓
10. Returns issuer URL, provider ARN, and discovery URLs to user
```

## Commands

### Create OIDC Lambda Function (One-time setup)

```bash
rosactl lambda create oidc
```

Creates a Python Lambda function with:
- **Runtime**: Python 3.12
- **Memory**: 256 MB (for S3/IAM operations)
- **Timeout**: 60 seconds
- **Role**: `rosactl-lambda-oidc-execution-role` with permissions for:
  - S3 bucket operations (`oidc-issuer-*` prefix)
  - IAM OIDC provider management
  - CloudWatch Logs
- **Environment Variables**:
  - `JWK_N`: RSA modulus (base64url-encoded)
  - `JWK_E`: RSA exponent (base64url-encoded)
  - `JWK_KID`: Key ID

**Output:**
```
Generating RSA key pair for OIDC Lambda...
Generated RSA key pair (kid: a1b2c3d4e5f6g7h8)
Creating OIDC issuer Lambda function...
Successfully created Lambda function: oidc
ARN: arn:aws:lambda:us-east-1:123456789012:function:oidc (version: 1)

To create an OIDC issuer, use:
  rosactl oidc create <bucket-name>

Example:
  rosactl oidc create oidc-issuer-my-cluster
```

### Create S3-Backed OIDC Issuer

```bash
# Basic usage (auto-prefixed to oidc-issuer-my-cluster)
rosactl oidc create my-cluster

# With specific region (auto-prefixed to oidc-issuer-hcp-prod)
rosactl oidc create hcp-prod --region us-west-2

# Use a different Lambda function
rosactl oidc create my-cluster --function my-oidc-lambda
```

**Note:** Bucket names are automatically prefixed with `oidc-issuer-` to match the IAM policy scope. For example, `my-cluster` becomes `oidc-issuer-my-cluster`.

**Output:**
```json
{
  "bucket_name": "oidc-issuer-my-cluster",
  "issuer_url": "https://oidc-issuer-my-cluster.s3.us-east-1.amazonaws.com",
  "provider_arn": "arn:aws:iam::123456789012:oidc-provider/oidc-issuer-my-cluster.s3.us-east-1.amazonaws.com",
  "discovery_url": "https://oidc-issuer-my-cluster.s3.us-east-1.amazonaws.com/.well-known/openid-configuration",
  "jwks_url": "https://oidc-issuer-my-cluster.s3.us-east-1.amazonaws.com/keys.json",
  "message": "OIDC issuer created successfully"
}
```

### List OIDC Issuers

```bash
# Table format (default)
rosactl oidc list

# JSON format
rosactl oidc list --output json
```

**Table Output:**
```
BUCKET NAME              ISSUER URL                                           STATUS  PROVIDER ARN
oidc-issuer-my-cluster   https://oidc-issuer-my-cluster.s3.us-east-1...     Active  arn:aws:iam::123...
```

**Status Values:**
- `Active` - Both S3 bucket and IAM provider exist
- `S3 Only` - Bucket exists but no IAM provider
- `IAM Only` - IAM provider exists but bucket not found

## Implementation Details

### File Structure

```
internal/
├── aws/
│   ├── s3/
│   │   ├── client.go      # S3 client wrapper
│   │   ├── bucket.go      # Bucket operations
│   │   └── errors.go      # S3 error types
│   ├── oidc/
│   │   ├── client.go      # IAM OIDC client wrapper
│   │   ├── list.go        # List/delete OIDC providers
│   │   └── errors.go      # OIDC error types
│   └── lambda/
│       ├── create_oidc.go # CreateOIDCFunction method
│       └── role.go        # OIDC execution role
├── crypto/
│   └── rsa.go             # RSA key generation
├── commands/
│   └── oidc/
│       ├── oidc.go        # Command group
│       ├── create.go      # Create issuer command
│       └── list.go        # List issuers command
└── python/
    └── handler.go         # Python Lambda code
```

### Lambda Execution Role Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:CreateBucket",
        "s3:PutObject",
        "s3:PutBucketPolicy",
        "s3:PutPublicAccessBlock",
        "s3:DeleteObject",
        "s3:DeleteBucket",
        "s3:GetBucketLocation"
      ],
      "Resource": [
        "arn:aws:s3:::oidc-issuer-*",
        "arn:aws:s3:::oidc-issuer-*/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "iam:CreateOpenIDConnectProvider",
        "iam:GetOpenIDConnectProvider",
        "iam:ListOpenIDConnectProviders",
        "iam:DeleteOpenIDConnectProvider"
      ],
      "Resource": "*"
    }
  ]
}
```

### RSA Key Generation

Keys are generated in Go using the standard library:

```go
// Generate 2048-bit RSA key pair
privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

// Extract public key components
publicKey := &privateKey.PublicKey
n := publicKey.N  // Modulus
e := publicKey.E  // Exponent

// Convert to base64url for JWKS
nBase64 := base64.RawURLEncoding.EncodeToString(n.Bytes())
eBase64 := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(e)).Bytes())

// Generate key ID
kid := sha256(n.Bytes())[:16]
```

### Security Considerations

1. **S3 Bucket Access**
   - Buckets are publicly readable (required for OIDC discovery)
   - Only `s3:GetObject` allowed (read-only)
   - No public write access
   - Server-side encryption can be enabled if needed

2. **RSA Key Security**
   - Keys generated using `crypto/rand` (cryptographically secure)
   - Private keys can be optionally saved locally
   - Only public keys uploaded to S3 (in JWKS format)
   - Keys are NOT used for actual JWT signing (workload identity provider handles that)

3. **IAM Permissions**
   - OIDC Lambda role scoped to `oidc-issuer-*` buckets only
   - No broader S3 or IAM permissions

## Dependencies

### Go Packages

- `github.com/aws/aws-sdk-go-v2/service/s3` - S3 operations
- `github.com/aws/aws-sdk-go-v2/service/iam` - IAM OIDC provider operations
- `crypto/rsa` - RSA key generation (standard library)
- `crypto/x509` - PEM encoding (standard library)

### Python Packages

- `boto3` - AWS SDK (included in Lambda runtime)
- **No external dependencies** - `cryptography` package eliminated

## Benefits of This Approach

1. **No Lambda Layers** - Eliminates need for `cryptography` Lambda layer
2. **Smaller Lambda Package** - No heavy cryptographic libraries (~50KB vs ~10MB+)
3. **Faster Cold Starts** - Less code to initialize
4. **Better Security** - Keys generated locally, embedded once
5. **Simple UX** - Just `rosactl oidc create <bucket>` after initial setup
6. **Consistent Keys** - All issuers from same Lambda share keys (simpler key management)
7. **No Key Transport** - Keys never transmitted over network after Lambda creation

## Future Enhancements

1. Add `rosactl oidc delete <bucket-name>` command
2. Support key rotation for JWKS
3. Add OIDC issuer validation command
4. Support custom domains via CloudFront
5. Multi-region bucket replication
6. CloudWatch monitoring/alarms
7. Integration with ROSA cluster creation workflow
