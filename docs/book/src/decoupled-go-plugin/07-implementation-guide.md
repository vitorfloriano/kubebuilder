# Chapter 7 — Implementation Guide

This chapter walks you through building `decoupled-go/v1` step by step.
By the end you will have a working external plugin that scaffolds a
decoupled Go operator project.

---

## Prerequisites

- Go 1.21+ installed.
- `kubebuilder` v4 binary in your `$PATH`.
- A dedicated Git repository for your plugin (e.g. `github.com/myorg/decoupled-go-plugin`).

---

## Step 1: Bootstrap the Plugin Repository

```bash
mkdir decoupled-go-plugin && cd decoupled-go-plugin
go mod init github.com/myorg/decoupled-go-plugin
```

Create the top-level directory layout:

```
decoupled-go-plugin/
├── cmd/
│   └── main.go          ← plugin entry point
├── internal/
│   ├── command/         ← one file per subcommand
│   │   ├── init.go
│   │   ├── api.go
│   │   ├── webhook.go
│   │   └── edit.go
│   ├── generator/       ← template rendering engine
│   │   └── generator.go
│   └── lock/            ← lock file read/write
│       └── lockfile.go
├── templates/           ← Go text/template files
│   ├── platform/
│   └── app/
└── go.mod
```

Add the Kubebuilder dependency:

```bash
go get sigs.k8s.io/kubebuilder/v4@latest
```

---

## Step 2: Implement the Entry Point

`cmd/main.go` is the binary's entry point. It reads the `PluginRequest`
from stdin, dispatches to the correct command handler, and writes the
`PluginResponse` to stdout.

```go
// cmd/main.go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"

    "github.com/myorg/decoupled-go-plugin/internal/command"
)

func main() {
    var req external.PluginRequest
    if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
        writeError(req, fmt.Sprintf("decode request: %v", err))
        os.Exit(1)
    }

    resp := external.PluginResponse{
        APIVersion: req.APIVersion,
        Command:    req.Command,
        Universe:   cloneUniverse(req.Universe),
    }

    var err error
    switch req.Command {
    case "init":
        err = command.Init(&req, &resp)
    case "create api":
        err = command.CreateAPI(&req, &resp)
    case "create webhook":
        err = command.CreateWebhook(&req, &resp)
    case "edit":
        err = command.Edit(&req, &resp)
    case "flags":
        err = command.Flags(&req, &resp)
    case "metadata":
        err = command.Metadata(&req, &resp)
    // unknown commands: pass universe through unchanged
    }

    if err != nil {
        resp.Error = true
        resp.ErrorMsgs = []string{err.Error()}
    }

    if encErr := json.NewEncoder(os.Stdout).Encode(resp); encErr != nil {
        fmt.Fprintf(os.Stderr, "encode response: %v\n", encErr)
        os.Exit(1)
    }
}

func cloneUniverse(u map[string]string) map[string]string {
    out := make(map[string]string, len(u))
    for k, v := range u {
        out[k] = v
    }
    return out
}

func writeError(req external.PluginRequest, msg string) {
    resp := external.PluginResponse{
        APIVersion: req.APIVersion,
        Command:    req.Command,
        Universe:   req.Universe,
        Error:      true,
        ErrorMsgs:  []string{msg},
    }
    _ = json.NewEncoder(os.Stdout).Encode(resp)
}
```

---

## Step 3: Implement the `flags` and `metadata` Commands

