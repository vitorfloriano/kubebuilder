# Chapter 5 — Software Architecture

## Layered Architecture

`decoupled-go/v1` applies a **layered architecture** to the generated
project. The key principle: **layers communicate downward only**. A higher
layer may call a lower layer, but never the reverse.

```
┌─────────────────────────────────────────────┐
│  cmd/main.go   (entry point — wires layers) │  ← user-owned
├─────────────────────────────────────────────┤
│  internal/app/   (app/domain layer)         │  ← user-owned
│  - hook implementations                     │
│  - domain types and helpers                 │
├─────────────────────────────────────────────┤
│  internal/platform/  (platform/infra layer) │  ← plugin-owned
│  - reconciler shells                        │
│  - webhook stubs                            │
│  - manager setup                            │
│  - hook interfaces (public API)             │
├─────────────────────────────────────────────┤
│  Kubernetes API / controller-runtime        │  ← external library
└─────────────────────────────────────────────┘
```

The **app layer** imports from the **platform layer** (to implement
interfaces) but the **platform layer** never imports from the **app layer**
(it calls the interface, not the concrete type).

---

## The Dependency Rule

The dependency rule (from Clean Architecture) states:

> Source code dependencies must point **inward** — toward higher-level policy,
> not toward lower-level details.

In this context:

- **App layer** = high-level policy (business rules).
- **Platform layer** = lower-level detail (Kubernetes plumbing).
- **Kubernetes API** = the lowest-level detail.

Imports flow:

```
cmd → app → platform → controller-runtime → k8s.io/api
```

**Never:**

```
platform → app   ← this would break the regeneration contract
```

This single rule is why regenerating the platform layer is safe: it has no
knowledge of what specific app code exists.

---

## Module Boundaries

### The Plugin's Perspective

When the plugin runs, it sees the project as a set of files in a `universe`
map. It operates on this map:

```
Plugin input:  universe (map[string]string) + request args
Plugin output: updated universe
```

The plugin is itself a layered program:

```
┌──────────────────────────────────┐
│  main.go  (CLI entry point)      │
├──────────────────────────────────┤
│  commands/  (init, api, etc.)    │
├──────────────────────────────────┤
│  generator/ (template engine)    │
├──────────────────────────────────┤
│  templates/ (Go text/template)   │
└──────────────────────────────────┘
```

### The Generated Project's Perspective

```
cmd/
  main.go              ← user-owned entry point (wires platform + app)

internal/
  platform/            ← plugin-owned (generated)
    manager/
      manager.go       ← sets up ctrl.Manager, registers schemes
    reconciler/
      <kind>_reconciler.go  ← reconcile skeleton, calls hooks
      hooks.go              ← defines ReconcileHooks interface
    webhook/
      <kind>_webhook.go     ← webhook stubs, calls hooks
      hooks.go              ← defines ValidationHooks, DefaultingHooks
    testing/
      builder.go            ← test helpers
  app/                 ← user-owned (never overwritten)
    <kind>/
      hooks.go         ← user implements ReconcileHooks
      webhook_hooks.go ← user implements ValidationHooks, DefaultingHooks

gen/kb/decoupled-go.v1/
  .kb-scaffold-lock.yaml  ← scaffold manifest
```

---

## SOLID Principles in Practice

### Single Responsibility Principle (SRP)

Each file has exactly one reason to change:

- `reconciler.go` changes only when the reconcile algorithm skeleton changes.
- `hooks.go` changes only when the hook interface contract changes.
- User's `hooks.go` changes only when business logic changes.

### Open/Closed Principle (OCP)

The platform layer is **closed for modification** (users shouldn't edit it)
but **open for extension** (users extend it by implementing interfaces).

### Liskov Substitution Principle (LSP)

Any type that implements `ReconcileHooks` must be substitutable for any
other. The generated code must not rely on concrete types, only on the
interface contract.

### Interface Segregation Principle (ISP)

Keep interfaces small and focused:

```go
// Good: focused interfaces
type BeforeReconciler interface {
    BeforeReconcile(ctx context.Context, req reconcile.Request) error
}

type MainReconciler interface {
    Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error)
}

// ReconcileHooks composes focused interfaces
type ReconcileHooks interface {
    BeforeReconciler
    MainReconciler
    AfterReconciler
    ErrorHandler
}
```

Users who need only `BeforeReconciler` for a specific adapter can implement
only that.

### Dependency Inversion Principle (DIP)

High-level modules (platform reconciler) depend on **abstractions**
(interfaces), not concretions. Low-level modules (user hooks) implement
those abstractions.

```go
// Platform depends on abstraction
type Reconciler struct {
    hooks ReconcileHooks  // ← interface (abstraction)
}

// User provides concretion
type MyAppHooks struct{}
func (h *MyAppHooks) Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error) {
    // concrete business logic
}
```

---

## Hexagonal Architecture Angle

You can also think of the generated project in hexagonal architecture terms:

- **Core (hexagon):** user's `internal/app/` — pure business logic.
- **Ports (interfaces):** `ReconcileHooks`, `ValidationHooks` — defined in platform layer.
- **Adapters (generated):** platform reconciler, webhook stubs — connect Kubernetes events to ports.
- **External (Kubernetes API, databases, etc.):** called from adapters or app layer.

The plugin generates the adapters. The user fills in the core.

---

## Observability Architecture

The generated manager sets up structured logging, metrics, and tracing
infrastructure. The platform layer passes these to hooks via the context:

```go
// generated — passes observability to user code via context
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)
    log.Info("Starting reconciliation", "name", req.Name, "namespace", req.Namespace)

    // ... call hooks
}
```

User code retrieves them from the context:

```go
// user-owned hooks
func (h *MyAppHooks) Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error) {
    log := log.FromContext(ctx)  // structured logger, already configured
    log.Info("Reconciling MyApp", "generation", obj.GetGeneration())
    // ...
}
```

---

## Testing Architecture

The generated project follows a three-layer test strategy:

| Layer | Location | What it tests | Speed |
|---|---|---|---|
| Unit | `internal/app/<kind>/*_test.go` | Hook logic in isolation | Fast |
| Integration | `internal/platform/*_test.go` | Reconciler + fake client | Medium |
| E2E | `test/e2e/` | Full operator in a cluster | Slow |

The `internal/platform/testing/builder.go` provides helpers that make
unit-testing user hook implementations straightforward without needing a
running cluster.

---

## ✅ Checkpoint

- [ ] Describe the four-layer architecture of the generated project.
- [ ] Explain the dependency rule and why `platform` must not import `app`.
- [ ] How does the Dependency Inversion Principle appear in the reconciler?
- [ ] What is the hexagonal architecture and how does it map to the plugin structure?

## Next Steps

→ [Chapter 6: Operator Best Practices](./06-operator-best-practices.md)
