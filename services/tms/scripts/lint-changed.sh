#!/usr/bin/env bash
# Lints the Go packages a change touched, not the whole module. With no arguments the packages
# are those holding a .go file changed since the merge base with LINT_BASE (default
# origin/master): commits not yet pushed, staged and unstaged edits, and new untracked files.
# Arguments name packages or directories to lint instead (./internal/foo, internal/foo/...).
#
#   LINT_JOBS     golangci-lint --concurrency (default 4)
#   LINT_TIMEOUT  golangci-lint --timeout (default 5m)
#   LINT_FAST     non-empty runs only the linters golangci-lint marks fast (--fast-only)
#   LINT_BASE     the ref changes are measured from (default origin/master)
set -euo pipefail

cd "$(dirname "$0")/.."

jobs="${LINT_JOBS:-4}"
timeout="${LINT_TIMEOUT:-5m}"
base_ref="${LINT_BASE:-origin/master}"

packages=()
if [ "$#" -gt 0 ]; then
	for arg in "$@"; do
		case "$arg" in
		./* | /*) packages+=("$arg") ;;
		*) packages+=("./$arg") ;;
		esac
	done
else
	base="$(git merge-base HEAD "$base_ref" 2>/dev/null || git rev-parse HEAD)"
	mapfile -t packages < <(
		{
			git diff --relative --name-only --diff-filter=d "$base" -- .
			git ls-files --others --exclude-standard -- .
		} |
			grep -E '\.go$' |
			grep -vE '(^|/)testdata/' |
			xargs -r -n1 dirname |
			sort -u |
			while read -r dir; do
				if [ -d "$dir" ]; then
					printf './%s\n' "$dir"
				fi
			done
	)
fi

if [ "${#packages[@]}" -eq 0 ]; then
	echo "lint: no Go package changed since ${base_ref}; nothing to lint"
	exit 0
fi

echo "lint: ${#packages[@]} package(s), concurrency ${jobs}, timeout ${timeout}${LINT_FAST:+, fast linters only}"

args=(run "--concurrency=${jobs}" "--timeout=${timeout}")
if [ -n "${LINT_FAST:-}" ]; then
	args+=(--fast-only)
fi

exec golangci-lint "${args[@]}" "${packages[@]}"
