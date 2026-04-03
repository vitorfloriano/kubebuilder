# Chapter 1 — Kubebuilder Plugin Model

## What is a Kubebuilder Plugin?

A Kubebuilder plugin is a unit of scaffolding behaviour. When you run a
`kubebuilder` command (e.g. `init`, `create api`), Kubebuilder delegates
the actual file generation to one or more plugins that have been configured
for the project. Plugins are composable: you can chain them so that each
one contributes its own files and configuration without knowing the
internals of other plugins.

---

## Plugin Phases

Kubebuilder's plugin architecture evolved in three major phases. Understanding
the differences is important for deciding how to build and deploy your own
plugin.

### Phase 1 — Compiled-in Plugins

In Phase 1, plugins are Go packages that implement a set of interfaces and
are **compiled into the `kubebuilder` binary**. Examples: `go/v4`,
`kustomize/v2`, `grafana/v1-alpha`.

**Pros:** fast, no IPC overhead, easy to test.  
**Cons:** only Kubebuilder maintainers can ship them; updating requires a
new binary release.

### Phase 1.5 — Plugin Chaining

Phase 1.5 introduced the ability to **chain multiple plugins** for a single
command. When a command runs, each plugin in the chain receives the
accumulated state (the *universe*) built by previous plugins and may add
to or modify it. The chain is recorded in the `PROJECT` file's `layout`
field.

The key mental model:

```
kubebuilder init --plugins go/v4,kustomize/v2,myPlugin/v1
```

Execution order for `init`:
```
go/v4.Init → kustomize/v2.Init → myPlugin/v1.Init
            ↓ each sees the growing universe
```

**Key rule for plugin authors:** only touch files your plugin owns. Pass
the rest of the universe through unchanged.

### Phase 2 — External Plugins

Phase 2 allows plugins that live **outside the `kubebuilder` binary** as
standalone executables. Kubebuilder communicates with them via JSON over
`stdin`/`stdout`. They can be written in any language.

This is the phase that enables:
- Third-party plugins maintained by anyone.
- Plugins shipped as binaries, scripts, or containers.
- The `decoupled-go/v1` plugin described in this guide.

---

## Core Plugin Interfaces

Every compiled-in plugin implements at least the `plugin.Plugin` base
interface. Optional interfaces add support for specific subcommands.

```go
// plugin.Plugin — every plugin must implement this
type Plugin interface {
    Name() string
    Version() Version
    SupportedProjectVersions() []config.Version
}

// Optional subcommand interfaces
type HasInit interface {
    GetInitSubcommand() InitSubcommand
}
type HasCreateAPI interface {
    GetCreateAPISubcommand() CreateAPISubcommand
}
type HasCreateWebhook interface {
    GetCreateWebhookSubcommand() CreateWebhookSubcommand
}
type HasEdit interface {
    GetEditSubcommand() EditSubcommand
}

// plugin.Full combines all four
type Full interface {
    Plugin
    HasInit
    HasCreateAPI
    HasCreateWebhook
    HasEdit
}
```

For **external plugins** (Phase 2), these interfaces are implemented
implicitly: your executable is called with a JSON `PluginRequest` and must
return a JSON `PluginResponse`. See [Chapter 2](./02-external-plugin-lifecycle.md).

---

## Plugin Naming and Versioning

Plugin names follow the pattern:

```
<name>.<qualifier>/<version>
```

Examples:
- `go.kubebuilder.io/v4`
- `kustomize.common.kubebuilder.io/v2`
- `decoupled-go.example.com/v1`

**Rules:**
- The name must be a valid DNS label.
- The qualifier (domain part) prevents conflicts between unrelated plugin
  ecosystems.
- The version is semantic; breaking changes require a major version bump.

---

## The PROJECT File

When `kubebuilder init` runs, it creates a `PROJECT` file that records:

```yaml
domain: example.com
layout:
  - go.kubebuilder.io/v4
  - kustomize.common.kubebuilder.io/v2
  - decoupled-go.example.com/v1   # ← your plugin added itself here
projectName: my-operator
repo: github.com/example/my-operator
version: "3"
resources:
  - group: apps
    version: v1
    kind: MyApp
    api:
      crdVersion: v1
    controller: true
```

This file is passed to every subsequent plugin command as part of the
`PluginRequest.Config` field, allowing plugins to inspect what other
plugins have already set up.

---

## Plugin Chaining — Practical Implications

When designing `decoupled-go/v1`, keep these chaining rules in mind:

1. **Read the full `universe`** — previous plugins may have created files
   you depend on.
2. **Write only to your own paths** — do not overwrite files owned by
   other plugins.
3. **Append, don't replace, config keys** — add entries under
   `plugins.<your-key>` in the PROJECT config; don't touch other keys.
4. **Be additive** — your plugin's absence shouldn't break projects that
   don't use it.
5. **Fail loudly** — if required markers or files are missing, return a
   clear error rather than silently skipping.

---

## ✅ Checkpoint

After this chapter you should be able to answer:

- [ ] What is the difference between Phase 1 and Phase 2 plugins?
- [ ] What does plugin chaining mean?
- [ ] What does the `layout` field in the PROJECT file represent?
- [ ] What interface does a Phase 2 external plugin implicitly implement?

## Next Steps

→ [Chapter 2: External Plugin Lifecycle](./02-external-plugin-lifecycle.md)
