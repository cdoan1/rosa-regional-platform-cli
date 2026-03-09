# E2E Tests for rosactl

End-to-end tests for the rosactl Lambda commands using the Ginkgo testing framework.

## Overview

These tests validate the complete workflow of rosactl Lambda commands against real AWS services (Lambda, IAM). They ensure that:

- Lambda functions can be created, invoked, and deleted
- ZIP-based deployments work correctly
- List operations return accurate results
- Error cases are handled properly
- Version management works as expected

## Prerequisites

### Required

1. **AWS Profile Configuration** - The tests require `AWS_PROFILE` to be set:
   ```bash
   export AWS_PROFILE=your-profile-name
   ```

2. **AWS Credentials** - Your `~/.aws/credentials` file must contain the profile:
   ```ini
   [your-profile-name]
   aws_access_key_id = YOUR_ACCESS_KEY
   aws_secret_access_key = YOUR_SECRET_KEY
   region = us-east-1
   ```

3. **AWS Permissions** - Your AWS credentials must have permissions for:
   - Lambda: CreateFunction, DeleteFunction, GetFunction, InvokeFunction, ListFunctions, UpdateFunctionCode, ListVersionsByFunction
   - IAM: CreateRole, DeleteRole, AttachRolePolicy, DetachRolePolicy, PutRolePolicy, CreateOpenIDConnectProvider, DeleteOpenIDConnectProvider, ListOpenIDConnectProviders, GetOpenIDConnectProvider
   - S3: CreateBucket, DeleteBucket, PutObject, DeleteObject, ListBucket, HeadBucket, HeadObject, PutBucketPolicy, PutPublicAccessBlock (for OIDC tests)

4. **Go 1.24+** - Required by Ginkgo v2
   ```bash
   go version
   ```

5. **Ginkgo CLI** - Install test dependencies:
   ```bash
   make test-deps
   ```

## Running Tests

### Run all E2E tests

```bash
# Make sure AWS_PROFILE is set first!
export AWS_PROFILE=your-profile-name

# Run tests via Makefile (recommended)
make test-e2e
```

### Run tests directly with Ginkgo

```bash
# Make sure AWS_PROFILE is set first!
export AWS_PROFILE=your-profile-name

# Run all tests
ginkgo -v ./test/e2e

# Run specific test suite
ginkgo --focus="ZIP-based" ./test/e2e

# Run with more verbose output
ginkgo -v --trace ./test/e2e
```

## Environment Variables

### Required

- `AWS_PROFILE` - AWS profile name (MANDATORY)
  ```bash
  export AWS_PROFILE=your-profile-name
  ```

### Optional

- `E2E_BINARY_PATH` - Path to rosactl binary
  - Default: `./bin/rosactl`
  - Set by Makefile automatically

## Test Structure

```
test/e2e/
├── e2e_suite_test.go      # Ginkgo suite initialization
├── e2e_test.go            # Lambda test specifications
├── oidc_test.go           # OIDC test specifications
├── fixtures/
│   ├── cli.go             # CLI execution helper
│   ├── aws.go             # AWS verification helper
│   └── cleanup.go         # Resource cleanup utilities
└── README.md              # This file
```

## How Tests Work

1. **Setup (BeforeSuite)**
   - Validates `AWS_PROFILE` is set (fails immediately if not)
   - Builds or locates the rosactl binary
   - Verifies binary is executable

2. **Test Execution**
   - Each test creates resources with unique timestamped names
   - CLI commands are executed as subprocesses (real user experience)
   - AWS SDK verifies resources independently
   - Resources are tracked for cleanup

3. **Cleanup (AfterEach)**
   - All tracked resources are deleted
   - Cleanup happens even if tests fail
   - Prevents orphaned AWS resources

## Test Coverage

### Lambda Tests (`e2e_test.go`)
- **ZIP-based Lambda**: Create → Invoke → Delete workflow (default Python runtime)
- **List operations**: Table and JSON output formats
- **Error handling**: Duplicate creation, non-existent function invocation
- **Version management**: ~~Update function → List versions~~ (Currently disabled due to AWS Lambda update timing issues)

### OIDC Tests (`oidc_test.go`)
- **OIDC Lambda creation**: Create OIDC Lambda with embedded RSA keys, verify environment variables
- **OIDC delete Lambda creation**: Create deletion Lambda function
- **OIDC issuer lifecycle**: Create → List → Delete workflow
  - Verifies S3 bucket creation with OIDC discovery documents
  - Verifies IAM OIDC provider creation
  - Tests cleanup of all resources
- **Multiple issuers**: Create and manage multiple OIDC issuers
- **JSON output**: List OIDC issuers in JSON format
- **Error handling**: Missing Lambda function, auto-prefixing behavior

