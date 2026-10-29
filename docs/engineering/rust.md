# Rust in Trenova

How Rust fits into the repository: where each kind of Rust code goes, how to choose between
the four ways Rust reaches the product, the two crates every service is built on, and the
conventions every crate follows. Read this before adding or changing Rust code.

The first service built this way is the imaging worker; see
[imaging-worker.md](imaging-worker.md). The survey that chose it, and the candidates after it,
is in [../notes/rust/opportunities.md](../notes/rust/opportunities.md).

## The decision

- **`services/` is the backend.** It holds every backend service, whatever its language, and
  the libraries and contracts those services share. `services/tms` is the Go service;
  `services/imaging` is the first Rust one.
- **One Cargo workspace, rooted at `services/`,** covers every backend Rust crate: one
  `Cargo.lock`, one toolchain (`services/rust-toolchain.toml`), one lint set, one dependency and
  supply-chain policy (`services/deny.toml`). `services/tms` and `services/postgres` are not
  members.
- **Every deployable is its own crate, binary and container,** a peer of `services/tms`.
  Unrelated Rust services are never combined into one process: they scale on different
  resources, and a fault in one must not take the other with it.
- **Pure computation Go or the browser needs is a library crate** in `services/crates`,
  compiled to WebAssembly (below). It is not a service.
- **Code that needs native libraries or process isolation is a service,** called over gRPC.
- **`desktop/capture` stays where it is,** with its own workspace: a Windows product with its own
  release tags, MSI, signing and version flow. It depends on shared crates in `services/crates`
  by path (today `trenova-imaging-core`), and CI fails when its toolchain pin differs from
  `services/rust-toolchain.toml`.

Why one workspace rather than one per service: shared crates (`service-kit`, `sandbox`,
`proto`, `imaging-core`) are edited together with their users, a dependency is audited and
upgraded once, and two services cannot quietly drift onto different majors of `tonic` or
`rustls`. Why services stay separate processes: the imaging worker is sized by CPU and memory
per document, a request-serving service by request rate; one container cannot be sized for both,
and a crash in a PDF parser must never stop anything else.

## Layout

```text
services/
  Cargo.toml              [workspace] members = ["imaging", "crates/*"]; dependencies, lints, profiles
  Cargo.lock
  rust-toolchain.toml     1.96 (the channel desktop/capture pins), rustfmt, clippy, the wasm32-unknown-unknown and wasm32-wasip1 targets
  deny.toml               cargo-deny: advisories, licences, duplicate majors, sources
  clippy.toml             words doc comments may use unquoted (PDFium, WebP, ...)
  .cargo/config.toml      only what every build needs
  proto/                  contracts between services (buf); packages trenova.<domain>.v1
  crates/                 library crates, each named trenova-<name>
    proto/                trenova-proto: Rust code for services/proto, generated in build.rs
    service-kit/          trenova-service-kit: the chassis every service runs on
    sandbox/              trenova-sandbox: a supervised pool of hardened child processes
    imaging-core/         trenova-imaging-core: platform-neutral page imaging (shared with capture)
    rules/                trenova-rules: the rules engine, one WebAssembly module for the TMS and the browser
    scripts/              trenova-scripts: agent scripts in Monty, a WebAssembly module only the TMS runs
    edi/                  trenova-edi: X12 reading, guides, validation and acknowledgments, one WebAssembly module for the TMS and the browser
    module-kit/           trenova-module-kit: the ABI macro, operation dispatch and `message!` every browser module shares
  scripts/wasm-module.sh  the reproducible WebAssembly build every module's build-wasm.sh sources
  vendor/monty/           Monty with one patch (vendor/monty/TRENOVA.md); excluded from the workspace
  imaging/                trenova-imaging: the imaging worker (a deployable)
  tms/                    the Go service
  postgres/<ext>/         future pgrx extensions; outside the workspace, built by deploy/Dockerfile.postgres
desktop/capture/          own workspace; path dependency on ../../services/crates/imaging-core
deploy/Dockerfile.<svc>   one image per deployable; build context services/
```

