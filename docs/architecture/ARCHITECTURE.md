# System Architecture

## Overview

`rosactl` is a command-line tool for managing AWS infrastructure for ROSA Regional HCP (Hosted Control Plane) clusters. It uses a **container-based Lambda + CloudFormation** approach to deploy cluster IAM resources in customer AWS accounts.

## Architecture Principles

1. **Managed OIDC**: Uses Red Hat's CloudFront-backed OIDC issuer (not customer-hosted)
2. **Declarative Infrastructure**: All IAM resources defined in CloudFormation templates
3. **Container-Based Lambda**: Single Go binary runs in both CLI and Lambda modes
4. **Transparency**: CloudFormation templates are auditable files in the repository
5. **No Private Keys**: HyperShift operator manages RSA keys in Management Cluster

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Customer Environment                        │
│                                                                 │
│  ┌──────────────┐                                               │
│  │   Developer  │                                               │
│  │   Machine    │                                               │
│  └──────┬───────┘                                               │
│         │                                                       │
│         │ 1. rosactl bootstrap create                          │
│         │    --image-uri <ECR_URI>                              │
│         ▼                                                       │
│  ┌─────────────────┐                                            │
│  │  CloudFormation │                                            │
│  │  Stack (Lambda) │                                            │
│  └────────┬────────┘                                            │
│           │ Creates                                             │
│           ▼                                                     │
│  ┌──────────────────────┐                                       │
│  │  Lambda Function     │                                       │
│  │  (Container-based)   │                                       │
│  │  - Go Binary         │                                       │
│  │  - CF Templates      │                                       │
│  └──────────┬───────────┘                                       │
│             │                                                   │
│             │ 2. rosactl cluster-iam create                     │
│             │    Invokes Lambda                                 │
│             ▼                                                   │
│  ┌──────────────────────┐                                       │
│  │  CloudFormation      │                                       │
│  │  Stack (Cluster IAM) │                                       │
│  └──────────┬───────────┘                                       │
│             │ Creates                                           │
│             ▼                                                   │
│  ┌─────────────────────────────────────────┐                    │
│  │  Cluster IAM Resources                  │                    │
│  │  - IAM OIDC Provider                    │                    │
│  │  - 7 Control Plane Roles                │                    │
│  │  - Worker Node Role + Instance Profile  │                    │
│  └─────────────────────────────────────────┘                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│               Red Hat Management Cluster                        │
│                                                                 │
│  ┌──────────────────────────────────────┐                       │
│  │  OIDC Issuer Infrastructure          │                       │
│  │  - CloudFront Distribution           │                       │
│  │  - Private S3 Bucket + OAC           │                       │
│  │  - HyperShift-managed RSA Keys       │                       │
│  └──────────────────────────────────────┘                       │
│                                                                 │
│  Customer's IAM OIDC Provider points here ──────────────────────┤
│  (e.g., https://d1234.cloudfront.net/cluster-name)             │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### 1. CLI Layer (`cmd/rosactl`)

**Responsibility**: Parse user commands, manage AWS resources via SDK

**Key Commands**:
- `rosactl bootstrap create` - Deploy Lambda function via CloudFormation
- `rosactl bootstrap delete` - Remove Lambda infrastructure
- `rosactl bootstrap status` - Show Lambda stack status
- `rosactl cluster-iam create` - Invoke Lambda to create IAM resources
- `rosactl cluster-iam delete` - Invoke Lambda to delete IAM resources
- `rosactl cluster-iam list` - List cluster IAM stacks
- `rosactl cluster-iam describe` - Show cluster IAM details
- `rosactl version` - Show CLI version

**Implementation**:
- `internal/commands/bootstrap/` - Bootstrap command group
- `internal/commands/clusteriam/` - Cluster IAM command group
- `internal/commands/version/` - Version command

---

### 2. Lambda Handler (`internal/lambda/handler.go`)

**Responsibility**: Execute in Lambda to apply CloudFormation templates

**Dual-Mode Binary**:
The same Go binary runs in two modes:
- **CLI Mode**: When `AWS_LAMBDA_RUNTIME_API` is not set
- **Lambda Mode**: When `AWS_LAMBDA_RUNTIME_API` is set

**Lambda Actions**:
- `apply-cluster-iam` - Creates CloudFormation stack with IAM resources
- `delete-cluster-iam` - Deletes CloudFormation stack

**Implementation**:
```go
func Handler(ctx context.Context, event Event) (Response, error) {
    switch event.Action {
    case "apply-cluster-iam":
        return applyClusterIAM(ctx, event)
    case "delete-cluster-iam":
        return deleteClusterIAM(ctx, event)
    }
}
```

---

### 3. AWS Service Clients

**CloudFormation Client** (`internal/aws/cloudformation/`):
- CreateStack, UpdateStack, DeleteStack
- DescribeStack, ListStacks, GetStackEvents
- Waiters for stack completion

**Lambda Client** (`internal/aws/lambda/`):
- InvokeFunction with JSON payload
- Used by cluster-iam commands to trigger Lambda

**IAM Client** (via CloudFormation):
- All IAM operations done through CloudFormation
- No direct IAM SDK calls in CLI

---

### 4. Crypto Layer (`internal/crypto/`)

**TLS Thumbprint Fetching**:
- Connects to OIDC issuer URL via HTTPS
- Extracts root CA certificate
- Calculates SHA-1 fingerprint
- Mimics Terraform's `data.tls_certificate` behavior

**Implementation**:
```go
func GetOIDCThumbprint(ctx context.Context, issuerURL string) (string, error) {
    // Connect to issuer, get TLS cert, return SHA-1 hash
}
```

---

### 5. CloudFormation Templates

**Bootstrap Template** (`templates/lambda-bootstrap.yaml`):
- Lambda execution IAM role with CloudFormation + IAM permissions
- Lambda function using container image from ECR
- Parameters: ContainerImageURI, FunctionName

**Cluster IAM Template** (`templates/cluster-iam.yaml`):
- IAM OIDC Provider pointing to Red Hat's CloudFront URL
- 7 control plane IAM roles with OIDC trust policies:
  - `ingress` - ROSAIngressOperatorPolicy
  - `cloud-controller-manager` - ROSAKubeControllerPolicy
  - `ebs-csi` - ROSAAmazonEBSCSIDriverOperatorPolicy
  - `image-registry` - ROSAImageRegistryOperatorPolicy
  - `network-config` - ROSACloudNetworkConfigOperatorPolicy
  - `control-plane-operator` - ROSAControlPlaneOperatorPolicy + supplemental policies
  - `node-pool-management` - ROSANodePoolManagementPolicy + supplemental policies
- Worker node IAM role + instance profile with EC2 trust

Parameters: ClusterName, OIDCIssuerURL, OIDCThumbprint

---

### 6. Container Image

**Dockerfile** (UBI9-based multi-stage build):
- Build stage: golang:1.24 compiles Go binary
- Runtime stage: ubi9-minimal with Go binary + CloudFormation templates
- Templates copied to `/app/templates/` for Lambda access

**Build and Push**:
```bash
docker build -f Dockerfile -t rosa-cli:latest .
docker tag rosa-cli:latest <account>.dkr.ecr.us-east-1.amazonaws.com/rosa-cli:latest
docker push <account>.dkr.ecr.us-east-1.amazonaws.com/rosa-cli:latest
```

---

## Data Flow

### Flow 1: Bootstrap Lambda (One-Time Setup)

```
1. User runs: rosactl bootstrap create --image-uri <ECR_URI> --region us-east-1
   ↓
2. CLI reads templates/lambda-bootstrap.yaml
   ↓
3. CLI creates CloudFormation stack with parameters:
   - ContainerImageURI: <ECR_URI>
   - FunctionName: rosa-regional-platform-lambda
   ↓
4. CloudFormation creates:
   - Lambda execution IAM role (with CF + IAM permissions)
   - Lambda function (container image from ECR)
   ↓
5. Stack outputs:
   - LambdaFunctionArn
   - LambdaFunctionName
   - ExecutionRoleArn
```

---

### Flow 2: Create Cluster IAM (Per Cluster)

```
1. User runs: rosactl cluster-iam create my-cluster \
              --oidc-issuer-url https://d1234.cloudfront.net/my-cluster \
              --region us-east-1
   ↓
2. CLI validates inputs (cluster name, OIDC URL format)
   ↓
3. CLI fetches TLS thumbprint from OIDC issuer URL
   ↓
4. CLI invokes Lambda with payload:
   {
     "action": "apply-cluster-iam",
     "cluster_name": "my-cluster",
     "oidc_issuer_url": "https://d1234.cloudfront.net/my-cluster",
     "oidc_thumbprint": "a1b2c3d4..."
   }
   ↓
5. Lambda handler receives event:
   - Reads /app/templates/cluster-iam.yaml
   - Applies CloudFormation stack: rosa-my-cluster-iam
   - Waits for CREATE_COMPLETE
   ↓
6. CloudFormation creates:
   - IAM OIDC Provider
   - 7 control plane IAM roles
   - Worker node IAM role + instance profile
   ↓
7. Lambda returns outputs to CLI:
   - StackID
   - Outputs: All role ARNs, OIDC provider ARN, instance profile name
   ↓
8. CLI displays formatted output to user
```

---

### Flow 3: Delete Cluster IAM

```
1. User runs: rosactl cluster-iam delete my-cluster --region us-east-1
   ↓
2. CLI invokes Lambda with payload:
   {
     "action": "delete-cluster-iam",
     "cluster_name": "my-cluster"
   }
   ↓
3. Lambda deletes CloudFormation stack: rosa-my-cluster-iam
   ↓
4. CloudFormation deletes all IAM resources:
   - Worker instance profile
   - Worker IAM role
   - 7 control plane IAM roles
   - IAM OIDC Provider
   ↓
5. Lambda returns success to CLI
```

---

## Design Patterns

### Pattern 1: Dual-Mode Binary
**Implementation**: Single Go binary detects execution environment
```go
if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
    lambdaHandler.Start()  // Lambda mode
} else {
    commands.Execute()     // CLI mode
}
```

**Benefits**:
- Single binary to build and distribute
- No code duplication between CLI and Lambda
- Same CloudFormation logic in both contexts

---

### Pattern 2: Template-Based Infrastructure
**Implementation**: CloudFormation templates as files, not inline code

**Benefits**:
- No 4096 character limits (inline Python Lambda limitation)
- Fully auditable (Git-tracked YAML files)
- Security review before deployment
- Reusable across tools (Terraform, CDK, manual deployment)

---

### Pattern 3: Lambda as CloudFormation Executor
**Implementation**: Lambda simply applies CloudFormation templates

**Benefits**:
- Lambda invocations logged in CloudWatch
- CloudFormation provides rollback on failure
- CloudFormation drift detection available
- Permissions scoped to Lambda execution role

---

## Security Architecture

### IAM Permissions Separation

**CLI User Permissions** (customer developer):
- CloudFormation: CreateStack, DescribeStacks, DeleteStack
- Lambda: InvokeFunction (for cluster-iam commands)
- ECR: GetAuthorizationToken, BatchGetImage (for pushing container images)

**Lambda Execution Role Permissions**:
- CloudFormation: CreateStack, UpdateStack, DeleteStack, DescribeStacks
- IAM: CreateOpenIDConnectProvider, CreateRole, AttachRolePolicy, CreateInstanceProfile, etc.
- Scoped to `rosa-*` CloudFormation stacks

**No Long-Lived Credentials**:
- OIDC uses federated trust (no AWS credentials in workloads)
- Lambda uses execution role (no credentials in environment variables)

---

### OIDC Trust Chain

```
1. HyperShift Operator (Management Cluster)
   ↓ Generates RSA key pair
   ↓ Stores in Kubernetes Secret
   ↓
2. Public Key Published
   ↓ CloudFront: https://d1234.cloudfront.net/my-cluster/keys.json
   ↓ S3 (private with OAC): oidc-issuer-bucket/my-cluster/
   ↓
3. Customer IAM OIDC Provider
   ↓ Issuer URL: https://d1234.cloudfront.net/my-cluster
   ↓ Thumbprint: SHA-1 of CloudFront TLS cert
   ↓
4. Control Plane Pods (Management Cluster)
   ↓ ServiceAccount token signed by HyperShift
   ↓ AssumeRoleWithWebIdentity to customer IAM roles
   ↓
5. Customer IAM Roles
   ↓ Trust policy: OIDC provider + specific ServiceAccount
   ↓ Temporary credentials granted to control plane pods
```

---

## Monitoring & Observability

**CloudWatch Logs**:
- Lambda execution logs: `/aws/lambda/rosa-regional-platform-lambda`
- CloudFormation stack events visible in AWS Console

**CloudWatch Metrics**:
- Lambda invocation count, duration, errors
- CloudFormation stack status (via CloudWatch Events)

**CLI Logging**:
- stdout/stderr for user-facing messages
- `--verbose` flag for debug output (future enhancement)

---

## Trade-offs & Constraints

### Container-Based Lambda vs Inline Python

**Decision**: Use container-based Lambda with Go binary

**Rationale**:
- CloudFormation templates can be full files (no 4096 char limit)
- Single codebase for CLI and Lambda
- UBI9 base image for security scanning
- Faster cold starts than Python with large dependencies

**Trade-offs**:
- ✅ Unlimited template size
- ✅ Full auditability (templates in Git)
- ✅ Single binary to maintain
- ⚠️ Requires ECR push step (container image must be available)
- ⚠️ Larger Lambda package (container image vs ZIP)

---

### CloudFormation for All Resources

**Decision**: All IAM resources defined in CloudFormation, not direct SDK calls

**Rationale**:
- Declarative infrastructure (GitOps-friendly)
- Automatic rollback on failure
- Drift detection available
- Change sets for previewing updates

**Trade-offs**:
- ✅ Full rollback support
- ✅ Change sets for safety
- ✅ Templates auditable before deployment
- ⚠️ CloudFormation quotas (200 resources per stack - not an issue for us)
- ⚠️ Slower than direct SDK calls (stack creation ~30-60s)

---

### Managed OIDC (Red Hat-Hosted) Only

**Decision**: No support for customer-hosted OIDC issuers

**Rationale**:
- Aligns with ROSA HCP service architecture
- Simpler key management (HyperShift handles it)
- No RSA private keys in customer accounts
- Eliminates S3 bucket creation in customer accounts

**Trade-offs**:
- ✅ No key management complexity for customers
- ✅ No S3 buckets to manage
- ✅ Consistent with ROSA service model
- ⚠️ Requires Red Hat infrastructure to be available
- ⚠️ No offline/air-gapped support

---

## Future Enhancements

1. **VPC Management**: Add CloudFormation templates for VPC creation (from rosa-regional-platform Terraform)
2. **Multi-Region**: Support replicating IAM resources across regions
3. **Dry-Run Mode**: Preview CloudFormation changes before applying
4. **CloudFormation Change Sets**: Use change sets for safer updates
5. **CloudWatch Alarms**: Add alarms for Lambda failures
6. **Rollback Support**: Add `cluster-iam rollback` command
7. **Template Validation**: Validate templates before deployment

---

## References

- [ROSA Regional Platform Terraform](https://github.com/openshift-online/rosa-regional-platform)
- [HyperShift OIDC Implementation](https://github.com/openshift/hypershift)
- [AWS CloudFormation Best Practices](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/best-practices.html)
- [AWS Lambda Container Images](https://docs.aws.amazon.com/lambda/latest/dg/images-create.html)
