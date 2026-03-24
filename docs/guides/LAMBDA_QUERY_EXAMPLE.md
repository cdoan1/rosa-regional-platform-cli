# Lambda Query Example: Complete Workflow

This guide demonstrates how to create cluster resources and use a Lambda function to query their configuration for a cluster named `test-cluster`.

## Prerequisites

- AWS credentials configured
- Container image for rosactl pushed to ECR (e.g., `123456789012.dkr.ecr.us-east-2.amazonaws.com/rosactl:latest`)
- OIDC issuer URL for your cluster (e.g., `https://oidc.example.com/test-cluster`)

## Step 1: Create Cluster VPC Resources

Create the VPC, subnets, NAT gateways, and networking infrastructure:

```bash
rosactl cluster-vpc create test-cluster --region us-east-2
```

**Expected Output:**
```
🌐 Creating cluster VPC resources for: test-cluster
   Region: us-east-2
   VPC CIDR: 10.0.0.0/16
   Single NAT Gateway: true

📄 Loading CloudFormation template...
☁️  Creating CloudFormation stack: rosa-test-cluster-vpc
   This may take several minutes...

✅ Cluster VPC resources created successfully!
   Stack ID: arn:aws:cloudformation:us-east-2:123456789012:stack/rosa-test-cluster-vpc/abc123

Outputs:
  VpcId: vpc-0abcd1234efgh5678
  VpcCidr: 10.0.0.0/16
  PublicSubnetIds: subnet-111,subnet-222,subnet-333
  PrivateSubnetIds: subnet-444,subnet-555,subnet-666
  PrivateHostedZoneId: Z1234567890ABC
  InternetGatewayId: igw-0123456789abcdef0
  NatGatewayId: nat-0123456789abcdef0
```

## Step 2: Create Cluster IAM Resources

Create the OIDC provider and IAM roles for the cluster:

```bash
rosactl cluster-iam create test-cluster \
  --oidc-issuer-url https://oidc.example.com/test-cluster \
  --region us-east-2
```

**Expected Output:**
```
🔐 Creating cluster IAM resources...
   Cluster: test-cluster
   OIDC Issuer: https://oidc.example.com/test-cluster
   Region: us-east-2

🔍 Fetching TLS thumbprint from OIDC issuer...
   Thumbprint: a1b2c3d4e5f67890a1b2c3d4e5f67890a1b2c3d4

📄 Loading CloudFormation template...
☁️  Creating CloudFormation stack: rosa-test-cluster-iam
   This may take several minutes...

✅ Cluster IAM resources created successfully!
   Stack ID: arn:aws:cloudformation:us-east-2:123456789012:stack/rosa-test-cluster-iam/xyz789

Created Resources:
  OIDCProviderArn: arn:aws:iam::123456789012:oidc-provider/oidc.example.com/test-cluster
  IngressOperatorRoleArn: arn:aws:iam::123456789012:role/test-cluster-ingress-operator
  KubeControllerManagerRoleArn: arn:aws:iam::123456789012:role/test-cluster-kube-controller-manager
  EBSCSIDriverOperatorRoleArn: arn:aws:iam::123456789012:role/test-cluster-ebs-csi
  ImageRegistryOperatorRoleArn: arn:aws:iam::123456789012:role/test-cluster-image-registry
  CloudNetworkConfigOperatorRoleArn: arn:aws:iam::123456789012:role/test-cluster-network-config
  ControlPlaneOperatorRoleArn: arn:aws:iam::123456789012:role/test-cluster-control-plane-operator
  NodePoolManagementRoleArn: arn:aws:iam::123456789012:role/test-cluster-node-pool-management
  WorkerRoleArn: arn:aws:iam::123456789012:role/test-cluster-worker
  WorkerInstanceProfileName: test-cluster-worker
```

## Step 3: Create Lambda Function

Create a Lambda function that can query cluster resources:

```bash
rosactl lambda create cluster-query-lambda \
  --image-uri 123456789012.dkr.ecr.us-east-2.amazonaws.com/rosactl:latest \
  --region us-east-2
```

**For cross-account access:**
```bash
rosactl lambda create cluster-query-lambda \
  --image-uri 123456789012.dkr.ecr.us-east-2.amazonaws.com/rosactl:latest \
  --region us-east-2 \
  --allow-cross-account
```