## CLI Interface Notes

The `rosactl` lambda commands use a simple interface:

### Create
- **ZIP-based functions**: `rosactl lambda create <name>` (default)
- **Handler options**: Use `--handler` flag to specify handler type (default, oidc, oidc-delete)

### Invoke
- **Basic invocation**: `rosactl lambda invoke <name>`
- **Specific version**: `rosactl lambda invoke <name> --version <version>`
- No `--payload` flag - invokes with empty event by default

### Other Commands
- **Delete**: `rosactl lambda delete <name>`
- **List**: `rosactl lambda list` or `rosactl lambda list --output json`
- **Update**: `rosactl lambda update <name>`
- **Versions**: `rosactl lambda versions <name>`

## Troubleshooting

### Error: "AWS_PROFILE environment variable must be set"

**Solution**: Export your AWS profile before running tests:
```bash
export AWS_PROFILE=your-profile-name
make test-e2e
```

### Error: "no valid credentials"

**Solution**: Verify your AWS credentials file:
```bash
cat ~/.aws/credentials
```

Ensure the profile exists and has valid credentials.

### Error: "failed to build rosactl binary"

**Solution**: Build the binary manually first:
```bash
make build
ls -la bin/rosactl
```

### Tests timeout or hang

**Possible causes**:
- AWS API throttling (wait a few minutes)
- Lambda function stuck in Pending state (check AWS console)
- Network connectivity issues

**Solution**: Run with more verbose output:
```bash
ginkgo -v --trace ./test/e2e
```

### Error: "ResourceConflictException: An update is in progress"

**Cause**: Attempting to update/invoke a function before it's fully ready. Lambda functions can be "Active" but still processing the initial creation or a previous update.

**Solution**: The tests use `WaitForFunctionReady()` which:
1. Checks both `State=Active` and `LastUpdateStatus=Successful`
2. Ensures the function remains stable for 5 seconds
3. This accounts for AWS Lambda's background processing

If you still see this error, increase the timeout or the stability period in `aws.go`.

### Resources not cleaned up

**Check**: Log into AWS console and look for resources with `test-` prefix.

**Manual cleanup**:
```bash
# List test functions
aws lambda list-functions --query 'Functions[?starts_with(FunctionName, `test-`)].FunctionName'

# Delete manually if needed
aws lambda delete-function --function-name test-xxx-123456789
```

## Cost Considerations

- Tests use real AWS resources but are designed for minimal cost
- Lambda invocations: ~10-20 per test run (free tier: 1M requests/month)
- Lambda compute: ~1-2 seconds per test (free tier: 400,000 GB-seconds/month)
- ECR storage: Minimal, repositories deleted immediately
- Expected cost: $0 for accounts within free tier

## Writing New Tests

### Basic test structure

```go
It("should do something", func(ctx SpecContext) {
    functionName := fmt.Sprintf("test-mytest-%d", time.Now().Unix())
    tracker.TrackLambda(functionName)

    By("Creating ZIP-based function")
    stdout, err := cli.ExpectSuccess("lambda", "create", functionName)
    Expect(err).NotTo(HaveOccurred())

    By("Verifying in AWS")
    exists, err := awsHelper.VerifyFunctionExists(ctx, functionName)
    Expect(err).NotTo(HaveOccurred())
    Expect(exists).To(BeTrue())

    By("Waiting for function to be ready")
    err = awsHelper.WaitForFunctionReady(ctx, functionName, 2*time.Minute)
    Expect(err).NotTo(HaveOccurred())

    // Now you can safely invoke or update the function
    // Test continues...
}, SpecTimeout(5*time.Minute))
```

### Key principles

1. **Always track resources**: Call `tracker.TrackLambda()` or `tracker.TrackECR()` immediately
2. **Use unique names**: Include timestamp in resource names
3. **Set timeouts**: Use `SpecTimeout()` for long-running tests
4. **Verify independently**: Use `awsHelper` to verify, not just CLI output
5. **Clean descriptions**: Use `By()` to document test steps
6. **Wait for function ready**: Use `WaitForFunctionReady()` before operations that modify the function (update, invoke) to avoid 409 ResourceConflictException errors. This ensures both `State=Active` and `LastUpdateStatus=Successful`, and that the function remains stable for at least 5 seconds (to avoid race conditions with AWS Lambda's background processing).

## CI/CD Integration

To run tests in CI:

```yaml
# GitHub Actions example
- name: Run E2E Tests
  env:
    AWS_PROFILE: ci-profile
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    AWS_REGION: us-east-1
  run: make test-e2e
```

## References

- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matcher Reference](https://onsi.github.io/gomega/)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
