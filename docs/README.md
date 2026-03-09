# Project Documentation

Welcome to the rosactl documentation! This directory contains all project documentation, specifications, and guides.

## 📚 Quick Links

| What do you want to do? | Start here |
|--------------------------|------------|
| **Learn what rosactl is** | [specs/OVERVIEW.md](specs/OVERVIEW.md) |
| **Understand the architecture** | [architecture/ARCHITECTURE.md](architecture/ARCHITECTURE.md) |
| **Build and contribute** | [guides/DEVELOPMENT.md](guides/DEVELOPMENT.md) |
| **Manage versions** | [guides/VERSIONING.md](guides/VERSIONING.md) |
| **Implement OIDC features** | [specs/feature-oidc.md](specs/feature-oidc.md) |
| **Run end-to-end tests** | [specs/feature-e2e.md](specs/feature-e2e.md) or [../test/e2e/README.md](../test/e2e/README.md) |

## 📁 Documentation Structure

```
docs/
├── README.md                    # This file
├── architecture/                # System design and architecture
│   ├── ARCHITECTURE.md          # System architecture overview
│   └── DECISIONS.md             # Architecture decision records (ADRs)
├── guides/                      # User and developer guides
│   ├── DEVELOPMENT.md           # Developer setup and workflows
│   ├── VERSIONING.md            # Semantic versioning guide
│   └── DOCUMENTATION.md         # Documentation writing guidelines
└── specs/                       # Feature specifications and requirements
    ├── OVERVIEW.md              # Project overview
    ├── REQUIREMENTS.md          # Functional requirements
    ├── USER_STORIES.md          # User stories and use cases
    ├── CLI_SPEC.md              # CLI command specifications
    ├── TECHNICAL_SPEC.md        # Technical implementation details
    ├── feature-lambda.md        # Lambda management feature spec
    ├── feature-oidc.md          # OIDC provider feature spec
    ├── feature-e2e.md           # End-to-end testing spec
    └── references.md            # External references and links
```

## 🏗️ Architecture Documentation

### [ARCHITECTURE.md](architecture/ARCHITECTURE.md)
Complete system architecture including:
- Component diagrams and data flow
- AWS service integrations (Lambda, S3, IAM)
- RSA private key management
- Security architecture
- Design trade-offs and decisions

### [DECISIONS.md](architecture/DECISIONS.md)
Architecture decision records (ADRs) documenting:
- Why certain technical choices were made
- Alternatives considered
- Trade-offs and consequences

## 📖 User & Developer Guides

### [DEVELOPMENT.md](guides/DEVELOPMENT.md)
Developer setup and contribution guide:
- Local development environment setup
- Build and test instructions
- Code organization and patterns
- Pull request workflow

### [VERSIONING.md](guides/VERSIONING.md)
Semantic versioning with conventional commits:
- How to use `make release` for version management
- Conventional commit message format
- Version bump rules (feat, fix, BREAKING CHANGE)
- Release workflow

### [DOCUMENTATION.md](guides/DOCUMENTATION.md)
Documentation writing guidelines:
- Documentation standards and style
- How to write effective docs
- Examples and anti-patterns

## 📋 Specifications

### Core Specifications

| Document | Purpose |
|----------|---------|
| [OVERVIEW.md](specs/OVERVIEW.md) | High-level project goals and scope |
| [REQUIREMENTS.md](specs/REQUIREMENTS.md) | Functional and non-functional requirements |
| [USER_STORIES.md](specs/USER_STORIES.md) | User personas and use cases |
| [CLI_SPEC.md](specs/CLI_SPEC.md) | Command-line interface design |
| [TECHNICAL_SPEC.md](specs/TECHNICAL_SPEC.md) | Implementation details and APIs |

### Feature Specifications

| Document | Purpose |
|----------|---------|
| [feature-lambda.md](specs/feature-lambda.md) | Lambda function management (create, invoke, delete, list, versions) |
| [feature-oidc.md](specs/feature-oidc.md) | S3-backed OIDC provider management |
| [feature-e2e.md](specs/feature-e2e.md) | End-to-end testing framework and test cases |

### References

