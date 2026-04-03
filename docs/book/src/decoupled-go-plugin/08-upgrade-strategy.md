# Chapter 8 — Upgrade & Migration Strategy

## Design Principle: Upgrades Are Free

The entire point of `decoupled-go/v1` is to make upgrades **cheap and
predictable**. This chapter explains how that promise is kept in practice.

---

## Version Policy

### Plugin Version vs Scaffold Version

The plugin has two version dimensions:

| Version | What it tracks | Who bumps it |
|---|---|---|
| **Plugin version** (semver) | The plugin binary itself | Plugin maintainer |
| **Scaffold schema version** | The template output format | Plugin maintainer |

These are often the same, but can differ. A bug-fix release (patch bump)
may produce identical scaffold output. A minor or major bump may change
template layout.

### Semantic Versioning Rules

Follow standard semver (`MAJOR.MINOR.PATCH`):

| Change type | Bump |
|---|---|
| Adding a new optional hook method | MINOR |
| Adding a new file to platform layer | MINOR |
| Removing or renaming a hook method | MAJOR |
| Changing hook method signatures | MAJOR |
| Changing the lock file format | MAJOR |
| Bug fixes in generated code | PATCH |
| Security fixes | PATCH (or MINOR if new required field) |

A MAJOR version bump means users may need to update their app layer code.
The plugin must emit **migration warnings** that list what changed and what
the user needs to do.

---

## The Upgrade Workflow

### Step 1: Update the plugin binary

```bash
# If installed via Go toolchain
go install github.com/myorg/decoupled-go-plugin/cmd@v1.2.0

# Copy to Kubebuilder plugin discovery path
cp $(go env GOBIN)/decoupled-go-plugin \
   "${XDG_DATA_HOME:-$HOME/.local/share}/kubebuilder/plugins/decoupled-go.example.com/v1/"
```

### Step 2: Regenerate the platform layer

```bash
kubebuilder edit \
    --plugins decoupled-go.example.com/v1 \
    --regenerate
```

Behind the scenes, the plugin:
1. Reads the existing `.kb-scaffold-lock.yaml`.
2. Compares the old plugin version with the new one.
3. If MAJOR or MINOR bump: emits migration warnings to stderr.
4. Regenerates all `platform`-zone files (overwrites).
5. Skips all `app`-zone files.
6. Updates the lock file.

### Step 3: Review diffs

```bash
git diff internal/platform/
```

The diff shows only generated code changes. Your business logic in
`internal/app/` is untouched.

### Step 4: Handle migration warnings (if any)

If the plugin printed migration warnings, review them and update your
app layer accordingly. Migration warnings look like:

```
[decoupled-go/v1] MIGRATION REQUIRED
  ReconcileHooks interface changed in v1.2.0:
  + AfterReconcile now returns (reconcile.Result, error) instead of error
  
  Update files:
  - internal/app/myapp/hooks.go: update AfterReconcile signature
```

---

## Interface Stability Contract

The plugin makes the following guarantees about the hook interfaces:

### Stable (no breaking changes within a MAJOR version)

- Method signatures of all `*Hooks` interfaces.
- File paths for `app`-zone files.
- Lock file format (minor additions are backward compatible).

### May change between MINOR versions (backward compatible)

- New optional methods added to interfaces (with default no-op provided).
- New files added to platform layer.
- New flags available.

### Will change on MAJOR version bump

- Removed or renamed hook methods.
- Changed method signatures.
- Restructured file paths.

---

## Handling Interface Evolution

### Adding a new optional hook method (MINOR)

When adding a new method to `ReconcileHooks`, provide a `NoopHooks` adapter
that implements the old interface so existing user implementations continue
to compile:

