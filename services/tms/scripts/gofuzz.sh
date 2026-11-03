#!/usr/bin/env bash
# Runs the Go fuzz tests: files tagged `//go:build fuzz` in the TMS and shared modules.
#
#   scripts/gofuzz.sh seeds                 every target's seed inputs, as ordinary tests
#   scripts/gofuzz.sh run [time] [name]     fuzz each target (or only `name`) for `time` (30s)
#
# Run from services/tms. Targets fuzz one at a time: `go test -fuzz` takes one target in one
# package, and running them side by side saturates the machine.
set -euo pipefail

mode="${1:-seeds}"
fuzztime="${2:-30s}"
only="${3:-}"
tags="fuzz,nomsgpack"
modules=("../../shared" ".")

fuzz_files() {
  (cd "$1" && git grep --untracked -l '^//go:build fuzz' -- '*_test.go' || true)
}

case "$mode" in
  seeds)
    for module in "${modules[@]}"; do
      packages=$(fuzz_files "$module" | xargs -r -n1 dirname | sort -u | sed 's|^|./|')
      if [ -n "$packages" ]; then
        # shellcheck disable=SC2086 # one argument per package
        (cd "$module" && go test -tags="$tags" -run '^Fuzz' $packages)
      fi
    done
    ;;
  run)
    found=0
    for module in "${modules[@]}"; do
      while IFS= read -r file; do
        [ -n "$file" ] || continue
        for name in $(grep -oE '^func Fuzz[A-Za-z0-9_]+' "$module/$file" | sed 's/^func //'); do
          if [ -n "$only" ] && [ "$name" != "$only" ]; then
            continue
          fi
          found=1
          echo "::group::$name ($file)"
          (cd "$module" && go test -tags="$tags" -run '^$' -fuzz "^${name}\$" \
            -fuzztime "$fuzztime" "./$(dirname "$file")")
          echo "::endgroup::"
        done
      done < <(fuzz_files "$module")
    done
    if [ "$found" -eq 0 ]; then
      echo "no fuzz target named ${only:-anything}" >&2
      exit 1
    fi
    ;;
  *)
    echo "usage: scripts/gofuzz.sh seeds | run [time] [name]" >&2
    exit 2
    ;;
esac
