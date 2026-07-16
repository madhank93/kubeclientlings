# KubeClientlings — Rename + iximiuz Playground

**Date:** 2026-07-16
**Status:** Approved design, pending implementation plan

## Goal

Two independent deliverables in one session:

1. **Rename** the project `clientlings` → `KubeClientlings` across the whole repo (branding, Go module, binary, directory, state file, CRD group / namespace label).
2. **iximiuz playground**: ship a `playground.yaml` manifest + `hack/iximiuz-init.sh` bootstrap so anyone can run the exercises in a hosted browser lab with no local setup.

Plus two cleanups folded in: fix stale splash strings, and rip out the dead docs-site tasks.

## Non-Goals

- Building the docs website (`web/`) — it does not exist; we remove the broken references instead.
- Renaming the GitHub remote or `learn-client-go` working directory (out of scope; module path is what matters for `go` tooling).
- Any new exercises or curriculum changes.
- controller-runtime operator topic (previously and still deliberately omitted).

## Naming Convention

| Layer | Old | New |
|---|---|---|
| Display / brand | `clientlings` | `KubeClientlings` |
| Go module path | `github.com/madhank93/clientlings` | `github.com/madhank93/kubeclientlings` |
| Binary | `clientlings` | `kubeclientlings` |
| Source dir | `clientlings/` | `kubeclientlings/` |
| State file | `.clientlings-state.json` | `.kubeclientlings-state.json` |
| CRD group / ns label | `clientlings.dev` | `kubeclientlings.dev` |

Rule: lowercase `kubeclientlings` for all identifiers/paths; CamelCase `KubeClientlings` only for human-facing display text (splash title, README headings, taglines).

## Part A — Rename

### Scope (measured)
- 124 files contain `clientlings`/`Clientlings`; 252 total string occurrences.
- 111 Go files import `madhank93/clientlings`.
- CRD group `clientlings.dev` appears in ~20 paired starter+solution files (crds, dynamic, subresources). These MUST change together or e2e breaks.

### Execution order (each step must leave the tree buildable before the next)
1. `go.mod`: module path → `github.com/madhank93/kubeclientlings`.
2. `git mv clientlings/ kubeclientlings/` (preserve history).
3. Rewrite all Go imports `madhank93/clientlings` → `madhank93/kubeclientlings`.
4. Rewrite identifiers/paths: binary name, state file const (`StateFile`), ns label const (`ExerciseLabel`), CRD group strings in exercises **and** matching solutions.
5. Rewrite branding strings: splash title/tagline, README(s), mise.toml header/comments, info.toml.
6. Fix stale splash: `"112 exercises"` → real count (derive from loaded exercise list, not hardcoded); drop "Learn Go the rustlings way" generic-Go framing → client-go/Kubernetes framing.
7. Update `mise.toml` task command paths (`./bin/clientlings` → `./bin/kubeclientlings`, `go build -o bin/kubeclientlings ./kubeclientlings`, lint/test package globs).

### Verification (gate — all must pass)
- `mise run build` — compiles.
- `mise run test` — unit tests green (race).
- `mise run e2e` — overlay solutions onto exercises, verify all 49 against kind, restore. This is the real proof the CRD-group rename stayed in sync.
- `mise run lint` — clean.
- `grep -ri clientlings` returns only intentional history/spec references (ideally zero in source).

### Risks
- **CRD group drift**: a starter using `kubeclientlings.dev` but its solution still on `clientlings.dev` (or vice-versa) fails silently at e2e. Mitigation: single sed pass over both trees, then e2e.
- **Orphaned progress**: state-file rename resets any in-progress `.clientlings-state.json`. Acceptable for a rename; note in commit.
- **Import path partial rewrite**: a missed import fails the build immediately (step 3 gated by `go build`).

## Part B — iximiuz Playground

### Deliverables
- `playground.yaml` — iximiuz Labs playground manifest describing the machine(s), base image, and init hook.
- `hack/iximiuz-init.sh` — bootstrap run on playground start.

### init script flow
1. Ensure toolchain: install `mise` (or rely on base image), `mise install` to pull go/kind/kubectl pinned in `mise.toml`.
2. Bring up the cluster: `mise run up` (kind `kind-kubeclientlings`) — or bind to an iximiuz-provided cluster if the playground supplies one (decide at implementation from iximiuz machine capabilities).
3. Build: `mise run build`.
4. Greeting: print how to start (`mise run watch`) or auto-launch the TUI in the initial shell.

### playground.yaml shape (to confirm against iximiuz schema at implementation)
- Single Linux machine with Docker (kind needs a container runtime).
- Repo made available (git clone on init, or mounted).
- `initTasks` / init hook → `hack/iximiuz-init.sh`.
- Welcome markdown pointing the learner at `mise run watch`.

### Open implementation detail
iximiuz playground manifest schema specifics (field names, machine kinds, whether kind-in-Docker is supported vs a provided k8s) must be checked against current iximiuz docs during implementation. The init script is the stable part; the manifest is thin glue.

### Verification
- `bash -n hack/iximiuz-init.sh` (syntax) + shellcheck clean.
- Manifest validates against iximiuz schema (or documented as best-effort if schema unavailable offline).
- Dry-run the init flow locally where possible (mise install / up / build / watch already proven by Part A).

## Cleanup — Dead docs site

- Remove `gen-site`, `site`, `site-build` mise tasks (point at non-existent `web/` dir).
- Remove the `clientlings.madhan.app` site link from the TUI splash.
- Leave `Repo` link (rewritten to kubeclientlings) and maintainer line.

## Rollout

Single feature branch off `main`. Commit sequence:
1. `refactor: rename clientlings → KubeClientlings` (Part A, after e2e green).
2. `chore: remove dead docs-site tasks and splash link`.
3. `feat: add iximiuz playground manifest and init script` (Part B).

Each commit independently builds; e2e green before the first commit lands.
