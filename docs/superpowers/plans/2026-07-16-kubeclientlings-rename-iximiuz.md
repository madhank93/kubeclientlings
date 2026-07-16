# KubeClientlings Rename + iximiuz Playground Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename the project `clientlings` → `KubeClientlings` across the whole repo, remove the dead docs-site tasks, and add an iximiuz playground so the exercises run in a hosted browser lab.

**Architecture:** A refactor with no behavior change (the existing `mise run test` + `mise run e2e` suite is the safety net), landed in green-at-every-step slices, followed by two new files for the iximiuz playground. The rename is split so each task leaves the tree buildable and e2e-green before the next begins.

**Tech Stack:** Go 1.26, cobra CLI, bubbletea TUI, mise (task runner + toolchain), kind, client-go, iximiuz Labs playground manifest, bash.

## Global Constraints

- Go module path: `github.com/madhank93/kubeclientlings` (was `github.com/madhank93/clientlings`).
- Display / brand text (human-facing only): `KubeClientlings`.
- Identifiers, paths, binary, cluster name, config keys: lowercase `kubeclientlings`.
- Binary: `bin/kubeclientlings`. Source dir: `kubeclientlings/`.
- State file: `.kubeclientlings-state.json`. CRD group / ns label domain: `kubeclientlings.dev`.
- CRD group appears in 12 paired files (6 in `exercises/`, 6 in `solutions/`) — starter and solution MUST use the same group or e2e fails.
- Verification suite: `mise run build`, `mise run test`, `mise run e2e`, `mise run lint`. `mise run e2e` requires a running kind cluster (`mise run up` first).
- Every commit must independently build; e2e green before Task 2's commit lands.

---

## Baseline (do once before Task 1)

- [ ] **Step 1: Confirm a clean, green starting point**

Run:
```bash
mise run up        # create kind cluster if not already up
mise run build && mise run test && mise run e2e && mise run lint
```
Expected: all four succeed (build OK, tests PASS, e2e verifies all 49 exercises then restores, lint clean). This is the baseline the rename must preserve. If any fail here, STOP — the failure predates this work.

---

## Task 1: Rename Go module, imports, and source directory

Moves `clientlings/` → `kubeclientlings/` and rewrites the module path + all imports. Runtime strings (cluster name, state file, CRD group) stay `clientlings` for now — internally consistent, so the tree still builds and e2e still passes.

**Files:**
- Modify: `go.mod` (line 1, module path)
- Move: `clientlings/` → `kubeclientlings/` (whole directory)
- Modify: all `*.go` files importing `github.com/madhank93/clientlings/...` (111 files)

**Interfaces:**
- Produces: import prefix `github.com/madhank93/kubeclientlings` and source dir `kubeclientlings/` used by every later task.

- [ ] **Step 1: Move the source directory (preserve git history)**

```bash
git mv clientlings kubeclientlings
```

- [ ] **Step 2: Rewrite the module path in go.mod**

Replace the first line of `go.mod`:
```
module github.com/madhank93/clientlings
```
with:
```
module github.com/madhank93/kubeclientlings
```

- [ ] **Step 3: Rewrite every import of the module path**

```bash
grep -rl "github.com/madhank93/clientlings" --include="*.go" . \
  | xargs sed -i '' 's|github.com/madhank93/clientlings|github.com/madhank93/kubeclientlings|g'
```
(On GNU sed drop the `''` after `-i`.)

- [ ] **Step 4: Rewrite mise.toml build/run/lint/test paths that point at the old dir/binary**

In `mise.toml`, replace every `./clientlings` package path and `bin/clientlings` binary path with `./kubeclientlings` and `bin/kubeclientlings`. Affected tasks: `build`, `watch`, `list`, `up`, `down`, `doctor`, `lint`, `test`, `e2e`, `nuke` (context flag `kind-clientlings` stays until Task 2). Concretely:
```bash
sed -i '' 's|bin/clientlings|bin/kubeclientlings|g; s|./clientlings|./kubeclientlings|g' mise.toml
```
Then hand-verify the `lint` task's package globs now read `./kubeclientlings ./kubeclientlings/cmd …` and `test` reads `./kubeclientlings/... ./internal/...`.

