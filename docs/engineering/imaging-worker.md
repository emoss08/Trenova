# The imaging worker

`services/imaging` (`trenova-imaging`) does every piece of document and image computation the
TMS used to do in-process: probing PDFs, drawing thumbnails and page views, inspecting
captured pages, assembling PDFs and normalizing images. It replaced PDFium on wazero (four
1 GB WebAssembly instances in every TMS process), pdfcpu, gozxing, disintegration/imaging and
gen2brain/webp. Read this before changing the worker, its contract
(`services/proto/trenova/imaging/v1`), the TMS adapter
(`services/tms/internal/infrastructure/imaging`) or anything that renders, inspects or
assembles a document. The Rust conventions it follows are in [rust.md](rust.md).

The worker reads no text and runs no OCR: docling-serve does both, and is the only extraction
engine.

## Architecture and boundary

```text
TMS api / worker ──gRPC, mTLS, streamed──▶ imaging supervisor ──framed IPC, sealed memfd──▶ child (hardened)
  storage, decryption,                       admission, priorities,                         PDFium, libwebp,
  tenancy, RLS, permissions,                 streaming, metrics, health                     libheif, rxing,
  audit, Temporal, caching                   (never parses a document)                      imaging-core
```

- **The TMS keeps** storage access, encryption and decryption, tenancy and row-level security,
  permissions, the document-view and download audit, Temporal orchestration and caching.
- **The worker holds** no database access, no object-storage credentials, no key access and no
  outbound network. It receives plaintext over gRPC and returns derived artifacts. It is
  stateless and tenant-agnostic, so it scales horizontally behind a DNS name.

## The API

`trenova.imaging.v1.ImagingService`, five operations. Every call streams its input in: a header
(the operation's parameters, each document's content type and exact declared size), then the
bytes in chunks of 1 MiB, each document opened by a start message when a call takes several.
Results stream back per page or per artifact: an `ArtifactHeader` followed by `ArtifactChunk`s,
the last flagged, so neither side holds a whole result set.

| Operation | Takes | Returns |
|---|---|---|
| `Probe` | one PDF or image | page count; each page's size, rotation and (with `text_profile`) whether it has native text; encryption; PDF version and producer; an image's format, upright size and EXIF orientation |
| `Render` | one PDF or image; pages; a DPI or a fit box; WebP, JPEG or PNG | each page as an artifact, a page that cannot be drawn as an in-band `PageError`, then a summary |
| `InspectCapture` | up to 1,000 captured pages (one-page PDFs or images) | per page: a thumbnail, ink coverage, every QR and barcode, or a `PageError`; then a summary |
| `Assemble` | sources (PDFs and images); outputs, each a list of page ranges with a rotation | each output PDF as an artifact; page counts in the summary |
| `NormalizeImage` | one image; PNG or JPEG; an optional box | the upright image without metadata; the source's format and size |

