---
title: Getting started
description: Install KubeClientlings, create the kind cluster, and start the interactive exercise runner.
---

## Prerequisites

- **Docker** (or a Docker-compatible runtime) — [kind](https://kind.sigs.k8s.io/)
  runs the cluster in a container.
- **[mise](https://mise.jdx.dev/)** — pins the whole toolchain (Go, kind,
  kubectl, golangci-lint), so you don't need any of it installed globally.

```sh
brew install mise        # macOS; see mise docs for other platforms
```

## Set up

```sh
git clone https://github.com/madhank93/kubeclientlings
cd kubeclientlings
mise install             # provisions Go, kind, kubectl, golangci-lint
mise run up              # creates the kind-kubeclientlings cluster
mise run watch           # launches the interactive TUI
```

## Run it in the browser

No Docker locally? The
[iximiuz Labs playground](https://labs.iximiuz.com/playgrounds/kubeclientlings)
provisions a Linux machine with the kind cluster and CLI ready — nothing to
install.

## How it works

1. The TUI highlights the next unfinished exercise and shows its file path.
2. Open that file and fix the code — each exercise is broken on purpose.
3. Remove the `// I AM NOT DONE` marker when you think it's done.
4. **Save** — the exercise re-runs. It only advances when it compiles, the
   linter is clean, and the checks against the kind cluster pass.

Useful commands:

```sh
mise run list                          # all exercises + progress
./bin/kubeclientlings run <exercise>   # run one exercise
./bin/kubeclientlings hint <exercise>  # show its hint
./bin/kubeclientlings reset <exercise> # restore the original broken file
mise run doctor                        # check tooling + cluster health
mise run down                          # delete the kind cluster
```

## Stuck?

Every exercise has a hint in the TUI (or `kubeclientlings hint <name>`), and the
[catalog](/catalog/) shows each exercise's code and hint with a link to the
worked solution on GitHub.
