# Aura API Client

## Overview

A Go package that enables the use of Neo4j Aura API in a friendly way, e.g.
`client.Instances.List(ctx)` to return a list of v1 instances in Aura. The same
module also exposes a beta `v2beta1` sub-package (import path
`github.com/neo4j-contrib/aura-go-sdk/v2beta1`) for the newer organization,
project, database and database-backup operations using a shared `ScopeOption`
calling pattern.

Client Id and Secret are required and these can be obtained from the [Neo4j Aura Console](https://neo4j.com/docs/aura/api/authentication/).

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Documentation](#documentation)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Complete Example Application](#complete-example-application)
- [Additional Resources](#additional-resources)
- [CI & Releases](#ci--releases)
- [Contributing](#contributing)
- [License](#license)

---

## Installation

```bash
go get github.com/neo4j-contrib/aura-go-sdk
```

The stable v1 API and the beta `v2beta1` sub-package are provided by the same
module; one `go get` fetches both.

---

## Quick Start

### Stable — v1 (import `aura`)

```go
package main

import (
    "context"
    "log"
    aura "github.com/neo4j-contrib/aura-go-sdk"
)

func main() {
    client, err := aura.NewClient(
        aura.WithCredentials("your-client-id", "your-client-secret"),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    ctx := context.Background()

    instances, err := client.Instances.List(ctx)
    if err != nil {
        log.Fatalf("Failed to list instances: %v", err)
    }

    for _, instance := range instances.Data {
        log.Printf("Instance: %s (ID: %s)\n", instance.Name, instance.ID)
    }
}
```

For full reference on the v1 operations, client configuration and context /
timeout handling see [docs/v1.md](docs/v1.md).

### Beta — v2beta1 (import `v2beta1 "github.com/neo4j-contrib/aura-go-sdk/v2beta1"`)

```go
package main

import (
    "context"
    "fmt"
    "log"

    v2beta1 "github.com/neo4j-contrib/aura-go-sdk/v2beta1"
)

func main() {
    // Construct the client once and set a default org/project so repeat calls
    // inherit that scope instead of passing it every time. (See docs/v2beta1.md
    // for the full calling convention.)
    client, err := v2beta1.NewClient(
        v2beta1.WithCredentials("your-client-id", "your-client-secret"),
        v2beta1.WithDefaultOrg("your-org-uuid"),
        v2beta1.WithDefaultProject("your-project-uuid"),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    ctx := context.Background()

    // org + project are inherited from the default scope configured above.
    databases, err := client.Databases.List(ctx)
    if err != nil {
        log.Fatalf("Failed to list databases: %v", err)
    }

    for _, db := range databases.Data {
        fmt.Printf("Database ID: %s\n", db.ID)
    }
}
```

See [docs/v2beta1.md](docs/v2beta1.md) for the full v2beta1 reference,
including how to pass scope per call and how body-carrying operations place the
request body before the trailing scope options.

---

## Documentation

The operation catalogs are split out into dedicated files to keep this file a
short landing page:

- **[docs/v1.md](docs/v1.md)** — stable v1 API reference: client configuration,
  context & timeouts, tenant / instance / snapshot / CMEK / GDS session /
  Prometheus operations.
- **[docs/v2beta1.md](docs/v2beta1.md)** — beta `v2beta1` sub-package reference:
  client construction with default-scope, and every v2beta1 operation for
  organizations, projects, instances, databases, database backups, organization
  users, organization invites and project users, all via the shared
  `ScopeOption` pattern.

---

## Error Handling

### Basic Error Handling

```go
ctx := context.Background()

instance, err := client.Instances.Get(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("instance-id"))
if err != nil {
    log.Printf("Error: %v\n", err)
    return
}
```

### Typed API Errors

```go
ctx := context.Background()

instance, err := client.Instances.Get(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"), v2beta1.WithInstance("non-existent-id"))
if err != nil {
    if apiErr, ok := err.(*aura.Error); ok {
        fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)

        switch {
        case apiErr.IsNotFound():
            fmt.Println("Instance not found")
        case apiErr.IsUnauthorized():
            fmt.Println("Authentication failed - check credentials")
        case apiErr.IsBadRequest():
            fmt.Println("Invalid request parameters")
        }

        if apiErr.HasMultipleErrors() {
            fmt.Println("All errors:")
            for _, msg := range apiErr.AllErrors() {
                fmt.Printf("  - %s\n", msg)
            }
        }
        return
    }

    log.Printf("Unexpected error: %v\n", err)
    return
}
```

### Context Errors

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

instances, err := client.Instances.List(ctx, v2beta1.WithOrg("your-org-uuid"), v2beta1.WithProject("your-project-uuid"))
if err != nil {
    switch ctx.Err() {
    case context.DeadlineExceeded:
        log.Println("Request timed out")
    case context.Canceled:
        log.Println("Request was cancelled")
    default:
        log.Printf("Error: %v\n", err)
    }
    return
}
```

---

## Best Practices

### 1. Secure Credential Management

```go
clientID := os.Getenv("AURA_CLIENT_ID")
clientSecret := os.Getenv("AURA_CLIENT_SECRET")

if clientID == "" || clientSecret == "" {
    log.Fatal("Missing AURA credentials in environment")
}

client, err := aura.NewClient(
    aura.WithCredentials(clientID, clientSecret),
)
```

### 2. Save Instance Credentials Immediately After Creation

```go
ctx := context.Background()

instance, err := client.Instances.Create(ctx, config)
if err != nil {
    log.Fatal(err)
}

// ⚠️ CRITICAL: Save these immediately — they are only shown once!
credentials := map[string]string{
    "instance_id":    instance.Data.ID,
    "connection_url": instance.Data.ConnectionURL,
    "username":       instance.Data.Username,
    "password":       instance.Data.Password,
}
// Store in a secrets manager. Do NOT log passwords in production.
```

### 3. Polling for Async Operations

```go
ctx := context.Background()

instanceID := newInstance.Data.ID

for range 30 {
    inst, err := client.Instances.Get(ctx, instanceID)
    if err != nil {
        log.Printf("Error checking status: %v", err)
    } else if inst.Data.Status == aura.StatusRunning {
        fmt.Println("Instance is ready!")
        break
    } else {
        fmt.Printf("Status: %s, waiting...\n", inst.Data.Status)
    }
    time.Sleep(10 * time.Second)
}
```

### 4. Graceful Shutdown

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigChan
    fmt.Println("\nShutting down gracefully...")
    cancel()
}()

// Pass ctx to any in-flight calls — they will be cancelled on signal
instances, err := client.Instances.List(ctx)
```

### 5. Retry Logic for Transient Failures

```go
func retryOperation(maxRetries int, fn func() error) error {
    var err error
    for i := range maxRetries {
        err = fn()
        if err == nil {
            return nil
        }

        if apiErr, ok := err.(*aura.Error); ok {
            // Don't retry client errors (4xx except 429 Too Many Requests)
            if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != 429 {
                return err
            }
        }

        wait := time.Duration(math.Pow(2, float64(i))) * time.Second
        fmt.Printf("Attempt %d failed, retrying in %v...\n", i+1, wait)
        time.Sleep(wait)
    }
    return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}

// Usage
ctx := context.Background()
err := retryOperation(3, func() error {
    _, err := client.Instances.List(ctx)
    return err
})
```

---

## Complete Example Application

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    aura "github.com/neo4j-contrib/aura-go-sdk"
)

func main() {
    clientID := os.Getenv("AURA_CLIENT_ID")
    clientSecret := os.Getenv("AURA_CLIENT_SECRET")
    tenantID := os.Getenv("AURA_TENANT_ID")

    if clientID == "" || clientSecret == "" {
        log.Fatal("Missing required environment variables")
    }

    client, err := aura.NewClient(
        aura.WithCredentials(clientID, clientSecret),
        aura.WithTimeout(120 * time.Second),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    ctx := context.Background()

    fmt.Println("=== Current Instances ===")
    instances, err := client.Instances.List(ctx)
    if err != nil {
        log.Fatalf("Failed to list instances: %v", err)
    }

    for _, inst := range instances.Data {
        fmt.Printf("- %s: %s (%s)\n", inst.Name, inst.ID, inst.CloudProvider)
    }

    if tenantID != "" {
        fmt.Println("\n=== Tenant Configuration ===")
        tenant, err := client.Tenants.Get(ctx, tenantID)
        if err != nil {
            log.Printf("Warning: Could not get tenant: %v", err)
        } else {
            fmt.Printf("Tenant: %s\n", tenant.Data.Name)
            fmt.Printf("Available configurations: %d\n", len(tenant.Data.InstanceConfigurations))
        }
    }

    fmt.Println("\n✓ Client is working correctly!")
}
```

Run with:
```bash
export AURA_CLIENT_ID="your-client-id"
export AURA_CLIENT_SECRET="your-client-secret"
export AURA_TENANT_ID="your-tenant-id"
go run main.go
```

---

## Additional Resources

- [Neo4j Aura API Documentation](https://neo4j.com/docs/aura/platform/api/)
- [GitHub Repository](https://github.com/neo4j-contrib/aura-go-sdk)
- [Report Issues](https://github.com/neo4j-contrib/aura-go-sdk/issues)
- [Prometheus Metrics Guide](./docs/prometheus.md)
- [v1 API reference](docs/v1.md)
- [v2beta1 API reference](docs/v2beta1.md)

---

## CI & Releases

Three GitHub Actions workflows manage CI and the release process.

### Workflows

| Workflow | Trigger | What it does |
|---|---|---|
| **CI** | Push to `main`, every PR | Runs tests with the race detector, golangci-lint, and `go build ./...` |
| **Changelog check** | Every PR | Fails if the PR changes `.go` files but has no entry in `.changes/unreleased/` |
| **Release** | Push of a `vX.Y.Z` tag | Gates on tests, extracts the changelog section, creates a GitHub Release |

### API surface contract test

`go test` includes `TestAPISurface`, which snapshots every exported identifier in the `aura` and `v2beta1` packages and fails if it drifts from `testdata/api_surface.golden`. This catches accidental breaking changes that wouldn't otherwise fail the build — for example, an unintentional addition or removal of an exported method.

If a PR intentionally changes the public API, regenerate the snapshot and commit it:

```bash
UPDATE_GOLDEN=1 go test . ./v2beta1 -run TestAPISurface
```

Review the resulting diff in `testdata/api_surface.golden` as part of the code review — it's the actual contract change. Breaking changes (removed/renamed exports, new interface methods) still need a `Changed` changie entry.

### Making a release

Releases follow a three-step process. changie collects the unreleased fragment files and determines the correct semver bump automatically from the change kinds (`Added` → minor, `Fixed`/`Security` → patch, `Changed`/`Removed` → major).

There is **no manual version bump** required. `ClientVersion` uses `debug.ReadBuildInfo()` at runtime to read the module version that the Go toolchain embeds when a consumer builds their application. It falls back to `"development"` only in local and test builds.

**1. Batch and merge the changelog**

```bash
changie batch   # collects .changes/unreleased/*.yaml → .changes/vX.Y.Z.md
changie merge   # folds that file into CHANGELOG.md
```

**2. Commit and tag**

```bash
git add CHANGELOG.md .changes/
git commit -m "chore: release vX.Y.Z"
git tag vX.Y.Z
git push origin main --tags
```

**3. Workflow takes over**

Pushing the tag fires the Release workflow, which:
- Runs `go test -race ./...` — the release is aborted if any test fails
- Extracts the `## vX.Y.Z` section from `CHANGELOG.md`
- Creates a GitHub Release with that text as the release notes

Because this is a Go module with no compiled binaries, the tag itself is what consumers reference:

```bash
go get github.com/neo4j-contrib/aura-go-sdk@vX.Y.Z
```

### Adding a changelog entry

Every PR that changes Go source files needs a changie fragment. Run:

```bash
changie new
```

Choose a kind and write a one-line summary, then commit the generated `.yaml` file alongside your code changes. The Changelog check workflow will fail the PR if this step is skipped.

To bypass the check for a PR that genuinely needs no entry (docs-only, CI-only, or test-only changes), add the **`no-changelog`** label to the PR.

---

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

See [LICENSE](LICENSE) file for details.