- **Probe** replaces `pdfTextProfile` (docling's per-page native-text check) and capture's
  `checkPDF`. An encrypted PDF that needs a password is reported (`encrypted`), not failed.
- **Render** draws a PDF page straight at the wanted size with PDFium, onto white: a thumbnail
  is a 240 px wide fit, a page view a 1400 × 1800 fit, never larger than the page at 300 DPI. A
  DPI render over the pixel budget (25 megapixels by default) is drawn smaller rather than
  refused, as the TMS always has. Images are decoded at a reduced size when the codec can.
- **InspectCapture** draws each page once at 150 DPI and takes everything from that raster: the
  240 px JPEG thumbnail at quality 72, the ink coverage (the capture companion's measure from
  `trenova-imaging-core`, with the server's threshold of 160, 6% margins and a stride of 2, each
  overridable), and every QR, Data Matrix, PDF417, Aztec and 1D code rxing finds.
- **Assemble** fills each output straight from the sources with PDFium's page import, so a split
  copies each range once. A range's `last` of 4294967295 means the source's last page. An image
  source becomes a page through `trenova-imaging-core`'s PDF writer, fitted to a letter sheet: a
  JPEG byte for byte (its EXIF rotation becomes the page's `/Rotate`), a black-and-white image as
  CCITT G4, anything else as JPEG.
- **NormalizeImage** decodes JPEG, PNG, WebP, TIFF, GIF and HEIC/HEIF, turns the image upright,
  fits it and writes PNG or JPEG. The template asset inliner uses it for WebP and HEIC.

Metadata: `x-trenova-priority: interactive|batch` (anything else is batch) and the caller's
`grpc-timeout`, capped per operation by the worker. Errors are `google.rpc.Status` with an
`ErrorInfo` (domain `imaging.trenova.app`, reason one of `INVALID_DOCUMENT`, `ENCRYPTED`,
`TOO_LARGE`, `PIXEL_BUDGET`, `UNSUPPORTED_FORMAT`, `PAGE_OUT_OF_RANGE`, `INVALID_REQUEST`,
`TIMEOUT`, `RESOURCE_EXHAUSTED`, `DRAINING`, `WORKER_CRASHED`, `INTERNAL`) and a `RetryInfo` on
`RESOURCE_EXHAUSTED` and `DRAINING`. The TMS adapter maps each reason to a `kernelports.ErrImaging*`
sentinel.

## Process model and sandbox

The binary has two roles.

- **`trenova-imaging serve`** is the supervisor: tokio and tonic through `trenova-service-kit`.
  It validates headers, admits calls, writes each call's input into a sealed memfd as it
  arrives, runs the job on the `trenova-sandbox` pool, and forwards each encoded response the
  child produces through a bounded channel. It never parses a document.
- **`trenova-imaging child`** is what the pool runs, one per pool slot (one per CPU the process
  may use, by default). It loads PDFium (from bblanchon/pdfium-binaries) and libheif, then
  hardens itself and handles one job at a time.

Each child is hardened before it reads untrusted input:

- `PR_SET_NO_NEW_PRIVS`;
- landlock: the filesystem shrinks to read-only shared libraries and fonts; no writable path;
- seccomp-bpf: a kill list (sockets, exec, ptrace, namespaces, mounts, BPF, keys, `io_uring`,
  identity changes) and an allow list answering `EPERM` to anything else, with thread-only
  `clone` and `clone3` as `ENOSYS`;
- rlimits set between fork and exec: address space (3 GiB), lifetime CPU (1 hour), no file
  writes, 64 descriptors, no core dumps; `PR_SET_PDEATHSIG`, checked against the parent's pid
  after it is set;
- an empty environment.

The pool **fails closed**: a child whose report falls short of the policy is refused, so the
worker does not start. `pool.require_hardening = false` exists for development machines
without landlock or seccomp, is refused in production, and is logged at error level on every
spawn.

The death signal fires when the *thread* that forked the child exits, so every fork runs on
one `sandbox-spawner` thread that lives as long as the pool. Never fork a child from tokio's
blocking pool: it retires idle threads after ten seconds, which killed every child that waited
that long for its first job.

A child is killed through its pidfd at its job's deadline or when the caller goes away. A
crash, kill or protocol violation fails only that job (`WORKER_CRASHED` or `TIMEOUT`) and the
slot gets a fresh child. Healthy children retire after 500 jobs, past 768 MiB resident, or near
their CPU limit. An idle child that died anyway is found when it is next taken, reaped and
replaced, so the job goes to a fresh child. A capture batch is cut into sub-batches of
`ceil(pages / capture_parallelism)` pages, never more than `capture_batch`, so a small batch
spreads over every slot; a sub-batch whose child dies is rerun a page at a time, so one hostile
page fails alone. The containment suite (`services/crates/sandbox/tests/containment.rs`, and the
hostile PDFs in `services/imaging/tests`) covers aborts, wild writes, spins, allocation bombs,
socket attempts, reads outside landlock, cancellation and recycling.

## Admission, priorities and scaling

Every call is admitted on its estimated cost before any bytes are read: one cost unit (one
child for the length of a job) and memory (its input plus its largest page held decoded, at 12
bytes a pixel, plus 48 MiB). The default budget is two units per child (one running, one
waiting) and 75% of the pool's address-space allowance. Interactive calls (page views,
synchronous probes) may use the whole budget; batch calls (Temporal activities) may not use
the interactive reserve (25% by default). A call that does not fit waits up to 250 ms, with at
most 64 waiting; after that it is refused with `RESOURCE_EXHAUSTED` and a retry-after taken from
how long recent jobs held their permits. A job larger than its class's whole budget runs alone
rather than never.

The TMS client:

- keeps its own endpoint set: it resolves `dns:///imaging:50051` every `resolveInterval` (10 s),
  holds one connection per address, and sends each call to the next address in rotation whose
  breaker is closed. Five failures in a row that say something about the endpoint (unavailable,
  internal, a crash, a deadline) open its breaker for 5 s, doubling to 2 min while it keeps
  failing; a draining worker is taken out at once; a bad document says nothing about the
  endpoint. gRPC's own outlier detection is not used because grpc-go offers it only through
  the xDS balancer configuration;
- retries `RESOURCE_EXHAUSTED` and `UNAVAILABLE` up to 4 attempts within the call's deadline,
  each on another endpoint where there is one, honouring `RetryInfo`, with jittered exponential
  backoff, and never after results have started streaming (the source is reopened for each
  attempt);
- leaves durable retries to Temporal activities.

Readiness (`grpc.health.v1` and `/readyz`) is false while the pool is short of children, the
interactive budget is spent, or the worker is draining. SIGTERM stops admission and drains
in-flight calls within `drain_grace` (30 s).

**Scaling.** Run several replicas behind a headless service (Kubernetes) or a DNS name with one
A record per replica; the client spreads calls over all of them. Autoscale on
`admission_cost_units_in_use / admission_cost_units_capacity` (scale out above 0.7) or on the
rate of `admission_decisions_total{outcome="Rejected"}`; CPU alone lags. Give each replica one
CPU per child and about 1 GiB of memory per child.

## Configuration

`trenova-imaging serve --config <file>` or `TRENOVA_IMAGING_CONFIG`, with
`TRENOVA_IMAGING__SECTION__KEY` overrides. Unknown keys stop the worker.

| Key | Default | |
|---|---|---|
| `environment` | `production` in the image | `production` requires mutual TLS and hardening |
| `grpc.listen` | `0.0.0.0:50051` in the image | |
| `grpc.tls.cert`, `.key`, `.client_ca` | none | required in production |
| `grpc.max_message_bytes` | 4 MiB | |
| `metrics.listen` | `0.0.0.0:9464` | `/metrics`, `/livez`, `/readyz` |
| `telemetry.log_level`, `.log_format`, `.otlp_endpoint`, `.trace_sample_ratio` | `info`, `json`, none, 1.0 | |
| `pool.size` | CPUs available | |
| `pool.max_jobs_per_child`, `.rss_high_water`, `.address_space`, `.cpu_seconds`, `.open_files` | 500, 768 MiB, 3 GiB, 3600, 64 | |
| `pool.require_hardening` | true | false refused in production |
| `admission.cost_units`, `.memory_bytes`, `.interactive_reserve`, `.max_wait`, `.max_waiters` | 2 × pool, 75% of the pool's address space, 0.25, 250 ms, 64 | |
| `limits.max_input_bytes` | 256 MiB | per call; for a capture batch, per sub-batch |
| `limits.max_capture_page_bytes` | 20 MiB | per capture page, matching `capture.MaxPageBytes` |
| `limits.max_render_pixels`, `.max_image_pixels`, `.max_pages` | 25 M, 100 M, 2,000 | |
| `limits.chunk_bytes`, `.capture_batch`, `.capture_parallelism` | 256 KiB, 8, 4 | |
| `deadlines.probe`, `.render`, `.inspect_capture`, `.assemble`, `.normalize_image` | 30 s, 60 s, 10 min, 5 min, 60 s | caps on the caller's deadline |
| `native.pdfium_library`, `.heif_library`, `.read_only_paths` | the image's paths | |
| `drain_grace` | 30 s | |

The TMS's `imaging` section: `endpoint`, `serverName`, `tls.{enabled,caFile,certFile,keyFile}`
(required outside development), `chunkBytes` (1 MiB), `maxMessageBytes` (4 MiB), `maxAttempts`
(4), `initialBackoff`/`maxBackoff` (100 ms / 5 s), `keepaliveTime`/`keepaliveTimeout`,
`deadlines.*` (30 s probe, 60 s render, 10 min capture, 5 min assemble, 60 s normalize),
`breaker.{consecutiveFailures,openFor,maxOpenFor}` (5, 5 s, 2 min), `resolveInterval` (10 s)
and `prerenderPages` (20).

`trenova-imaging certs` issues a private CA and the server and client certificates into three
directories (server half, client half, CA key); the self-hosted compose file runs it once.

## Metrics and alerts

| Metric | Labels |
|---|---|
| `imaging_calls_total` | `operation`, `priority`, `code` |
| `imaging_call_duration_seconds` | `operation`, `priority` |
| `imaging_input_bytes_total`, `imaging_output_bytes_total`, `imaging_pages_total` | `operation` |
| `admission_decisions_total` | `priority`, `outcome` (Admitted, Waited, Rejected, Draining) |
| `admission_in_flight`, `admission_cost_units_in_use`, `admission_cost_units_capacity`, `admission_memory_bytes_in_use`, `admission_waiting` | |
| `sandbox_live`, `sandbox_size`, `sandbox_busy`, `sandbox_spawned_total`, `sandbox_spawn_failures_total` | |
| `sandbox_killed_total` | `reason` (Deadline, Cancelled, Protocol, Crash) |
| `sandbox_retired_total` | `reason` (Jobs, Rss, Cpu) |
| `sandbox_child_rss_bytes` | |

Traces: every call is a `grpc.call` span whose parent is the TMS's `otelgrpc` client span.
The dashboard is `deploy/observability/grafana/imaging-dashboard.json`; the alerts
(`deploy/observability/prometheus/imaging-alerts.yml`) cover a worker down, the error ratio,
capacity refusals (and any refused interactive call), crashing children and a short pool.

## Page renders in the TMS

Page views are not rendered on every request. After an upload, the thumbnail workflow
pre-renders the first `imaging.prerenderPages` pages' views (1400 × 1800 WebP) and the
thumbnail, and stores them encrypted under keys built from the document's checksum, the page,
the size and the format, so a new version never serves an old render. `GetPageImage` serves the
stored render; on a miss it renders the page at interactive priority behind a singleflight
(one render per key however many viewers ask) and stores it. ETag, immutable caching headers
and the document-view audit are unchanged. Deleting a document deletes its renders.

## Native libraries and licences

| Library | Version | Licence | How it ships |
|---|---|---|---|
| PDFium (bblanchon/pdfium-binaries) | chromium/7881 | BSD-3-Clause; build scripts MIT | downloaded in the image build, SHA-256 checked per architecture |
| libheif | 1.23.4 (Debian trixie) | LGPL-3.0 | Debian package, opened at run time (dynamically linked, replaceable) |
| libde265 (libheif's HEVC plugin) | 1.0.15 | LGPL-3.0 | Debian package |
| libwebp | as vendored by libwebp-sys 0.14.4 | BSD-3-Clause | compiled into the binary |
| Liberation fonts | 2.1.5 | SIL OFL 1.1 | Debian package |

PDFium is pinned to the build pdfium-render's newest bindings target; moving it means moving
pdfium-render with it.

## Blob encryption

Every stored object a document produces (the document itself, an upload session's object, a
capture page and its thumbnail, a document's thumbnail, its stored page renders and its
Docling conversion) is sealed as a **streaming blob** by `encryptionservice.SealBlob` and read
by `OpenBlob`. Small secrets (provider keys, OIDC secrets, tokens) keep the string envelope
(`trenova-envelope:v1:`); a stored object in that format is refused, and there is no reader for
it. Callers do not use the encryption service for objects directly: `storedobject.Write`,
`Open`, `Read` and `Source` (in `modules/documents/app/storedobject`) are the boundary, and the
email service opens a sealed attachment with `OpenBlob` as it downloads it.

**Format** (`encryptionservice/blob.go`), big-endian:

| Part | Bytes | |
|---|---|---|
| magic | 7 | `TRNBLOB` |
| version | 1 | `1` |
| chunk size | 4 | 65,536; anything else is refused |
| nonce prefix | 7 | random per blob |
| AAD hash | 32 | SHA-256 of the associated data |
| key section length | 2 | |
| key section | ≤ 10 KiB | provider, key id, wrapping algorithm, wrapping nonce and wrapped data key, each a 2-byte length and its bytes |
| chunks | 64 KiB + 16 each | AES-256-GCM, the last one shorter (or empty for an empty plaintext) |

- Each blob has its own random 256-bit data key, wrapped by the key manager under the same
  associated data: the local keyring (HKDF-derived key-encryption keys from
  `security.encryption.key`, with `retiredKeys` still opening older wraps) or GCP KMS, which
  fails closed at startup as it does for secrets.
- Chunk *i* is sealed with nonce `prefix ‖ uint32(i) ‖ last`, where `last` is 1 only on the
  final chunk, and with additional data `fixed header ‖ AAD` (the first 51 bytes, then the
  associated data's canonical bytes). Truncating at or inside a chunk, dropping the last chunk,
  appending one, reordering or swapping chunks within a blob, splicing chunks or a key section
  from another blob, and opening under another tenant's or path's associated data all fail
  authentication (`blob_test.go` covers each).
- The key section is not in the chunks' additional data, so a rewrap replaces only the
  header: `trenova encryption rewrap` reads the header, unwraps the data key under the retired
  key, wraps it under the active one, and writes the new header followed by the old chunks
  byte for byte. It spools the chunks to a temporary file rather than memory, because MinIO
  holds an object's lock while it is being read. An object that is not a blob (one stored in
  the retired envelope format) is never rewritten or read: the run counts it under "not
  blobs" and logs its key.
- The associated data is unchanged from the envelope: purpose, organization, business unit and
  the object key (`storedobject.AAD`, `captureObjectAAD`), so a copied object does not open
  under another name.
- `documents.crypto_mode` and `document_upload_sessions.crypto_mode` are `blob_v1` with
  `crypto_version` 2, and objects carry `crypto_mode: blob_v1` in their metadata.

**Streaming end to end.** Nothing in the TMS holds a whole document to encrypt or decrypt it:

- An upload (the session part, `documentservice.Upload` and generated documents stored with
  `Store`) is read from the request, scanned, hashed and sealed as it streams to object
  storage with its exact sealed size, so storage needs no buffering either. Memory is about one
  chunk on each side of the cipher.
- **Malware scanning still comes first.** The bytes go to clamd through `Gate.Stream` as they
  pass, and the sealer withholds the blob's last chunk until the scanner's verdict: an infected
  file or a scanner that cannot answer fails the read that would end the upload, so storage
  aborts it and the object never exists (a large file's earlier parts are discarded with the
  aborted multipart upload). Only after that does anything open the document, so no imaging
  call precedes the scan on any path: uploads and inbound attachments are scanned in the upload,
  and capture pages are scanned before `checkPDF`.
- Reads stream the other way: a download or view is the decrypting reader handed to the HTTP
  response, and `storedobject.Source` gives the imaging adapter a source that downloads and
  decrypts as the gRPC stream sends it, opening the object again for each retry. Thumbnails,
  page views, prerendering and Docling's text profile read through it.
- The Docling submission no longer buffers the file twice: `ConvertRequest` carries an `Open`,
  the client streams a multipart body between a prebuilt prefix and suffix with an exact
  `Content-Length`, and each retry or resubmission opens the document again.
- A tampered object is detected when the reader reaches the damaged chunk, after earlier
  chunks may already have been sent; a download is then cut off rather than completed, and an
  extraction fails as unreadable rather than retrying.

## Benchmarks

Measured with the corpus from `task imaging:corpus` on one machine (12th Gen Intel i7-1270P).
**Before** (`task imaging:bench-go`, since removed, results in
`services/tms/internal/testutil/imagingbench/results/go-baseline.json`): the Go implementation in
one 4-CPU container. **After** (`task imaging:bench`, `results/worker.json`): the same
operations through the TMS's imaging adapter, from a 2-CPU container, against a worker held to
4 CPUs with 4 children. Throughput per core counts the 4 CPUs doing the work in both. Latency is
p50 / p95 in milliseconds over 15 runs after a warm-up.

| Operation | Input | Before p50 / p95 | After p50 / p95 |
|---|---|---|---|
| thumbnail | native-1p.pdf | 46 / 48 | 6.6 / 7.0 |
| thumbnail | scan-bol-1p-gray.pdf | 366 / 373 | 13 / 15 |
| thumbnail | large-5mb.pdf | 522 / 556 | 36 / 40 |
| thumbnail | photo-bol.jpg | 186 / 194 | 19 / 20 |
| page view | native-1p.pdf | 194 / 214 | 106 / 117 |
| page view | scan-bol-1p-gray.pdf | 357 / 366 | 136 / 141 |
| page view | scan-bol-20p-g4.pdf | 614 / 637 | 131 / 148 |
| page view | large-20mb.pdf | 646 / 683 | 264 / 277 |
| capture inspection | capture-cover-2qr.pdf | 267 / 276 | 149 / 155 |
| capture inspection | capture-text-0.pdf | 279 / 286 | 68 / 70 |
| capture inspection | 5 pages, one batch call | (4 goroutines) | 258 / 277 |
| text profile | native-200p.pdf | 1,183 / 1,249 | 105 / 111 |
| text profile | native-10p.pdf | 75 / 78 | 6.8 / 8.9 |
| merge | 50 one-page PDFs | 13 / 15 | 3.8 / 4.9 |
| merge | 10p + 5p scan + rate confirmation | 5.7 / 7.5 | 5.6 / 7.8 |
| split | native-200p.pdf | 93 / 98 | 52 / 56 |
| split | native-50p.pdf | 24 / 30 | 15 / 16 |
| rotate | scan-bol-1p-gray.pdf | 0.7 / 1.9 | 1.8 / 2.8 |
| image to PDF | photo-bol.jpg | 0.6 / 2.6 | 2.6 / 3.6 |

| Throughput (per second, per core) | Concurrency | Before | After |
|---|---|---|---|
| page view, scan-bol-1p-gray.pdf | 1 / 4 / 8 | 2.8 / 7.4 / 7.5 (1.9 per core at 4) | 6.1 / 14.4 / 11.8 (3.6 per core at 4) |
| thumbnail, scan-bol-1p-gray.pdf | 1 / 4 / 8 | 2.9 / 7.6 / 7.6 | 67 / 161 / 140 |
| capture inspection, one page | 1 / 4 / 8 | 4.1 / 9.6 / 9.9 | 12.8 / 35.4 / 30.8 |

| TMS peak RSS, upload → encrypt → store → download → decrypt → thumbnail | Before (Go) | Worker, whole-file envelope | Worker, streaming blobs |
|---|---|---|---|
| idle, after a warm-up | 319–335 MB | 37 MB | 37 MB |
| 5 MB PDF | 319 MB | 87 MB | 41 MB |
| 20 MB PDF | 353 MB | 215 MB | 42 MB |
| 52 MB PDF | 551 MB | 440 MB | 43 MB |

Reading the numbers:

- A thumbnail is drawn straight at its size instead of rendering the page at full resolution and
  scaling it down in Go, which is most of the 10–30× gain.
- At concurrency 8 the worker's throughput falls a little below concurrency 4: four children
  are busy and the rest wait in admission, as designed.
- **Rotate and image to PDF are slower** (by 1–2 ms): they were sub-millisecond pdfcpu edits in
  process, and are now a gRPC call, an admission decision and a hand-off to a child. Nothing
  calls either in a loop; a capture batch's assembly is one call for the whole batch.
- **Peak RSS no longer grows with the document.** With the whole-file envelope the file, its
  ciphertext and its base64 JSON were held at once on both sides of storage; with streaming
  blobs a 52 MB document peaks 6 MB above idle, the chunk buffers and the gRPC stream's
  window. The idle footprint fell from about 330 MB (the WebAssembly heaps) to 37 MB.
