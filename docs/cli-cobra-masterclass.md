# Kubebuilder CLI Cobra Masterclass

> **Audience:** New contributors onboarding to Kubebuilder's CLI layer.
> **Goal:** Understand how Kubebuilder's plugin-based CLI is wired together, identify the
> antipatterns hiding in it, and ship three focused pull-requests that make it measurably
> better — using Cobra's native features rather than working around them.

---

## Table of Contents

1. [Why This Matters](#1-why-this-matters)
2. [Cobra Concepts You Need to Know](#2-cobra-concepts-you-need-to-know)
3. [How Kubebuilder's CLI Is Structured](#3-how-kubebuilders-cli-is-structured)
4. [Antipatterns and Issues in the Current CLI](#4-antipatterns-and-issues-in-the-current-cli)
5. [The 3-PR Execution Plan](#5-the-3-pr-execution-plan)
   - [PR 1 — UX: Groups, Help, Examples, Suggestion Tuning](#pr-1--ux-groups-help-examples-suggestion-tuning)
   - [PR 2 — Validation & Completion](#pr-2--validation--completion)
   - [PR 3 — Plugin Safety, Determinism & Architecture](#pr-3--plugin-safety-determinism--architecture)
6. [General Commit & Review Guidance](#6-general-commit--review-guidance)
7. [Open-Ended Exercises](#7-open-ended-exercises)

---

## 1. Why This Matters

Kubebuilder is *the* scaffolding tool for Kubernetes operators. Every new operator
author's first experience with Kubernetes custom controllers starts here. A confusing
CLI is a barrier to the entire ecosystem.

As a contributor, you have two responsibilities:

1. **Keep the existing code working** — operators teams depend on CLI stability.
2. **Pay down UX debt** — but surgically, using the tools Cobra already provides
   rather than reinventing them.

The three PRs in this guide are designed to be merged independently. Each one is
self-contained, reviewable, and verifiable without the others.

---

## 2. Cobra Concepts You Need to Know

This section covers the Cobra APIs we will use. Treat each sub-section as a building
block; the PR plan later will reference them by name.

### 2.1 Command Tree and Metadata Fields

Every `cobra.Command` has several metadata fields. Use them precisely:

| Field | Purpose | Example |
|---|---|---|
| `Use` | Syntax string shown in help | `"init [flags]"` |
| `Short` | One-line description (shown in lists) | `"Initialize a new project"` |
| `Long` | Full description shown on `cmd --help` | Multi-line prose |
| `Example` | Shown under `Examples:` in help | Real shell invocations |
| `Aliases` | Alternative names for the command | `[]string{"initialise"}` |
| `Deprecated` | Non-empty string marks command as deprecated; message shown | `"use 'init' instead"` |
| `Hidden` | Hides from help/completion but still executable | `true` |
| `Annotations` | Arbitrary key/value pairs (used for grouping, etc.) | — |

```go
cmd := &cobra.Command{
    Use:   "init [flags]",
    Short: "Initialize a new project",
    Long: `Initialize a new project in the current directory.

This command scaffolds the project structure and writes a PROJECT file
that records your plugin chain and domain.`,
    Example: `  # Initialize with the default plugin bundle
  kubebuilder init --domain example.com --repo github.com/example/myop

  # Initialize with an optional Helm plugin layered on top
  kubebuilder init --domain example.com --plugins=helm/v2-alpha`,
}
```

### 2.2 `RunE` vs `Run`

Always prefer `RunE` over `Run` in application code.

```go
// ❌ BAD — error is silently swallowed by the caller
cmd.Run = func(cmd *cobra.Command, args []string) {
    if err := doWork(); err != nil {
        fmt.Println(err)  // you must handle it yourself, inconsistently
    }
}

// ✅ GOOD — error propagates to cobra; SilenceUsage / SilenceErrors apply
cmd.RunE = func(cmd *cobra.Command, args []string) error {
    return doWork()
}
```

**Kubebuilder's current gap:** `pkg/cli/init.go` uses bare `Run` for the init command
base definition. This means any early-return error in that anonymous function is lost.

### 2.3 Args Validators

Cobra has built-in validators for positional arguments. Use them instead of
manual `len(args)` checks inside `RunE`.

```go
// Reject any positional args
cmd.Args = cobra.NoArgs

// Require exactly N
cmd.Args = cobra.ExactArgs(2)

// Require at least N
cmd.Args = cobra.MinimumNArgs(1)

// Custom validator (e.g., reject unrecognised resource names)
cmd.Args = func(cmd *cobra.Command, args []string) error {
    for _, a := range args {
        if !isValidKindName(a) {
            return fmt.Errorf("invalid kind name %q: must start with uppercase letter", a)
        }
    }
    return nil
}
```

Using `cobra.NoArgs` on commands like `create api` and `create webhook` immediately
rejects typos such as `kubebuilder create api MyKind` before any file writes begin.

### 2.4 Persistent Flags and `PersistentPreRunE`

*Persistent* flags are inherited by all subcommands. `PersistentPreRunE` runs before
`PreRunE` for every command in the subtree — the ideal place for cross-cutting checks.

```go
// On the root command
root.PersistentFlags().StringSlice("plugins", nil, "plugin keys to use")

// Root-level hook that fires for EVERY subcommand
root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    // Validate the environment once, before any subcommand-specific logic
    if err := checkGoModulePresence(cmd); err != nil {
        return fmt.Errorf("%w\n\nHow to fix:\n  go mod init <module-path>", err)
    }
    return nil
}
```

> ⚠️ **Caveat:** If a *child* command defines its own `PersistentPreRunE`, it **replaces**
> (does not chain) the parent's hook by default. To chain them you must call the parent
> explicitly. Cobra v1.8+ adds `EnableTraverseRunHooks` to restore chaining, but
> requires opt-in. Design your hook hierarchy with this in mind.

### 2.5 Command Groups (`AddGroup` / `GroupID`)

Available since Cobra v1.6, groups let you organize commands in the help output under
labelled sections.

```go
root.AddGroup(
    &cobra.Group{ID: "setup",        Title: "Project Setup"},
    &cobra.Group{ID: "scaffold",     Title: "Scaffolding"},
    &cobra.Group{ID: "experimental", Title: "Experimental (use at your own risk)"},
)

initCmd.GroupID   = "setup"
createCmd.GroupID = "scaffold"
editCmd.GroupID   = "setup"
alphaCmd.GroupID  = "experimental"
```

Without groups, all subcommands appear in a flat list. With groups, a first-time user
immediately sees "Project Setup" vs "Scaffolding" and knows which to run first.

### 2.6 Help and Usage Template Customisation

Cobra lets you override the entire help template:

```go
root.SetHelpTemplate(`{{with .Long}}{{. | trimRightSpace}}

{{end}}Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

{{GroupsString .}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)
```

You can also add a custom help function for more dynamic content:

```go
root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
    fmt.Fprintln(cmd.OutOrStdout(), "=== Kubebuilder Quick Start ===")
    fmt.Fprintln(cmd.OutOrStdout(), "  1. kubebuilder init --domain example.com")
    fmt.Fprintln(cmd.OutOrStdout(), "  2. kubebuilder create api --group batch --version v1 --kind Job")
    fmt.Fprintln(cmd.OutOrStdout(), "  3. make build && make test")
    fmt.Fprintln(cmd.OutOrStdout())
    cmd.Usage()
})
```

### 2.7 Shell Completion

Cobra generates shell completion scripts automatically. Three hooks give you control:

#### 2.7.1 `ValidArgsFunction` — dynamic positional completion

```go
cmd.ValidArgsFunction = func(
    cmd *cobra.Command, args []string, toComplete string,
) ([]string, cobra.ShellCompDirective) {
    // Return candidates + a directive
    return []string{"Foo\tThe Foo kind", "Bar\tThe Bar kind"},
        cobra.ShellCompDirectiveNoFileComp
}
```

#### 2.7.2 `RegisterFlagCompletionFunc` — dynamic flag value completion

```go
_ = cmd.RegisterFlagCompletionFunc("plugins",
    func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        return []string{
            "go.kubebuilder.io/v4\tDefault Go+Kustomize bundle",
            "helm.kubebuilder.io/v2-alpha\tHelm chart generation",
            "grafana.kubebuilder.io/v1-alpha\tGrafana dashboard generation",
        }, cobra.ShellCompDirectiveNoFileComp
    },
)
```

#### 2.7.3 Shell completion directives

| Directive | Meaning |
|---|---|
| `ShellCompDirectiveDefault` | Standard file completion as fallback |
| `ShellCompDirectiveNoFileComp` | No file completion (use when values are enumerable) |
| `ShellCompDirectiveNoSpace` | No trailing space after completion (for partial tokens) |
| `ShellCompDirectiveFilterFileExt` | Complete only files with given extensions |
| `ShellCompDirectiveError` | Completion failed; do not show results |

### 2.8 Suggestions and Aliases

```go
// Tune how close a typo must be before a suggestion fires (default is 2)
root.SuggestionsMinimumDistance = 2

// Teach cobra about common alternative spellings
initCmd.Aliases = []string{"initialize", "new"}
```

### 2.9 Deprecation and Hiding

```go
// Deprecated: still runs but prints a warning + message
oldCmd.Deprecated = "use 'kubebuilder alpha generate' instead"

// Hidden: does not appear in help or completion but still works
internalCmd.Hidden = true
```

### 2.10 Error Handling Patterns

```go
// SilenceUsage prevents cobra from dumping usage on every RunE error.
// Set it on the root so all subcommands inherit the behaviour.
root.SilenceUsage = true

// SilenceErrors prevents cobra from printing the error twice
// (once by your code, once by cobra itself).
root.SilenceErrors = true
```

With both set you own the full error-printing experience. The caller in `cmd.go` then
logs via `slog.Error` before `os.Exit(1)`, which matches Kubebuilder's structured
logging strategy.

---

## 3. How Kubebuilder's CLI Is Structured

### 3.1 Package Map

```
main.go
  └── internal/cli/cmd/cmd.go   (Run function — wire everything here)
        ├── pkg/cli/             (core CLI package — command construction)
        │     ├── cli.go         CLI struct + New() + buildCmd() + Run()
        │     ├── root.go        newRootCmd(), plugin table helpers
        │     ├── init.go        newInitCmd()
        │     ├── create.go      newCreateCmd()
        │     ├── api.go         newCreateAPICmd()
        │     ├── webhook.go     newCreateWebhookCmd()
        │     ├── edit.go        newEditCmd()
        │     ├── alpha.go       newAlphaCmd(), addAlphaCmd()
        │     ├── completion.go  newCompletionCmd()
        │     ├── version.go     newVersionCmd()
        │     ├── options.go     CLI functional options (WithPlugins, etc.)
        │     └── cmd_helpers.go plugin hook chain (PreRunE, RunE, PostRunE)
        ├── pkg/plugin/          plugin interfaces, bundle, resolution, filter
        │     ├── plugin.go      Plugin, Init, CreateAPI, CreateWebhook, Edit interfaces
        │     ├── bundle.go      Bundle (composite plugin)
        │     ├── filter.go      FilterPluginsByKey, FilterPluginsByProjectVersion
        │     └── errors.go      typed errors (ExitError)
        └── pkg/plugins/         concrete plugin implementations
              ├── golang/v4/      go/v4 scaffolding
              ├── common/kustomize/v2/  kustomize/v2 scaffolding
              ├── external/       exec-based external plugin bridge
              └── optional/       helm, grafana, autoupdate, …
```

### 3.2 Lifecycle of a `kubebuilder init` Invocation

Understanding the full call chain helps you place validation at the right layer.

```
main()
  cmd.Run()
    cli.New(options...)
      newCLI()            — apply functional options, populate c.plugins map
      buildCmd()
        newRootCmd()      — build root cobra.Command
        parseRootFlags()  — parse --plugins, --project-version from os.Args
        resolvePlugins()  — match plugin keys to registered plugins (see §4.2)
        addSubcommands()  — build child commands (init, create, edit, alpha, …)
          newInitCmd()
            filterSubcommands()   — find plugins that implement plugin.Init
            applySubcommandHooks()
              initializationHooks()   — UpdateMetadata, BindFlags (merged)
              factory.preRunEFunc()   — load/create config, InjectConfig, PreScaffold
              factory.runEFunc()      — Scaffold (file writes happen HERE)
              factory.postRunEFunc()  — Save config, PostScaffold
    c.Run()
      cmd.Execute()
```

Key insight: **file writes happen inside `RunE`** (via `Scaffold`), but project
configuration is loaded/created inside `PreRunE`. Any validation you add in `PreRunE`
runs *before* any file is touched — that is the right place for all input checking.

### 3.3 How Plugins Are Bundled and Resolved

A *plugin* implements one or more of: `plugin.Init`, `plugin.CreateAPI`,
`plugin.CreateWebhook`, `plugin.Edit`.

A *bundle* (`plugin.Bundle`) wraps multiple plugins under a single key. The default
bundle `go.kubebuilder.io/v4` composes `go/v4` and `kustomize/v2`:

```go
// internal/cli/cmd/cmd.go
gov4Bundle, _ := plugin.NewBundleWithOptions(
    plugin.WithName(golang.DefaultNameQualifier),
    plugin.WithVersion(plugin.Version{Number: 4}),
    plugin.WithPlugins(kustomizecommonv2.Plugin{}, golangv4.Plugin{}),
    plugin.WithDescription("Default scaffold (go/v4 + kustomize/v2)"),
)
```

During `resolvePlugins()` in `cli.go`, the CLI iterates over `c.pluginKeys`
(parsed from `--plugins` or the PROJECT file) and calls `FilterPluginsByKey` for each.
If exactly one plugin matches, it is added to `c.resolvedPlugins`. If zero or more than
one match, an error is returned.

The resolved plugins are then iterated in order inside `filterSubcommands()` and
`applySubcommandHooks()` to build the hook chain that runs on every scaffold command.

---

## 4. Antipatterns and Issues in the Current CLI

The following issues were identified by reading the actual source. Each includes a
severity rating, the relevant file, and whether a Cobra native feature can fix it.

### 4.1 Global `alphaCommands` Slice (Global State / Package-Level Initialisation Risk)

**File:** `pkg/cli/alpha.go`

```go
// ❌ ANTIPATTERN — mutable package-level variable, initialised at package load time
var alphaCommands = []*cobra.Command{
    newAlphaCommand(),          // called at package load — not inside main()
    alpha.NewScaffoldCommand(),
    alpha.NewUpdateCommand(),
}
```

**Why it's a problem:**
- Package-level var is populated once at startup; tests that create multiple `CLI`
  instances share and mutate the same slice.
- `newAlphaCommand()` is called when the package is first imported (package-level var
  initialisation), before `main()` runs — any error in that constructor fires before
  the CLI can handle it gracefully.
- The slice is not protected by a mutex; tests running in parallel could race.

**Cobra-native fix:** None required for the API itself, but the pattern should change:
construct `alphaCommands` inside `newAlphaCmd()` from injected dependencies, not as a
package-level variable. Use `cli.WithExtraAlphaCommands(...)` for optional additions.

```go
// ✅ BETTER — build the list inside the factory method
func (c *CLI) newAlphaCmd() *cobra.Command {
    cmd := &cobra.Command{Use: "alpha", ...}
    cmd.AddCommand(alpha.NewScaffoldCommand())
    cmd.AddCommand(alpha.NewUpdateCommand())
    for _, extra := range c.extraAlphaCommands {
        cmd.AddCommand(extra)
    }
    return cmd
}
```

**Severity:** Medium — affects test reliability and startup error handling.

---

### 4.2 Ambiguous Plugin Resolution Without Conflict Detection

**File:** `pkg/cli/cli.go` (`resolvePlugins`)

```go
switch len(plugins) {
case 1:
    c.resolvedPlugins = append(c.resolvedPlugins, plugins[0])
case 0:
    return fmt.Errorf("no plugin could be resolved with key %q%s", pluginKey, extraErrMsg)
default:
    return fmt.Errorf("ambiguous plugin %q%s", pluginKey, extraErrMsg)
}
```

The `default` branch is correct but the error message gives no hint about *which*
plugins matched. A user who has an external plugin with a short key that partially
matches a built-in plugin gets a confusing "ambiguous" error with no remediation hint.

**Cobra-native partial fix:** `SuggestionsMinimumDistance` and `SuggestFor` cannot fix
this directly, but a well-formatted error with actionable guidance reduces friction
significantly. Additionally, `RegisterFlagCompletionFunc` on `--plugins` can enumerate
valid keys, preventing the ambiguity from occurring in the first place.

**Full fix direction:**

```go
default:
    matchedKeys := make([]string, 0, len(plugins))
    for _, p := range plugins {
        matchedKeys = append(matchedKeys, plugin.KeyFor(p))
    }
    slices.Sort(matchedKeys) // deterministic output
    return fmt.Errorf(
        "plugin key %q matches multiple plugins: %s\n\nUse a more specific key, e.g. one of:\n  %s",
        pluginKey, extraErrMsg,
        strings.Join(matchedKeys, "\n  "),
    )
```

**Severity:** High — silent ambiguity produces non-deterministic behaviour.

---

### 4.3 Late Validation — Resource Validation After Config Creation

**File:** `pkg/cli/cmd_helpers.go` (`preRunEFunc`)

```go
// Project configuration is CREATED here (file written to disk)
if err := factory.store.New(factory.projectVersion); err != nil { ... }

// ... and only then do we validate the resource
if err := res.Validate(); err != nil {
    return fmt.Errorf("%s: created invalid resource: %w", factory.errorMessage, err)
}
```

If `res.Validate()` fails, the PROJECT file has already been created on disk. The user
must manually delete it before retrying. This violates the **fail-fast before writes**
principle.

**Cobra-native fix:** Move resource/flag validation into a dedicated `PreRunE`
that runs *before* any filesystem operations. Cobra's `PreRunE` runs before `RunE` and
can return an error that aborts the command without any side effects.

```go
// ✅ validate options before touching the filesystem
cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
    if err := options.validate(); err != nil {
        return fmt.Errorf("invalid flags: %w\n\nRun '%s --help' for usage", err, cmd.CommandPath())
    }
    return nil
}
cmd.RunE = func(cmd *cobra.Command, args []string) error {
    // Only now create the config and scaffold
    return scaffold(options)
}
```

**Severity:** High — leaves partial state on failure.

---

### 4.4 `init.go` Uses `Run` Instead of `RunE`

**File:** `pkg/cli/init.go`

```go
cmd := &cobra.Command{
    ...
    Run: func(_ *cobra.Command, _ []string) {},  // ❌ bare Run, errors discarded
}
```

The init command's `Run` is later overwritten by `applySubcommandHooks` (which sets
`PreRunE` / `RunE` / `PostRunE`), so in practice no real error is silently swallowed
here. However, the pattern is misleading and inconsistent. If someone adds logic to
this `Run` callback in the future they will be surprised that errors are swallowed.

**Cobra-native fix:** Replace `Run` with `RunE` and return `nil` or a sentinel.

```go
Run: func(_ *cobra.Command, _ []string) {},
// →
RunE: func(_ *cobra.Command, _ []string) error { return nil },
```

**Severity:** Low — latent confusion risk.

---

### 4.5 External Plugin Trust Model Is Implicit

**File:** `pkg/plugins/external/plugin.go`, `pkg/plugins/external/helpers.go`

External plugins are discovered from the filesystem and executed as arbitrary binaries.
There is no explicit:
- allowlist / denylist mechanism
- binary ownership check (e.g., same UID as the kubebuilder process)
- warning to the user that an external binary is about to be executed
- path canonicalization (symlink resolution before exec)

A malicious plugin placed in the search path could be silently invoked with the same
privileges as the user running kubebuilder, and would receive the full project config
over stdin.

**Cobra-native partial fix:** None — this is an execution-boundary issue, not a
Cobra issue. However, using `PersistentPreRunE` on the root command to print a clear
warning before any external plugin executes gives users a chance to notice unexpected
plugins.

**Full fix direction (PR 3):**

```go
// Before exec-ing an external plugin, verify ownership and warn
func (p *ExternalPlugin) verifyTrust(execPath string) error {
    info, err := os.Stat(execPath)
    if err != nil {
        return fmt.Errorf("external plugin not found: %w", err)
    }
    if info.Mode()&0o022 != 0 {
        return fmt.Errorf(
            "external plugin %q is world- or group-writable; refusing to execute.\n"+
                "Fix: chmod o-w,g-w %s", execPath, execPath,
        )
    }
    return nil
}
```

**Severity:** High (security) — arbitrary code execution path.

---

### 4.6 Inconsistent Error UX (Mixed Remediation Hints)

Across the codebase, error messages vary between:
- `"failed to initialize project: failed to find configuration file, project must be initialized"`
- `"no resolved plugin, please verify the project version and plugins specified in flags or configuration file"`
- `"ambiguous plugin %q%s"` (no guidance)

Some errors include a "how to fix" hint; others do not. A user encountering an error
should always see:
1. What went wrong.
2. Why it went wrong (if non-obvious).
3. One concrete command to try next.

**Cobra-native fix:** `SilenceUsage` + `SilenceErrors` on root (so cobra doesn't print
usage on every RunE failure), paired with a consistent error format applied in
`PersistentPreRunE` or in a wrapper function.

```go
// Package-level helper — use everywhere in pkg/cli
func cliError(err error, hint string) error {
    if hint == "" {
        return err
    }
    return fmt.Errorf("%w\n\nHow to fix:\n  %s", err, hint)
}
```

**Severity:** Medium — affects learnability.

---

### 4.7 No Shell Completion for `--plugins` or Resource Flags

`--plugins` accepts a comma-separated list of plugin keys but provides no completion.
The user must know the exact key format (`go.kubebuilder.io/v4`) to avoid an
"ambiguous plugin" error. Similarly, `--group`, `--version`, and `--kind` on
`create api` / `create webhook` accept arbitrary strings with no completion.

**Cobra-native fix:** `RegisterFlagCompletionFunc` — the canonical solution.

**Severity:** Medium — reduces discoverability; forces doc lookups.

---

### 4.8 `create` Command Has No Guided `RunE`

**File:** `pkg/cli/create.go`

`create` is purely a namespace command. When invoked alone it shows help, but there is
no `RunE` that produces a decision guide. A user running `kubebuilder create` gets a
generic help page rather than a prompt like:
> "Did you mean `create api` or `create webhook`? See `kubebuilder create --help`."

**Cobra-native fix:** Add a `RunE` that returns a descriptive error with examples.

```go
RunE: func(cmd *cobra.Command, args []string) error {
    _ = cmd.Help()
    return fmt.Errorf(
        "choose a subcommand:\n" +
            "  kubebuilder create api     — scaffold a CRD + controller\n" +
            "  kubebuilder create webhook — scaffold an admission webhook",
    )
},
```

**Severity:** Low — minor UX friction.

---

### 4.9 `alpha` Commands Not Grouped as Experimental

**File:** `pkg/cli/alpha.go`

The `alpha` command is added to the root but without a `GroupID`. It appears in the
same flat list as stable commands like `init` and `create`. New users have no signal
that alpha commands carry a stability disclaimer.

**Cobra-native fix:** `AddGroup` + `GroupID = "experimental"`.

**Severity:** Low — discoverability/trust issue.

---

### 4.10 `SuggestionsMinimumDistance` Not Configured

**File:** `pkg/cli/root.go` (`newRootCmd`)

Cobra defaults to a Levenshtein distance of 2 for "did you mean?" suggestions. This is
fine for short commands but worth making explicit so contributors know it is a conscious
choice, and so it can be tuned per-command.

**Cobra-native fix:** Explicitly set it:

```go
cmd.SuggestionsMinimumDistance = 2
```

**Severity:** Low — documentation/intent clarity.

---

## 5. The 3-PR Execution Plan

Each PR is independent. They are ordered by risk (lowest first) so you can merge them
incrementally without destabilizing the CLI.

---

### PR 1 — UX: Groups, Help, Examples, Suggestion Tuning

**Goal:** Make `kubebuilder --help` tell a first-time user exactly what to do next,
organize commands into labelled sections, and improve typo recovery — all with zero
changes to plugin logic or scaffolding.

#### Checklist

- [ ] Add `cobra.Group` definitions to `newRootCmd()` in `pkg/cli/root.go`
- [ ] Assign `GroupID` to `initCmd`, `createCmd`, `editCmd`, `alphaCmd`, `completionCmd`
- [ ] Set `root.SuggestionsMinimumDistance = 2` explicitly with a comment
- [ ] Set `root.SilenceUsage = true` and `root.SilenceErrors = true` on root
- [ ] Add a "Quick Start" `Long` description to the root command
- [ ] Replace the existing `rootExamples()` output with grouped, annotated examples
- [ ] Expand `Long` / `Example` for `create`, `init`, `api`, `webhook`, `edit`
- [ ] Add `RunE` guidance to `create` command (§4.8)
- [ ] Move `alphaCommands` from package-level var into `newAlphaCmd()` body (§4.1)
- [ ] Add experimental-group warning to `alpha` `Long` description
- [ ] Replace `Run` with `RunE` in `init.go` base definition (§4.4)
- [ ] Update unit tests in `pkg/cli/*_test.go` to verify group IDs and help output

#### Acceptance Criteria

- `kubebuilder --help` shows commands in at least two groups.
- `kubebuilder cretae api` (typo) prints "did you mean 'create'?".
- `kubebuilder create` (no subcommand) prints a decision guide, not just usage.
- `kubebuilder alpha --help` shows an experimental-stability disclaimer.
- All existing `pkg/cli` unit tests continue to pass.

#### Representative Code Snippets

```go
// pkg/cli/root.go — inside newRootCmd(), after cmd construction

// Explicit suggestion tuning (Cobra default is 2; make it visible)
cmd.SuggestionsMinimumDistance = 2

// Silence cobra's default error/usage printing; we log via slog
cmd.SilenceUsage  = true
cmd.SilenceErrors = true
```

```go
// pkg/cli/cli.go — inside addSubcommands(), after building each sub-command

root.AddGroup(
    &cobra.Group{ID: "setup",        Title: "Project Setup"},
    &cobra.Group{ID: "scaffold",     Title: "Scaffolding"},
    &cobra.Group{ID: "experimental", Title: "Experimental"},
    &cobra.Group{ID: "other",        Title: "Other"},
)

initCmd.GroupID       = "setup"
editCmd.GroupID       = "setup"
createCmd.GroupID     = "scaffold"
alphaCmd.GroupID      = "experimental"
completionCmd.GroupID = "other"
versionCmd.GroupID    = "other"
```

```go
// pkg/cli/create.go — add guided RunE
func (c CLI) newCreateCmd() *cobra.Command {
    return &cobra.Command{
        Use:        "create",
        SuggestFor: []string{"new", "add"},
        Short:      "Scaffold a Kubernetes API or webhook",
        Long: `Scaffold a Kubernetes API or webhook.

Choose one of the available subcommands:

  api     — Scaffold a Custom Resource Definition (CRD) and its controller
  webhook — Scaffold an admission webhook for an existing API kind`,
        Example: fmt.Sprintf(`  # Scaffold a new API with a controller
  %[1]s create api --group batch --version v1 --kind CronJob --resource --controller

  # Scaffold a validating webhook for an existing kind
  %[1]s create webhook --group batch --version v1 --kind CronJob --defaulting`, c.commandName),
        RunE: func(cmd *cobra.Command, args []string) error {
            return fmt.Errorf(
                "subcommand required\n\n" +
                "  kubebuilder create api     — scaffold CRD + controller\n" +
                "  kubebuilder create webhook — scaffold admission webhook\n\n" +
                "Run 'kubebuilder create --help' for full usage",
            )
        },
    }
}
```

```go
// pkg/cli/alpha.go — remove global var; build inside factory

func (c *CLI) newAlphaCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:        alphaCommand,
        SuggestFor: []string{"experimental"},
        Short:      "Alpha-stage subcommands",
        Long: `Alpha-stage subcommands for unstable or experimental features.

⚠ WARNING: Alpha subcommands may change or be removed without notice.
           No backwards compatibility is guaranteed.`,
    }
    cmd.AddCommand(alpha.NewScaffoldCommand())
    cmd.AddCommand(alpha.NewUpdateCommand())
    for _, extra := range c.extraAlphaCommands {
        cmd.AddCommand(extra)
    }
    return cmd
}
```

#### Test Plan

```go
// pkg/cli/root_test.go
func TestCommandGroups(t *testing.T) {
    c, err := cli.New(
        cli.WithCommandName("kubebuilder"),
        cli.WithPlugins(/* minimal test plugin */),
    )
    require.NoError(t, err)

    root := c.Command()
    groups := root.Groups()
    groupIDs := make(map[string]bool)
    for _, g := range groups {
        groupIDs[g.ID] = true
    }
    assert.True(t, groupIDs["setup"],    "expected 'setup' group")
    assert.True(t, groupIDs["scaffold"], "expected 'scaffold' group")
}

func TestCreateWithNoSubcommandGivesGuidance(t *testing.T) {
    // capture stdout, execute "create", assert error message contains "api"
}
```

#### Minimizing Risk

- All changes are in `pkg/cli/` cosmetic/metadata fields or small structural refactors.
- No changes to plugin interfaces, scaffolding, or config.
- Each changed file should have a corresponding test change.
- Keep the PR diff under 300 lines; split further if needed.

---

### PR 2 — Validation & Completion

**Goal:** Add `RegisterFlagCompletionFunc` for `--plugins` and resource flags; add
`Args` validators; move input validation strictly before filesystem writes.

#### Checklist

- [ ] Add `RegisterFlagCompletionFunc` for `--plugins` on root command (enumerate registered keys)
- [ ] Add `RegisterFlagCompletionFunc` for `--project-version` on root/init commands
- [ ] Add `RegisterFlagCompletionFunc` for `--group`, `--version`, `--kind` on `create api` / `create webhook` (dynamic: read from PROJECT file if present)
- [ ] Add `cobra.NoArgs` to commands that should not accept positional args
- [ ] Move `options.validate()` and `res.Validate()` before `factory.store.New()` in `preRunEFunc` (§4.3)
- [ ] Apply `cliError(err, hint)` wrapper to all error returns in `pkg/cli/`
- [ ] Add `--group` / `--version` cross-validation (version must match `v` + digit pattern early)
- [ ] Add `ValidArgsFunction` to `alpha` subcommands returning `ShellCompDirectiveNoFileComp`
- [ ] Update `pkg/cli/completion.go` to document the `completion` command in root help
- [ ] Add/expand unit tests for completion callbacks and validation paths

#### Acceptance Criteria

- `kubebuilder init --plugins <TAB>` returns the list of registered plugin short-keys.
- `kubebuilder create api --group <TAB>` returns known groups from the project config (or empty + directive if no project found).
- Running `kubebuilder init` with an invalid `--domain` (e.g., containing spaces) fails immediately with a clear message before any file is written.
- Running `kubebuilder create api --group my group` fails at argument validation, not after PROJECT file creation.
- All `pkg/cli` unit tests pass.

#### Representative Code Snippets

```go
// pkg/cli/root.go — inside newRootCmd(), after PersistentFlags definition

// Completion for --plugins: enumerate all registered plugin short-keys
// getShortKey converts "go.kubebuilder.io/v4" → "go/v4" for readability
_ = cmd.RegisterFlagCompletionFunc(pluginsFlag,
    func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        // Sort keys for deterministic completion results (maps have non-deterministic iteration)
        keys := make([]string, 0, len(c.plugins))
        for key := range c.plugins {
            keys = append(keys, key)
        }
        slices.Sort(keys)

        candidates := make([]string, 0, len(keys))
        for _, key := range keys {
            p := c.plugins[key]
            desc := ""
            if d, ok := p.(plugin.Describable); ok {
                desc = d.Description()
            }
            candidates = append(candidates, getShortKey(key)+"\t"+desc)
        }
        return candidates, cobra.ShellCompDirectiveNoFileComp
    },
)

// Completion for --project-version
_ = cmd.RegisterFlagCompletionFunc(projectVersionFlag,
    func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        return c.getAvailableProjectVersions(), cobra.ShellCompDirectiveNoFileComp
    },
)
```

```go
// pkg/cli/api.go — dynamic completion for --group and --version

func completionFromProjectConfig(flagName string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
    return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        fs := machinery.Filesystem{FS: afero.NewOsFs()}
        store := yamlstore.New(fs)
        if err := store.Load(); err != nil {
            // No project found — return empty with no error directive
            return nil, cobra.ShellCompDirectiveNoFileComp
        }
        cfg := store.Config()
        seen := make(map[string]struct{})
        var results []string
        for _, res := range cfg.ListResources() {
            var val string
            switch flagName {
            case "group":
                val = res.Group
            case "version":
                val = res.Version
            }
            if _, ok := seen[val]; !ok && val != "" {
                seen[val] = struct{}{}
                results = append(results, val)
            }
        }
        return results, cobra.ShellCompDirectiveNoFileComp
    }
}

