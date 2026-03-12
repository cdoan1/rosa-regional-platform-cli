# System Architecture

## Overview

`rosactl` is a command-line tool for managing AWS infrastructure for ROSA Regional HCP (Hosted Control Plane) clusters. It provides **direct CloudFormation management** for VPC networking and IAM resources, with optional Lambda support for event-driven workflows.

## Architecture Principles

1. **Direct Execution**: CLI commands directly create CloudFormation stacks (no Lambda required)
2. **Embedded Templates**: CloudFormation templates embedded in binary using go:embed
3. **Declarative Infrastructure**: All resources defined in CloudFormation templates
4. **Optional Lambda**: Lambda available for automation but not required for basic operations
5. **Dual-Mode Architecture**: Same binary can run as CLI or Lambda function
6. **Managed OIDC**: Uses Red Hat's CloudFront-backed OIDC issuer (not customer-hosted)
7. **Transparency**: CloudFormation templates are auditable files in the repository

## High-Level Architecture

### Primary Mode: Direct CloudFormation

```
┌─────────────────────────────────────────────────────────────────┐
│                     Customer Environment                        │
│                                                                 │
│  ┌──────────────┐                                               │
│  │   Developer  │                                               │
│  │   Machine    │                                               │
│  └──────┬───────┘                                               │
│         │                                                       │
│         │ 1. rosactl cluster-vpc create my-cluster              │
│         │    (CLI with embedded CloudFormation templates)       │
│         ▼                                                       │
│  ┌──────────────────────┐                                       │
│  │  CloudFormation      │                                       │
│  │  Stack (VPC)         │                                       │
│  │  rosa-my-cluster-vpc │                                       │
│  └──────────┬───────────┘                                       │
│             │ Creates                                           │
│             ▼                                                   │
│  ┌─────────────────────────────────────────┐                    │
│  │  VPC Resources                          │                    │
│  │  - VPC (10.0.0.0/16)                    │                    │
│  │  - 3 Public + 3 Private Subnets         │                    │
│  │  - Internet Gateway, NAT Gateway(s)     │                    │
│  │  - Route Tables, Security Groups        │                    │
│  │  - Route53 Private Hosted Zone          │                    │
│  └─────────────────────────────────────────┘                    │
│                                                                 │
│         │                                                       │
│         │ 2. rosactl cluster-iam create my-cluster              │
│         │    --oidc-issuer-url https://d1234.cloudfront...      │
│         ▼                                                       │
│  ┌──────────────────────┐                                       │
│  │  CloudFormation      │                                       │
│  │  Stack (IAM)         │                                       │
│  │  rosa-my-cluster-iam │                                       │
│  └──────────┬───────────┘                                       │
│             │ Creates                                           │
│             ▼                                                   │
│  ┌─────────────────────────────────────────┐                    │
│  │  IAM Resources                          │                    │
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

### Optional Mode: Lambda for Event-Driven Workflows

Lambda bootstrap is **optional** and used for CI/CD integration or event-driven automation. The same rosactl binary runs as a Lambda function.

```
┌──────────────┐
│   AWS Event  │
│   Source     │
└──────┬───────┘
       │
       ▼
┌──────────────────────┐
│  Lambda Function     │
│  (Container: rosactl)│
│  - Embedded Templates│
│  - Same Binary       │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  CloudFormation      │
│  Stacks (VPC + IAM)  │
└──────────────────────┘
```

## Components

### 1. CLI Layer (`cmd/rosactl`)

**Responsibility**: Parse user commands, directly manage CloudFormation stacks

**Key Commands**:

**Cluster VPC Management**:
- `rosactl cluster-vpc create CLUSTER_NAME` - Create VPC networking via CloudFormation
- `rosactl cluster-vpc delete CLUSTER_NAME` - Delete VPC stack
- `rosactl cluster-vpc list` - List all VPC stacks

**Cluster IAM Management**:
- `rosactl cluster-iam create CLUSTER_NAME --oidc-issuer-url URL` - Create IAM resources
- `rosactl cluster-iam delete CLUSTER_NAME` - Delete IAM stack
- `rosactl cluster-iam list` - List all IAM stacks

**Optional Lambda Bootstrap** (for event-driven workflows):
- `rosactl lambda create` - Deploy Lambda container (optional)
- `rosactl lambda delete` - Remove Lambda function (optional)

**Other**:
- `rosactl version` - Show CLI version

**Implementation**:
- `internal/commands/clustervpc/` - VPC management command group
- `internal/commands/clusteriam/` - IAM management command group
- `internal/commands/lambda/` - Lambda bootstrap (optional)
- `internal/commands/version/` - Version command

---

### 2. CloudFormation Client (`internal/aws/cloudformation/`)

**Responsibility**: Direct CloudFormation stack management

**Operations**:
- `CreateStack()` - Create CloudFormation stack with parameters, tags, and capabilities
- `UpdateStack()` - Update existing stack with new parameters or template
- `DeleteStack()` - Delete stack and wait for completion
- `DescribeStacks()` - Get stack status and outputs
- `ListStacks()` - List all stacks in region
- `GetStackEvents()` - Retrieve stack events for debugging
- `GetStackOutputs()` - Extract stack outputs

**Error Handling**:
Custom typed errors for graceful handling:
- `StackAlreadyExistsError` - Stack exists, automatically try update instead
- `NoChangesError` - No changes to apply, inform user
- `StackNotFoundError` - Stack doesn't exist, handle gracefully in delete

**Implementation**:
```go
type Client struct {
    cfnClient *cloudformation.Client
}

