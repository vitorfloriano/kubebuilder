# Plugin Specification: `decoupled-go/v1`

> **Status:** Draft  
> **Plugin key:** `decoupled-go.example.com/v1`  
> **Type:** External plugin (Phase 2)  
> **Supported project version:** `3` (PROJECT file v3)  
> **Language:** Go

---

## 1. Scope

`decoupled-go/v1` scaffolds Go-based Kubernetes operators with a strict
**platform/app separation**:

- **Platform layer** (`internal/platform/`) — generated framework code,
  fully managed by the plugin and regenerated on every run.
- **App layer** (`internal/app/`) — user business logic, created once and
  never overwritten.

### Goals

- ✅ Scaffold a complete, production-ready Go operator project.
- ✅ Enforce a strict boundary between generated and user-owned code.
- ✅ Make scaffold upgrades as simple as bumping a version and running one command.
- ✅ Integrate cleanly with other Kubebuilder plugins (`kustomize/v2`, `grafana/v1-alpha`, etc.).
- ✅ Be modular: users can adopt individual parts without the full stack.
- ✅ Follow all Kubebuilder plugin chaining rules.

### Non-Goals

- ❌ Replace `kustomize/v2` — this plugin does not scaffold Kustomize manifests.
- ❌ Replace `go/v4` — this is a new plugin with different file ownership semantics.
- ❌ Support non-Go operators (Helm, Ansible, etc.).
- ❌ Provide a UI or web interface.
- ❌ Auto-generate business logic — only structural stubs.
- ❌ Manage CI/CD pipeline configuration (that is for `autoupdate/v1-alpha`).

---

## 2. Command Support

### `init`

**Invocation:** `kubebuilder init --plugins decoupled-go.example.com/v1 [flags]`

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--domain` | string | `example.com` | Domain for API groups |
| `--repo` | string | *(required)* | Go module path |
| `--project-name` | string | *(directory name)* | Project name |

**Files generated:**

| Path | Zone | Policy |
|---|---|---|
| `go.mod` | platform | always-overwrite |
| `cmd/main.go` | platform | always-overwrite |
| `internal/platform/doc.go` | platform | always-overwrite |
| `internal/platform/manager/manager.go` | platform | always-overwrite |
| `internal/app/doc.go` | app | create-only |
| `Makefile` | platform | always-overwrite |
| `Dockerfile` | platform | always-overwrite |
| `.gitignore` | platform | always-overwrite |
| `gen/kb/decoupled-go.v1/.kb-scaffold-lock.yaml` | platform | always-overwrite |

### `create api`

**Invocation:** `kubebuilder create api --group G --version V --kind K [flags]`

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--namespaced` | bool | `true` | Resource is namespace-scoped |
| `--force` | bool | `false` | Overwrite if resource already exists |

**Files generated:**

| Path | Zone | Policy |
|---|---|---|
| `api/<group>/<version>/<kind>_types.go` | platform | always-overwrite |
| `api/<group>/<version>/groupversion_info.go` | platform | always-overwrite |
| `internal/platform/reconciler/<kind>_reconciler.go` | platform | always-overwrite |
| `internal/platform/reconciler/hooks.go` | platform | always-overwrite |
| `internal/platform/reconciler/noop.go` | platform | always-overwrite |
| `internal/platform/testing/builder.go` | platform | always-overwrite |
| `internal/app/<kind>/hooks.go` | app | create-only |
| `internal/app/<kind>/hooks_test.go` | app | create-only |

### `create webhook`

**Invocation:** `kubebuilder create webhook --group G --version V --kind K [flags]`

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--defaulting` | bool | `false` | Scaffold defaulting webhook |
| `--validating` | bool | `false` | Scaffold validating webhook |
| `--programmatic-validation` | bool | `false` | Scaffold custom validator (CEL alternative) |

**Files generated:**

| Path | Zone | Policy |
|---|---|---|
| `internal/platform/webhook/<kind>_webhook.go` | platform | always-overwrite |
| `internal/platform/webhook/hooks.go` | platform | always-overwrite |
| `internal/app/<kind>/webhook_hooks.go` | app | create-only |
| `internal/app/<kind>/webhook_hooks_test.go` | app | create-only |

### `edit`

**Invocation:** `kubebuilder edit --plugins decoupled-go.example.com/v1 [flags]`

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--regenerate` | bool | `false` | Regenerate all platform-zone files |

When `--regenerate` is set: regenerates all `platform`-zone files, skips
all `app`-zone files, and updates the lock file. Emits migration warnings
if the plugin version has changed since the last generation.

---

## 3. Generated Project Layout

