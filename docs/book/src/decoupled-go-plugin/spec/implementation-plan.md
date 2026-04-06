# Implementation Plan: `decoupled-go/v1`

> **Goal:** Ship a production-quality external Kubebuilder plugin that
> scaffolds decoupled Go operator projects.

---

## Milestones Overview

| Milestone | Description | Target |
|---|---|---|
| M0 | Repository bootstrap + CI | Week 1 |
| M1 | MVP: `init` + `create api` commands | Week 3 |
| M2 | Complete: `create webhook` + `edit --regenerate` | Week 5 |
| M3 | Integration + E2E tests | Week 7 |
| M4 | Documentation + v1.0.0 release | Week 8 |

---

## Phase 1 (M0): Repository Bootstrap

### Tasks

- [ ] Create plugin repository (`github.com/myorg/decoupled-go-plugin`).
- [ ] Initialize Go module with correct import path.
- [ ] Set up directory structure (see Chapter 7 layout).
- [ ] Add `sigs.k8s.io/kubebuilder/v4` as a dependency.
- [ ] Configure GitHub Actions CI:
  - `go vet ./...`
  - `go test ./...`
  - `golangci-lint run`
  - Build matrix for Linux/macOS.
- [ ] Add `Makefile` with targets: `build`, `test`, `lint`, `install`, `release`.
- [ ] Add `LICENSE` (Apache 2.0 recommended for Kubernetes ecosystem).
- [ ] Add `CONTRIBUTING.md` and `CHANGELOG.md`.

### Done Criteria

- CI passes on empty plugin that returns valid JSON.
- Binary builds for linux/amd64 and darwin/arm64.

---

## Phase 2 (M1): MVP Commands

### Tasks

#### Entry Point

- [ ] Implement `cmd/main.go` with stdin/stdout JSON dispatch.
- [ ] Implement error handling (set `resp.Error`, never exit non-zero).
- [ ] Implement `flags` command for all subcommands.
- [ ] Implement `metadata` command for all subcommands.

#### `init` Command

- [ ] Parse flags: `--domain`, `--repo`, `--project-name`.
- [ ] Generate `internal/platform/doc.go`.
- [ ] Generate `internal/platform/manager/manager.go`.
- [ ] Generate `cmd/main.go` (project main, not plugin main).
- [ ] Generate `internal/app/doc.go` (create-only).
- [ ] Generate `go.mod`, `Makefile`, `Dockerfile`, `.gitignore`.
- [ ] Write initial lock file.
- [ ] Add provenance headers to all platform files.

#### `create api` Command

- [ ] Parse flags: `--group`, `--version`, `--kind`, `--namespaced`, `--force`.
- [ ] Validate GROUP/VERSION/KIND (DNS label format).
- [ ] Generate `api/<group>/<version>/<kind>_types.go`.
- [ ] Generate `internal/platform/reconciler/<kind>_reconciler.go`.
- [ ] Generate `internal/platform/reconciler/hooks.go`.
- [ ] Generate `internal/platform/reconciler/noop.go`.
- [ ] Generate `internal/platform/testing/builder.go`.
- [ ] Generate `internal/app/<kind>/hooks.go` (create-only).
- [ ] Generate `internal/app/<kind>/hooks_test.go` (create-only).
- [ ] Update lock file with new files.

### Unit Tests (M1)

- [ ] Test `flagValue()` helper.
- [ ] Test `cloneUniverse()` — must deep copy.
- [ ] Test `Init()` — verify all expected keys in returned universe.
- [ ] Test `CreateAPI()` — platform files overwrite; app files create-only.
- [ ] Test `Flags()` and `Metadata()` — verify correct structs returned.
- [ ] Test lock file marshal/unmarshal roundtrip.
- [ ] Test template rendering — verify no template parse/execution errors.
- [ ] Test provenance header presence in all platform files.

### Done Criteria

- `kubebuilder init` produces a compilable Go project.
- `kubebuilder create api` adds a compilable controller skeleton.
- Unit test coverage ≥ 80%.
- CI passes.

---

## Phase 3 (M2): Complete Commands

### Tasks

#### `create webhook` Command

- [ ] Parse flags: `--defaulting`, `--validating`, `--programmatic-validation`.
- [ ] Generate `internal/platform/webhook/<kind>_webhook.go`.
- [ ] Generate `internal/platform/webhook/hooks.go` (ValidationHooks, DefaultingHooks).
- [ ] Generate `internal/app/<kind>/webhook_hooks.go` (create-only).
- [ ] Generate `internal/app/<kind>/webhook_hooks_test.go` (create-only).
- [ ] Update lock file.

#### `edit` Command with `--regenerate`

- [ ] Read existing lock file.
- [ ] Detect plugin version change (old vs new).
- [ ] Emit migration warnings to stderr for MAJOR/MINOR bumps.
- [ ] Regenerate all `platform`-zone files.
- [ ] Skip all `app`-zone files.
- [ ] Update lock file with new checksums and version.

#### `NoopHooks` Pattern

- [ ] Generate `noop.go` for reconciler hooks.
- [ ] Generate `noop.go` for webhook hooks.
- [ ] Document embedding pattern in generated stub comments.

### Unit Tests (M2)

- [ ] Test `CreateWebhook()` — platform overwrite, app create-only.
- [ ] Test `Edit()` with `--regenerate` — verify app files unchanged.
- [ ] Test migration warning emission.
- [ ] Test lock file read/write/update cycle.
- [ ] Test `NoopHooks` implements all hook interfaces (compile test).

