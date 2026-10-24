# Shared by the WebAssembly modules' build scripts (services/crates/*/scripts/build-wasm.sh):
# the pinned binaryen release, the target features wasm-opt must keep, and a reproducible
# `cargo build` of one crate followed by wasm-opt. A script sets WASM_TASK (the name its
# messages carry) and SERVICES_DIR, sources this file, and calls build_wasm_module.
#
# The build is reproducible: the same commit gives the same bytes on any machine and in any
# checkout path. The services checkout, the cargo target directory and CARGO_HOME are remapped
# out of every path rustc records, symbols are stripped, and wasm-opt comes from one pinned
# binaryen release, verified against its SHA-256 and cached under services/target/binaryen.
#
# Environment:
#   CARGO_BUILD_JOBS        compile jobs (default 1)
#   TRENOVA_WASM_OPT        a wasm-opt to use instead of the pinned download; the check still
#                           requires it to report the pinned version

BINARYEN_VERSION="133"
BINARYEN_SHA256_X86_64_LINUX="2dc9c7813f5375db93d96ead4b78222fcc3e2677bbb832297af4797782a37489"
BINARYEN_SHA256_AARCH64_LINUX="89c07ea56faf38d0fbecf36ca8ec0721756716185f265b568e133d427f299bf8"
BINARYEN_SHA256_ARM64_MACOS="ad66da82ac13f163e424b1643f16c6dfcccc98b5966296b43e52d3cab04f84a8"
BINARYEN_SHA256_X86_64_MACOS="13a9b90be775c6389ce3d1f879cb8627bea56708ba8c122983941d53a8199b95"

# Exactly the features rustc 1.96 (LLVM 22) enables by default for wasm32-unknown-unknown and
# wasm32-wasip1, which are the same (`rustc --print cfg --target <target>`, plus the two LLVM
# implies with bulk-memory and reference-types), so wasm-opt neither lowers nor rejects any
# instruction.
WASM_FEATURES=(
  --mvp-features
  --enable-bulk-memory
  --enable-bulk-memory-opt
  --enable-call-indirect-overlong
  --enable-multivalue
  --enable-mutable-globals
  --enable-nontrapping-float-to-int
  --enable-reference-types
  --enable-sign-ext
)

die() {
  echo "${WASM_TASK}: $*" >&2
  exit 1
}

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  else
    shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

binaryen_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"
  case "$os/$arch" in
    Linux/x86_64) echo "x86_64-linux" ;;
    Linux/aarch64 | Linux/arm64) echo "aarch64-linux" ;;
    Darwin/arm64) echo "arm64-macos" ;;
    Darwin/x86_64) echo "x86_64-macos" ;;
    *) die "no pinned binaryen release for $os/$arch; set TRENOVA_WASM_OPT to a wasm-opt version_$BINARYEN_VERSION" ;;
  esac
}

binaryen_sha256() {
  case "$1" in
    x86_64-linux) echo "$BINARYEN_SHA256_X86_64_LINUX" ;;
    aarch64-linux) echo "$BINARYEN_SHA256_AARCH64_LINUX" ;;
    arm64-macos) echo "$BINARYEN_SHA256_ARM64_MACOS" ;;
    x86_64-macos) echo "$BINARYEN_SHA256_X86_64_MACOS" ;;
  esac
}