```
<project-root>/
├── api/
│   └── <group>/
│       └── <version>/
│           ├── <kind>_types.go           [platform] CRD type definitions
│           └── groupversion_info.go      [platform] scheme registration
├── cmd/
│   └── main.go                           [platform] manager entry point
├── internal/
│   ├── platform/                         [platform] PLUGIN-MANAGED — DO NOT EDIT
│   │   ├── doc.go
│   │   ├── manager/
│   │   │   └── manager.go                manager setup, scheme, RBAC
│   │   ├── reconciler/
│   │   │   ├── hooks.go                  ReconcileHooks interface
│   │   │   ├── noop.go                   NoopHooks default implementation
│   │   │   ├── <kind>_reconciler.go      reconcile skeleton
│   │   │   └── builder.go               test builder helpers
│   │   └── webhook/
│   │       ├── hooks.go                  ValidationHooks, DefaultingHooks
│   │       └── <kind>_webhook.go         webhook stubs
│   └── app/                              [app] USER-OWNED — NEVER OVERWRITTEN
│       ├── doc.go
│       └── <kind>/
│           ├── hooks.go                  implement ReconcileHooks here
│           ├── hooks_test.go             unit tests for hooks
│           ├── webhook_hooks.go          implement webhook hooks here
│           └── webhook_hooks_test.go
├── config/                               [kustomize/v2] — if chained
│   └── ...
├── gen/
│   └── kb/
│       └── decoupled-go.v1/
│           └── .kb-scaffold-lock.yaml    scaffold manifest
├── Dockerfile                            [platform]
├── Makefile                              [platform]
├── go.mod                                [platform]
└── .gitignore                            [platform]
```

---

## 4. Extension-Point Interfaces

### ReconcileHooks

```go
// Defined in: internal/platform/reconciler/hooks.go
type ReconcileHooks interface {
    BeforeReconcile(ctx context.Context, req reconcile.Request) error
    Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error)
    AfterReconcile(ctx context.Context, obj client.Object, result reconcile.Result) (reconcile.Result, error)
    OnError(ctx context.Context, obj client.Object, err error) (reconcile.Result, error)
}
```

### ValidationHooks

```go
// Defined in: internal/platform/webhook/hooks.go
type ValidationHooks interface {
    ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error)
    ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error)
    ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error)
}
```

### DefaultingHooks

```go
// Defined in: internal/platform/webhook/hooks.go
type DefaultingHooks interface {
    Default(ctx context.Context, obj runtime.Object) error
}
```

---

## 5. Chaining and Modularity

`decoupled-go/v1` is designed to chain with:

- `kustomize/v2` — for Kustomize manifest generation (recommended).
- `grafana/v1-alpha` — for dashboard scaffolding.
- `autoupdate/v1-alpha` — for GitHub Actions workflow.
- Any other plugin that follows chaining rules.

**Chaining rules respected by this plugin:**

1. Reads the full incoming universe and passes it through.
2. Writes only to paths under `internal/platform/`, `internal/app/`,
   `api/`, `cmd/`, and `gen/kb/decoupled-go.v1/`.
3. Adds plugin config only under `plugins.decoupled-go.example.com/v1`
   in the PROJECT file.
4. Does not modify `layout` — Kubebuilder manages this.

**Recommended chain (init):**

```bash
kubebuilder init \
    --plugins "decoupled-go.example.com/v1,kustomize.common.kubebuilder.io/v2"
```

---

## 6. Regeneration Guarantees and Overwrite Policy

| Zone | Policy | Triggered by |
|---|---|---|
| `platform` | Fully overwritten | Every scaffold command AND `edit --regenerate` |
| `app` | Created once, never overwritten | First scaffold command that generates the file |
| `bridge` | Marker-driven (future) | Regenerated only in marker regions |

### Determinism

Given the same inputs (flags, PROJECT config, universe), the plugin must
produce byte-for-byte identical output. This is required for:

- Meaningful `git diff` reviews.
- Reproducible builds.
- Safe CI checks that detect unintentional changes.

### Provenance Headers

Every platform-zone file begins with:

```go
// Code generated by decoupled-go/v1 ({{ .PluginVersion }}). DO NOT EDIT.
// Source: {{ .Command }}
// Regenerate: kubebuilder edit --plugins decoupled-go.example.com/v1 --regenerate
```

---

## 7. Scaffold Lock File Format

```yaml
# Scaffold lock — maintained by decoupled-go/v1. DO NOT EDIT.
pluginVersion: v1.0.0
pluginKey: decoupled-go.example.com/v1
generatedAt: "2024-01-15T10:30:00Z"
files:
  - path: internal/platform/reconciler/myapp_reconciler.go
    sha256: abc123...
    zone: platform
    policy: always-overwrite
    pluginVersionAtGeneration: v1.0.0
  - path: internal/app/myapp/hooks.go
    sha256: def456...
    zone: app
    policy: create-only
    pluginVersionAtGeneration: v1.0.0
```

---

## 8. Security Principles

1. **No shell injection** — generated Makefiles use `$(MAKE)` and explicit
   commands; no user-provided strings are interpolated into shell scripts
   without sanitization.
2. **Minimal RBAC** — generated markers request only the minimum permissions
   needed for each resource.
3. **Non-root container** — generated Dockerfile runs as a non-root user.
4. **No secrets in generated code** — generated code never embeds secrets,
   tokens, or credentials.
5. **Deterministic output** — no timestamps or random values in generated
   file contents (except the lock file's `generatedAt`, which is
   informational only).
6. **Dependency pinning** — generated `go.mod` pins all dependencies to
   tested versions.

---

## 9. Maintenance Principles

1. **Plugin source code is versioned separately** from generated project code.
2. **Breaking changes require a MAJOR version bump** with migration guide.
3. **Security fixes are released as PATCH versions** and regenerating picks
   them up immediately.
4. **The plugin itself is tested** with unit, integration, and E2E tests
   before every release.
5. **Generated projects are tested** using the testdata pattern — a
   sample project in `testdata/decoupled-project-v1/` is regenerated and
   tested on every CI run.
