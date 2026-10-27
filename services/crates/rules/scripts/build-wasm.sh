#!/usr/bin/env bash
# Builds the trenova-rules WebAssembly module and installs it, or checks the committed copies.
#
#   build-wasm.sh install   build, optimise and write both committed copies and their .sha256,
#                           and the browser's unit table generated from the crate
#   build-wasm.sh check     build into a temporary directory and fail, naming the file, when a
#                           committed copy or its .sha256 differs from the fresh build, or the
#                           unit table differs from the crate's
#
# One wasm32-unknown-unknown module serves both hosts (docs/engineering/rules-engine.md):
#
#   services/tms/pkg/rules/rules.wasm                 embedded by the Go host (wazero)
#   client/packages/shared/src/rules/rules.wasm       loaded by the browser
#
# The browser also reads the unit table before the module loads, so
# client/packages/shared/src/rules/tariff-units.generated.ts is written from the crate's
# (tests/client_units.rs, run with TRENOVA_WRITE_GENERATED=1).
#
# The build itself (pinned binaryen, path remapping, reproducibility) is services/scripts/wasm-module.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
SERVICES_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd -P)"
REPO_ROOT="$(cd "$SERVICES_DIR/.." && pwd -P)"
WASM_TASK="rust:wasm-rules"
# shellcheck source=../../../scripts/wasm-module.sh
source "$SERVICES_DIR/scripts/wasm-module.sh"
DESTINATIONS=(
  "services/tms/pkg/rules"
  "client/packages/shared/src/rules"
)

# Writes the browser's unit table from the crate's, or with "check" fails when it differs.
unit_table() {
  cd "$SERVICES_DIR"
  if [ "$1" = "write" ]; then
    CARGO_BUILD_JOBS="${CARGO_BUILD_JOBS:-1}" TRENOVA_WRITE_GENERATED=1 \
      cargo test --locked -p trenova-rules --test client_units >&2
    echo "rust:wasm-rules: wrote client/packages/shared/src/rules/tariff-units.generated.ts"
  else
    CARGO_BUILD_JOBS="${CARGO_BUILD_JOBS:-1}" \
      cargo test --locked -p trenova-rules --test client_units >&2 ||
      die "client/packages/shared/src/rules/tariff-units.generated.ts differs from the crate's unit table"
  fi
}

write_sha256() {
  echo "$(sha256_of "$1")  rules.wasm" >"$2"
}

WORK=""
cleanup() {
  [ -z "$WORK" ] || rm -rf "$WORK"
}
trap cleanup EXIT

install() {
  WORK="$(mktemp -d)"
  local work="$WORK"
  build_wasm_module "$work/rules.wasm" trenova-rules wasm32-unknown-unknown wasm -Os
  for dest in "${DESTINATIONS[@]}"; do
    mkdir -p "$REPO_ROOT/$dest"
    cp "$work/rules.wasm" "$REPO_ROOT/$dest/rules.wasm"
    write_sha256 "$work/rules.wasm" "$REPO_ROOT/$dest/rules.wasm.sha256"
    echo "rust:wasm-rules: wrote $dest/rules.wasm ($(wc -c <"$work/rules.wasm" | tr -d ' ') bytes, sha256 $(sha256_of "$work/rules.wasm"))"
  done
  unit_table write
}

check() {
  local failed=0
  WORK="$(mktemp -d)"
  local work="$WORK"
  build_wasm_module "$work/rules.wasm" trenova-rules wasm32-unknown-unknown wasm -Os
  write_sha256 "$work/rules.wasm" "$work/rules.wasm.sha256"
  for dest in "${DESTINATIONS[@]}"; do
    for name in rules.wasm rules.wasm.sha256; do
      if ! cmp -s "$work/$name" "$REPO_ROOT/$dest/$name"; then
        echo "rust:wasm-rules: $dest/$name differs from a fresh build of services/crates/rules" >&2
        failed=1
      fi
    done
  done
  if [ "$failed" -ne 0 ]; then
    die "run 'task rust:wasm-rules' and commit both copies; never edit rules.wasm by hand"
  fi
  unit_table check
  echo "rust:wasm-rules: both committed copies match a fresh build (sha256 $(sha256_of "$work/rules.wasm"))"
}

case "${1:-}" in
  install) install ;;
  check) check ;;
  *) die "usage: build-wasm.sh install|check" ;;
esac
