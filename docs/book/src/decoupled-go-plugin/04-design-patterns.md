# Chapter 4 — Design Patterns

This chapter explains the software design patterns used in `decoupled-go/v1`
and why each was chosen. Understanding these patterns will help you build,
extend, and maintain the plugin confidently.

---

## Pattern 1: Template Method

### What it is

The Template Method pattern defines the **skeleton of an algorithm** in a base
type (or generated code), and lets subclasses (or user implementations) fill in
the specific steps.

### Where it appears

The generated reconciler is a template method:

```go
// Generated (platform layer) — defines the algorithm skeleton
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Step 1: Load object (fixed — always done)
    obj := &appsv1.MyApp{}
    if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Step 2: Before hook (user-defined step)
    if err := r.hooks.BeforeReconcile(ctx, req); err != nil {
        return r.hooks.OnError(ctx, obj, err)
    }

    // Step 3: Main reconcile (user-defined step)
    result, err := r.hooks.Reconcile(ctx, obj)
    if err != nil {
        return r.hooks.OnError(ctx, obj, err)
    }

    // Step 4: After hook (user-defined step)
    return r.hooks.AfterReconcile(ctx, obj, result)
}
```

The user fills in steps 2, 3, and 4 by implementing the `ReconcileHooks`
interface — without touching the skeleton.

### Why it matters

- The upgrade path for the skeleton is clear: update step 1 (e.g., add
  pagination support) without affecting user-defined steps.
- Users can never break the skeleton; they can only affect their own steps.

---

## Pattern 2: Strategy

### What it is

The Strategy pattern defines a **family of algorithms**, encapsulates each one,
and makes them interchangeable. The context calls the strategy via an interface,
not a concrete type.

### Where it appears

`ReconcileHooks`, `ValidationHooks`, and `DefaultingHooks` are all strategy
interfaces:

```go
// ReconcileHooks is the strategy interface for reconciliation logic
type ReconcileHooks interface {
    BeforeReconcile(ctx context.Context, req reconcile.Request) error
    Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error)
    AfterReconcile(ctx context.Context, obj client.Object, result reconcile.Result) error
    OnError(ctx context.Context, obj client.Object, err error) (reconcile.Result, error)
}
```

The platform-layer reconciler selects the strategy at construction time:

```go
func NewReconciler(client client.Client, hooks ReconcileHooks) *Reconciler {
    return &Reconciler{client: client, hooks: hooks}
}
```

Different users can provide completely different reconcile strategies without
modifying the generated scaffolding.

### Why it matters

- Users can swap strategies in tests (e.g., a `NoopHooks` for unit tests).
- Multiple operators can reuse the same platform scaffold with different app logic.

---

## Pattern 3: Composition over Inheritance

### What it is

Rather than inheriting behaviour from a base class, objects are **composed** of
smaller, focused components.

### Where it appears

The platform reconciler composes several concerns:

```go
// Platform-layer reconciler — composed, not inherited
type Reconciler struct {
    client   client.Client
    scheme   *runtime.Scheme
    recorder record.EventRecorder
    hooks    ReconcileHooks      // user strategy
    logger   logr.Logger
}
```

The user's `hooks.go` also composes:

```go
// User-owned hooks — app layer
type MyAppHooks struct {
    client client.Client
    db     *mydb.Client           // user's custom dependency
    cache  *mycache.Cache         // user's custom dependency
}
```

There is no base class. Dependencies are injected at construction time.

### Why it matters

- Easy to test each component in isolation.
- Adding a new concern (e.g., feature flags) means adding a field,
  not modifying a class hierarchy.
- No fragile base class problem.

---

## Pattern 4: Adapter

### What it is

An Adapter wraps an existing type to implement a different interface, making
incompatible types work together.

### Where it appears

Imagine a user who has an existing domain service `MyDomainService` with its
own method signatures. They can write a thin adapter to make it implement
`ReconcileHooks` without modifying the domain service:

```go
// Adapter (user-owned, in internal/app/<kind>/)
type domainServiceAdapter struct {
    svc *MyDomainService
}

func (a *domainServiceAdapter) BeforeReconcile(ctx context.Context, req reconcile.Request) error {
    return nil // no-op, not needed
}

func (a *domainServiceAdapter) Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error) {
    myObj := obj.(*appsv1.MyApp)
    return a.svc.ProcessMyApp(ctx, myObj)
}

// ... implement remaining methods
```

The scaffold calls the `ReconcileHooks` interface; the user adapts their
existing code to fit.

### Why it matters

- Users aren't forced to redesign their domain layer to fit the plugin's
  interface shape.
- Adapters are cheap to write and easy to test.

---

## Pattern 5: Registry (for modularity)

### What it is

A Registry maintains a collection of named components and allows callers to
look them up by key. This is the mechanism that makes the plugin **modular**.

### Where it appears

When a user wants to add multiple reconcile extensions (e.g., a logging
module and a metrics module), they register them:

```go
// internal/platform/manager/setup.go (generated)
func Setup(mgr ctrl.Manager, registry *hooks.Registry) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&appsv1.MyApp{}).
        Complete(platform.NewReconciler(mgr.GetClient(), registry.Hooks("myapp")))
}
```

```go
// internal/app/setup.go (user-owned, created once)
func RegisterHooks(registry *hooks.Registry) {
    registry.Register("myapp", &myapp.MyAppHooks{
        // inject user dependencies here
    })
}
```

The platform layer doesn't know about specific implementations — it works
through the registry.

### Why it matters

- Users can add or remove extension modules without touching generated code.
- Multiple resources can be wired independently through the same registry.
- Testing a specific hook in isolation doesn't require wiring the whole manager.

---

## Pattern 6: Builder (for test setup)

### What it is

The Builder pattern constructs complex objects step by step, providing a
fluent API.

### Where it appears

The plugin can generate a test builder for easy controller testing:

```go
// internal/platform/testing/builder.go (generated)
type ControllerTestBuilder struct {
    scheme *runtime.Scheme
    hooks  ReconcileHooks
}

func NewControllerTest() *ControllerTestBuilder {
    return &ControllerTestBuilder{
        scheme: runtime.NewScheme(),
        hooks:  &NoopHooks{},
    }
}

func (b *ControllerTestBuilder) WithHooks(h ReconcileHooks) *ControllerTestBuilder {
    b.hooks = h
    return b
}

func (b *ControllerTestBuilder) Build(t *testing.T) (*Reconciler, client.Client) {
    // ... set up envtest, return reconciler and client
}
```

### Why it matters

- Tests are readable, composable, and don't repeat setup code.
- Generated builders make it easy to test user hook implementations.

---

## Pattern Summary

| Pattern | Used in | Why |
|---|---|---|
| Template Method | Platform reconciler skeleton | Stable upgrade path for fixed steps |
| Strategy | `ReconcileHooks`, `ValidationHooks` | Swap user logic without touching scaffold |
| Composition | Reconciler, user hooks structs | Testable, no fragile base class |
| Adapter | User code wrapping existing services | Integrate existing code with plugin interfaces |
| Registry | Hook registration in `main`/`setup` | Modular, multi-resource, order-independent |
| Builder | Test setup helpers | Readable, reusable test setup |

---

## ✅ Checkpoint

- [ ] Explain the Template Method pattern and where it appears in the reconciler.
- [ ] What is the Strategy pattern and why are hook interfaces a strategy?
- [ ] Why is composition preferred over inheritance in Go?
- [ ] How does the Registry pattern make the plugin modular?

## Next Steps

→ [Chapter 5: Software Architecture](./05-software-architecture.md)