```go
// internal/command/flags.go
package command

import "sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"

var initFlags = []external.Flag{
    {Name: "domain", Type: "string", Default: "example.com",
        Usage: "Domain for your operator's API groups"},
    {Name: "repo", Type: "string", Default: "",
        Usage: "Go module path for the generated project"},
    {Name: "project-name", Type: "string", Default: "",
        Usage: "Name of the project (defaults to current directory name)"},
}

var apiFlags = []external.Flag{
    {Name: "namespaced", Type: "bool", Default: "true",
        Usage: "Resource is namespace-scoped (false = cluster-scoped)"},
    {Name: "force", Type: "bool", Default: "false",
        Usage: "Attempt to create resource even if it already exists"},
}

func Flags(req *external.PluginRequest, resp *external.PluginResponse) error {
    // req.Args contains ["--init"], ["--create api"], etc.
    for _, arg := range req.Args {
        switch arg {
        case "--init":
            resp.Flags = initFlags
            return nil
        case "--create api":
            resp.Flags = apiFlags
            return nil
        }
    }
    return nil
}
```

```go
// internal/command/metadata.go
package command

import "sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"

var descriptions = map[string]external.SubcommandMetadata{
    "init": {
        Description: `Initialize a new decoupled Go operator project.

The decoupled-go/v1 plugin scaffolds a project with a strict platform/app
separation: generated framework code lives in internal/platform/ and your
business logic lives in internal/app/. Upgrades are as simple as bumping
the plugin version and re-running kubebuilder edit --regenerate.`,
        Examples: `  # Initialize a new project
  kubebuilder init --plugins decoupled-go.example.com/v1 \
    --domain my.org --repo github.com/myorg/my-operator`,
    },
    "create api": {
        Description: `Scaffold a new API (CRD + controller) with decoupled business logic hooks.`,
        Examples: `  kubebuilder create api --group apps --version v1 --kind MyApp`,
    },
}

func Metadata(req *external.PluginRequest, resp *external.PluginResponse) error {
    for _, arg := range req.Args {
        switch arg {
        case "--init":
            resp.Metadata = descriptions["init"]
        case "--create api":
            resp.Metadata = descriptions["create api"]
        }
    }
    return nil
}
```

---

## Step 4: Implement the `init` Command

The `init` command creates the base project structure:

```go
// internal/command/init.go
package command

import (
    "fmt"
    "path/filepath"

    "sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"

    "github.com/myorg/decoupled-go-plugin/internal/generator"
    "github.com/myorg/decoupled-go-plugin/internal/lock"
)

func Init(req *external.PluginRequest, resp *external.PluginResponse) error {
    // Parse flags from req.Args
    domain := flagValue(req.Args, "--domain", "example.com")
    repo := flagValue(req.Args, "--repo", "")
    if repo == "" {
        return fmt.Errorf("--repo is required for init")
    }

    data := generator.InitData{
        Domain: domain,
        Repo:   repo,
    }

    // Generate platform files (always overwrite)
    files, err := generator.RenderInit(data)
    if err != nil {
        return fmt.Errorf("render init templates: %w", err)
    }

    for path, content := range files {
        resp.Universe[path] = content
    }

    // Write lock file
    lf := lock.NewLockFile("v1.0.0", "decoupled-go.example.com/v1")
    for path := range files {
        lf.AddFile(path, content, lock.ZonePlatform, lock.PolicyAlwaysOverwrite)
    }
    lockContent, err := lf.Marshal()
    if err != nil {
        return fmt.Errorf("marshal lock file: %w", err)
    }
    resp.Universe[filepath.Join("gen", "kb", "decoupled-go.v1", ".kb-scaffold-lock.yaml")] = lockContent

    return nil
}

// flagValue extracts the value of a named flag from an args slice.
func flagValue(args []string, flag, defaultVal string) string {
    for i, arg := range args {
        if arg == flag && i+1 < len(args) {
            return args[i+1]
        }
    }
    return defaultVal
}
```

---

## Step 5: Implement the Template Generator

```go
// internal/generator/generator.go
package generator

import (
    "bytes"
    "text/template"
)

// InitData holds the data passed to init templates.
type InitData struct {
    Domain string
    Repo   string
}

// RenderInit renders the init templates and returns a universe fragment.
func RenderInit(data InitData) (map[string]string, error) {
    return renderTemplates(initTemplates, data)
}

func renderTemplates(templates map[string]string, data any) (map[string]string, error) {
    out := make(map[string]string, len(templates))
    for path, tmplStr := range templates {
        tmpl, err := template.New(path).Parse(tmplStr)
        if err != nil {
            return nil, err
        }
        var buf bytes.Buffer
        if err = tmpl.Execute(&buf, data); err != nil {
            return nil, err
        }
        out[path] = buf.String()
    }
    return out, nil
}
```