func (c *Client) CreateStack(ctx context.Context, params *CreateStackParams) (*CreateStackOutput, error) {
    // Creates stack and waits for CREATE_COMPLETE
}
```

---

### 3. Template Management (`internal/cloudformation/templates/`)

**Responsibility**: Embed and read CloudFormation templates

**go:embed Directive**:
```go
//go:embed *.yaml
var templateFS embed.FS

func Read(filename string) (string, error) {
    data, err := templateFS.ReadFile(filename)
    return string(data), err
}
```

**Templates Embedded**:
- `cluster-vpc.yaml` - VPC networking stack
- `cluster-iam.yaml` - IAM roles and OIDC provider stack
- `lambda-bootstrap.yaml` - Lambda function stack (optional)

**Benefits**:
- Single portable binary (no external template files needed)
- Templates embedded at compile time
- No runtime file I/O errors
- Consistent across all environments

---

### 4. Lambda Handler (`internal/lambda/handler.go`) - Optional

**Responsibility**: Execute in Lambda for event-driven workflows (optional feature)

**Dual-Mode Binary**:
The same Go binary runs in two modes:
- **CLI Mode**: When `AWS_LAMBDA_RUNTIME_API` is not set (default)
- **Lambda Mode**: When `AWS_LAMBDA_RUNTIME_API` is set

**Lambda Actions**:
- `apply-cluster-vpc` - Creates VPC CloudFormation stack
- `delete-cluster-vpc` - Deletes VPC stack
- `apply-cluster-iam` - Creates IAM CloudFormation stack
- `delete-cluster-iam` - Deletes IAM stack

**Implementation**:
```go
func Handler(ctx context.Context, event Event) (Response, error) {
    switch event.Action {
    case "apply-cluster-vpc":
        return applyClusterVPC(ctx, event)
    case "apply-cluster-iam":
        return applyClusterIAM(ctx, event)
    // ...
    }
}
```

**Note**: Lambda is **not required** for basic cluster management. It's an optional deployment mode for CI/CD integration.

---

### 5. Crypto Layer (`internal/crypto/`)

**TLS Thumbprint Fetching**:
- Connects to OIDC issuer URL via HTTPS
- Extracts root CA certificate from TLS handshake
- Calculates SHA-1 fingerprint of the root CA
- Mimics Terraform's `data.tls_certificate` behavior
- Used by `cluster-iam create` to auto-fetch thumbprint

**OIDC Domain Extraction**:
- Strips `https://` prefix from OIDC issuer URL
- Validates URL format
- Used as parameter for CloudFormation template

**Implementation**:
```go
func GetOIDCThumbprint(ctx context.Context, issuerURL string) (string, error) {
    // Connect to issuer, get TLS cert, return SHA-1 hash
}

func GetOIDCIssuerDomain(issuerURL string) (string, error) {
    // Strip https:// prefix and validate
    return strings.TrimPrefix(issuerURL, "https://"), nil
}
```

---

### 6. CloudFormation Templates

All templates are embedded in the binary using `//go:embed` directive.

**Cluster VPC Template** (`internal/cloudformation/templates/cluster-vpc.yaml`):
- VPC with configurable CIDR block (default: 10.0.0.0/16)
- 3 public subnets + 3 private subnets across availability zones
- Internet Gateway for public subnets
- NAT Gateway(s) - single (cost savings) or per-AZ (HA)
- Route tables and routes
- Security groups for cluster nodes
- Route53 private hosted zone for internal DNS

**Parameters**: ClusterName, VpcCidr, PublicSubnetCidrs, PrivateSubnetCidrs, SingleNatGateway, AvailabilityZone1/2/3