// Inside newCreateAPICmd(), after cmd construction:
_ = cmd.RegisterFlagCompletionFunc("group",   completionFromProjectConfig("group"))
_ = cmd.RegisterFlagCompletionFunc("version", completionFromProjectConfig("version"))
```

```go
// pkg/cli/cmd_helpers.go — preRunEFunc, move validation before store.New()

func (factory *executionHooksFactory) preRunEFunc(...) func(...) error {
    return func(cmd *cobra.Command, _ []string) error {
        // 1. Sync duplicate flag values
        syncDuplicateFlags(cmd.Flags(), factory.duplicateFlagValues)

        // 2. Validate options BEFORE touching the filesystem ← FIX for §4.3
        var res *resource.Resource
        if options != nil {
            options.Domain = factory.domain // injected earlier
            if err := options.validate(); err != nil {
                return cliError(
                    fmt.Errorf("invalid resource flags: %w", err),
                    fmt.Sprintf("%s create api --help", factory.commandName),
                )
            }
            res = options.newResource()
            if err := res.Validate(); err != nil {
                return cliError(
                    fmt.Errorf("resource validation failed: %w", err),
                    fmt.Sprintf("%s create api --help", factory.commandName),
                )
            }
        }

        // 3. NOW touch the filesystem
        if createConfig {
            if err := factory.store.New(factory.projectVersion); err != nil { ... }
        } else {
            if err := factory.store.Load(); err != nil { ... }
        }
        ...
    }
}
```

```go
// pkg/cli/api.go — reject positional args
cmd.Args = cobra.NoArgs
```

#### Test Plan

```go
// pkg/cli/completion_test.go
func TestPluginsFlagCompletion(t *testing.T) {
    c, _ := cli.New(cli.WithPlugins(golangv4.Plugin{}), ...)
    root := c.Command()

    // Simulate shell completion call
    results, directive := root.ValidateRequiredFlags() // placeholder — use cobra test helper
    // Verify "go/v4" appears in results
    // Verify directive == cobra.ShellCompDirectiveNoFileComp
}