**Expected Output:**
```
Creating Lambda function for cluster resource queries...
   Function name: cluster-query-lambda
   Stack name: rosa-lambda-cluster-query-lambda
   Container image: 123456789012.dkr.ecr.us-east-2.amazonaws.com/rosactl:latest
   Region: us-east-2

Creating CloudFormation stack...

Lambda function created successfully!

Outputs:
  LambdaFunctionArn: arn:aws:lambda:us-east-2:123456789012:function:cluster-query-lambda
  LambdaFunctionName: cluster-query-lambda
  ExecutionRoleArn: arn:aws:iam::123456789012:role/cluster-query-lambda-execution-role

To invoke the Lambda function:
  rosactl lambda invoke cluster-query-lambda --cluster-name test-cluster --region us-east-2
```

## Step 4: Invoke Lambda to Query Cluster Configuration

Invoke the Lambda function to retrieve all cluster resource information:

```bash
rosactl lambda invoke cluster-query-lambda \
  --cluster-name test-cluster \
  --region us-east-2
```

**Expected Output:**
```
Invoking Lambda function: cluster-query-lambda
   Cluster: test-cluster
   Region: us-east-2

Cluster: test-cluster
Region: us-east-2

IAM Resources:
───────────────────────────────────────────────────────────────
  Stack Name: rosa-test-cluster-iam
  Status: CREATE_COMPLETE
  Created: 2024-01-15T10:30:00Z

  Outputs:
    OIDCProviderArn                    : arn:aws:iam::123456789012:oidc-provider/oidc.example.com/test-cluster
    OIDCProviderURL                    : oidc.example.com/test-cluster
    IngressOperatorRoleArn             : arn:aws:iam::123456789012:role/test-cluster-ingress-operator
    KubeControllerManagerRoleArn       : arn:aws:iam::123456789012:role/test-cluster-kube-controller-manager
    EBSCSIDriverOperatorRoleArn        : arn:aws:iam::123456789012:role/test-cluster-ebs-csi
    ImageRegistryOperatorRoleArn       : arn:aws:iam::123456789012:role/test-cluster-image-registry
    CloudNetworkConfigOperatorRoleArn  : arn:aws:iam::123456789012:role/test-cluster-network-config
    ControlPlaneOperatorRoleArn        : arn:aws:iam::123456789012:role/test-cluster-control-plane-operator
    NodePoolManagementRoleArn          : arn:aws:iam::123456789012:role/test-cluster-node-pool-management
    WorkerRoleArn                      : arn:aws:iam::123456789012:role/test-cluster-worker
    WorkerInstanceProfileName          : test-cluster-worker

VPC Resources:
───────────────────────────────────────────────────────────────
  Stack Name: rosa-test-cluster-vpc
  Status: CREATE_COMPLETE
  Created: 2024-01-15T10:25:00Z

  Outputs:
    VpcId                              : vpc-0abcd1234efgh5678
    VpcCidr                            : 10.0.0.0/16
    PublicSubnetIds                    : subnet-111,subnet-222,subnet-333
    PrivateSubnetIds                   : subnet-444,subnet-555,subnet-666
    PrivateHostedZoneId                : Z1234567890ABC
    InternetGatewayId                  : igw-0123456789abcdef0
    NatGatewayId                       : nat-0123456789abcdef0
```

## Step 5: Get Raw JSON Output (Optional)

For programmatic processing, get the raw JSON response:

```bash
rosactl lambda invoke cluster-query-lambda \
  --cluster-name test-cluster \
  --region us-east-2 \
  --output
```