**Cluster IAM Template** (`internal/cloudformation/templates/cluster-iam.yaml`):
- IAM OIDC Provider pointing to Red Hat's CloudFront URL
- 7 control plane IAM roles with OIDC trust policies:
  - `IngressOperatorRole` - ROSAIngressOperatorPolicy
  - `KubeControllerRole` - ROSAKubeControllerPolicy
  - `EBSCSIDriverRole` - ROSAAmazonEBSCSIDriverOperatorPolicy
  - `ImageRegistryOperatorRole` - ROSAImageRegistryOperatorPolicy
  - `CloudNetworkConfigOperatorRole` - ROSACloudNetworkConfigOperatorPolicy
  - `ControlPlaneOperatorRole` - ROSAControlPlaneOperatorPolicy + supplemental policies
  - `NodePoolManagementRole` - ROSANodePoolManagementPolicy + supplemental policies
- Worker node IAM role + instance profile with EC2 trust

**Parameters**: ClusterName, OIDCIssuerURL, OIDCIssuerDomain, OIDCThumbprint

**Lambda Bootstrap Template** (`internal/cloudformation/templates/lambda-bootstrap.yaml`) - Optional:
- Lambda execution IAM role with CloudFormation + IAM permissions
- Lambda function using container image from ECR
- Parameters: ContainerImageURI, FunctionName

---

### 7. Container Image (Optional - for Lambda deployment)

**Dockerfile** (UBI9-based multi-stage build):
- Build stage: golang:1.24 compiles Go binary with embedded templates
- Runtime stage: ubi9-minimal with Go binary only (templates are embedded)
- No separate template files needed in container

**Build and Push** (only needed for Lambda deployment):
```bash
docker build -f Dockerfile -t rosa-cli:latest .
docker tag rosa-cli:latest <account>.dkr.ecr.us-east-1.amazonaws.com/rosa-cli:latest
docker push <account>.dkr.ecr.us-east-1.amazonaws.com/rosa-cli:latest
```

**Note**: Container image is only required if deploying rosactl as a Lambda function. For CLI usage, the standalone binary is sufficient.

---

## Data Flow

### Flow 1: Create Cluster VPC

```
1. User runs: rosactl cluster-vpc create my-cluster --region us-east-1
   ↓
2. CLI validates inputs (cluster name, CIDR ranges)
   ↓
3. CLI reads embedded template via templates.Read("cluster-vpc.yaml")
   ↓
4. CLI loads AWS config with region
   ↓
5. CLI creates CloudFormation client
   ↓
6. CLI calls CreateStack with parameters:
   - StackName: rosa-my-cluster-vpc
   - TemplateBody: <embedded cluster-vpc.yaml>
   - Parameters: ClusterName, VpcCidr, PublicSubnetCidrs, PrivateSubnetCidrs, SingleNatGateway
   - Tags: Cluster=my-cluster, ManagedBy=rosactl, red-hat-managed=true
   ↓
7. CloudFormation creates resources:
   - VPC
   - 3 public subnets + 3 private subnets
   - Internet Gateway
   - NAT Gateway(s)
   - Route tables and routes
   - Security groups
   - Route53 private hosted zone
   ↓
8. CLI waits for CREATE_COMPLETE (15 minute timeout)
   ↓
9. CLI displays stack outputs:
   - VpcId
   - PublicSubnetIds
   - PrivateSubnetIds
   - PrivateHostedZoneId
```

---

### Flow 2: Create Cluster IAM

```
1. User runs: rosactl cluster-iam create my-cluster \
              --oidc-issuer-url https://d1234.cloudfront.net/my-cluster \
              --region us-east-1
   ↓
2. CLI validates inputs (cluster name, OIDC URL format)
   ↓
3. CLI fetches TLS thumbprint from OIDC issuer URL via HTTPS
   ↓
4. CLI derives OIDC issuer domain (strips https:// prefix)
   ↓
5. CLI reads embedded template via templates.Read("cluster-iam.yaml")
   ↓
6. CLI loads AWS config with region
   ↓
7. CLI creates CloudFormation client
   ↓
8. CLI calls CreateStack with parameters:
   - StackName: rosa-my-cluster-iam
   - TemplateBody: <embedded cluster-iam.yaml>
   - Parameters: ClusterName, OIDCIssuerURL, OIDCIssuerDomain, OIDCThumbprint
   - Capabilities: CAPABILITY_IAM, CAPABILITY_NAMED_IAM
   - Tags: Cluster=my-cluster, ManagedBy=rosactl, red-hat-managed=true
   ↓
9. CloudFormation creates resources:
   - IAM OIDC Provider
   - 7 control plane IAM roles with OIDC trust policies
   - Worker node IAM role + instance profile
   ↓
10. CLI waits for CREATE_COMPLETE (15 minute timeout)
   ↓
11. CLI displays stack outputs:
   - OIDCProviderArn
   - IngressOperatorRoleArn
   - KubeControllerRoleArn
   - EBSCSIDriverRoleArn
   - ImageRegistryOperatorRoleArn
   - CloudNetworkConfigOperatorRoleArn
   - ControlPlaneOperatorRoleArn
   - NodePoolManagementRoleArn
   - WorkerRoleArn
   - WorkerInstanceProfileArn
```