### Done Criteria

- All four commands functional.
- `--regenerate` preserves user code.
- Migration warnings appear for version changes.

---

## Phase 4 (M3): Integration and E2E Tests

### Integration Tests

- [ ] Full `init` → compile project → verify it compiles.
- [ ] Full `create api` → compile project → verify controller is registered.
- [ ] `create api` twice → second run preserves app hooks stub.
- [ ] `edit --regenerate` → verify platform files updated, app files preserved.
- [ ] Chain with `kustomize/v2` — verify universe merges correctly.

### E2E Tests

- [ ] Create a Kind cluster in CI.
- [ ] Init project with plugin.
- [ ] Create API.
- [ ] Implement minimal hooks stub.
- [ ] `make build` — builds operator binary.
- [ ] `make manifests` — generates CRD manifests.
- [ ] Deploy to Kind cluster.
- [ ] Verify controller starts and watches resources.
- [ ] Create a CR — verify reconcile loop runs.
- [ ] Delete CR — verify cleanup.
- [ ] Upgrade plugin version — verify `--regenerate` does not break user code.

### Done Criteria

- All integration tests pass.
- E2E tests pass on Kind cluster.
- Generated project compiles and works end-to-end.

---

## Phase 5 (M4): Documentation and Release

### Tasks

- [ ] Write `CHANGELOG.md` v1.0.0 entry.
- [ ] Write `MIGRATION.md` (empty for v1.0.0, template for future).
- [ ] Write plugin README with installation instructions.
- [ ] Submit plugin to Kubebuilder's external plugins list (if applicable).
- [ ] Create GitHub Release v1.0.0 with pre-built binaries for:
  - `linux/amd64`
  - `linux/arm64`
  - `darwin/amd64`
  - `darwin/arm64`
- [ ] Set up `goreleaser` for automated release builds.
- [ ] Tag `v1.0.0` and push.

### Done Criteria

- Release page has binaries for all platforms.
- Installation instructions tested from scratch on a clean machine.

---

## Testing Strategy

### Test Pyramid

```
        /\
       /E2E\        ← few, slow, high confidence
      /──────\
     /Integra-\
    / tion     \    ← moderate, tests interactions
   /────────────\
  /  Unit Tests  \  ← many, fast, test individual functions
 /────────────────\
```

### Test Matrix

| Command | Unit | Integration | E2E |
|---|---|---|---|
| `init` | ✅ | ✅ | ✅ |
| `create api` | ✅ | ✅ | ✅ |
| `create webhook` | ✅ | ✅ | ✅ |
| `edit --regenerate` | ✅ | ✅ | ✅ |
| Plugin chaining | ✅ | ✅ | — |
| `--force` flag | ✅ | ✅ | — |
| Version upgrade | ✅ | ✅ | ✅ |
| Determinism | ✅ | — | — |

### Test Tooling

- **Unit:** `go test` with standard `testing` package (add `github.com/stretchr/testify` if preferred)
- **Integration:** `go test` with `//go:build integration` tag + temp directory scaffolding
- **E2E:** Kind cluster + Kubebuilder `utils.TestContext` pattern
- **Coverage:** `go test -coverprofile` + `go tool cover`
- **Race detection:** `go test -race ./...`

---

## Versioning and Release Plan

### Version Cadence

| Release type | Trigger | Includes |
|---|---|---|
| Patch (v1.x.Y) | Bug fix, security fix | Fixes only |
| Minor (v1.X.0) | New feature, new hook | Features + fixes, backward compatible |
| Major (vX.0.0) | Breaking interface change | Migration required |

### Release Checklist

- [ ] All tests pass.
- [ ] `CHANGELOG.md` updated.
- [ ] `MIGRATION.md` updated (for MAJOR/MINOR).
- [ ] Version constant in plugin source updated.
- [ ] Tag pushed to GitHub.
- [ ] `goreleaser` triggered — binaries built and attached to release.
- [ ] Plugin README updated with new version installation instructions.

---

## Risk List and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Kubebuilder external plugin protocol changes | Low | High | Pin to a specific `kubebuilder/v4` minor; update on each minor release |
| Users accidentally edit platform files | Medium | Medium | Lint check in Makefile that verifies provenance headers; CI enforcement |
| Interface breaking change frustrates early adopters | Medium | High | Stable v1 interfaces; NoopHooks for optional methods; clear migration docs |
| Generated code has security vulnerability | Low | High | Template review in CI; goreleaser builds trigger on security patches |
| Plugin binary not found by Kubebuilder | Medium | Medium | Clear installation docs; diagnostic mode prints discovery paths |
| Template rendering produces non-compiling Go | Medium | High | Compile-test in CI: render → `go build` → verify |
| Determinism broken (non-reproducible output) | Low | Medium | Determinism test: render twice, assert byte-for-byte equal |
| Chaining conflicts with `kustomize/v2` | Low | High | Integration test that chains both plugins and verifies universe merge |

---

## Success Criteria for v1.0.0

- [ ] A new Go operator project can be scaffolded in under 5 minutes.
- [ ] `kubebuilder edit --regenerate` takes under 10 seconds.
- [ ] Upgrading plugin minor version requires zero changes to app layer code.
- [ ] Upgrading plugin major version requires only the changes listed in `MIGRATION.md`.
- [ ] Generated project compiles and deploys to Kubernetes without errors.
- [ ] Unit test coverage ≥ 80%.
- [ ] E2E test passes on Kind cluster in CI.
- [ ] Zero known security issues in generated code.