func TestCreateAPIRejectsPositionalArgs(t *testing.T) {
    c, _ := cli.New(...)
    root := c.Command()
    root.SetArgs([]string{"create", "api", "unexpected-arg"})
    err := root.Execute()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "unknown command")
}
```

#### Minimizing Risk

- Completion functions are additive — they do not change existing command behaviour.
- The validation-ordering fix in `preRunEFunc` is the highest-risk change; isolate it
  as a separate commit with focused unit tests verifying the "no side effects on error"
  guarantee.
- Avoid reading the project config file in completion functions if it may not exist;
  always handle `os.ErrNotExist` gracefully.

---

### PR 3 — Plugin Safety, Determinism & Architecture

**Goal:** Harden the external plugin execution path with an explicit trust check,
improve plugin resolution error messages with candidate lists, eliminate the global
`alphaCommands` side effect (if not already done in PR 1), and reduce other global
state.

#### Checklist

- [ ] Add `verifyExternalPluginTrust(path string) error` to `pkg/plugins/external/` (world-writable check, symlink resolution)
- [ ] Call `verifyExternalPluginTrust` before exec in every external plugin subcommand (`init.go`, `api.go`, `webhook.go`, `edit.go`)
- [ ] Print an explicit warning before executing any external plugin binary (using `PersistentPreRunE` on the command or via `slog.Warn`)
- [ ] Improve `resolvePlugins()` error message for the `default` (ambiguous) case to list matched keys (§4.2)
- [ ] Add `cliError()` helper to `pkg/cli/` and replace raw `fmt.Errorf` calls in all error paths in `pkg/cli/`
- [ ] Document the external plugin trust model in `pkg/plugins/external/plugin.go` header comment and in `CONTRIBUTING.md`
- [ ] Remove `alphaCommands` package-level var if not removed in PR 1
- [ ] Verify idempotency: running `kubebuilder create api` twice for the same GVK returns a clear "already exists" message without corrupting state
- [ ] Add unit tests for `verifyExternalPluginTrust` covering: missing file, world-writable, symlink to world-writable target
- [ ] Add integration test that asserts `resolvePlugins` with two matching plugins returns both keys in the error

#### Acceptance Criteria

- Executing `kubebuilder init` with an external plugin that is world-writable prints an error and exits without scaffolding.
- `resolvePlugins` ambiguity error lists all matched plugin keys.
- `cliError` is used consistently in at least the three most common error paths.
- No package-level mutable state remains in `pkg/cli/alpha.go`.
- All existing unit and integration tests pass.

#### Representative Code Snippets

```go
// pkg/plugins/external/helpers.go — trust verification