| Kind | Where | Named | Ships as |
|---|---|---|---|
| Deployable service | `services/<name>/` | `trenova-<name>` | a container image (`deploy/Dockerfile.<name>`) |
| Library | `services/crates/<name>/` | `trenova-<name>` | nothing on its own; linked into services, desktop or WASM |
| Contract | `services/proto/trenova/<domain>/v1/` | package `trenova.<domain>.v1` | Rust at build time, Go checked in |
| Postgres extension | `services/postgres/<ext>/` | the extension's name | built into the Postgres image |
| Fuzz targets | `<crate>/fuzz/` | `<crate>-fuzz` | CI only; outside the workspace |

## Choosing a seam

Rust reaches the product four ways. Choose by what the code needs and what crossing the
boundary costs.

| Seam | Use it for | Costs |
|---|---|---|
| WebAssembly in the Go process (`wasm32-unknown-unknown`, run by wazero) | Pure computation the TMS calls often: parsers, rule engines, encoders, checks | Slower than native (no SIMD by default, a copy across linear memory); every boundary call costs, so the API must take batches; no threads, no native libraries |
| WebAssembly in the browser (the same module) | Logic the browser and the server must agree on to the cent | A download of its own, kept out of the first-load budget (`client/apps/web/bundle-budget.json`) by loading it lazily |
| A gRPC service | Native libraries (PDFium, codecs), untrusted input that needs process isolation, work that must scale apart from the TMS | Another container to run, size and monitor; a contract to version; plaintext crossing a network link (mutual TLS) |
| A Postgres extension (pgrx) | Functions on every statement's path where a round trip is the cost | Rules out managed Postgres; native code inside the database's trust boundary; upgrades tied to the Postgres image |

A rule of thumb: if it needs no native library and no isolation from the TMS, it is a library
compiled to WASM (one module for both hosts, below); if it parses files nobody vetted, it is a service, and its parsing runs in
`trenova-sandbox`.

## The building blocks every service uses

**`trenova-service-kit`** is the chassis. A service does not set up these things itself:

- `config`: one typed configuration from an optional TOML file (`--config` or
  `<PREFIX>_CONFIG`) and `<PREFIX>__SECTION__KEY` environment overrides, decoded with unknown
  fields denied and validated before anything starts, every problem reported at once.
- `telemetry`: JSON logs on stderr, OTLP traces, and the W3C trace context Go's `otelgrpc`
  sends taken as each call's parent.
- `metrics`: a shared Prometheus registry and a listener for `/metrics`, `/livez` and
  `/readyz`, apart from the gRPC port so scrapers and healthchecks need no client certificate.
- `grpc`: the tonic server with mutual TLS (required in production), message and stream limits,
  keepalives and a span per call. Never server reflection.
- `shutdown`: SIGTERM and SIGINT start a drain bounded by a grace period.
- `admission`: a per-process budget of cost units and memory, interactive and batch priority
  classes with a reserve batch work cannot use, a bounded wait, and a `RESOURCE_EXHAUSTED`
  refusal with a retry-after taken from how long recent jobs held their permits.

**`trenova-sandbox`** runs untrusted-input parsing in disposable child processes of the
service's own binary: framed IPC on the child's stdin socket with inputs as sealed memfds,
no-new-privs, landlock, stacked seccomp filters, rlimits set between fork and exec, an empty
environment, recycling by job count, RSS and CPU, deadline kills through a pidfd, and crash
containment (one crash fails one job). It fails closed. Any service that parses files from
outside, and the future code-mode sandbox, is built on it. Children are forked only from the
pool's own long-lived spawner thread: their death signal fires when the forking *thread* exits,
and a child forked from tokio's blocking pool was killed ten idle seconds later.

## Contracts