```go
// internal/platform/reconciler/noop.go (generated)
// NoopHooks implements ReconcileHooks with no-op defaults.
// Embed this in your Hooks struct to satisfy new optional methods.
type NoopHooks struct{}

func (NoopHooks) BeforeReconcile(_ context.Context, _ reconcile.Request) error { return nil }
func (NoopHooks) Reconcile(_ context.Context, _ client.Object) (reconcile.Result, error) {
    return reconcile.Result{}, nil
}
func (NoopHooks) AfterReconcile(_ context.Context, _ client.Object, r reconcile.Result) (reconcile.Result, error) {
    return r, nil
}
func (NoopHooks) OnError(_ context.Context, _ client.Object, err error) (reconcile.Result, error) {
    return reconcile.Result{}, err
}
// NEW in v1.2.0:
func (NoopHooks) OnDelete(_ context.Context, _ client.Object) error { return nil }
```

Users who want the default behaviour can embed `NoopHooks`:

```go
// internal/app/myapp/hooks.go (user-owned)
type MyAppHooks struct {
    reconciler.NoopHooks  // ← embed for defaults; override only what you need
    client client.Client
}

func (h *MyAppHooks) Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error) {
    // my custom logic
}
```

### Breaking change (MAJOR)

1. Bump MAJOR version of the plugin.
2. Generate a detailed `MIGRATION.md` document in the plugin repository.
3. Print migration warnings during `--regenerate` that link to the guide.

---

## Compatibility Policy Summary

| Guarantee | Scope |
|---|---|
| `app`-zone files are never overwritten | Always |
| Platform-zone files are fully regenerated | On every scaffold command |
| Interface signature stability | Within a MAJOR version |
| Lock file backward compatibility | Within a MAJOR version |
| Migration warnings on MAJOR/MINOR bump | Always |

---

## Multi-Version Support

When users have a multi-group or multi-resource project with different
resources scaffolded at different plugin versions, the lock file tracks
per-file versions:

```yaml
pluginVersion: v1.2.0
pluginKey: decoupled-go.example.com/v1
generatedAt: "2024-06-01T12:00:00Z"
files:
  - path: internal/platform/reconciler/myapp_reconciler.go
    sha256: abc...
    zone: platform
    policy: always-overwrite
    pluginVersionAtGeneration: v1.0.0   # when this file was last generated
  - path: internal/app/myapp/hooks.go
    sha256: def...
    zone: app
    policy: create-only
    pluginVersionAtGeneration: v1.0.0
```

The plugin can detect that a specific file is stale (generated by an older
version) and warn accordingly.

---

## Security and Maintenance

### Security Fix Releases

Security fixes are released as PATCH versions. The generated platform layer
picks up fixes immediately on the next regeneration. Users never need to
manually patch generated code — that is the entire point.

### Dependency Updates

The plugin's Go module dependencies (especially `controller-runtime` and
`k8s.io/api`) are updated on every minor release. The generated `go.mod`
reflects the latest compatible versions tested by the plugin.

### Deprecation Policy

When a feature or hook is deprecated:
1. Emit a deprecation warning during scaffold.
2. Keep the deprecated API for at least one MAJOR version.
3. Document the migration path in `MIGRATION.md`.

---

## ✅ Checkpoint

- [ ] Explain the two-step upgrade workflow: update binary, regenerate.
- [ ] What is the difference between MINOR and MAJOR bump from the user's perspective?
- [ ] How does `NoopHooks` enable backward-compatible interface evolution?
- [ ] What does the lock file's `pluginVersionAtGeneration` field track?

## Congratulations!

You have completed the learning path. You now understand:

- ✅ Kubebuilder plugin phases and chaining
- ✅ External plugin protocol (PluginRequest/PluginResponse)
- ✅ Decoupled file ownership model
- ✅ Design patterns used in the plugin
- ✅ Software architecture principles
- ✅ Operator development best practices
- ✅ Step-by-step implementation
- ✅ Upgrade and migration strategy

**Continue to:**
- [Plugin Specification](./spec/plugin-spec.md) — the full `decoupled-go/v1` spec
- [Implementation Plan](./spec/implementation-plan.md) — milestones and testing strategy
- The bootstrap skeleton in `pkg/plugins/decoupled-go/v1/` in this repository