import (
    "fmt"
    "os"
    "path/filepath"
)

// VerifyExternalPluginTrust checks that the plugin binary at execPath is safe to execute.
// It resolves symlinks, verifies the file exists, and rejects world- or group-writable binaries.
func VerifyExternalPluginTrust(execPath string) error {
    // Resolve symlinks to catch attacks via symlink redirection
    resolved, err := filepath.EvalSymlinks(execPath)
    if err != nil {
        return fmt.Errorf("external plugin %q not found or unresolvable: %w", execPath, err)
    }

    info, err := os.Stat(resolved)
    if err != nil {
        return fmt.Errorf("external plugin %q: stat failed: %w", resolved, err)
    }

    mode := info.Mode()
    if mode&0o022 != 0 {
        return fmt.Errorf(
            "external plugin %q is writable by group or others (mode %04o); refusing to execute.\n\n"+
                "How to fix:\n  chmod go-w %s",
            resolved, mode, resolved,
        )
    }
    return nil
}
```

```go
// pkg/plugins/external/init.go — call trust check before exec

func (p *ExternalPlugin) GetInitSubcommand() plugin.Subcommand {
    return &initSubcommand{plugin: p}
}

func (s *initSubcommand) PreScaffold(fs machinery.Filesystem) error {
    if err := VerifyExternalPluginTrust(s.plugin.Path); err != nil {
        return err
    }
    slog.Warn("executing external plugin",
        "path", s.plugin.Path,
        "subcommand", "init",
    )
    return nil
}
```

```go
// pkg/cli/cli.go — improved ambiguity error in resolvePlugins()

