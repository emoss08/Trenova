#!/usr/bin/env bash
# Builds the trenova-edi WebAssembly module and installs it, or checks the committed copy.
#
#   build-wasm.sh install   build, optimise and write the committed copy and its .sha256
#   build-wasm.sh check     build into a temporary directory and fail when the committed copy
#                           or its .sha256 differs from the fresh build
#
# One wasm32-unknown-unknown module with no imports (docs/engineering/edi-x12.md), built with
# the size-tuned wasm profile because the browser will run the same bytes:
#
#   services/tms/pkg/x12/edi.wasm                     embedded by the Go host (wazero)
#
# The build itself (pinned binaryen, path remapping, reproducibility) is services/scripts/wasm-module.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
SERVICES_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd -P)"
REPO_ROOT="$(cd "$SERVICES_DIR/.." && pwd -P)"
WASM_TASK="rust:wasm-edi"
# shellcheck source=../../../scripts/wasm-module.sh
source "$SERVICES_DIR/scripts/wasm-module.sh"
DESTINATION="services/tms/pkg/x12"

WORK=""
cleanup() {
  [ -z "$WORK" ] || rm -rf "$WORK"
}
trap cleanup EXIT

build() {
  build_wasm_module "$1" trenova-edi wasm32-unknown-unknown wasm -Os
}

write_sha256() {
  echo "$(sha256_of "$1")  edi.wasm" >"$2"
}

install() {
  WORK="$(mktemp -d)"
  build "$WORK/edi.wasm"
  mkdir -p "$REPO_ROOT/$DESTINATION"
  cp "$WORK/edi.wasm" "$REPO_ROOT/$DESTINATION/edi.wasm"
  write_sha256 "$WORK/edi.wasm" "$REPO_ROOT/$DESTINATION/edi.wasm.sha256"
  echo "rust:wasm-edi: wrote $DESTINATION/edi.wasm ($(wc -c <"$WORK/edi.wasm" | tr -d ' ') bytes, sha256 $(sha256_of "$WORK/edi.wasm"))"
}

check() {
  local failed=0
  WORK="$(mktemp -d)"
  build "$WORK/edi.wasm"
  write_sha256 "$WORK/edi.wasm" "$WORK/edi.wasm.sha256"
  for name in edi.wasm edi.wasm.sha256; do
    if ! cmp -s "$WORK/$name" "$REPO_ROOT/$DESTINATION/$name"; then
      echo "rust:wasm-edi: $DESTINATION/$name differs from a fresh build of services/crates/edi" >&2
      failed=1
    fi
  done
  if [ "$failed" -ne 0 ]; then
    die "run 'task rust:wasm-edi' and commit the copy; never edit edi.wasm by hand"
  fi
  echo "rust:wasm-edi: the committed copy matches a fresh build (sha256 $(sha256_of "$WORK/edi.wasm"))"
}

case "${1:-}" in
  install) install ;;
  check) check ;;
  *) die "usage: build-wasm.sh install|check" ;;
esac