---

### Flow 3: Delete Cluster Resources

```
1. User runs: rosactl cluster-iam delete my-cluster --region us-east-1
   ↓
2. CLI loads AWS config with region
   ↓
3. CLI creates CloudFormation client
   ↓
4. CLI calls DeleteStack:
   - StackName: rosa-my-cluster-iam
   ↓
5. CloudFormation deletes all resources:
   - Worker instance profile
   - Worker IAM role
   - 7 control plane IAM roles
   - IAM OIDC Provider
   ↓
6. CLI waits for DELETE_COMPLETE (15 minute timeout)
   ↓
7. CLI displays success message

Similar flow for cluster-vpc delete (stack name: rosa-my-cluster-vpc)
```

---

### Flow 4: Optional Lambda Deployment

For event-driven workflows, rosactl can be deployed as a Lambda function:

```
1. User builds container: docker build -t rosactl:latest .
   ↓
2. User pushes to ECR: docker push <account>.dkr.ecr.../rosactl:latest
   ↓
3. User runs: rosactl lambda create rosactl-bootstrap --handler default
   ↓
4. CLI creates CloudFormation stack with Lambda function
   ↓
5. Lambda function can be invoked via AWS events, Step Functions, etc.
   - Same binary runs in Lambda mode
   - Same embedded templates
   - Same CloudFormation logic
```

---

## Design Patterns

### Pattern 1: Embedded Templates with go:embed
**Implementation**: CloudFormation templates embedded at compile time
```go
//go:embed *.yaml
var templateFS embed.FS

func Read(filename string) (string, error) {
    data, err := templateFS.ReadFile(filename)
    return string(data), nil
}
```

**Benefits**:
- Single portable binary (no external files needed)
- No runtime file I/O errors
- Templates versioned with code
- Works in any environment (local, Lambda, container)

---

### Pattern 2: Direct CloudFormation Management
**Implementation**: CLI commands directly call CloudFormation API

**Benefits**:
- No Lambda required for basic operations
- Simpler architecture (fewer moving parts)
- Faster execution (no Lambda cold start)
- Direct user feedback via CloudFormation events
- Same permissions as user's AWS credentials

---

### Pattern 3: Typed Error Handling
**Implementation**: Custom error types for CloudFormation states
```go
type StackAlreadyExistsError struct { StackName string }
type NoChangesError struct { StackName string }
type StackNotFoundError struct { StackName string }
```

**Benefits**:
- Graceful handling of common scenarios
- Automatic update when stack exists
- Friendly messages for "no changes" case
- Safe deletion of non-existent stacks

---

### Pattern 4: Dual-Mode Binary (Optional)
**Implementation**: Single Go binary detects execution environment
```go
if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
    lambdaHandler.Start()  // Lambda mode
} else {
    commands.Execute()     // CLI mode
}
```

**Benefits**:
- Single binary for CLI and Lambda
- No code duplication
- Same CloudFormation logic in both modes
- Lambda deployment is optional, not required

---

## Security Architecture

### IAM Permissions Required

**CLI User Permissions** (customer developer/automation):

**CloudFormation**:
- CreateStack, UpdateStack, DeleteStack
- DescribeStacks, ListStacks, DescribeStackEvents
- DescribeStackResources, ListStackResources

**EC2** (for VPC creation):
- CreateVpc, DeleteVpc, CreateSubnet, DeleteSubnet
- CreateSecurityGroup, DeleteSecurityGroup
- CreateNatGateway, DeleteNatGateway
- CreateInternetGateway, DeleteInternetGateway
- CreateRoute, DeleteRoute, CreateRouteTable, DeleteRouteTable
- AuthorizeSecurityGroupEgress, AuthorizeSecurityGroupIngress

**IAM** (for cluster IAM creation):
- CreateRole, DeleteRole, AttachRolePolicy, DetachRolePolicy
- CreateInstanceProfile, DeleteInstanceProfile
- AddRoleToInstanceProfile, RemoveRoleFromInstanceProfile
- CreateOpenIDConnectProvider, DeleteOpenIDConnectProvider
- GetOpenIDConnectProvider, ListOpenIDConnectProviders

