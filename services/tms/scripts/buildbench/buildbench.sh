#!/usr/bin/env bash
#
# Measures how long services/tms takes to rebuild after representative edits.
#
#   buildbench.sh [--target cli|api|worker] [--mode release|air] [--strip] [--cold]
#   buildbench.sh --fanin
#   buildbench.sh --deps-of <target> <pattern>
#
# Each scenario appends a probe to a real source file, rebuilds, and restores the
# file from a byte-for-byte backup. The restore also runs from an EXIT trap, so an
# interrupted run never leaves a probe behind. The script never runs git checkout,
# git restore or git stash, and refuses to touch a file with uncommitted changes.
#
# Builds use their own cache, $(go env GOCACHE)-buildbench, so they neither reuse
# nor evict the cache your editor and `task` share. --cold empties only that cache.
# Builds run under nice with -p set to the CPU count minus one.
#
# Results are printed as a table and written to
# scripts/buildbench/results/<git-sha>-<target>-<mode>[-strip][-cold].json, so a
# cold run and the warm run after it are kept side by side. The first edit after a
# cold build recompiles more than the same edit does later, so compare S1-S6 from
# warm runs.

set -euo pipefail
export LC_ALL=C

readonly TAGS=nomsgpack

die() {
	printf 'buildbench: %s\n' "$*" >&2
	exit 1
}

usage() {
	sed -n '3,22p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

((BASH_VERSINFO[0] >= 5)) || die "needs bash 5 or later for EPOCHREALTIME"

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
module_dir=$(cd "$script_dir/../.." && pwd)
cd "$module_dir"

action=bench
target=cli
mode=release
strip=false
cold=false
deps_target=
deps_pattern=

while (($# > 0)); do
	case "$1" in
	--target)
		(($# >= 2)) || die "--target needs a value"
		target=$2
		shift 2
		;;
	--mode)
		(($# >= 2)) || die "--mode needs a value"
		mode=$2
		shift 2
		;;
	--strip)
		strip=true
		shift
		;;
	--cold)
		cold=true
		shift
		;;
	--fanin)
		action=fanin
		shift
		;;
	--deps-of)
		(($# >= 3)) || die "--deps-of needs a target and a pattern"
		action=deps-of
		deps_target=$2
		deps_pattern=$3
		shift 3
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		die "unknown argument: $1 (see --help)"
		;;
	esac
done

target_package() {
	case "$1" in
	cli | api | worker) ;;
	*) die "unknown target '$1' (cli, api or worker)" ;;
	esac
	[[ -d "cmd/$1" ]] || die "target '$1' needs ./cmd/$1, which does not exist yet"
	printf './cmd/%s' "$1"
}

case "$mode" in
release | air) ;;
*) die "unknown mode '$mode' (release or air)" ;;
esac

export GOFLAGS=
cpus=$(getconf _NPROCESSORS_ONLN)
jobs=$((cpus > 1 ? cpus - 1 : 1))
nice_cmd=(nice -n 10)

work=$(mktemp -d "${TMPDIR:-/tmp}/buildbench.XXXXXX")
active_file=
active_backup=

restore_probe() {
	[[ -n "$active_file" ]] || return 0
	cp -p -- "$active_backup" "$active_file"
	if ! cmp -s -- "$active_backup" "$active_file"; then
		printf 'buildbench: could not restore %s; its original is at %s\n' \
			"$active_file" "$active_backup" >&2
		return 1
	fi
	active_file=
}

cleanup() {
	local status=$?
	if ! restore_probe; then
		exit 1
	fi
	rm -rf -- "$work"
	exit "$status"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

helper="$work/buildbench"
"${nice_cmd[@]}" go build -p "$jobs" -o "$helper" ./scripts/buildbench

case "$action" in
fanin)
	"${nice_cmd[@]}" "$helper" fanin -tags "$TAGS" -top 30
	exit 0
	;;
deps-of)
	"${nice_cmd[@]}" "$helper" deps-of -tags "$TAGS" \
		-target "$(target_package "$deps_target")" -pattern "$deps_pattern"
	exit 0
	;;