- Protobuf plus buf is the one contract between Go and Rust services. Definitions live in
  `services/proto` (`buf.yaml`: STANDARD lint, FILE breaking rules).
- Rust code is generated at build time by `trenova-proto` with protox, so no `protoc` is
  installed anywhere.
- Go code is generated by buf into the TMS adapter package that consumes it (for imaging,
  `services/tms/internal/infrastructure/imaging/imagingpb`) and checked in. The plugins are
  pinned and run through `go run`.
- `task rust:proto` lints and regenerates; `task rust:proto-breaking` fails on a breaking change
  against `origin/master`. CI runs both and fails when the checked-in Go is stale.
- A breaking change to a released contract is a new version package (`trenova.imaging.v2`).
- Errors are `google.rpc.Status` with an `ErrorInfo` (a fixed domain, a reason from the
  contract's enum) and, when the caller should come back, a `RetryInfo`.

## Conventions

- Edition 2024, the workspace lint set (clippy `pedantic` as warnings, `unwrap_used`,
  `undocumented_unsafe_blocks`, `print_stdout`/`print_stderr`, `missing_debug_implementations`);
  CI runs clippy with `-D warnings`.
- Rust follows desktop/capture's style, including its doc comments: a module says what it is
  for and why it works the way it does; functions say what a caller needs to know. Go code
  follows AGENTS.md (no comments).
- `thiserror` in libraries; structured errors at service edges (the contract's reasons).
- No document content in any log, metric label or error message: sizes, counts, durations,
  page numbers and error kinds only.
- Every `unsafe` block carries a `// SAFETY:` comment saying why it holds. Prefer a safe crate
  (`rustix`, `libloading`, `pdfium-render`) to raw `libc`; FFI to a native library is bound in
  one module that wraps it in safe types.

### Profiles

| Profile | Use | Settings |
|---|---|---|
| `release` | images, benchmarks of record | fat LTO, one codegen unit, `panic = "abort"`, stripped |
| `service-dev` | a quick local service build (`task rust:build-imaging`) | release without LTO, 16 codegen units, incremental |
| `wasm` | WebAssembly modules (`task rust:wasm-rules`) | release with `opt-level = "s"`, since the browser downloads the module |
| `bench` | criterion | thin LTO |

Fat-LTO builds are slow; do not run them over and over. Cap parallelism with
`CARGO_BUILD_JOBS` (the `rust:` tasks default to 4) or `LOW_RESOURCE=true`.

### Dependencies

- A new dependency goes into `[workspace.dependencies]` in `services/Cargo.toml`, with
  `default-features = false` unless a default feature is needed, and crates refer to it with
  `.workspace = true`.
- `cargo deny check` (`task rust:deny`) must pass: no advisories, only allow-listed licences,
  no two majors of a crate without a `skip` entry giving the reason, crates.io only.
- A C or C++ library is pinned by version and checksum where the image installs it, and its
  licence is recorded at the top of the Dockerfile and in the service's doc. An LGPL library is
  linked dynamically (opened at run time) so it can be replaced.

### Testing

- Unit tests next to the code; integration tests under `<crate>/tests/` drive the real thing
  (the imaging worker's start the real supervisor with hardened children).
- Property tests (`proptest`) for every parser and every invariant a type promises.
- Golden tests compare produced artifacts with a tolerance where pixels are involved.
- `cargo-fuzz` targets under `<crate>/fuzz/` for every parser that reads untrusted bytes; CI
  runs each for a bounded time.
- Tests that need a native library read its path from the environment
  (`TRENOVA_PDFIUM_LIBRARY`) and skip without it; the image's `test` stage and CI provide them.
- `criterion` benchmarks for hot paths, run with the `bench` profile.

## WebAssembly modules

`trenova-rules` (`services/crates/rules`, [rules-engine.md](rules-engine.md)) is the first
WebAssembly crate, and the pattern for any later one. `trenova-scripts`
(`services/crates/scripts`, [agent-code-mode.md](agent-code-mode.md)) is the second, and
`trenova-edi` (`services/crates/edi`, [edi-x12.md](edi-x12.md)) the third.

- **A browser module builds on `trenova-module-kit`** (`services/crates/module-kit`): its
  `export_abi!` macro writes the three exports, `ops::dispatch` decodes the envelope and runs the
  operation table, and `message!` makes user-facing text an extractable template. A new module
  writes its operation table and nothing else at the boundary.
- **The Go side is `pkg/wasmmodule`**: the wazero runtime, the compile cache, the instance pool,
  the error sentinels and the metrics observer, described once in rules-engine.md ("The Go
  host"). A module's package (`pkg/rules`, `pkg/x12`) is a `wasmmodule.Spec` (name, bytes,
  version operation, cache directory and its environment variable) and typed calls on a
  `Runtime` that embeds `*wasmmodule.Runtime`. Metrics come from one `platformmetrics.NewWasmModule`
  per module (`trenova_rules_*`, `trenova_edi_module_*`).

- **A module the browser runs is `wasm32-unknown-unknown` with no imports at all** (no WASI, no
  clock, no randomness, no I/O), so the bytes the TMS runs in wazero are the bytes the browser
  runs. `wasm32-wasip1` imports WASI functions a browser does not have, and a second build per
  host would turn "the same answer to the cent" back into something tests have to keep true.
- **A module only the TMS runs may be `wasm32-wasip1`** when a dependency needs the standard
  library's clock or randomness (Monty's `chrono` pulls browser bindings on
  `unknown-unknown`). The host gives it a WASI with nothing in it: no directories, no
  environment, no arguments, and it refuses a module importing anything but
  `wasi_snapshot_preview1` (`pkg/scriptvm`, `checkABI`).
- **Untrusted code may run in a module rather than a service** when the module itself grants
  nothing: an instance's linear memory is its whole world, wazero stops it when its context
  ends, and a fresh instance per run leaves nothing behind. That is how agent scripts run.
  Code that needs a native library, or the outside world, is still a service.
- **The ABI** is three exports and JSON in and out (`rules_alloc`, `rules_call`,
  `rules_free`; see "The ABI" in rules-engine.md). Nothing else crosses the boundary.
- **Two committed copies**, each with a `rules.wasm.sha256` sidecar in `sha256sum` format:
  - `services/tms/pkg/rules/rules.wasm`, embedded with `//go:embed` by the Go host;
  - `client/packages/shared/src/rules/rules.wasm`, a content-hashed Vite asset the browser
    loader fetches.

  Never edit either by hand. The Go and the client tests both check that the bytes match the
  sidecar, and the Go test that the two copies are identical.
- **The build** is `services/crates/rules/scripts/build-wasm.sh`, run by `task rust:wasm-rules`,
  on the shared `services/scripts/wasm-module.sh` (the scripts module's build sources the same
  file, with its own target, profile and optimisation level):
  - `cargo build --locked -p trenova-rules --profile wasm --target wasm32-unknown-unknown`, with
    `--remap-path-prefix` for the services checkout (`/trenova/services`), the cargo target
    directory (`/trenova/target`), `CARGO_HOME` (`/cargo`) and the toolchain's `rust-src`
    (`/rustc/<commit-hash>`, which rustc records for the standard library only when `rust-src`
    is not installed), and `-C strip=symbols`;
  - then `wasm-opt -Os` from binaryen **version_133**, downloaded once into
    `services/target/binaryen/` and verified against the SHA-256 pinned in the script for each
    platform (Linux x86_64 and aarch64, macOS arm64 and x86_64; elsewhere set
    `TRENOVA_WASM_OPT` to a wasm-opt that reports version 133);
  - with exactly the features rustc 1.96 enables for the target, so nothing is lowered or
    rejected: `--mvp-features` plus bulk memory (and its `-opt` subset), overlong
    `call_indirect`, multi-value, mutable globals, non-trapping float-to-int, reference types
    and sign extension;
  - stripping debug info, the producers section and the target-features section;
  - then the browser's unit table, `client/packages/shared/src/rules/tariff-units.generated.ts`,
    written from the crate's `SIMPLE_UNITS` by `tests/client_units.rs` run with
    `TRENOVA_WRITE_GENERATED=1` (the web app formats wire values before the module loads, so it
    cannot read `tariff.catalog`'s `units`; generated, it cannot drift).

  The same commit gives the same bytes on any machine and in any checkout path. A toolchain,
  dependency or binaryen change changes the bytes, so the same commit carries the rebuilt
  copies. To move binaryen, change `BINARYEN_VERSION` and the four checksums in the script
  (taken from the release's `.sha256` files and checked against your own download), then run
  the task.
- **The check.** `task rust:wasm-rules-check` rebuilds into a temporary directory and fails,
  naming the file, when a committed copy or its sidecar differs, or the unit table is not the
  crate's (that test also runs in `cargo test`). The `wasm-rules` job in
  `rust.yml` runs it, and clippy for `--target wasm32-unknown-unknown`, the only build that
  compiles `abi.rs`. The Go host's tests run with the rest of the TMS in `test-tms.yml`, the
  loader's with the client in `test-client.yml`.
- **Calls take batches.** Every call copies its request and its response through linear memory
  and holds an instance from the pool while it runs. One call per request, never one per item:
  an operation takes the whole list (every charge line, every lane, every band, every program of
  a rating) and answers for all of it. A loop that calls the module once per row is a review
  failure, like an N+1 query.

## Tasks

| Task | Does |
|---|---|
| `task rust:fmt` | `cargo fmt --all` (`-- --check` to check) |
| `task rust:lint` | clippy over the workspace, warnings as errors |
| `task rust:test` | `cargo test --workspace` |
| `task rust:deny` | `cargo deny check` |
| `task rust:proto` | buf lint and regenerate the Go contracts |
| `task rust:proto-breaking` | buf breaking against `origin/master` |
| `task rust:build-imaging` | the imaging worker with the `service-dev` profile |
| `task rust:image-imaging` | the imaging worker's image |
| `task rust:wasm-rules` | build `trenova-rules` to WebAssembly and install both committed copies with their `.sha256` |
| `task rust:wasm-rules-check` | rebuild it and fail when a committed copy or its `.sha256` differs |
| `task rust:wasm-scripts` | build `trenova-scripts` to WebAssembly (`wasm32-wasip1`, profile `wasm-server`, `wasm-opt -O3`) and install `services/tms/pkg/scriptvm/scripts.wasm` with its `.sha256` |
| `task rust:wasm-scripts-check` | rebuild it and fail when the committed copy or its `.sha256` differs |
| `task rust:wasm-edi` | build `trenova-edi` to WebAssembly (`wasm32-unknown-unknown`, profile `wasm`, `wasm-opt -Os`) and install `services/tms/pkg/x12/edi.wasm` with its `.sha256` |
| `task rust:wasm-edi-check` | rebuild it and fail when the committed copy or its `.sha256` differs |

## CI

`.github/workflows/rust.yml` runs on changes under `services/` outside `services/tms`, the
generated Go contracts, the committed `rules.wasm` copies, the toolchain pins and the imaging
Dockerfile: the toolchain-pin check, `cargo fmt --check`, clippy with `-D warnings`,
`cargo test --workspace` with PDFium and libheif installed, the three WebAssembly modules (clippy
for `wasm32-unknown-unknown` and `wasm32-wasip1`, and each `build-wasm.sh check`), `cargo deny check`, buf lint and
breaking, the generated-Go check, bounded fuzzing (after each merge to master and weekly, not on
pull requests), and the imaging image (its test stage, the runtime image, and a start-up check that it refuses
to run in production without TLS and reports ready in development). `native-capture.yml` also
runs when `services/crates/imaging-core`, the workspace manifest or the toolchain pin changes.
