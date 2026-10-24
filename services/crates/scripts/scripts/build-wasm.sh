#!/usr/bin/env bash
# Builds the trenova-scripts WebAssembly module and installs it, or checks the committed copy.
#
#   build-wasm.sh install   build, optimise and write the committed copy and its .sha256
#   build-wasm.sh check     build into a temporary directory and fail when the committed copy
#                           or its .sha256 differs from the fresh build
#
# One wasm32-wasip1 module, run only by the TMS (docs/engineering/agent-code-mode.md):
#
#   services/tms/pkg/scriptvm/scripts.wasm            embedded by the Go host (wazero)
#
# The build itself (pinned binaryen, path remapping, reproducibility) is services/scripts/wasm-module.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
SERVICES_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd -P)"
REPO_ROOT="$(cd "$SERVICES_DIR/.." && pwd -P)"
WASM_TASK="rust:wasm-scripts"
# shellcheck source=../../../scripts/wasm-module.sh
source "$SERVICES_DIR/scripts/wasm-module.sh"
DESTINATION="services/tms/pkg/scriptvm"

WORK=""
cleanup() {
  [ -z "$WORK" ] || rm -rf "$WORK"
}
trap cleanup EXIT

build() {
  build_wasm_module "$1" trenova-scripts wasm32-wasip1 wasm-server -O3
}

write_sha256() {
  echo "$(sha256_of "$1")  scripts.wasm" >"$2"
}

install() {
  WORK="$(mktemp -d)"
  build "$WORK/scripts.wasm"
  mkdir -p "$REPO_ROOT/$DESTINATION"
  cp "$WORK/scripts.wasm" "$REPO_ROOT/$DESTINATION/scripts.wasm"
  write_sha256 "$WORK/scripts.wasm" "$REPO_ROOT/$DESTINATION/scripts.wasm.sha256"
  echo "rust:wasm-scripts: wrote $DESTINATION/scripts.wasm ($(wc -c <"$WORK/scripts.wasm" | tr -d ' ') bytes, sha256 $(sha256_of "$WORK/scripts.wasm"))"
}

check() {
  local failed=0
  WORK="$(mktemp -d)"
  build "$WORK/scripts.wasm"
  write_sha256 "$WORK/scripts.wasm" "$WORK/scripts.wasm.sha256"
  for name in scripts.wasm scripts.wasm.sha256; do
    if ! cmp -s "$WORK/$name" "$REPO_ROOT/$DESTINATION/$name"; then
      echo "rust:wasm-scripts: $DESTINATION/$name differs from a fresh build of services/crates/scripts" >&2
      failed=1
    fi
  done
  if [ "$failed" -ne 0 ]; then
    die "run 'task rust:wasm-scripts' and commit the copy; never edit scripts.wasm by hand"
  fi
  echo "rust:wasm-scripts: the committed copy matches a fresh build (sha256 $(sha256_of "$WORK/scripts.wasm"))"
}

case "${1:-}" in
  install) install ;;
  check) check ;;
  *) die "usage: build-wasm.sh install|check" ;;
esac