| Document | Purpose |
|----------|---------|
| [references.md](specs/references.md) | External links, APIs, and documentation |

## 🚀 Getting Started

### For New Users

1. Read [OVERVIEW.md](specs/OVERVIEW.md) to understand what rosactl does
2. Check [../README.md](../README.md) for installation and quick start
3. Review [specs/CLI_SPEC.md](specs/CLI_SPEC.md) for command reference

### For Contributors

1. Read [DEVELOPMENT.md](guides/DEVELOPMENT.md) for setup instructions
2. Review [ARCHITECTURE.md](architecture/ARCHITECTURE.md) to understand the system
3. Follow [VERSIONING.md](guides/VERSIONING.md) for commit message format
4. Check [DOCUMENTATION.md](guides/DOCUMENTATION.md) for documentation standards

### For Feature Implementation

1. Read the relevant feature spec:
   - Lambda: [feature-lambda.md](specs/feature-lambda.md)
   - OIDC: [feature-oidc.md](specs/feature-oidc.md)
   - Testing: [feature-e2e.md](specs/feature-e2e.md)
2. Review [ARCHITECTURE.md](architecture/ARCHITECTURE.md) for integration points
3. Check [TECHNICAL_SPEC.md](specs/TECHNICAL_SPEC.md) for implementation details

## 🔄 Documentation Lifecycle

### Creating New Documentation

1. **Choose the right location**:
   - Architecture design → `architecture/`
   - User/developer guide → `guides/`
   - Feature spec → `specs/`

2. **Follow the style guide**: See [DOCUMENTATION.md](guides/DOCUMENTATION.md)

3. **Update this index**: Add your new doc to this README

4. **Use conventional commits**:
   ```bash
   git commit -m "docs: add JWT signing guide"
   ```

### Updating Existing Documentation

When the code changes:

1. **Update affected docs** immediately (don't let docs drift)
2. **Test all examples** (ensure commands still work)
3. **Update version badges** if needed (see [VERSIONING.md](guides/VERSIONING.md))
4. **Review for accuracy** before committing

### Documentation Review Checklist

Before merging documentation changes:

- [ ] Spell-checked (no typos)
- [ ] Links work (all references valid)
- [ ] Code examples tested (commands actually work)
- [ ] Output current (matches actual tool behavior)
- [ ] Diagrams render correctly
- [ ] Added to this index (if new file)

## 📊 Documentation Status

| Category | Status | Last Updated |
|----------|--------|--------------|
| Architecture | ✅ Complete | 2026-02-22 |
| Developer Guides | ✅ Complete | 2026-02-22 |
| Feature Specs | ✅ Complete | 2026-02-22 |
| User Stories | ⚠️ Needs Update | - |
| Requirements | ⚠️ Needs Update | - |
| Technical Spec | ⚠️ Needs Update | - |

## 🤝 Contributing to Documentation

Documentation improvements are always welcome! To contribute:

1. **Fix typos and errors**: Just submit a PR
2. **Add examples**: Include working code and expected output
3. **New features**: Update both code and docs in the same PR
4. **Improve clarity**: Simplify complex explanations

See [DOCUMENTATION.md](guides/DOCUMENTATION.md) for detailed guidelines.

## 📝 Documentation Conventions

### File Naming

- Use `UPPERCASE.md` for top-level docs (OVERVIEW.md, README.md)
- Use `lowercase-with-dashes.md` for feature specs (feature-oidc.md)
- Use `PascalCase.md` for specific topics (VERSIONING.md)

### Markdown Style

- Use ATX-style headers (`#` not underlines)
- Fenced code blocks with language hints (```bash not ```)
- Tables for comparisons and structured data
- Emoji for visual categorization (📚 📁 🚀 etc.)

### Code Examples

- Always include expected output
- Use real commands that actually work
- Show both success and error cases
- Include comments explaining non-obvious steps

## 🔗 External Resources

- [Main README](../README.md) - Project README
- [E2E Test README](../test/e2e/README.md) - End-to-end testing guide
- [Makefile](../Makefile) - Build targets and commands
- [GitHub Repository](https://github.com/openshift-online/rosa-regional-platform-cli)

---

**Questions or suggestions?** Open an issue or submit a PR!
