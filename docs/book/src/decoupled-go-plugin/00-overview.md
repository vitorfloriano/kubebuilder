# Decoupled Go Operator Plugin — Learning & Development Guide

> **Who this guide is for:** Go developers new to Kubebuilder plugin development
> who want to build (and learn from) a production-quality, upgrade-friendly
> external plugin for scaffolding Go-based Kubernetes operators.
>
> **What you will build:** `decoupled-go/v1` — a Kubebuilder external plugin that
> keeps generated framework code and user business code in **strictly separate
> directories**, so upgrading is as easy as bumping a version and re-running
> the scaffolder.

---

## Why This Guide Exists

The default `go/v4` plugin mixes generated scaffolding with user-written
business logic in the same files. As a result, every scaffold upgrade forces
manual merge work. The `decoupled-go/v1` plugin solves this by enforcing a
clear ownership boundary:

| Layer | Directory | Owned by | Regenerated? |
|---|---|---|---|
| **Platform layer** | `internal/platform/` | Plugin | Yes — fully replaceable |
| **App layer** | `internal/app/` | User | No — preserved forever |

Generated code calls user interfaces. Users never edit generated files.
Upgrades become: **bump plugin version → regenerate**.

---

## Learning Path

Work through the chapters in order. Each chapter ends with a **✅ Checkpoint**
and a list of **Next Steps**.

| # | Chapter | What you'll learn |
|---|---|---|
| 1 | [Kubebuilder Plugin Model](./01-kubebuilder-plugin-model.md) | Phases 1/1.5/2, plugin interfaces, chaining |
| 2 | [External Plugin Lifecycle](./02-external-plugin-lifecycle.md) | stdin/stdout protocol, PluginRequest/Response |
| 3 | [Decoupled Scaffold Philosophy](./03-decoupled-philosophy.md) | File ownership model, regeneration contract |
| 4 | [Design Patterns](./04-design-patterns.md) | Composition, adapter, strategy, template method |
| 5 | [Software Architecture](./05-software-architecture.md) | Layered arch, dependency rule, interface segregation |
| 6 | [Operator Best Practices](./06-operator-best-practices.md) | Reconciliation, idempotency, status, finalizers, security |
| 7 | [Implementation Guide](./07-implementation-guide.md) | Step-by-step build of `decoupled-go/v1` |
| 8 | [Upgrade & Migration Strategy](./08-upgrade-strategy.md) | Version policy, compatibility, migration workflow |

Specification and plan documents:

- [Plugin Specification — decoupled-go/v1](./spec/plugin-spec.md)
- [Implementation Plan & Milestones](./spec/implementation-plan.md)

---

## Quick Reference — Key Concepts

### Plugin Phases

- **Phase 1** — Plugin implements Go interfaces compiled into the binary.
- **Phase 1.5** — Plugins are chained; each sees the universe built by previous ones.
- **Phase 2** — Plugins run as external executables communicating via JSON on stdin/stdout.

`decoupled-go/v1` is a **Phase 2 external plugin**.

### File Ownership Zones

```
<project-root>/
├── internal/
│   ├── platform/          # ← PLUGIN-OWNED  (regenerate freely)
│   │   ├── manager/       #   Manager setup, scheme registration
│   │   └── reconciler/    #   Generated reconciler shell
│   └── app/               # ← USER-OWNED    (never overwritten)
│       └── <kind>/        #   Business logic, hooks, custom types
├── gen/kb/                # ← PLUGIN-OWNED  manifest + lock file
│   └── decoupled-go.v1/
│       └── .kb-scaffold-lock.yaml
└── ...
```

### The Core Contract

> **Generated files call user interfaces. User files implement them.**

The scaffolded reconciler calls:

```go
type ReconcileHooks interface {
    BeforeReconcile(ctx context.Context, req reconcile.Request) error
    Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error)
    AfterReconcile(ctx context.Context, obj client.Object, result reconcile.Result) error
    OnError(ctx context.Context, obj client.Object, err error) (reconcile.Result, error)
}
```

Users implement this interface in `internal/app/<kind>/hooks.go` — a
file the plugin creates **once** and never overwrites.

---

## Prerequisites

- Go 1.21+
- `kubebuilder` v4 binary installed
- Basic familiarity with Kubernetes controllers (you don't need to be an expert)
- Willingness to read some Go code

---

## Getting Help

- [Kubebuilder Book](https://book.kubebuilder.io)
- [External Plugins documentation](../plugins/extending/external-plugins.md)
- [Phase 2 Design document](../../../designs/extensible-cli-and-scaffolding-plugins-phase-2.md)
- [controller-runtime documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