---

## Step 6: Define Init Templates

```go
// internal/generator/init_templates.go
package generator

var initTemplates = map[string]string{
    "internal/platform/doc.go": platformDocGo,
    "internal/app/doc.go":      appDocGo,
    "go.mod":                   goModTmpl,
    "Makefile":                 makefileTmpl,
}

const platformDocGo = `// Code generated by decoupled-go/v1. DO NOT EDIT.
// Regenerate: kubebuilder edit --plugins decoupled-go.example.com/v1 --regenerate
package platform
`

const appDocGo = `// Package app contains user-owned business logic.
// This file is created once by decoupled-go/v1 and never overwritten.
package app
`

const goModTmpl = `module {{ .Repo }}

go 1.21

require (
    sigs.k8s.io/controller-runtime v0.18.0
)
`

const makefileTmpl = `# Generated by decoupled-go/v1. DO NOT EDIT.
.PHONY: generate manifests build test

generate:
	go generate ./...

manifests:
	controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

build:
	go build ./cmd/...

test:
	go test ./... -coverprofile cover.out
`
```

---

## Step 7: Implement the `create api` Command

The `create api` command generates the platform reconciler (overwrite) and
the app hooks stub (create-only):

```go
// internal/command/api.go
package command

import (
    "fmt"

    "sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"

    "github.com/myorg/decoupled-go-plugin/internal/generator"
)

func CreateAPI(req *external.PluginRequest, resp *external.PluginResponse) error {
    group   := flagValue(req.Args, "--group", "")
    version := flagValue(req.Args, "--version", "")
    kind    := flagValue(req.Args, "--kind", "")

    if group == "" || version == "" || kind == "" {
        return fmt.Errorf("--group, --version, and --kind are required")
    }

    data := generator.APIData{
        Group:   group,
        Version: version,
        Kind:    kind,
    }

    // Platform files — always overwrite
    platformFiles, err := generator.RenderAPIPlatform(data)
    if err != nil {
        return fmt.Errorf("render platform API templates: %w", err)
    }
    for path, content := range platformFiles {
        resp.Universe[path] = content
    }

    // App stub files — create only if not already in universe
    appFiles, err := generator.RenderAPIApp(data)
    if err != nil {
        return fmt.Errorf("render app API templates: %w", err)
    }
    for path, content := range appFiles {
        if _, exists := resp.Universe[path]; !exists {
            resp.Universe[path] = content
        }
        // if exists: user owns it — skip
    }

    return nil
}
```

---

## Step 8: Define API Templates

```go
// internal/generator/api_templates.go
package generator

import "strings"

// APIData holds the data passed to API templates.
type APIData struct {
    Group   string
    Version string
    Kind    string
}

// KindLower returns the lowercase Kind name.
func (d APIData) KindLower() string { return strings.ToLower(d.Kind) }

// RenderAPIPlatform renders platform-layer API templates.
func RenderAPIPlatform(data APIData) (map[string]string, error) {
    return renderTemplates(platformAPITemplates, data)
}

// RenderAPIApp renders app-layer stub templates (create-only).
func RenderAPIApp(data APIData) (map[string]string, error) {
    return renderTemplates(appAPITemplates, data)
}

var platformAPITemplates = map[string]string{
    "internal/platform/reconciler/{{ .KindLower }}_reconciler.go": platformReconcilerTmpl,
    "internal/platform/reconciler/hooks.go":                        platformHooksTmpl,
}

var appAPITemplates = map[string]string{
    "internal/app/{{ .KindLower }}/hooks.go": appHooksTmpl,
}

const platformHooksTmpl = `// Code generated by decoupled-go/v1. DO NOT EDIT.
package reconciler

