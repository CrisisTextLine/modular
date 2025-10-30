# Go Module Versioning Guide

This document explains how the Modular framework handles Go semantic versioning for the core framework and its modules.

## Overview

Go modules follow semantic versioning (semver) with a special requirement for major versions 2 and above: the module path in `go.mod` must include a version suffix (e.g., `/v2`, `/v3`).

## Version Naming Rules

### For v0.x.x and v1.x.x

**No module path suffix is required.**

```go
// go.mod
module github.com/CrisisTextLine/modular
```

Tags:
- Core: `v1.0.0`, `v1.1.0`, `v1.2.3`
- Modules: `modules/reverseproxy/v1.0.0`, `modules/auth/v1.2.3`

### For v2.x.x and Higher

**Module path MUST include the `/vN` suffix.**

```go
// go.mod for v2
module github.com/CrisisTextLine/modular/v2
```

Tags:
- Core: `v2.0.0`, `v2.1.0`, `v3.0.0`
- Modules: `modules/reverseproxy/v2.0.0`, `modules/auth/v3.1.0`

## Automated Workflow Behavior

The release workflows automatically handle major version transitions:

### When Releasing v2.0.0 or Higher

1. **Version Determination**: The workflow calculates the next version based on contract changes and user input
2. **Module Path Validation**: Before creating a release:
   - Checks if releasing v2+ version
   - Validates current module path in `go.mod`
   - If no version suffix exists, adds `/vN` to the module path
   - If version suffix exists but doesn't match, fails with error
3. **Auto-Update**: If needed, updates `go.mod` and commits the change
4. **Release Creation**: Creates the GitHub release with the correct tag
5. **Go Proxy Announcement**: Announces to Go proxy using the correct module path

### Example Workflow

**Initial State (v1.x.x):**
```go
// modules/reverseproxy/go.mod
module github.com/CrisisTextLine/modular/modules/reverseproxy
```

**After Releasing v2.0.0:**
```go
// modules/reverseproxy/go.mod
module github.com/CrisisTextLine/modular/modules/reverseproxy/v2
```

The workflow:
1. Detects that next version is v2.0.0
2. Updates `go.mod` module path to include `/v2`
3. Commits the change
4. Creates tag `modules/reverseproxy/v2.0.0`
5. Announces `github.com/CrisisTextLine/modular/modules/reverseproxy/v2@v2.0.0` to Go proxy

## Manual Version Updates

If you need to manually prepare for a v2+ release:

### Core Framework

```bash
# 1. Update go.mod
sed -i 's|^module github.com/CrisisTextLine/modular$|module github.com/CrisisTextLine/modular/v2|' go.mod

# 2. Update import paths in all .go files (if any self-imports)
find . -name "*.go" -type f -not -path "*/modules/*" -not -path "*/examples/*" \
  -exec sed -i 's|github.com/CrisisTextLine/modular"|github.com/CrisisTextLine/modular/v2"|g' {} +

# 3. Run go mod tidy
go mod tidy

# 4. Test
go test ./...
```

### Module

```bash
MODULE_NAME="reverseproxy"  # Change this to your module name
MAJOR_VERSION="2"           # Change to your target major version

# 1. Update go.mod
sed -i "s|^module github.com/CrisisTextLine/modular/modules/${MODULE_NAME}$|module github.com/CrisisTextLine/modular/modules/${MODULE_NAME}/v${MAJOR_VERSION}|" \
  modules/${MODULE_NAME}/go.mod

# 2. Update import paths (if module has self-imports - rare)
find modules/${MODULE_NAME} -name "*.go" -type f \
  -exec sed -i "s|github.com/CrisisTextLine/modular/modules/${MODULE_NAME}\"|github.com/CrisisTextLine/modular/modules/${MODULE_NAME}/v${MAJOR_VERSION}\"|g" {} +

# 3. Run go mod tidy
cd modules/${MODULE_NAME}
go mod tidy

# 4. Test
go test ./...
```

## Importing v2+ Modules

When using v2+ versions in your code:

```go
// For v1.x.x
import "github.com/CrisisTextLine/modular/modules/reverseproxy"

// For v2.x.x
import "github.com/CrisisTextLine/modular/modules/reverseproxy/v2"

// For v3.x.x
import "github.com/CrisisTextLine/modular/modules/reverseproxy/v3"
```

In `go.mod`:
```go
require (
    github.com/CrisisTextLine/modular/v2 v2.0.0
    github.com/CrisisTextLine/modular/modules/reverseproxy/v2 v2.0.0
)
```

## Breaking Changes and Major Versions

According to semantic versioning:
- **Breaking changes** require a major version bump (e.g., v1.5.0 → v2.0.0)
- **Backward-compatible additions** require a minor version bump (e.g., v1.5.0 → v1.6.0)
- **Bug fixes** require a patch version bump (e.g., v1.5.0 → v1.5.1)

Our workflows use contract-based detection to suggest appropriate version bumps, but you can override this with manual version input or by selecting a different release type.

## Troubleshooting

### Error: "module contains a go.mod file, so module path must match major version"

This error occurs when:
1. You're trying to release v2.0.0 or higher
2. The module path in `go.mod` doesn't include the `/vN` suffix

**Solution**: The workflow should handle this automatically. If you see this error, it means the auto-update step failed. Manually update the module path as described above.

### Error: "Module path has /vX but releasing vY"

This error occurs when the module path already has a version suffix, but it doesn't match the version you're trying to release.

**Solution**: Either:
1. Update the version number to match the existing suffix, or
2. Manually update the module path suffix to match your target version

### Downgrading Major Versions

**You cannot downgrade major versions** (e.g., from v3 to v2). If you need to maintain an older major version:
1. Create a branch from the appropriate tag (e.g., `v2-maintenance` from `v2.5.0`)
2. Apply fixes to that branch
3. Release patch versions on that branch (e.g., v2.5.1, v2.5.2)

## References

- [Go Modules: v2 and Beyond](https://go.dev/blog/v2-go-modules)
- [Go Module Reference](https://go.dev/ref/mod)
- [Semantic Versioning](https://semver.org/)

## Testing

To test the version handling logic locally, run:

```bash
./scripts/test-version-handling.sh
```

This demonstrates how versions are mapped to module paths.
