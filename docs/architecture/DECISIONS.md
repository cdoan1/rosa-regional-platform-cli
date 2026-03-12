# Architecture Decision Records (ADRs)

## ADR Template

```markdown
# ADR-XXX: [Title]

Date: YYYY-MM-DD

## Status
[Proposed | Accepted | Deprecated | Superseded]

## Context
[What is the issue that we're seeing that is motivating this decision or change?]

## Decision
[What is the change that we're proposing and/or doing?]

## Consequences
[What becomes easier or more difficult to do because of this change?]

### Positive
- [Benefit 1]
- [Benefit 2]

### Negative
- [Drawback 1]
- [Drawback 2]

### Neutral
- [Impact 1]

## Alternatives Considered
- [Alternative 1] - [Why not chosen]
- [Alternative 2] - [Why not chosen]
```

---

## ADR-001: Direct CloudFormation Management from CLI

Date: 2026-03-11

### Status
Accepted

### Context
Originally, rosactl required users to bootstrap a Lambda function before managing cluster resources. This meant:
- Extra setup step (build container, push to ECR, deploy Lambda)
- Two-hop execution (CLI → Lambda → CloudFormation)
- Lambda cold starts impacting performance
- More complex troubleshooting

Users wanted a simpler experience where they could run `rosactl cluster-vpc create` and `rosactl cluster-iam create` directly without Lambda bootstrap.

### Decision
CLI commands now directly call the CloudFormation API to create/update/delete stacks. Lambda deployment becomes optional and is only used for event-driven workflows (CI/CD, Step Functions, etc.).

The same Go binary can run in two modes:
- **CLI mode** (default): Direct CloudFormation management
- **Lambda mode** (optional): Event-driven execution

### Consequences

**Positive**:
- No Lambda bootstrap required for basic operations
- Faster execution (no Lambda cold start)
- Simpler architecture with fewer moving parts
- Direct CloudFormation error messages visible to users
- Works with user's AWS credentials (no separate Lambda role needed)
- Lower AWS costs (no Lambda invocations for basic usage)

**Negative**:
- Requires AWS credentials configured locally (AWS profile or environment variables)
- CLI execution time limited by user's session (not serverless)
- No built-in event-driven execution without Lambda

**Neutral**:
- Same CloudFormation templates used in both modes
- Same IAM permissions required (just scoped to user instead of Lambda role)

### Alternatives Considered
- **Lambda-only approach** - Rejected because it adds unnecessary complexity for basic operations
- **Terraform provider** - Rejected because it duplicates existing rosa-regional-platform Terraform and doesn't align with service tooling
- **Direct AWS SDK calls** - Rejected because CloudFormation provides rollback, drift detection, and declarative infrastructure

---

## ADR-002: Embed CloudFormation Templates using go:embed

Date: 2026-03-11

### Status
Accepted

### Context
CloudFormation templates needed to be available at runtime for both CLI and Lambda execution modes. Options considered:
1. Read from file system (`templates/` directory)
2. Embed in binary using go:embed
3. Inline as Go strings

Reading from file system caused issues:
- Path resolution problems (working directory, relative vs absolute paths)
- Binary must be run from specific directory
- Templates not versioned with binary
- Deployment complexity (must ship templates with binary)

### Decision
Use go:embed to embed CloudFormation templates directly in the binary at compile time:

```go
//go:embed *.yaml
var templateFS embed.FS

func Read(filename string) (string, error) {
    data, err := templateFS.ReadFile(filename)
    return string(data), nil
}
```

Templates are embedded from `internal/cloudformation/templates/` directory.

### Consequences

**Positive**:
- Single portable binary (no external file dependencies)
- No runtime file path issues
- Templates always versioned with code
- Works in any execution environment (local, container, Lambda)
- No file I/O errors at runtime
- Simpler deployment (just the binary)

**Negative**:
- Templates fixed at compile time (requires rebuild to update)
- Binary size slightly larger (~10-20KB for YAML files - negligible)

**Neutral**:
- Template files still exist in Git for auditing and review
- Can still view templates in repository

### Alternatives Considered
- **File system reading** - Rejected due to path resolution issues and deployment complexity
- **Inline Go strings** - Rejected because it makes templates hard to read, edit, and review
- **Download from S3 at runtime** - Rejected because it requires internet access and adds latency

---

## ADR-003: Optional Lambda Deployment for Event-Driven Workflows

Date: 2026-03-11

### Status
Accepted

### Context
While direct CLI execution is simpler for most use cases, some workflows benefit from event-driven execution:
- CI/CD pipelines triggered by git events
- AWS Step Functions orchestration
- S3 event triggers
- CloudWatch Events/EventBridge integration

We needed to support both direct execution and event-driven workflows without duplicating code.

### Decision
Make Lambda deployment **optional** via the `lambda` command group:

```bash
# Optional: Deploy rosactl as Lambda function
rosactl lambda create rosactl-bootstrap --handler default

# Lambda can then be invoked via AWS events
```

The same binary runs in both modes using runtime detection:
```go
if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
    lambdaHandler.Start()  // Lambda mode
} else {
    commands.Execute()     // CLI mode
}
```

### Consequences

**Positive**:
- Lambda is optional, not required
- Single binary for both CLI and Lambda
- No code duplication between modes
- Same CloudFormation templates in both modes
- Same error handling and logic
- Users choose deployment mode based on needs

**Negative**:
- Dual-mode detection adds minor complexity
- Lambda deployment requires container build and ECR push
- Two codepaths to maintain (though using same CloudFormation logic)

**Neutral**:
- Lambda mode uses same embedded templates
- Lambda requires execution role with CloudFormation/IAM permissions

### Alternatives Considered
- **CLI-only** - Rejected because event-driven workflows are valuable for automation
- **Lambda-only** - Rejected because it adds unnecessary complexity for basic usage (previous architecture)
- **Separate binaries** - Rejected because it causes code duplication and maintenance burden

---

## Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| 001 | Direct CloudFormation Management from CLI | Accepted | 2026-03-11 |
| 002 | Embed CloudFormation Templates using go:embed | Accepted | 2026-03-11 |
| 003 | Optional Lambda Deployment for Event-Driven Workflows | Accepted | 2026-03-11 |
