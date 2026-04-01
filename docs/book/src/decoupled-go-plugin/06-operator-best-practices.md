# Chapter 6 — Operator Best Practices

This chapter covers the practices that every production-quality Kubernetes
operator should follow. The `decoupled-go/v1` plugin scaffolds code that
embodies these practices by default.

---

## 1. Reconciliation — Level-Triggered, Not Edge-Triggered

Kubernetes controllers are **level-triggered**: they react to the *current
state* of the cluster, not to individual events. Your reconciler must be
able to run multiple times for the same object and produce the same result.

**Wrong (edge-triggered thinking):**
```go
// Only creates the Deployment on "object created" events
func (h *MyHooks) Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error) {
    if obj.GetGeneration() == 1 {
        return h.createDeployment(ctx, obj)
    }
    return reconcile.Result{}, nil
}
```

**Correct (level-triggered):**
```go
// Always ensures the Deployment exists with the right spec
func (h *MyHooks) Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error) {
    myApp := obj.(*appsv1.MyApp)
    deploy := buildDeployment(myApp)

    existing := &appsv1.Deployment{}
    err := h.client.Get(ctx, client.ObjectKeyFromObject(deploy), existing)
    if apierrors.IsNotFound(err) {
        return reconcile.Result{}, h.client.Create(ctx, deploy)
    }
    if err != nil {
        return reconcile.Result{}, err
    }

    // Update if spec differs
    if !equality.Semantic.DeepEqual(existing.Spec, deploy.Spec) {
        existing.Spec = deploy.Spec
        return reconcile.Result{}, h.client.Update(ctx, existing)
    }

    return reconcile.Result{}, nil
}
```

---

## 2. Idempotency

Every reconcile operation must produce the same result when applied multiple
times to the same state. Use **create-or-update** patterns:

```go
import "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

// CreateOrUpdate is idempotent — safe to call repeatedly
op, err := controllerutil.CreateOrUpdate(ctx, h.client, deploy, func() error {
    deploy.Spec = buildDeploymentSpec(myApp)
    return controllerutil.SetControllerReference(myApp, deploy, h.scheme)
})
log.FromContext(ctx).Info("Deployment reconciled", "operation", op)
```

---

## 3. Status Conditions

Use the standard `metav1.Condition` type to report operator-managed status.
Never invent custom status fields for readiness — use conditions.

```go
// In your API type (api/<group>/<version>/<kind>_types.go)
type MyAppStatus struct {
    // +listType=map
    // +listMapKey=type
    // +patchStrategy=merge
    // +patchMergeKey=type
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

Update conditions atomically after each reconcile:

```go
func (h *MyHooks) AfterReconcile(
    ctx context.Context,
    obj client.Object,
    result reconcile.Result,
) (reconcile.Result, error) {
    myApp := obj.(*appsv1.MyApp)

    meta.SetStatusCondition(&myApp.Status.Conditions, metav1.Condition{
        Type:               "Ready",
        Status:             metav1.ConditionTrue,
        Reason:             "Reconciled",
        Message:            "MyApp is ready",
        ObservedGeneration: myApp.Generation,
    })

    if err := h.client.Status().Update(ctx, myApp); err != nil {
        return reconcile.Result{}, err
    }
    return result, nil
}
```

Always set `ObservedGeneration` so users can detect stale conditions.

---

## 4. Finalizers

Use finalizers when your operator creates **external resources** (cloud
resources, databases, etc.) that must be cleaned up before the Kubernetes
object is deleted.

```go
const myFinalizer = "myapp.example.com/cleanup"

func (h *MyHooks) BeforeReconcile(ctx context.Context, req reconcile.Request) error {
    // Add finalizer on first reconcile
    obj := &appsv1.MyApp{}
    if err := h.client.Get(ctx, req.NamespacedName, obj); err != nil {
        return client.IgnoreNotFound(err)
    }

    if obj.DeletionTimestamp.IsZero() {
        // Object is not being deleted; ensure finalizer is present
        if !controllerutil.ContainsFinalizer(obj, myFinalizer) {
            controllerutil.AddFinalizer(obj, myFinalizer)
            return h.client.Update(ctx, obj)
        }
    } else {
        // Object is being deleted; run cleanup
        if controllerutil.ContainsFinalizer(obj, myFinalizer) {
            if err := h.cleanup(ctx, obj); err != nil {
                return err
            }
            controllerutil.RemoveFinalizer(obj, myFinalizer)
            return h.client.Update(ctx, obj)
        }
    }
    return nil
}
```

**Important:** Never use finalizers for resources owned by the object
(those are cleaned up by the garbage collector via owner references).

---

## 5. Owner References

Set owner references on resources your operator creates so they are
garbage-collected automatically when the parent object is deleted:

```go
if err := controllerutil.SetControllerReference(myApp, deploy, h.scheme); err != nil {
    return reconcile.Result{}, err
}
```

Use `SetControllerReference` for resources that have exactly one owner.
Use `SetOwnerReference` for additional non-controller owners.

---

## 6. Patch vs Update

Prefer **patch** over **update** to avoid conflicts and reduce server load:

```go
// Update replaces the whole object — risky if another controller modifies it
h.client.Update(ctx, obj) // avoid for spec