- [ ] **Step 5: Build to prove imports resolve**

Run: `mise run build`
Expected: compiles, produces `bin/kubeclientlings`. If any import is missed, the compiler names the file — fix and rebuild.

- [ ] **Step 6: Run the full suite (behavior unchanged)**

Run: `mise run test && mise run e2e && mise run lint`
Expected: tests PASS, e2e verifies all 49 and restores, lint clean. (CRD group still `clientlings.dev` everywhere — consistent, so green.)

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor: move clientlings dir and rename Go module to kubeclientlings"
```

---

## Task 2: Rename runtime identifiers and CRD group

Renames the strings that appear at runtime: kind cluster name, cobra command name, state file, namespace label, and the `clientlings.dev` CRD group in all 12 paired files. e2e is the gate that proves the CRD-group rename stayed in sync.

**Files:**
- Modify: `kubeclientlings/cluster/cluster.go:16` (`Name = "clientlings"`)
- Modify: `kubeclientlings/cmd/root.go:11` (`Use: "clientlings"`)
- Modify: `kubeclientlings/exercises/state.go:12` (`StateFile = ".clientlings-state.json"`)
- Modify: `internal/exkit/exkit.go:34` (`ExerciseLabel = "clientlings.dev/exercise"`)
- Modify: 12 CRD-group files — `exercises/{crds/crd1,crds/crd2,dynamic/dyn3,finalizers/fin1,finalizers/fin2,subresources/sub2}/main.go` and the same 6 paths under `solutions/`
- Modify: `mise.toml` `nuke` task (`kind-clientlings` context) and any remaining `kind-clientlings` refs

**Interfaces:**
- Consumes: source dir `kubeclientlings/` from Task 1.
- Produces: cluster name `kubeclientlings`, state file `.kubeclientlings-state.json`, label `kubeclientlings.dev/exercise`, CRD group `kubeclientlings.dev` — the final runtime contract.

- [ ] **Step 1: Rename the CRD group in exercises and solutions together**

```bash
grep -rl "clientlings.dev" exercises solutions \
  | xargs sed -i '' 's|clientlings\.dev|kubeclientlings.dev|g'
```
This rewrites `clientlings.dev/v1alpha1`, group `clientlings.dev`, and `widgets.clientlings.dev` in all 12 files at once.

- [ ] **Step 2: Rename runtime constants and command name**

```bash
sed -i '' 's|Name    = "clientlings"|Name    = "kubeclientlings"|' kubeclientlings/cluster/cluster.go
sed -i '' 's|Use:           "clientlings"|Use:           "kubeclientlings"|' kubeclientlings/cmd/root.go
sed -i '' 's|StateFile = ".clientlings-state.json"|StateFile = ".kubeclientlings-state.json"|' kubeclientlings/exercises/state.go
sed -i '' 's|ExerciseLabel = "clientlings.dev/exercise"|ExerciseLabel = "kubeclientlings.dev/exercise"|' internal/exkit/exkit.go
```
Note: the `ExerciseLabel` line is already covered by Step 1's `clientlings.dev` sed if `internal/exkit/exkit.go` were in scope — it is NOT (Step 1 scoped to exercises/solutions), so this explicit line is required.

- [ ] **Step 3: Update the kind context in mise.toml `nuke` task**

```bash
sed -i '' 's|kind-clientlings|kind-kubeclientlings|g' mise.toml
```
The cluster `Name` const (Step 2) drives the actual context `kind-kubeclientlings`; this aligns the raw kubectl call in `nuke`.

- [ ] **Step 4: Recreate the cluster under the new name**

The kind cluster is now named `kind-kubeclientlings`; the old `kind-clientlings` still exists from baseline.
```bash
kind delete cluster --name clientlings
mise run up
```
Expected: new cluster `kind-kubeclientlings` created.

- [ ] **Step 5: Build**

Run: `mise run build`
Expected: compiles clean.

- [ ] **Step 6: Run the full suite — proves CRD-group pairs stayed in sync**

Run: `mise run test && mise run e2e && mise run lint`
Expected: tests PASS; e2e overlays solutions onto exercises and verifies all 49 against the new cluster (any starter/solution CRD-group drift would fail here); lint clean.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor: rename runtime identifiers and CRD group to kubeclientlings"
```

