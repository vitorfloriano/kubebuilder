# Chapter 2 — External Plugin Lifecycle

## How External Plugins Work

An external plugin is a standalone executable that Kubebuilder discovers,
invokes, and communicates with via JSON over `stdin`/`stdout`. From
Kubebuilder's perspective, calling `your-plugin-binary` with a JSON request
on stdin and reading the JSON response from stdout is the entire protocol.

```
┌──────────────┐   JSON on stdin   ┌──────────────────────┐
│  kubebuilder │ ────────────────► │  your-plugin-binary  │
│              │ ◄──────────────── │                      │
└──────────────┘   JSON on stdout  └──────────────────────┘
                   errors on stderr (human-readable)
```

---

## PluginRequest — What Kubebuilder Sends

Every time Kubebuilder calls your plugin, it sends a `PluginRequest` JSON
object. The structure is defined in
`pkg/plugin/external/types.go`:

```go
type PluginRequest struct {
    // APIVersion is the schema version of this request (currently "v1alpha1").
    APIVersion string `json:"apiVersion"`

    // Command is the subcommand being executed:
    // "init", "create api", "create webhook", "edit",
    // "flags" (to list accepted flags), "metadata" (to describe the plugin).
    Command string `json:"command"`

    // Args are the CLI arguments passed by the user after flag parsing.
    Args []string `json:"args"`

    // Universe is a map[filePath]fileContents representing every file
    // in the project at the time your plugin is called.
    // Previous plugins in the chain have already contributed their files.
    Universe map[string]string `json:"universe"`

    // PluginChain lists the plugin keys that have run before yours
    // in the current command execution, in order.
    PluginChain []string `json:"pluginChain,omitempty"`

    // Config is the serialised PROJECT file contents.
    // It is absent on the very first "init" run because the PROJECT
    // file doesn't exist yet.
    Config map[string]any `json:"config,omitempty"`
}
```

### Special commands

| Command | When called | What to return |
|---|---|---|
| `flags` | Before any subcommand, to discover your flags | `Flags []Flag` |
| `metadata` | When `--help` is used | `Metadata SubcommandMetadata` |
| `init` | `kubebuilder init` | Files for the initial project scaffold |
| `create api` | `kubebuilder create api` | Files for a new API/controller |
| `create webhook` | `kubebuilder create webhook` | Files for a new webhook |
| `edit` | `kubebuilder edit` | Modified/new files |

---

## PluginResponse — What You Return

Your plugin must write a `PluginResponse` JSON object to stdout:

```go
type PluginResponse struct {
    // APIVersion must match the request's APIVersion.
    APIVersion string `json:"apiVersion"`

    // Command echoes back the command from the request.
    Command string `json:"command"`

    // Universe is the updated map[filePath]fileContents.
    // Include ALL files you want to exist after your plugin runs —
    // both files you created and files passed through unchanged.
    Universe map[string]string `json:"universe"`

    // Flags is only used when Command == "flags".
    Flags []Flag `json:"flags,omitempty"`

    // Metadata is only used when Command == "metadata".
    Metadata SubcommandMetadata `json:"metadata,omitempty"`

    // Error signals that your plugin encountered a fatal error.
    Error bool `json:"error,omitempty"`

    // ErrorMsgs provides human-readable error details when Error is true.
    ErrorMsgs []string `json:"errorMsgs,omitempty"`
}
```

> **Important:** The `universe` in your response is the **complete new state**
> of the project files. Pass through any files from the request's universe
> that you did not modify. Kubebuilder writes this universe to disk.

---

## A Minimal External Plugin in Go

Here is the smallest valid external plugin that handles the `init` command:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

func main() {
    // Decode request from stdin
    var req external.PluginRequest
    if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
        writeError(req, fmt.Sprintf("decode request: %v", err))
        os.Exit(1)
    }

    // Build response starting from the incoming universe
    resp := external.PluginResponse{
        APIVersion: req.APIVersion,
        Command:    req.Command,
        Universe:   req.Universe, // pass through existing files
    }

    switch req.Command {
    case "init":
        handleInit(&req, &resp)
    case "flags":
        handleFlags(&req, &resp)
    case "metadata":
        handleMetadata(&req, &resp)
    default:
        // Unknown command — return universe unchanged
    }

    if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
        fmt.Fprintf(os.Stderr, "encode response: %v\n", err)
        os.Exit(1)
    }
}

func handleInit(req *external.PluginRequest, resp *external.PluginResponse) {
    // Add files to resp.Universe
    resp.Universe["internal/platform/doc.go"] = `// Package platform contains generated framework code.
// DO NOT EDIT — regenerated by decoupled-go/v1.
package platform
`
}

func handleFlags(_ *external.PluginRequest, resp *external.PluginResponse) {
    resp.Flags = []external.Flag{
        {Name: "domain", Type: "string", Default: "example.com",
            Usage: "Domain for the operator's API groups"},
    }
}

func handleMetadata(_ *external.PluginRequest, resp *external.PluginResponse) {
    resp.Metadata = external.SubcommandMetadata{
        Description: "Scaffold a decoupled Go operator with a clean platform/app separation.",
        Examples:    "kubebuilder init --plugins decoupled-go.example.com/v1 --domain my.org",
    }
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

## Plugin Discovery

Kubebuilder discovers external plugins through the `--plugins` flag:

```bash
kubebuilder init \
    --plugins decoupled-go.example.com/v1 \
    --domain my.org
```

Kubebuilder resolves `decoupled-go.example.com/v1` to an executable by
searching:

1. `$XDG_DATA_HOME/kubebuilder/plugins/decoupled-go.example.com/v1/`
2. `~/.config/kubebuilder/plugins/decoupled-go.example.com/v1/`
3. `/usr/local/share/kubebuilder/plugins/decoupled-go.example.com/v1/`

Place your binary in one of these directories, or set `--plugin-path` when
calling Kubebuilder.

---

## The `flags` and `metadata` Commands

Kubebuilder calls these two special commands before running any scaffolding
subcommand:

- **`flags`** — called to get the list of CLI flags your plugin accepts.
  Return a `[]Flag` in `resp.Flags`. Kubebuilder uses these to bind flags
  and pass them to the real subcommand.
- **`metadata`** — called when the user passes `--help`. Return
  `resp.Metadata` with `Description` and `Examples`.

Both commands receive an empty `Universe`. Return the universe unchanged.

---

## Error Handling

When your plugin encounters an unrecoverable error:

1. Set `resp.Error = true`.
2. Add human-readable messages to `resp.ErrorMsgs`.
3. Return the current universe unchanged.
4. Write the response to stdout and exit with code 0.

> **Do not exit non-zero.** Kubebuilder reads your stdout; a non-zero exit
> code prevents the response from being read correctly. Signal errors via
> `resp.Error`, not via exit code.

Write **diagnostic** (non-fatal) output to stderr — Kubebuilder forwards
stderr to the terminal as-is.

---

## ✅ Checkpoint

- [ ] Explain the stdin/stdout JSON protocol.
- [ ] What is the `universe` and why must your response include files from the request's universe?
- [ ] What are the special `flags` and `metadata` commands for?
- [ ] How do you signal an error from an external plugin?

## Next Steps

→ [Chapter 3: Decoupled Scaffold Philosophy](./03-decoupled-philosophy.md)
