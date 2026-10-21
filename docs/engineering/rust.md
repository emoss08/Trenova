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
per document, a future rules engine by request rate; one container cannot be sized for both,
and a crash in a PDF parser must never stop anything else.

## Layout

```text
services/
  Cargo.toml              [workspace] members = ["imaging", "crates/*"]; dependencies, lints, profiles
  Cargo.lock
  rust-toolchain.toml     1.94 (the channel desktop/capture pins), rustfmt, clippy, wasm targets
  deny.toml               cargo-deny: advisories, licences, duplicate majors, sources
  clippy.toml             words doc comments may use unquoted (PDFium, WebP, ...)
  .cargo/config.toml      only what every build needs
  proto/                  contracts between services (buf); packages trenova.<domain>.v1
  crates/                 library crates, each named trenova-<name>
    proto/                trenova-proto: Rust code for services/proto, generated in build.rs
    service-kit/          trenova-service-kit: the chassis every service runs on
    sandbox/              trenova-sandbox: a supervised pool of hardened child processes
    imaging-core/         trenova-imaging-core: platform-neutral page imaging (shared with capture)
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
| WebAssembly in the Go process (`wasm32-wasip1`, run by wazero) | Pure computation the TMS calls often: parsers, rule engines, encoders, checks | Slower than native (no SIMD by default, a copy across linear memory); every boundary call costs, so the API must take batches; no threads, no native libraries |
| WebAssembly in the browser (`wasm32-unknown-unknown`) | Logic the browser and the server must agree on to the cent | Bundle weight against the first-load budget (`client/apps/web/bundle-budget.json`), so load it lazily |
| A gRPC service | Native libraries (PDFium, codecs), untrusted input that needs process isolation, work that must scale apart from the TMS | Another container to run, size and monitor; a contract to version; plaintext crossing a network link (mutual TLS) |
| A Postgres extension (pgrx) | Functions on every statement's path where a round trip is the cost | Rules out managed Postgres; native code inside the database's trust boundary; upgrades tied to the Postgres image |

A rule of thumb: if it needs no native library and no isolation from the TMS, it is a library
compiled to WASM; if it parses files nobody vetted, it is a service, and its parsing runs in
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
  follows CLAUDE.md (no comments).
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

## WebAssembly artifacts (when the first WASM crate arrives)

No WASM crate exists yet; this is how the first one is built and shipped.

- The crate lives in `services/crates/<name>` and builds for `wasm32-wasip1` (wazero in Go)
  or `wasm32-unknown-unknown` (the browser), both targets pinned in `rust-toolchain.toml`.
- The build is reproducible: release profile, `--locked`, `-C strip=symbols`,
  `--remap-path-prefix` for the workspace path, and `wasm-opt` at a pinned version.
- The artifact is committed next to its consumer (`services/tms/internal/infrastructure/<name>/<name>.wasm`
  embedded with `//go:embed`; for the browser, the client package that loads it), with its
  SHA-256 in a sibling file.
- A `task rust:wasm-<name>` rebuilds it, and CI rebuilds it and fails when the committed bytes
  differ, the same way the generated Go contracts are checked.
- The Go side calls it in batches: one boundary crossing per request, never one per item.

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

## CI

`.github/workflows/rust.yml` runs on changes under `services/` outside `services/tms`, the
generated Go contracts, the toolchain pins and the imaging Dockerfile: the toolchain-pin check,
`cargo fmt --check`, clippy with `-D warnings`, `cargo test --workspace` with PDFium and libheif
installed, `cargo deny check`, buf lint and breaking, the generated-Go check, bounded fuzzing,
and the imaging image (its test stage, the runtime image, and a start-up check that it refuses
to run in production without TLS and reports ready in development). `native-capture.yml` also
runs when `services/crates/imaging-core`, the workspace manifest or the toolchain pin changes.