import (
    "context"

    "sigs.k8s.io/controller-runtime/pkg/reconcile"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileHooks defines the extension points for {{ .Kind }} reconciliation.
// Implement this interface in internal/app/{{ .KindLower }}/hooks.go.
type ReconcileHooks interface {
    // BeforeReconcile is called before the main reconcile loop.
    BeforeReconcile(ctx context.Context, req reconcile.Request) error

    // Reconcile contains the main business logic for {{ .Kind }}.
    Reconcile(ctx context.Context, obj client.Object) (reconcile.Result, error)

    // AfterReconcile is called after a successful reconcile.
    AfterReconcile(ctx context.Context, obj client.Object, result reconcile.Result) (reconcile.Result, error)

    // OnError is called when BeforeReconcile or Reconcile returns an error.
    OnError(ctx context.Context, obj client.Object, err error) (reconcile.Result, error)
}
`

const platformReconcilerTmpl = `// Code generated by decoupled-go/v1. DO NOT EDIT.
package reconciler

import (
    "context"

    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"
    ctrl "sigs.k8s.io/controller-runtime"
)

// {{ .Kind }}Reconciler reconciles a {{ .Kind }} object.
// DO NOT EDIT — regenerated by decoupled-go/v1.
// Implement ReconcileHooks in internal/app/{{ .KindLower }}/hooks.go.
type {{ .Kind }}Reconciler struct {
    client client.Client
    hooks  ReconcileHooks
}

// New{{ .Kind }}Reconciler creates a new reconciler wired to the given hooks.
func New{{ .Kind }}Reconciler(c client.Client, hooks ReconcileHooks) *{{ .Kind }}Reconciler {
    return &{{ .Kind }}Reconciler{client: c, hooks: hooks}
}

// Reconcile is the main reconciliation loop.
func (r *{{ .Kind }}Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)
    log.Info("Reconciling {{ .Kind }}", "name", req.Name, "namespace", req.Namespace)

    if err := r.hooks.BeforeReconcile(ctx, req); err != nil {
        return r.hooks.OnError(ctx, nil, err)
    }

    var obj client.Object // TODO: replace with concrete type from api/<group>/<version>
    if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    result, err := r.hooks.Reconcile(ctx, obj)
    if err != nil {
        return r.hooks.OnError(ctx, obj, err)
    }

    return r.hooks.AfterReconcile(ctx, obj, result)
}

