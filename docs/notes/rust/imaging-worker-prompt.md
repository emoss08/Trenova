# Rust platform and imaging worker: implementation prompt

The brief handed to an implementation session for the first Rust port in
[opportunities.md](opportunities.md) (candidate R1). Phase 0 sets the layout every later Rust port
follows; Phase 1 builds the document and imaging worker on it.

```text
You are building Trenova's Rust platform and its first service, a document and imaging worker that replaces the Go document and imaging stack. Trenova is pre-launch.
- Break whatever needs breaking.
- No backward compatibility, migration guides, upgrade notes, feature flags for the old path, dual-read code or legacy readers. Delete the old implementation outright; development data gets reseeded.
- "Production" here means built to run at enterprise scale, hardened and observable. It does not mean preserving compatibility.

Before writing anything, read in full:
- CLAUDE.md
- docs/engineering/module-architecture.md, row-level-security.md, security-audit.md, generated-artifacts.md, desk-artifacts.md
- desktop/capture/README.md
- desktop/capture/Cargo.toml and the CI in .github/workflows/native-capture.yml

Follow CLAUDE.md exactly for Go. Its DO NOT rules apply: cap build parallelism (CARGO_BUILD_JOBS, LOW_RESOURCE) and do not run fat-LTO release builds over and over.

# Phase 0 — The Rust platform under services/ (the architecture is decided; implement it)

## Decision
- **services/ is the backend.** It holds every backend service, whatever its language, plus the libraries and contracts those services share.
- **One Cargo workspace rooted at services/,** covering every backend Rust crate. One Cargo.lock, one toolchain, shared lints, one dependency and supply-chain policy.
- **Every deployable is its own crate, binary and container,** a peer of services/tms. Never combine unrelated Rust services into one process; they scale on different resources.
- **Pure computation that Go or the browser needs is a library crate** in services/crates. It is compiled to wasm32-wasip1 (run in-process by wazero) or wasm32-unknown-unknown (browser). Such crates are not services. None are built in this task, but the layout and docs must accommodate them.
- **Code that needs native libraries or process isolation is a service** called over gRPC.
- **desktop/capture stays where it is, with its own workspace.** It is a Windows product with its own release tags, MSI, signing and version flow. It depends on shared crates in services/crates by path. CI keeps its toolchain pin identical to services/rust-toolchain.toml.

## Layout

    services/
      Cargo.toml                  [workspace]: resolver 3, members = ["imaging", "crates/*"], exclude = ["tms", "postgres"]; [workspace.dependencies], [workspace.lints], profiles
      Cargo.lock
      rust-toolchain.toml         pinned channel (keep desktop's 1.94), rustfmt, clippy, wasm32-wasip1, wasm32-unknown-unknown
      deny.toml                   cargo-deny: advisories, licenses (allow-list compatible with FSL-1.1-ALv2), bans (duplicate majors), sources
      .cargo/config.toml          only what every build needs
      proto/                      contracts between services: buf.yaml, buf.gen.yaml; packages trenova.<domain>.v1
        trenova/imaging/v1/*.proto
      crates/                     library crates, each named trenova-<name>
        proto/                    trenova-proto: tonic/prost code generated in build.rs from ../proto (use protox, so no protoc install is needed)
        service-kit/              trenova-service-kit: the chassis every Rust service uses
                                    - config (env plus optional TOML, strict validation, fail-fast)
                                    - tracing with JSON logs and OTLP export, W3C context
                                    - Prometheus /metrics
                                    - tonic server setup: mTLS, grpc.health.v1, message and stream limits, no reflection in production
                                    - graceful drain on SIGTERM
                                    - admission control and priority primitives
        sandbox/                  trenova-sandbox: a supervised pool of hardened child processes
                                    - framed IPC and memfd input passing
                                    - seccomp-bpf, landlock, rlimits, NO_NEW_PRIVS
                                    - recycling by job count or RSS, deadline kills, crash containment
                                  Written generically, so the future code-mode sandbox reuses it
        imaging-core/             trenova-imaging-core: the platform-neutral pixel and PDF-writing code moved out of desktop/capture/crates/capture-imaging
      imaging/                    trenova-imaging: the deployable (binary trenova-imaging; the child worker is a subcommand of the same binary, or a second bin in this crate)
      tms/                        the Go service (unchanged location)
      postgres/<ext>/             future pgrx extensions; excluded from the workspace, each with its own, built by deploy/Dockerfile.postgres
    desktop/capture/              own workspace; path dependency on ../../services/crates/imaging-core
    deploy/Dockerfile.imaging     build context services/ (it needs crates/ and proto/)

Rules for the workspace:
- **Moving capture-imaging.** Move the code the server can use (raster and luma, binarize, previews, JPEG and CCITT G4 encoding, single-image and multi-page PDF writing) into services/crates/imaging-core.
  - Remove its dependency on capture-protocol by moving any shared types into imaging-core or passing primitives.
  - Device formats only capture uses (DIB, PWG) stay in a capture crate.
  - Update every capture crate and bin to depend on imaging-core by path, and keep the Windows build, the MSI and the release flow working.
- **Lints.** Copy desktop's lint set into services/Cargo.toml (pedantic, unwrap_used, undocumented_unsafe_blocks, …). Every unsafe block carries a SAFETY comment. Rust follows desktop/capture's conventions, including its doc comments; Go follows CLAUDE.md (no comments).
- **Release profile.** lto = "fat", codegen-units = 1, panic = "abort" for binaries. Add a dev-friendly profile for local service builds.
- **Errors and logs.** thiserror in libraries, structured errors at service edges. No document content ever logged.
- **Dependencies.** A new dependency goes into [workspace.dependencies] with default-features off unless a feature is needed. C or C++ native libraries are pinned by version and SHA-256 in the Dockerfile and documented with their license.
- **Tasks.** Add a `rust:` task namespace to the root Taskfile.yml: fmt, lint, test, deny, proto, proto-breaking, build-imaging, image-imaging.

## Contracts
- protobuf plus buf is the one Go↔Rust service contract, defined in services/proto.
- Go code for a contract is generated into the TMS adapter package that consumes it (for imaging, under services/tms/internal/infrastructure/imaging), placed so archlint passes. It is checked in and verified by the Codegen Checks CI like other generated code.
- Rust code is generated at build time by trenova-proto.
- Add `task rust:proto` (buf lint, buf generate) and `task rust:proto-breaking` (buf breaking against origin/master). Add both to CI.
- Go uses google.golang.org/grpc (already in the module graph) with otelgrpc for trace propagation.

## CI
- A new `.github/workflows/rust.yml` triggers on services/** excluding services/tms/**.
- It runs on Linux from services/:
  - `cargo fmt --check`
  - `cargo clippy --workspace --all-targets -- -D warnings`
  - `cargo test --workspace`
  - `cargo deny check`
  - building each service image
  - buf lint and buf breaking
  - generated Go up to date
  - a check that services/rust-toolchain.toml and desktop/capture/rust-toolchain.toml pin the same channel
- native-capture.yml keeps its Windows matrix and installer and widens its path filter to services/crates/imaging-core/**.
- Use Swatinem/rust-cache or equivalent, and cargo-chef in the Dockerfiles.

## Docs for Phase 0
- Write docs/engineering/rust.md, the guide every later Rust port follows:
  - the decision above and why
  - the layout and what goes where (deployable vs library vs contract vs extension)
  - how to choose a seam: in-process WASM via wazero, browser WASM, gRPC service, or Postgres extension, with each one's costs
  - service-kit and sandbox as the required building blocks for services
  - conventions, unsafe policy, dependency policy, testing (unit, golden, property, cargo-fuzz)
  - how WASM artifacts will be built and embedded when the first WASM crate arrives: reproducible build, committed artifact, CI freshness check
- Update CLAUDE.md:
  - the repository structure ("services/ — backend services in any language plus their shared crates and contracts")
  - commands
  - a "Read docs/engineering/rust.md before adding or changing Rust code" pointer
  - the references to desktop/capture being the only Rust

# Phase 1 — The imaging worker (services/imaging)

## Measure first
Before changing any Go code, build a benchmark harness and record a baseline of the current Go implementation.
- **Corpus:** tools/docling-fixtures generates synthetic documents. Add real-world-shaped samples: scanned BOLs, phone photos, 1–200 page PDFs, malformed and hostile PDFs.
- **Measure:**
  - p50 and p95 latency, and throughput, for: thumbnail, page view render, capture page inspection, text profile, merge, split and rotate
  - peak TMS RSS while processing 5, 20 and 50 MB documents end to end (upload → store → thumbnail)
- Record the numbers. They go in the final doc next to the new ones.

## What it replaces
Delete these Go implementations and dependencies; their responsibilities move to the worker:
- `internal/infrastructure/pdfrender` (go-pdfium on wazero, the 4 × 1 GB WASM pool) and the `kernelports.PDFReader` / `PDFDocument` ports
- `internal/infrastructure/pdfassembly` (pdfcpu: PageCount, Assemble, Split, ImageToPDF)
- `internal/infrastructure/captureimaging` (gozxing multi-QR decode, ink coverage, thumbnail)
- the pixel work in `modules/documents/app/thumbnailservice/generator.go` (disintegration/imaging, gen2brain/webp)
- the Legacy extraction engine and all of its OCR, deleted outright and not ported (see "Remove the Legacy extraction engine" below). Docling becomes the only extraction engine and does all OCR.
- `pdfTextProfile` in `documentextractionjobs/conversion.go`
- the WebP→PNG transcode in `infrastructure/templating/assetinliner/webp.go`
- the go-pdfium, pdfcpu, disintegration/imaging, gen2brain/webp and gozxing modules, and wazero if nothing else uses it. For QR *encoding* in `infrastructure/captureqr`, keep one Go library (boombuler/barcode, already used by `shared/qrutils`) and drop the duplicate encoder.
- every caller adapts, including:
  - `documentservice` (GetPageImage)
  - `documentuploadservice`
  - `thumbnailjobs`, `documentuploadjobs`
  - `captureservice` (processing, batches checkPDF, filing, storage)
  - `triageworkservice` (split)
  - `billing/app/factoringsubmissionservice` (submission packets)
  - `agent/app/messagefileservice`
  - `internal/testutil/pdftest`
  - `bootstrap/modules/infrastructure/pdfrender.go`

## Boundary
The worker is stateless, tenant-agnostic compute.
- **Go keeps:** storage access, encryption and decryption, tenancy, RLS, permissions, audit (document views and downloads), Temporal orchestration and caching.
- **The worker holds:** no database access, no object-storage credentials, no KMS access and no outbound network. It receives plaintext bytes over gRPC and returns derived artifacts. That keeps the attack surface minimal and lets it scale horizontally behind DNS.

## gRPC API (services/proto/trenova/imaging/v1)
Design the API for streaming and batching:
- **Input** is client-streamed in chunks (1 MiB default) behind a header message: content type, declared size, operations and limits.
- **Output** is server-streamed per page or per artifact, so the client never buffers a whole result set.
- **Per-call deadlines** propagate. Structured errors use google.rpc status details: invalid document, encrypted or password-protected, too large, pixel budget exceeded, timeout, resource exhausted with retry-after, internal.

Operations. There is no text-extraction or OCR operation: docling does all of that, and once the Legacy engine is gone nothing else reads page text.
1. **Probe.** Validate the document. Return page count, page sizes and rotation, whether each page has native text (replaces pdfTextProfile and checkPDF), encryption, PDF version, and producer.
2. **Render.** Render pages to an image, chosen by DPI or by a fit box (thumbnail 240 px wide; page view fit 1400×1800). Formats: WebP (libwebp, lossy and lossless), JPEG and PNG. Flatten onto white and enforce a pixel-budget cap.
3. **Capture inspection.** One pass per page: thumbnail, ink coverage and blank score, and multi-QR plus 1D barcode decoding with rxing.
   - Use imaging-core's luma and threshold code. Keep the server's ink threshold and margins as parameters, so desktop and server use one implementation.
   - Batch form: a capture batch of up to 1,000 pages streams in, and results stream back per page.
4. **Assemble.** Using PDFium's page import, rotation and save APIs:
   - merge, split into ranges (import the subset straight from the source; no split-then-merge duplicating shared resources), per-page rotation, page extraction
   - image(s) → PDF through imaging-core's pdf-writer path (JPEG passed through without re-encoding, bilevel as G4)
5. **Image normalization.**
   - Decode JPEG, PNG, WebP, TIFF, GIF and HEIC/HEIF (libheif, dynamically linked; record the LGPL licensing in docs). HEIC matters for phone photos from Dash.
   - Honor EXIF orientation, decode with downscaling where the codec supports it, and strip metadata on derived artifacts.
   - Transcode to PNG or JPEG for the template asset inliner.
   - Add HEIC/HEIF to the server upload MIME allow-list.

## Process model and isolation (built on trenova-sandbox)
PDFium and the native codecs are C/C++ parsing untrusted input, and PDFium is not thread-safe. Today WASM sandboxes them; native must be at least as contained.
- **The service binary is a supervisor:** tokio and tonic through trenova-service-kit, admission control, streaming, metrics, health.
- **Document work runs in the trenova-sandbox pool** (one child per allotted core by default). Each child handles one job at a time and is recycled after N jobs or when it passes an RSS high-water mark. It is killed at its deadline, and on a crash or kill its job fails with a typed error while the pool replaces the process.
- **Input:** pass it to the child through a memfd (SCM_RIGHTS) or a per-job tmpfs directory. Use a length-prefixed framed protocol on the child's stdio.
- **Hardening each child:**
  - PR_SET_NO_NEW_PRIVS
  - a seccomp-bpf allow-list (seccompiler) with no network syscalls
  - landlock restricting the filesystem to its job directory and read-only library paths
  - rlimits on address space, CPU, file size and open files
  - an empty environment
- **Fail closed.** The service refuses to start without these protections unless explicitly configured otherwise, and logs loudly when it is.
- **Fuzz:** the framed protocol and every Rust decoder (cargo-fuzz targets checked in). Add a crash-containment test suite: hostile PDFs that crash, hang or exhaust memory must take down only one child.

## Scaling and resilience
- **Admission control** from a per-pod budget of cores and memory (service-kit primitives). Estimate each job's cost from page pixels and declared size. Over budget, answer RESOURCE_EXHAUSTED with retry-after rather than queueing without bound.
- **Two priority classes,** carried in request metadata: interactive (API page views, synchronous probes) and batch (Temporal activities). Reserve a configurable share of capacity for interactive.
- **Go client:**
  - gRPC client-side round_robin over DNS (headless service), keepalives, and retries on RESOURCE_EXHAUSTED or UNAVAILABLE to another pod within the deadline
  - circuit breaking per endpoint
  - Temporal activities keep their own durable retry policies
- **Health and shutdown:**
  - grpc.health.v1; readiness turns false when the pool is saturated or unhealthy
  - SIGTERM drains in-flight jobs within a grace period
- **Observability:**
  - Prometheus /metrics: RED per operation and priority, queue and admission, pool size, restarts, kills by reason, child RSS, bytes in and out
  - OTLP traces with W3C context taken from Go's otelgrpc
  - Add a Grafana dashboard and alerts beside the existing ones in deploy/observability
- **Security of the link:**
  - mTLS between TMS and worker, configurable on both sides and required in production config
  - max message and stream sizes enforced on both ends
- **Container** (deploy/Dockerfile.imaging, build context services/):
  - multi-stage with cargo-chef
  - runtime on debian-slim or distroless/cc with libpdfium from bblanchon/pdfium-binaries, libwebp and libheif, all pinned and checksummed
  - non-root, read-only root filesystem, tmpfs for job directories, an SBOM
- **Compose:** add the service to dev/compose.yml and deploy/selfhost/compose.yml with CPU and memory limits and a healthcheck. tms-api and tms-worker depend on it being healthy. Publish the image from release.yml.
- **Scaling docs:** document horizontal scaling (replicas behind a headless DNS name, autoscaling on in-flight or admission-rejection metrics) in the engineering doc.

## Remove the Legacy extraction engine (do this before the Go rewiring)
Docling is the only extraction engine. Delete the Legacy engine end to end, so the worker rewiring has fewer callers to move:
- **The setting.** `tenant.ExtractionEngine` (`identity/domain/tenant/documentcontrol.go`), `DocumentControl.UsesDocling`, the `document_controls.extraction_engine` column (a migration drops it), the documentcontrolservice validator and service handling, the GraphQL field and generated code, `client/apps/web/src/types/document-control.ts`, the engine choice in `routes/admin/document-intelligence/_components/document-intelligence-form.tsx` and the tests that cover it (including `capture-settings-form.test.tsx`), and seeds.
- **The route.** `ExtractionRouteLegacy` in `documentextractionjobs/conversion.go` (PlanDocumentExtractionActivity) and `extractionworkflow.go`, `ProcessDocumentIntelligenceActivity`, and every function only it reaches: PDF text and layout extraction, `runOCRPage`, OCR image preprocessing and thresholding, Tesseract TSV parsing, `pdfLayout`, and the `ai.ocrCommand` config. Find them by call graph, not by name.
- **Office files.** `internal/infrastructure/officedoc` and `kernelports.OfficeDocument` serve only the Legacy route; docling reads DOCX, XLSX, PPTX and email itself. Delete them.
- **Keep** what the docling path shares (AnalyzeConvertedDocumentActivity, classification, field extraction, AI normalization, capture analysis). Confirm each by call graph before deleting anything near it.
- **Behavior without docling.** When document intelligence is on but docling, `ai.documentExtraction` or an extraction provider is missing, do what the Docling engine does today: store the document, mark extraction unavailable, and name what is missing on the upload screens and the document's failure message.
- **Generated artifacts and text.** Read docs/engineering/generated-artifacts.md before removing the field and column: regenerate buncolgen, gqlgen, the client GraphQL codegen, i18n catalogs and the product guide (`pnpm --filter @trenova/web guide:generate`). The schema change is breaking by design; Trenova is pre-launch, so follow the repo's process for accepting a breaking change in `task gqlschema-diff`.
- **Docs.** Rewrite the extraction section of docs/self-hosting/README.md: Docling is the engine, with its requirements (the service, `ai.documentExtraction`, an extraction provider). Remove every Legacy and Tesseract mention.

## Go side
- **Port.** Replace PDFReader with one imaging port in kernelports, since documents, billing and agent all use it. Implement it in services/tms/internal/infrastructure/imaging as a gRPC adapter using the generated client.
  - Obey archlint and the CLAUDE.md generics rule.
  - Provide an in-memory fake for unit tests.
  - Integration tests run against the real worker image in the repo's Docker-based integration harness.
- **Config.** An `imaging` section: endpoints, mTLS, per-operation deadlines, retry and backoff, message limits, priority. Validate strictly. Remove the old pdfrender keys and `ai.ocrCommand`.
- **Streaming blob encryption.** Replace the stored-blob encryption format (kernel/app/encryptionservice: base64(JSON{base64 ciphertext}), about 1.78× size, whole-file buffers) with a binary, chunked, streaming AEAD format for blobs: documents, capture pages, thumbnails and rendered pages.
  - Header: magic, version, key provider and id, wrapped DEK, nonce prefix, chunk size, AAD hash.
  - Chunks: 64 KiB AES-256-GCM in the STREAM construction (nonce = prefix ‖ counter ‖ last-chunk flag). Truncation, reordering and chunk swaps must fail authentication.
  - Provide io.Reader and io.Writer APIs. Keep the AAD binding.
  - Small secrets may keep the string envelope. No legacy reader.
- **Stream end to end.** Upload → encrypt → object storage (multipart) without io.ReadAll. Storage → decrypt → gRPC stream to the worker. Worker results → encrypt → storage.
  - Enforce size limits with bounded readers.
  - Remove the duplicate buffering in the docling submission path (conversion.go readAllBody, docling client Submit) while you are there.
- **Page views are no longer rendered on every request.**
  - After upload, a Temporal activity pre-renders every page's view image and thumbnail (bounded pages per document, rest on demand) and stores them encrypted, keyed by document version, page, size and format.
  - GetPageImage serves the stored render. On a miss it renders synchronously at interactive priority behind singleflight and stores the result.
  - Keep ETag and immutable caching headers, and the document-view audit events.
- **Capture.** captureservice streams a whole batch through the batch inspection call instead of 4 goroutines each rendering through PDFium.
- **Remove** every deleted dependency from go.mod and go.sum, and keep `go build` and `go test` green with the nomsgpack tag as the Taskfile does.

## Verification
- Rust:
  - unit and property tests
  - golden-image tests with a perceptual tolerance
  - fuzz targets run for a bounded time in CI
  - crash-containment suite
  - criterion benchmarks
- Go: unit tests with the fake, and integration tests against the worker image for every rewired caller.
- Desktop: the capture workspace builds and tests on Linux and Windows against the moved imaging-core.
- Re-run the Phase 1 benchmark harness and compare with the baseline: latency, throughput per core, TMS peak RSS.
  - Any regression must be fixed or explained.
  - Check that the TMS process no longer holds multi-GB WASM heaps or whole-document copies.
- Run `task lint`, `task test` and the relevant integration tests in services/tms, plus the full Rust CI locally (fmt, clippy, test, deny, buf). Report anything you could not run.

## Docs for Phase 1
- docs/engineering/imaging-worker.md:
  - architecture and boundary
  - the API
  - the process model and sandbox
  - admission, priorities and scaling
  - configuration
  - metrics and alerts
  - the blob encryption format
  - benchmark results, before and after
- Update docs/engineering/desk-artifacts.md where it describes page rendering and prefetch.
- Update docs/self-hosting with the new service and its resource needs (the extraction section is rewritten by the Legacy removal step).
- Add CLAUDE.md pointers ("Read docs/engineering/imaging-worker.md before changing…").
- No migration guide.

## Out of scope (the structure must make these easy later)
- the capture companion uploading its own thumbnails, ink scores and QR results
- WASM imaging in the Dash app
- Gotenberg, docling's models, and other Rust ports

## Delivery
- Commit in logical steps: workspace and move, service-kit, sandbox, contracts, service skeleton, each operation, Legacy engine removal, Go rewiring, encryption format, deletion of old code, deploy and CI, docs. The tree builds and tests pass at each step.
- Finish with a summary:
  - what changed
  - a before/after benchmark table
  - every default or judgement call you made
  - native libraries and licenses added
  - anything left for a human
```