**Expected JSON Output:**
```json
{
  "cluster_info": {
    "cluster_name": "test-cluster",
    "region": "us-east-2",
    "iam": {
      "stack_name": "rosa-test-cluster-iam",
      "stack_id": "arn:aws:cloudformation:us-east-2:123456789012:stack/rosa-test-cluster-iam/xyz789",
      "status": "CREATE_COMPLETE",
      "creation_time": "2024-01-15T10:30:00Z",
      "outputs": {
        "CloudNetworkConfigOperatorRoleArn": "arn:aws:iam::123456789012:role/test-cluster-network-config",
        "ControlPlaneOperatorRoleArn": "arn:aws:iam::123456789012:role/test-cluster-control-plane-operator",
        "EBSCSIDriverOperatorRoleArn": "arn:aws:iam::123456789012:role/test-cluster-ebs-csi",
        "ImageRegistryOperatorRoleArn": "arn:aws:iam::123456789012:role/test-cluster-image-registry",
        "IngressOperatorRoleArn": "arn:aws:iam::123456789012:role/test-cluster-ingress-operator",
        "KubeControllerManagerRoleArn": "arn:aws:iam::123456789012:role/test-cluster-kube-controller-manager",
        "NodePoolManagementRoleArn": "arn:aws:iam::123456789012:role/test-cluster-node-pool-management",
        "OIDCProviderArn": "arn:aws:iam::123456789012:oidc-provider/oidc.example.com/test-cluster",
        "OIDCProviderURL": "oidc.example.com/test-cluster",
        "WorkerInstanceProfileName": "test-cluster-worker",
        "WorkerRoleArn": "arn:aws:iam::123456789012:role/test-cluster-worker"
      }
    },
    "vpc": {
      "stack_name": "rosa-test-cluster-vpc",
      "stack_id": "arn:aws:cloudformation:us-east-2:123456789012:stack/rosa-test-cluster-vpc/abc123",
      "status": "CREATE_COMPLETE",
      "creation_time": "2024-01-15T10:25:00Z",
      "outputs": {
        "InternetGatewayId": "igw-0123456789abcdef0",
        "NatGatewayId": "nat-0123456789abcdef0",
        "PrivateHostedZoneId": "Z1234567890ABC",
        "PrivateSubnetIds": "subnet-444,subnet-555,subnet-666",
        "PublicSubnetIds": "subnet-111,subnet-222,subnet-333",
        "VpcCidr": "10.0.0.0/16",
        "VpcId": "vpc-0abcd1234efgh5678"
      }
    }
  }
}
```

## Alternative: Direct CLI Query (Without Lambda)

You can also query cluster resources directly without using Lambda:

```bash
# Query IAM resources
rosactl cluster-iam describe test-cluster --region us-east-2

# Query VPC resources
rosactl cluster-vpc describe test-cluster --region us-east-2
```

## External Invocation via AWS CLI

External systems can invoke the Lambda directly using the AWS CLI:

```bash
aws lambda invoke \
  --function-name cluster-query-lambda \
  --region us-east-2 \
  --payload '{"action":"describe-cluster","cluster_name":"test-cluster","region":"us-east-2"}' \
  response.json

cat response.json | jq .
```

**Cross-Account Invocation:**
```bash
# From a different AWS account (requires --allow-cross-account)
aws lambda invoke \
  --function-name arn:aws:lambda:us-east-2:123456789012:function:cluster-query-lambda \
  --region us-east-2 \
  --payload '{"action":"describe-cluster","cluster_name":"test-cluster"}' \
  response.json

cat response.json | jq .
```

## Cleanup

When done, clean up resources in reverse order:

```bash
# Delete Lambda function
rosactl lambda delete cluster-query-lambda --region us-east-2

# Delete IAM resources
rosactl cluster-iam delete test-cluster --region us-east-2

# Delete VPC resources
rosactl cluster-vpc delete test-cluster --region us-east-2
```

## Use Cases

This Lambda-based query approach is useful for:

1. **CI/CD Pipelines**: Retrieve cluster configuration as part of automated workflows
2. **Multi-Account Architectures**: Query cluster resources from a central management account
3. **Configuration Management**: Export cluster resource ARNs for use in other systems
4. **Audit & Compliance**: Programmatically verify cluster resource configuration
5. **Event-Driven Workflows**: Trigger downstream processes based on cluster state

## Security Considerations

- **Same Account (default)**: Lambda can only be invoked by IAM principals in the same AWS account
- **Cross-Account**: Use `--allow-cross-account` flag to allow any AWS account to invoke
- **IAM Permissions**: Ensure the Lambda execution role has sufficient CloudFormation read permissions
- **Network**: Lambda runs in AWS's managed VPC by default (no VPC configuration needed)