---

## Task 3: Rebrand splash + docs, fix stale strings, remove dead docs-site

Rewrites human-facing branding to `KubeClientlings`, fixes the hardcoded `112` and generic-Go tagline, and rips out the `web/`-dependent mise tasks + splash site link that reference a site that does not exist.

**Files:**
- Modify: `kubeclientlings/tui/view.go` — welcome() splash (title line ~97, tagline line ~98, meta links lines ~101-103)
- Modify: `mise.toml` — remove `gen-site`, `site`, `site-build` tasks
- Modify: any `README*.md` / `info.toml` header text carrying `clientlings` brand or the `clientlings.madhan.app` domain
- Modify: `mise.toml` line 1 comment (`clientlings learning repo` → `kubeclientlings learning repo`)

**Interfaces:**
- Consumes: `Model.total` (already the live exercise count) for the dynamic splash count.

- [ ] **Step 1: Fix the tagline + dynamic exercise count in the splash**

In `kubeclientlings/tui/view.go`, `welcome()`, replace:
```go
	tagline := dimStyle.Render("Learn Go the rustlings way — 112 exercises, basics → advanced")
```
with:
```go
	tagline := dimStyle.Render(fmt.Sprintf("Learn Kubernetes client-go the rustlings way — %d exercises, basics → advanced", m.total))
```
(`fmt` is already imported in view.go.)

- [ ] **Step 2: Update the splash title + repo link, drop the dead site link**

In the same `welcome()`, change the title `"🐹  clientlings"` → `"🐹  KubeClientlings"`. In the `meta` block, rewrite the repo link to `https://github.com/madhank93/kubeclientlings` and DELETE the `Site` line (`clientlings.madhan.app` points at a site that does not exist):
```go
	meta := lipgloss.JoinVertical(lipgloss.Left,
		labelStyle.Render("Repo")+linkStyle.Render("https://github.com/madhank93/kubeclientlings"),
		labelStyle.Render("Maintainer")+"Madhan Kumaravelu  "+dimStyle.Render("(@madhank93)"),
	)
```

- [ ] **Step 3: Remove the dead docs-site mise tasks**

Delete the `[tasks.gen-site]`, `[tasks.site]`, and `[tasks.site-build]` blocks from `mise.toml` (they run `go run ./web/gen` / `npm` in a `web/` dir that does not exist). Update the line-1 comment brand to `kubeclientlings`.

- [ ] **Step 4: Rebrand remaining docs strings**

```bash
grep -rl "clientlings" README*.md info.toml 2>/dev/null \
  | xargs sed -i '' 's|Clientlings|KubeClientlings|g; s|clientlings|kubeclientlings|g'
```
Then eyeball the diff: fix any spot where `KubeClientlings` reads wrong mid-sentence, and confirm no `clientlings.madhan.app` remains.

- [ ] **Step 5: Confirm no stray old references remain in source**

Run:
```bash
grep -rin "clientlings" --include="*.go" --include="*.toml" --include="*.md" . \
  | grep -vi kubeclientlings | grep -v docs/superpowers
```
Expected: no output (only the spec/plan under `docs/superpowers/` may still say `clientlings` in prose — excluded above).

- [ ] **Step 6: Build + test + lint**