# Prints the path of the pinned wasm-opt, downloading and verifying it on first use.
ensure_wasm_opt() {
  if [ -n "${TRENOVA_WASM_OPT:-}" ]; then
    echo "$TRENOVA_WASM_OPT"
    return
  fi
  local platform expected cache archive staging url
  platform="$(binaryen_platform)"
  expected="$(binaryen_sha256 "$platform")"
  cache="$SERVICES_DIR/target/binaryen/version_${BINARYEN_VERSION}-${platform}"
  if [ -x "$cache/bin/wasm-opt" ] && [ "$(cat "$cache/.sha256" 2>/dev/null)" = "$expected" ]; then
    echo "$cache/bin/wasm-opt"
    return
  fi

  mkdir -p "$SERVICES_DIR/target/binaryen"
  staging="$(mktemp -d "$SERVICES_DIR/target/binaryen/.download.XXXXXX")"
  archive="$staging/binaryen.tar.gz"
  url="https://github.com/WebAssembly/binaryen/releases/download/version_${BINARYEN_VERSION}/binaryen-version_${BINARYEN_VERSION}-${platform}.tar.gz"
  echo "${WASM_TASK}: downloading binaryen version_${BINARYEN_VERSION} ($platform)" >&2
  curl -fsSL --retry 3 --retry-delay 2 -o "$archive" "$url" || {
    rm -rf "$staging"
    die "could not download $url"
  }
  local actual
  actual="$(sha256_of "$archive")"
  if [ "$actual" != "$expected" ]; then
    rm -rf "$staging"
    die "binaryen archive checksum mismatch: expected $expected, got $actual"
  fi
  mkdir -p "$staging/unpacked"
  tar -xzf "$archive" -C "$staging/unpacked" --strip-components=1 \
    "binaryen-version_${BINARYEN_VERSION}/bin/wasm-opt" "binaryen-version_${BINARYEN_VERSION}/lib"
  echo "$expected" >"$staging/unpacked/.sha256"
  rm -rf "$cache"
  mv "$staging/unpacked" "$cache"
  rm -rf "$staging"
  echo "$cache/bin/wasm-opt"
}

# Builds one crate to WebAssembly and optimises it into a file:
#   build_wasm_module <out> <package> <target> <profile> <wasm-opt level>
build_wasm_module() {
  local out="$1" package="$2" target="$3" profile="$4" level="$5"
  local wasm_opt target_dir cargo_home flags version
  wasm_opt="$(ensure_wasm_opt)"
  version="$("$wasm_opt" --version)"
  case "$version" in
    *"version $BINARYEN_VERSION "* | *"version $BINARYEN_VERSION") ;;
    *) die "wasm-opt reports \"$version\"; the build needs binaryen version $BINARYEN_VERSION" ;;
  esac

  cd "$SERVICES_DIR"
  target_dir="$(cargo metadata --format-version 1 --no-deps --offline 2>/dev/null |
    grep -o '"target_directory":"[^"]*"' | cut -d'"' -f4 || true)"
  [ -n "$target_dir" ] || target_dir="$SERVICES_DIR/target"
  cargo_home="${CARGO_HOME:-$HOME/.cargo}"
  local cargo_home_physical
  cargo_home_physical="$(cd "$cargo_home" 2>/dev/null && pwd -P || echo "$cargo_home")"

  # The last matching prefix wins, so the target directory (normally inside services/) is
  # remapped after the checkout. The flags are \x1f-separated so a path may contain spaces.
  local sep=$'\x1f'
  flags="--remap-path-prefix=${SERVICES_DIR}=/trenova/services"
  flags+="${sep}--remap-path-prefix=${cargo_home}=/cargo"
  if [ "$cargo_home_physical" != "$cargo_home" ]; then
    flags+="${sep}--remap-path-prefix=${cargo_home_physical}=/cargo"
  fi
  flags+="${sep}--remap-path-prefix=${target_dir}=/trenova/target"
  # With the rust-src component installed, rustc records the standard library's panic paths
  # under the local sysroot instead of /rustc/<commit>; map them back to what a toolchain
  # without rust-src records.
  local sysroot commit
  sysroot="$(rustc --print sysroot)"
  commit="$(rustc -vV | sed -n 's/^commit-hash: //p')"
  [ -n "$commit" ] || die "rustc -vV reports no commit-hash"
  flags+="${sep}--remap-path-prefix=${sysroot}/lib/rustlib/src/rust=/rustc/${commit}"
  flags+="${sep}-Cstrip=symbols"

  CARGO_BUILD_JOBS="${CARGO_BUILD_JOBS:-1}" \
    CARGO_ENCODED_RUSTFLAGS="$flags" \
    cargo build --locked -p "$package" --profile "$profile" --target "$target" >&2

  local raw="$target_dir/$target/$profile/${package//-/_}.wasm"
  [ -f "$raw" ] || die "cargo produced no $raw"
  "$wasm_opt" "$raw" "$level" "${WASM_FEATURES[@]}" \
    --strip-debug --strip-producers --strip-target-features -o "$out"
}