default:
    matchedKeys := make([]string, 0, len(plugins))
    for _, p := range plugins {
        matchedKeys = append(matchedKeys, "  "+plugin.KeyFor(p))
    }
    slices.Sort(matchedKeys) // deterministic output
    return fmt.Errorf(
        "plugin key %q is ambiguous%s; it matches:\n%s\n\n"+
            "Use the full plugin key with --plugins, e.g.:\n  --plugins=%s",
        pluginKey, extraErrMsg,
        strings.Join(matchedKeys, "\n"),
        strings.TrimPrefix(matchedKeys[0], "  "), // safe prefix removal
    )
```

```go
// pkg/cli/cmd_helpers.go — cliError helper

// cliError wraps err with an actionable hint for the user.
// hint should be a concrete command or instruction, not a sentence.
func cliError(err error, hint string) error {
    if hint == "" {
        return err
    }
    return fmt.Errorf("%w\n\nHow to fix:\n  %s", err, hint)
}
```

#### Test Plan

```go
// pkg/plugins/external/helpers_test.go
func TestVerifyExternalPluginTrust(t *testing.T) {
    t.Run("missing file", func(t *testing.T) {
        err := VerifyExternalPluginTrust("/nonexistent/binary")
        require.Error(t, err)
        assert.Contains(t, err.Error(), "not found")
    })

    t.Run("world-writable", func(t *testing.T) {
        f, _ := os.CreateTemp(t.TempDir(), "plugin-*")
        _ = f.Close()
        _ = os.Chmod(f.Name(), 0o777)
        err := VerifyExternalPluginTrust(f.Name())
        require.Error(t, err)
        assert.Contains(t, err.Error(), "writable")
    })

    t.Run("safe binary", func(t *testing.T) {
        f, _ := os.CreateTemp(t.TempDir(), "plugin-*")
        _ = f.Close()
        _ = os.Chmod(f.Name(), 0o755)
        err := VerifyExternalPluginTrust(f.Name())
        require.NoError(t, err)
    })
}
```

```go
// pkg/cli/cli_test.go — ambiguous resolution error includes keys
func TestResolvePluginsAmbiguousError(t *testing.T) {
    // Register two plugins with keys that share a prefix
    c, err := cli.New(
        cli.WithPlugins(fakePluginA{}, fakePluginB{}), // both match "fake"
        ...
    )
    require.NoError(t, err) // construction succeeds

    c.Command().SetArgs([]string{"init", "--plugins=fake"})
    err = c.Command().Execute()
    require.Error(t, err)
    assert.Contains(t, err.Error(), "fake.a.io/v1")
    assert.Contains(t, err.Error(), "fake.b.io/v1")
}
```

#### Minimizing Risk

- The trust check is additive — it only adds a new error path before the existing exec.
  It does not change the exec mechanism itself.
- Start with a warning (not an error) for the world-writable check in the first merge,
  then graduate to an error in a follow-up, giving users time to fix permissions.
- Document the trust model change in `CHANGELOG` / release notes.
- Keep the `cliError` helper change as a pure refactor commit (no logic changes).

---

## 6. General Commit & Review Guidance

### Commit structure within a PR

Prefer one logical commit per PR; squash if the CI is clean. If the PR touches
unrelated areas, keep them in separate commits with clear messages:

```
feat(cli): add command groups and help redesign