Run: `mise run build && mise run test && mise run lint`
Expected: compiles, tests PASS, lint clean. (No cluster interaction needed here; e2e already proven in Task 2.)

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: rebrand splash/docs to KubeClientlings and remove dead docs-site tasks"
```

---

## Task 4: iximiuz playground manifest + init script

Adds the hosted-lab glue: a bootstrap script (the stable part) and a thin playground manifest.

**Files:**
- Create: `hack/iximiuz-init.sh`
- Create: `playground.yaml`

**Interfaces:**
- Consumes: `mise run up` (creates `kind-kubeclientlings`), `mise run build`, `mise run watch` from the renamed repo.

- [ ] **Step 1: Write the init script**

Create `hack/iximiuz-init.sh`:
```bash
#!/usr/bin/env bash
# Bootstrap KubeClientlings inside an iximiuz Labs playground:
# provision the pinned toolchain, bring up the kind cluster, build the CLI.
set -euo pipefail

REPO_URL="https://github.com/madhank93/kubeclientlings"
WORKDIR="${HOME}/kubeclientlings"

echo "→ Cloning KubeClientlings…"
if [ ! -d "${WORKDIR}/.git" ]; then
  git clone "${REPO_URL}" "${WORKDIR}"
fi
cd "${WORKDIR}"

echo "→ Installing mise + pinned toolchain (go, kind, kubectl)…"
if ! command -v mise >/dev/null 2>&1; then
  curl -fsSL https://mise.run | sh
  export PATH="${HOME}/.local/bin:${PATH}"
fi
eval "$(mise activate bash)"
mise install

echo "→ Creating the kind cluster (kind-kubeclientlings)…"
mise run up

echo "→ Building the kubeclientlings CLI…"
mise run build

cat <<'EOF'

✓ KubeClientlings ready.
  Start the interactive TUI:   mise run watch
  List all exercises:          mise run list
  Check tooling/cluster:       mise run doctor

EOF
```

- [ ] **Step 2: Make it executable and syntax-check it**

```bash
chmod +x hack/iximiuz-init.sh
bash -n hack/iximiuz-init.sh
```
Expected: no output (syntax OK). If `shellcheck` is available: `shellcheck hack/iximiuz-init.sh` — expected no errors (info-level SC notes acceptable).

- [ ] **Step 3: Write the playground manifest**

Create `playground.yaml`:
```yaml
# iximiuz Labs playground for KubeClientlings.
# A single Linux machine with Docker so `kind` can run a local Kubernetes
# cluster; the init task provisions the toolchain and builds the CLI.
kind: playground
name: kubeclientlings
title: KubeClientlings — Learn Kubernetes client-go the rustlings way
description: |
  Fix broken client-go programs to learn the Kubernetes API, one bite-sized
  exercise at a time. The kind cluster and CLI are provisioned on start.
machines:
  - name: dev
    kind: linux
    resources:
      cpuCount: 2
      ramSize: 4Gi
    users:
      - name: laborant
        default: true
initTasks:
  init-kubeclientlings:
    machine: dev
    init: true
    user: laborant
    run: |
      curl -fsSL https://raw.githubusercontent.com/madhank93/kubeclientlings/main/hack/iximiuz-init.sh | bash
```

- [ ] **Step 4: Validate the manifest is well-formed YAML**

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('playground.yaml')); print('yaml ok')"
```
Expected: `yaml ok`. (The iximiuz field schema — machine `kind`, `initTasks` shape — is best-effort per current iximiuz Labs docs; the init script is the load-bearing part and is independently testable.)

- [ ] **Step 5: Commit**

```bash
git add hack/iximiuz-init.sh playground.yaml
git commit -m "feat: add iximiuz playground manifest and bootstrap script"
```

---

## Final verification

- [ ] **Step 1: Full green sweep on the renamed repo**

```bash
mise run build && mise run test && mise run e2e && mise run lint
```
Expected: all pass on `kind-kubeclientlings`.

- [ ] **Step 2: Confirm the rename is total**

```bash
grep -rin "clientlings" --include="*.go" --include="*.toml" --include="*.md" . \
  | grep -vi kubeclientlings | grep -v docs/superpowers
```
Expected: no output.
