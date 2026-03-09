# Documentation Guidelines

This guide defines the standards and best practices for writing documentation in the rosactl project.

## Core Principles

### 1. Conciseness
- **Be direct and concise** - Remove unnecessary words
- **One concept per paragraph** - Don't mix multiple ideas
- **Use active voice** - "The Lambda invokes the function" not "The function is invoked by Lambda"

### 2. Visual Over Text
- **Prioritize diagrams** - Use ASCII diagrams, flowcharts, and swimlanes
- **Show, don't tell** - Include code examples and command outputs
- **Tables over lists** - Use tables for comparisons and structured data

### 3. Examples First
- **Start with examples** - Show working code before explaining theory
- **Real-world scenarios** - Use actual use cases, not abstract examples
- **Include expected output** - Always show what the user should see

## Documentation Types

### Architecture Documentation (`docs/architecture/`)

**Purpose**: Explain system design and structure

**Format**:
```markdown
# Component Name

## Overview
[1-2 sentence description]

## Architecture Diagram
[ASCII or Mermaid diagram]

## Key Components
- Component A: [Purpose]
- Component B: [Purpose]

## Data Flow
[Swimlane or sequence diagram]

## Trade-offs
[Design decisions and rationale]
```

**Examples**:
- `ARCHITECTURE.md` - System-wide architecture
- `DECISIONS.md` - Architecture decision records (ADRs)

---

### User Guides (`docs/guides/`)

**Purpose**: Help users accomplish specific tasks

**Format**:
```markdown
# Task Name

## Quick Start
[30-second example]

## Detailed Steps
1. Step 1
   ```bash
   command example
   ```
   Expected output:
   ```
   output example
   ```

2. Step 2
   ...

## Common Issues
[Troubleshooting tips]
```

**Examples**:
- `VERSIONING.md` - How to manage versions
- `DEVELOPMENT.md` - Developer setup guide

---

### Feature Specifications (`docs/specs/`)

**Purpose**: Define requirements and implementation details

**Format**:
```markdown
# Feature Name

## Summary
[What this feature does in 2-3 sentences]

## User Stories
- As a [role], I want [feature] so that [benefit]

## Requirements
- MUST: [Critical requirement]
- SHOULD: [Important requirement]
- COULD: [Nice-to-have]

## Implementation
[Technical approach]

## Examples
[Usage examples]
```

**Examples**:
- `feature-oidc.md` - OIDC provider management
- `feature-lambda.md` - Lambda function management
- `feature-e2e.md` - End-to-end testing

---

## Writing Style

### Commands and Code

✅ **Good**:
```markdown
Create a Lambda function:
```bash
rosactl lambda create my-function --handler oidc
```

Output:
```
Successfully created Lambda function: my-function
ARN: arn:aws:lambda:us-east-1:123456789012:function:my-function
```
```

❌ **Bad**:
```markdown
You can create a Lambda function by running the command to create it with the handler option set to oidc.
```

### Diagrams

✅ **Good** (Swimlane diagram):
```markdown
User                CLI             Lambda          AWS
  |                  |                |              |
  |-- create cmd --->|                |              |
  |                  |-- package ---->|              |
  |                  |                |-- deploy --->|
  |                  |                |<-- ARN ------|
  |<--- success -----|                |              |
```

❌ **Bad** (Wall of text):
```markdown
When the user runs the create command, the CLI packages the code and sends it to Lambda, which then deploys it to AWS and returns an ARN that the CLI shows to the user.
```

### Comparisons

✅ **Good** (Table):
```markdown
| Handler | Purpose | Use Case |
|---------|---------|----------|
| default | Hello world | Testing, demos |
| oidc | Create OIDC providers | Production identity |
| oidc-delete | Remove OIDC providers | Cleanup |
```

❌ **Bad** (Long paragraphs):
```markdown
The default handler is used for hello world and testing. The oidc handler creates OIDC providers for production identity management. The oidc-delete handler removes OIDC providers for cleanup purposes.
```

## Documentation Checklist

Before committing documentation, verify:

- [ ] **Spell-checked** - No typos (use `aspell` or IDE spellchecker)
- [ ] **Links work** - All internal references point to existing files
- [ ] **Code examples tested** - All commands actually work
- [ ] **Output current** - Example outputs match actual tool behavior
- [ ] **Diagrams clear** - ASCII diagrams render correctly in monospace font
- [ ] **Navigation clear** - Easy to find related docs
- [ ] **No dead ends** - Each doc links to related docs

## File Organization

```
docs/
├── README.md                    # Documentation index
├── architecture/
│   ├── ARCHITECTURE.md          # System architecture
│   └── DECISIONS.md             # ADRs
├── guides/
│   ├── VERSIONING.md            # How to version releases
│   ├── DEVELOPMENT.md           # Developer setup
│   └── DOCUMENTATION.md         # This file
└── specs/
    ├── OVERVIEW.md              # Project overview
    ├── REQUIREMENTS.md          # Requirements
    ├── USER_STORIES.md          # User stories
    ├── CLI_SPEC.md              # CLI specification
    ├── TECHNICAL_SPEC.md        # Technical details
    ├── feature-lambda.md        # Lambda feature spec
    ├── feature-oidc.md          # OIDC feature spec
    ├── feature-e2e.md           # E2E testing spec
    └── references.md            # External references
```

## Commit Messages for Documentation

Follow conventional commits:

```bash
# Documentation updates
git commit -m "docs: update OIDC guide with new examples"

# New documentation
git commit -m "docs: add troubleshooting guide for S3 timeouts"

# Fix typos
git commit -m "docs: fix typos in architecture documentation"
```

## Common Mistakes to Avoid

### 1. Too Much Theory, Not Enough Practice

❌ **Bad**:
> "The OIDC provider uses the S3-backed discovery document pattern, which is a common approach in cloud-native architectures for implementing federated identity..."

✅ **Good**:
```bash
# Create an OIDC provider
rosactl oidc create my-cluster --function my-oidc

# What this creates:
# - S3 bucket: oidc-issuer-my-cluster
# - Discovery doc: .well-known/openid-configuration
# - IAM provider: arn:aws:iam::...
```

### 2. Missing Context

❌ **Bad**:
```bash
rosactl lambda create my-function --handler oidc
```

✅ **Good**:
```bash
# Create an OIDC management Lambda
# This Lambda can create S3-backed OIDC issuers
rosactl lambda create my-function --handler oidc

# Private key saved to: /tmp/oidc-private-key-abc123.pem
# Keep this secure! It signs JWTs for the OIDC issuer.
```

### 3. Outdated Examples

❌ **Bad** (deprecated syntax):
```bash
rosactl lambda create oidc  # Old: name-based detection
```

✅ **Good** (current syntax):
```bash
rosactl lambda create my-oidc --handler oidc  # New: explicit --handler flag
```

## References

- [Google Developer Documentation Style Guide](https://developers.google.com/style)
- [Kubernetes Documentation Style Guide](https://kubernetes.io/docs/contribute/style/style-guide/)
- [The Documentation System](https://documentation.divio.com/) - Four types of docs framework