// Patch sends only the diff — preferred
patch := client.MergeFrom(obj.DeepCopy())
obj.Spec.Replicas = pointer.Int32(3)
h.client.Patch(ctx, obj, patch)

// Status subresource must be updated separately
h.client.Status().Patch(ctx, obj, statusPatch)
```

Use `SSA` (Server-Side Apply) for complex multi-owner scenarios:

```go
apply := &appsv1.Deployment{...}
h.client.Patch(ctx, apply, client.Apply,
    client.ForceOwnership,
    client.FieldOwner("myapp-operator"))
```

---

## 7. Error Handling and Requeue

```go
func (h *MyHooks) OnError(
    ctx context.Context,
    obj client.Object,
    err error,
) (reconcile.Result, error) {
    log := log.FromContext(ctx)

    // Transient errors — requeue with backoff
    if isTransient(err) {
        log.Error(err, "Transient error, will retry")
        return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
    }

    // Permanent errors — update status, don't requeue
    log.Error(err, "Permanent error")
    h.setFailedCondition(ctx, obj, err)
    return reconcile.Result{}, nil  // don't return the error to avoid exponential backoff
}
```

**Requeue strategies:**

| Situation | Strategy |
|---|---|
| Transient error (network, timeout) | `RequeueAfter: backoff` |
| Waiting for external resource | `RequeueAfter: poll interval` |
| Permanent error | Update status, no requeue |
| Successful | `reconcile.Result{}` (no requeue unless needed) |

---

## 8. Security Best Practices

### RBAC Markers

Use `+kubebuilder:rbac` markers on your reconciler to declare minimum
permissions. Never request more permissions than needed:

```go
// +kubebuilder:rbac:groups=apps.example.com,resources=myapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.example.com,resources=myapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
```

### Secret Handling

- Never log secrets or sensitive data.
- Use `client.IgnoreNotFound` when fetching secrets to avoid leaking their
  existence in error messages.
- Prefer workload identity or projected service account tokens over
  long-lived secrets.

### Image Security

Generated `Dockerfile` should use:
- Distroless or minimal base image.
- Non-root user.
- Read-only root filesystem.
- No shell unless strictly necessary.

---

## 9. Observability

### Structured Logging

Use the `logr` logger from context. Follow Kubernetes logging conventions:

```go
log := log.FromContext(ctx)
// Values are key-value pairs
log.Info("Reconciling MyApp",
    "name", obj.GetName(),
    "namespace", obj.GetNamespace(),
    "generation", obj.GetGeneration(),
)
log.Error(err, "Failed to update Deployment", "deployment", deploy.Name)
```

**Convention:** Start messages with a capital letter, no trailing period.

### Metrics

The generated manager exposes Prometheus metrics via controller-runtime.
Add custom metrics in your app layer:

```go
var myAppReconcileTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "myapp_reconcile_total",
        Help: "Total number of MyApp reconciliations.",
    },
    []string{"namespace", "result"},
)

func init() {
    metrics.Registry.MustRegister(myAppReconcileTotal)
}
```

### Events

Record Kubernetes Events for significant state changes:

```go
h.recorder.Eventf(myApp, corev1.EventTypeNormal, "Reconciled",
    "Successfully reconciled MyApp %s", myApp.Name)
h.recorder.Eventf(myApp, corev1.EventTypeWarning, "ReconcileFailed",
    "Failed to reconcile MyApp %s: %v", myApp.Name, err)
```

---

## 10. Testing Best Practices

### Unit Tests (fast, no cluster)

Test hook logic with a fake client:

```go
func TestMyAppHooks_Reconcile(t *testing.T) {
    myApp := &appsv1.MyApp{...}
    fakeClient := fake.NewClientBuilder().WithObjects(myApp).Build()

    hooks := &MyAppHooks{client: fakeClient}
    result, err := hooks.Reconcile(context.Background(), myApp)

    assert.NoError(t, err)
    assert.Equal(t, reconcile.Result{}, result)

    // verify Deployment was created
    deploy := &appsv1.Deployment{}
    err = fakeClient.Get(context.Background(),
        types.NamespacedName{Name: "myapp", Namespace: "default"}, deploy)
    assert.NoError(t, err)
}
```

### Integration Tests (envtest)

Use `controller-runtime/envtest` for tests that need a real API server:

```go
var testEnv *envtest.Environment

func TestMain(m *testing.M) {
    testEnv = &envtest.Environment{
        CRDDirectoryPaths: []string{"../../config/crd/bases"},
    }
    cfg, _ := testEnv.Start()
    defer testEnv.Stop()

    // run tests
    os.Exit(m.Run())
}
```

### E2E Tests

Use the `test/e2e/utils.TestContext` pattern from the Kubebuilder repository
to run full operator lifecycle tests against a real cluster.

---

## ✅ Checkpoint

- [ ] Explain level-triggered reconciliation and why idempotency is required.
- [ ] When should you use finalizers? When should you not?
- [ ] Why is `Patch` preferred over `Update` for spec changes?
- [ ] What logging convention does Kubernetes use for structured logs?

## Next Steps

→ [Chapter 7: Implementation Guide](./07-implementation-guide.md)