- Define setup/scaffold/experimental/other groups on root
- Assign GroupID to init, create, edit, alpha, completion, version
- Replace bare global alphaCommands slice with factory construction
- Tune SuggestionsMinimumDistance to 2 (explicit)
- Replace Run with RunE in init base definition

Fixes #<issue>
```

### What reviewers will look for

1. **Behaviour change surface:** Any change to `PreRunE`, `RunE`, `PostRunE` or flag
   parsing changes the command's observable behaviour. Make those explicit in your PR
   description.
2. **Test coverage:** Every new code path must have a unit test. Completion functions
   and validation helpers are easy to unit-test because they are pure functions.
3. **Backward compatibility:** Existing `--flags` must keep working. Do not rename or
   remove flags; deprecate them with `cmd.Flags().MarkDeprecated("old", "use --new")`.
4. **Error message quality:** Run through the failure cases manually and paste the
   output in the PR description.

### How to verify locally

```bash
# Build and install
make build && make install

# Check help output
kubebuilder --help
kubebuilder create --help
kubebuilder alpha --help

# Test completion (bash)
source <(kubebuilder completion bash)
kubebuilder init --plugins <TAB>

# Test typo suggestion
kubebuilder initt

# Run unit tests
make test-unit

# Run lint
make lint-fix
```

---

## 7. Open-Ended Exercises

These exercises have no single right answer. They are meant to stretch your thinking
about CLI design trade-offs.

1. **Exercise — Cobra vs. custom:** Kubebuilder uses a custom plugin-key completion
   function. Cobra's `completion` command already generates shell-native completions.
   What would it take to make `kubebuilder completion bash | source /dev/stdin` give
   tab-complete for `--plugins` on a fresh shell without any custom code in Kubebuilder?

2. **Exercise — PersistentPreRunE chaining:** The root command's `PersistentPreRunE`
   currently checks for help flags in `--plugins`. If you add a `PersistentPreRunE` to
   the `alpha` command, will the root hook still fire? Write a test that proves your
   answer. (Hint: look at `EnableTraverseRunHooks`.)

3. **Exercise — idempotency audit:** Run `kubebuilder create api --group batch
   --version v1 --kind CronJob` twice in the same project. What happens? Is the second
   run idempotent, partially idempotent, or destructive? What Cobra hook would be the
   right place to detect "already exists" and abort cleanly?

4. **Exercise — external plugin path injection:** An attacker who can write to a
   directory that appears in kubebuilder's plugin search path can introduce a malicious
   binary. Map out the full discovery path in `DiscoverExternalPlugins` (in `pkg/cli/`)
   and identify every directory that is searched. Propose a mitigation strategy that does
   not break legitimate external plugin workflows.

5. **Exercise — deprecation lifecycle:** `helm/v1-alpha` is deprecated in favour of
   `helm/v2-alpha`. Currently the deprecation warning is printed but the v1-alpha plugin
   still runs. Design a complete deprecation schedule (warn → warn louder → error →
   remove) using only Cobra's `Deprecated` field, `PersistentPreRunE`, and semver plugin
   versioning. Write the `PreRunE` hook that implements stage two ("warn louder").

6. **Exercise — determinism test:** The plugin resolution order is determined by
   iteration over `c.plugins` (a `map[string]plugin.Plugin`). Go maps do not guarantee
   iteration order. Under what conditions could this produce non-deterministic scaffold
   output? Write a test that reliably detects non-determinism in plugin resolution.

---

*This document is a living guide. As Kubebuilder's CLI evolves, update the antipatterns
section when issues are resolved and add new exercises as the plugin ecosystem grows.*
