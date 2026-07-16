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