esac

pkg=$(target_package "$target")
module=$(sed -n 's/^module[[:space:]]\{1,\}//p' go.mod | head -n 1)
[[ -n "$module" ]] || die "cannot read the module path from go.mod"
cache="$(go env GOCACHE)-buildbench"
bin="$work/$target"

build_args=(-p "$jobs" -tags "$TAGS")
mode_name=$mode
if [[ "$mode" == air ]]; then
	build_args+=("-gcflags=$module/...=-N -l")
fi
if [[ "$strip" == true ]]; then
	build_args+=("-ldflags=-s -w")
	mode_name="$mode-strip"
fi

probes="$work/probes.tsv"
"${nice_cmd[@]}" "$helper" probes -tags "$TAGS" -target "$pkg" >"$probes"

while IFS=$'\t' read -r _ _ _ file _; do
	[[ -f "$file" ]] || die "probe file $file does not exist"
	if [[ -n "$(git status --porcelain -- "$file")" ]]; then
		die "$file has uncommitted changes; commit or set them aside first"
	fi
done <"$probes"

sha=$(git rev-parse --short=12 HEAD)
dirty=false
if [[ -n "$(git status --porcelain)" ]]; then
	dirty=true
fi
started=$(date -u +%Y-%m-%dT%H:%M:%SZ)

if [[ "$cold" == true ]]; then
	mode_name="$mode_name-cold"
fi
if [[ "$cold" == true && -d "$cache" ]]; then
	GOCACHE="$cache" go clean -cache
fi
mkdir -p "$cache"

results=()

measure() {
	local id=$1 kind=${2:-} probed=${3:-} file=${4:-} dependents=${5:-0}
	local graph="$work/$id.actiongraph.json" result="$work/$id.json" start end

	printf 'buildbench: %s\n' "$id" >&2
	start=${EPOCHREALTIME/./}
	GOCACHE="$cache" "${nice_cmd[@]}" go build "${build_args[@]}" \
		-debug-actiongraph="$graph" -o "$bin" "$pkg"
	end=${EPOCHREALTIME/./}

	"$helper" analyze -graph "$graph" -module "$module" -binary "$bin" \
		-wall-ns "$(((end - start) * 1000))" -id "$id" -kind "$kind" -package "$probed" \
		-file "$file" -dependents "$dependents" >"$result"
	results+=("$result")
}

apply_probe() {
	local kind=$1 file=$2 nonce=${EPOCHREALTIME/./}
	if [[ -n "$(git status --porcelain -- "$file")" ]]; then
		die "$file changed during the run; not probing it"
	fi
	active_backup="$work/probe.backup"
	cp -p -- "$file" "$active_backup"
	active_file=$file
	case "$kind" in
	unexported-func) printf '\nfunc buildbenchProbe%s() int { return 0 }\n' "$nonce" >>"$file" ;;
	exported-func) printf '\nfunc BuildbenchProbe%s() int { return 0 }\n' "$nonce" >>"$file" ;;
	exported-type) printf '\ntype BuildbenchProbe%s struct{}\n' "$nonce" >>"$file" ;;
	*) die "unknown probe kind $kind" ;;
	esac
}

if [[ "$cold" == true ]]; then
	measure cold
else
	measure warm
fi
measure S1

while IFS=$'\t' read -r id kind probed file dependents; do
	apply_probe "$kind" "$file"
	measure "$id" "$kind" "$probed" "$file" "$dependents"
	restore_probe
done <"$probes"

go_version=$(go env GOVERSION)
"$helper" report \
	-out "$script_dir/results/$sha-$target-$mode_name.json" \
	-sha "$sha" -dirty="$dirty" -target "$target" -package "$pkg" -mode "$mode" \
	-strip="$strip" -build-args "$(
		IFS=$'\t'
		printf '%s' "${build_args[*]}"
	)" \
	-jobs "$jobs" -cpus "$cpus" -go-version "$go_version" -gocache "$cache" \
	-started "$started" "${results[@]}"