**Route53** (for VPC):
- CreateHostedZone, DeleteHostedZone

**Optional - Lambda Bootstrap**:
- Lambda: CreateFunction, DeleteFunction, InvokeFunction
- ECR: GetAuthorizationToken, BatchGetImage (for container images)

**No Long-Lived Credentials**:
- OIDC uses federated trust (no AWS credentials in workloads)
- CLI uses AWS credential chain (profile, environment variables, IAM role)

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

**CloudFormation Stack Events**:
- Real-time stack events displayed during creation/deletion
- Detailed error messages for failed resources
- Stack status polling with timeout

**CLI Output**:
- Emoji-based progress indicators (🌐, 📄, ☁️, ✅, ❌)
- Structured output with stack ID and resource outputs
- Clear error messages with remediation suggestions

**CloudFormation Console**:
- Full stack history and drift detection
- Resource visualization
- Change sets for preview

**Optional - CloudWatch Logs** (when using Lambda):
- Lambda execution logs: `/aws/lambda/<function-name>`
- CloudFormation stack events from Lambda invocations

**CLI Logging**:
- stdout for user-facing messages
- stderr for errors
- `--verbose` flag for debug output (future enhancement)

---

## Trade-offs & Constraints

### Direct CloudFormation vs Lambda Invocation

**Decision**: CLI commands directly call CloudFormation API (Lambda is optional)

**Rationale**:
- Simpler architecture (fewer moving parts)
- No Lambda cold start delays
- Direct user feedback
- Fewer AWS resources to manage
- Lambda available for event-driven workflows when needed

**Trade-offs**:
- ✅ Faster execution (no Lambda cold start)
- ✅ Simpler setup (no Lambda bootstrap required)
- ✅ Direct CloudFormation error messages
- ✅ Works with user's AWS credentials
- ⚠️ Requires AWS credentials configured locally
- ⚠️ No built-in event-driven execution (unless using optional Lambda)

---

### Embedded Templates with go:embed

**Decision**: Embed CloudFormation templates in binary using go:embed

**Rationale**:
- Single portable binary
- No runtime file path issues
- Templates versioned with code
- Works in any environment

**Trade-offs**:
- ✅ No external file dependencies
- ✅ Single binary distribution
- ✅ No file I/O errors
- ✅ Templates always in sync with code
- ⚠️ Templates fixed at compile time (requires rebuild to update)
- ⚠️ Binary size slightly larger (but negligible for YAML files)

---

### CloudFormation for All Resources

**Decision**: All resources defined in CloudFormation, not direct SDK calls

**Rationale**:
- Declarative infrastructure (GitOps-friendly)
- Automatic rollback on failure
- Drift detection available
- Change sets for previewing updates
- Consistent with ROSA service model

**Trade-offs**:
- ✅ Full rollback support
- ✅ Change sets for safety
- ✅ Templates auditable before deployment
- ✅ Stack-based lifecycle management
- ⚠️ CloudFormation quotas (200 resources per stack - not an issue)
- ⚠️ Slower than direct SDK calls (stack creation ~2-5 minutes)

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
- ✅ No S3 buckets to manage in customer accounts
- ✅ Consistent with ROSA service model
- ✅ Auto-fetch TLS thumbprint from OIDC URL
- ⚠️ Requires Red Hat OIDC infrastructure to be available
- ⚠️ No offline/air-gapped support

---

## Future Enhancements

1. **CloudFormation Change Sets**: Use change sets for safer updates before applying
2. **Dry-Run Mode**: Preview what resources will be created/deleted
3. **Multi-Region**: Support replicating IAM/VPC resources across regions
4. **Stack Outputs Export**: Save stack outputs to file (JSON/YAML)
5. **Template Validation**: Validate templates before deployment (cfn-lint integration)
6. **Rollback Support**: Add `rollback` subcommand for failed stacks
7. **Parallel Stack Management**: Create VPC and IAM stacks concurrently
8. **LocalStack Full Support**: Enhanced testing with complete CloudFormation feature set
9. **Verbose Logging**: Add `--verbose` flag for detailed AWS SDK logging
10. **Cost Estimation**: Estimate AWS costs before creating resources

---

## References

- [ROSA Regional Platform Terraform](https://github.com/openshift-online/rosa-regional-platform)
- [HyperShift OIDC Implementation](https://github.com/openshift/hypershift)
- [AWS CloudFormation Best Practices](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/best-practices.html)
- [AWS Lambda Container Images](https://docs.aws.amazon.com/lambda/latest/dg/images-create.html)