// SetupWithManager registers the reconciler with the controller manager.
func (r *{{ .Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        Complete(r)
}
`

const appHooksTmpl = `// Package {{ .KindLower }} contains user-owned business logic for {{ .Kind }}.
// This file was created by decoupled-go/v1 and will NOT be overwritten.
// Implement the ReconcileHooks interface below.
package {{ .KindLower }}

import (
    "context"

    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Hooks implements platform/reconciler.ReconcileHooks for {{ .Kind }}.
type Hooks struct {
    Client client.Client
    // Add your dependencies here (e.g. database clients, external services).
}

// BeforeReconcile is called before the main reconcile step.
// Use this for pre-flight checks, leader-election guards, etc.
func (h *Hooks) BeforeReconcile(_ context.Context, _ reconcile.Request) error {
    return nil
}

// Reconcile contains your main business logic for {{ .Kind }}.
// obj is the current {{ .Kind }} object fetched from the API server.
func (h *Hooks) Reconcile(_ context.Context, _ client.Object) (reconcile.Result, error) {
    // TODO: implement your business logic here
    return reconcile.Result{}, nil
}

// AfterReconcile is called after a successful reconcile.
// Use this to update status conditions, emit events, etc.
func (h *Hooks) AfterReconcile(_ context.Context, _ client.Object, result reconcile.Result) (reconcile.Result, error) {
    return result, nil
}

// OnError is called when BeforeReconcile or Reconcile returns an error.
// Use this to update status conditions and decide whether to requeue.
func (h *Hooks) OnError(_ context.Context, _ client.Object, err error) (reconcile.Result, error) {
    return reconcile.Result{}, err
}
`
```

---

## Step 9: Implement the Lock File

```go
// internal/lock/lockfile.go
package lock

import (
    "crypto/sha256"
    "fmt"
    "time"

    "sigs.k8s.io/yaml"
)

const (
    ZonePlatform = "platform"
    ZoneApp      = "app"
    PolicyAlwaysOverwrite = "always-overwrite"
    PolicyCreateOnly      = "create-only"
)

// LockFile tracks plugin-managed files and their ownership.
type LockFile struct {
    PluginVersion string     `yaml:"pluginVersion"`
    PluginKey     string     `yaml:"pluginKey"`
    GeneratedAt   string     `yaml:"generatedAt"`
    Files         []FileEntry `yaml:"files"`
}

// FileEntry describes a single managed file.
type FileEntry struct {
    Path    string `yaml:"path"`
    SHA256  string `yaml:"sha256"`
    Zone    string `yaml:"zone"`
    Policy  string `yaml:"policy"`
}

// NewLockFile creates a new lock file for the given plugin version and key.
func NewLockFile(version, key string) *LockFile {
    return &LockFile{
        PluginVersion: version,
        PluginKey:     key,
        GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
    }
}

// AddFile registers a file in the lock file.
func (lf *LockFile) AddFile(path, content, zone, policy string) {
    h := sha256.Sum256([]byte(content))
    lf.Files = append(lf.Files, FileEntry{
        Path:   path,
        SHA256: fmt.Sprintf("%x", h),
        Zone:   zone,
        Policy: policy,
    })
}

// Marshal serialises the lock file to YAML.
func (lf *LockFile) Marshal() (string, error) {
    header := "# Scaffold lock — maintained by decoupled-go/v1. DO NOT EDIT.\n"
    b, err := yaml.Marshal(lf)
    if err != nil {
        return "", err
    }
    return header + string(b), nil
}
```

---

## Step 10: Build and Install the Plugin

```bash
# Build the binary
go build -o decoupled-go-plugin ./cmd/

# Install it where Kubebuilder can discover it
PLUGIN_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/kubebuilder/plugins/decoupled-go.example.com/v1"
mkdir -p "$PLUGIN_DIR"
cp decoupled-go-plugin "$PLUGIN_DIR/"
```

---

## Step 11: Use Your Plugin

```bash
mkdir my-operator && cd my-operator
kubebuilder init \
    --plugins decoupled-go.example.com/v1 \
    --domain my.org \
    --repo github.com/myorg/my-operator

kubebuilder create api \
    --plugins decoupled-go.example.com/v1 \
    --group apps --version v1 --kind MyApp
```

Expected file tree after `create api`:

```
internal/
  platform/
    reconciler/
      myapp_reconciler.go   ← generated, DO NOT EDIT
      hooks.go              ← generated, DO NOT EDIT
  app/
    myapp/
      hooks.go              ← yours — implement ReconcileHooks here
gen/kb/decoupled-go.v1/
  .kb-scaffold-lock.yaml
```

Edit `internal/app/myapp/hooks.go` to add your business logic. Run
`kubebuilder edit --plugins decoupled-go.example.com/v1 --regenerate` at
any time to refresh the platform layer without losing your app code.

---

## ✅ Checkpoint

- [ ] You have a working plugin binary that handles `init`, `create api`, `flags`, and `metadata`.
- [ ] Platform files are overwritten on every run; app stubs are created once.
- [ ] The lock file is generated and tracks file zones and policies.
- [ ] You can install the binary and use it with Kubebuilder CLI.

## Next Steps

→ [Chapter 8: Upgrade & Migration Strategy](./08-upgrade-strategy.md)
