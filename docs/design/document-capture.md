# Document Capture — Scanning and Virtual Printing

> Status: design, not started. Purpose: let a person put paper or another program's output
> into Trenova without first producing a file on their own disk, from a scanner (including one
> behind Kofax VRS) or from the Windows print dialog of any application.

---

## 1. What this is, in one sentence

A Windows companion, **Trenova Capture**, acquires pages from a TWAIN/WIA scanner or from a
"Trenova" printer and streams them to the API as a **capture batch**; the batch is split into
proposed documents, and each is filed either straight onto the record the person was looking at
or, from an **intake queue**, onto whichever record it belongs to.

## 2. Why the current approach cannot do it

Every way a document enters Trenova today starts with a file:

| Path | Entry point | Assumes |
|---|---|---|
| Web upload | `documenthandler` `/documents/uploads/`, `useDocumentUpload` | a file on the user's disk |
| Driver portal | `driverportalservice/documents.go` | a photo on the phone |
| Inbound email | `inboundhandler` → `inboundmessageservice/attachments.go` | someone emailed a file |

A back-office user with a stack of PODs has to scan to a folder, find the files, and upload
them one at a time, then attach each to its shipment. A browser cannot talk to TWAIN or WIA,
and it cannot be a printer, so neither of those steps can be removed from inside the web app.
Something has to run on the machine.

## 3. Decisions made

These follow from the product answers (Windows only, our own companion, both landing options,
batch scanning with assignment). Everything below is built on them.

1. **The companion is written in Rust.** See §4.
2. **The browser never talks to the companion.** The web app creates a *capture request* on the
   server; the companion, which holds one authenticated stream to the API, receives it. There is
   no localhost port the browser calls. See §6.3.
3. **The virtual printer is an IPP printer on loopback**, driven by Windows' own inbox IPP Class
   Driver. We ship no print driver and no port monitor DLL. See §5.4.
4. **Pages, not files, are the unit of capture.** A batch is an ordered list of pages; a document
   is a range of them. Splitting, merging, reordering and rotating are edits to that list and
   never re-scan anything.
5. **The server is authoritative for splitting and routing.** The companion reports what the
   device told it (patch codes, TWAIN barcodes); separator and cover-sheet detection runs on
   the server, once, for scans and print jobs alike.
6. **A filed capture item becomes an ordinary `document.Document`** through the existing upload
   and intelligence pipeline. Capture adds a front door; it does not fork document storage,
   versioning, packet rules or extraction.

## 4. Why Rust for the companion

The companion's job is mostly FFI against C and COM ABIs, running in the same process as
third-party scanner drivers, while parsing input from other local processes and holding a
credential.

- **The ABIs.** TWAIN is a C ABI (`DSM_Entry` in `twaindsm.dll`); WIA 2.0, the print spooler,
  DPAPI and Credential Manager are Win32/COM. Rust reaches TWAIN through `bindgen` over
  `twain.h`, and everything else through `windows-rs`, which Microsoft maintains and generates
  from the Windows metadata. This is the same surface C++ would use, with no wrapper layer.
- **Memory safety where it matters.** The companion parses IPP requests from any local process,
  PDF and raster print data, and image buffers handed over by driver code we do not control. A
  memory bug there is a local privilege boundary problem in a process holding a Trenova
  credential. Rust removes that class of bug from our side of the boundary.
- **Deployment.** One statically linked `.exe` per architecture, static CRT, no runtime or VC++
  redistributable to install. That matters for GPO/Intune rollouts.
- **The 32-bit problem.** Many scanner TWAIN drivers, and older Kofax VRS installs, are 32-bit
  only. A 64-bit process cannot load them. `cargo build --target i686-pc-windows-msvc` produces
  the 32-bit scan helper from the same crate (§5.2). Go can target 386, but TWAIN wants its own
  OS thread with a message loop, plus callbacks arriving on driver threads, and doing that
  through cgo on both architectures is where Go costs more than it gives.
- **Why not C++.** It is the traditional language for this, and TWAIN's samples are C, but it
  offers nothing here that Rust lacks and gives up the memory-safety argument.
- **Why not C#/.NET.** NTwain makes .NET the common choice for TWAIN applications, but it
  brings a runtime dependency and a larger attack surface for no gain.

The cost is that the tray UI is written against Win32 directly. That is acceptable because the
companion's UI is deliberately tiny (§5.5): all real work (filing, review, splitting) happens
in the web app.

## 5. The companion

### 5.1 Processes

```
┌──────────────────────── Windows machine ─────────────────────────┐
│                                                                  │
│  TrenovaCaptureSvc (Windows service, LocalService)               │
│   └─ IPP listener 127.0.0.1:<port>  ◄── Windows spooler          │
│        (IPP Class Driver, "Trenova" printer)                     │
│   └─ per-user spool  %ProgramData%\Trenova\Capture\spool\<SID>\  │
│   └─ named pipe \\.\pipe\trenova-capture-<SID> (ACL: that SID)   │
│                                                                  │
│  trenova-capture.exe (per user, starts at logon, tray icon)      │
│   └─ device credential (Credential Manager, DPAPI)               │
│   └─ API stream + upload queue, local spool (DPAPI-encrypted)    │
│   └─ spawns per scan:                                            │
│        trenova-capture-scan-x64.exe  (64-bit TWAIN / WIA)        │
│        trenova-capture-scan-x86.exe  (32-bit TWAIN)              │
└──────────────────────────────────────────────────────────────────┘
```

- **The per-user agent** is where identity lives. It holds the device credential, keeps the
  stream to the API, owns the upload queue, and starts scans. It runs in the user's session,
  which TWAIN needs because data sources, VRS's QC window included, show UI.
- **The service** exists because the spooler, not the user, connects to the printer, and on an
  RDS/Citrix host many users share one machine. The service receives the job, attributes it
  to the submitting user (§5.4), and hands it to that user's agent over a named pipe whose
  ACL admits only that user's SID. If the agent is not running, the job waits in a spool
  directory ACL'd to that SID and the service, and is handed over at next logon.
- **Scan helpers** are short-lived child processes, one per scan, in the bitness of the chosen
  data source. A driver that crashes takes down the helper, not the agent. The helper speaks a
  length-prefixed protocol on an anonymous pipe (page metadata + page bytes) and knows nothing
  about the network.

### 5.2 Scanning

**Source enumeration.** The agent asks both helpers (`MSG_GETFIRST`/`MSG_GETNEXT` against
`twaindsm.dll` in each bitness) and WIA's device manager, and merges the lists. Each source is
reported to the API with its bitness, protocol and capabilities, so the web app can offer it.

**TWAIN session.** The helper walks the TWAIN state machine (states 1–7) on a dedicated thread
with a hidden window and message loop:

- Negotiate from the capture profile (§7.6): `ICAP_XRESOLUTION`/`YRESOLUTION`,
  `ICAP_PIXELTYPE`, `CAP_FEEDERENABLED`, `CAP_DUPLEXENABLED`, `CAP_XFERCOUNT = -1`, and, where
  the source supports them, `ICAP_AUTODISCARDBLANKPAGES`, `ICAP_PATCHCODEDETECTIONENABLED` and
  `ICAP_BARCODEDETECTIONENABLED`. A capability the source rejects is recorded, not fatal. VRS
  answers these itself, and its own blank-page removal and deskew are better than ours.
- **Show the driver UI or not** is a profile setting. With VRS, showing it gives the operator
  VRS's QC review.
- Transfer with `TWSX_MEMORY` (buffered memory). It is the mode every source must support,
  it handles large feeder batches without temp files, and it is the one VRS documents for
  high-volume integration. Read extended image info (`DAT_EXTIMAGEINFO`) per page for patch
  codes and barcodes.
- Paper jams, cover-open and double-feed conditions come back as TWAIN condition codes. The
  helper reports them; the agent keeps the pages already scanned and offers "continue batch"
  so a jam on page 40 does not throw away 39 pages.

**WIA 2.0** is the fallback for devices without a TWAIN driver: `IWiaDevMgr2` over `windows-rs`,
feeder and duplex through `WIA_IPS_DOCUMENT_HANDLING_SELECT`.

**Page encoding** happens in the helper, so raw bitmaps never cross a process boundary:

| Pixel type | Encoding | Letter @ 300 dpi |
|---|---|---|
| Black & white | CCITT Group 4 | ~40–80 KB |
| Grayscale / colour | JPEG, quality from profile (default 80) | ~200–600 KB |

Each page is wrapped as a single-page PDF (`pdf-writer`), and that is what gets uploaded. PDF is
the one format the rest of the document pipeline (thumbnails, `go-fitz` text extraction, OCR
fallback) already takes.

### 5.3 Upload and offline behaviour

- Pages go into a local spool (DPAPI-encrypted, per user) **before** upload, so an unplugged
  network, sleep, or a crash never loses a scan.
- Upload is per page and idempotent on `(batch_id, sequence)`. A retried page is a no-op.
  Exponential backoff with jitter; the tray shows "12 pages waiting to upload".
- A batch is **sealed** with its page count and a manifest digest once acquisition ends. The
  server processes a batch only after it is sealed and every page is present.

### 5.4 Virtual printing

The installer creates a printer named **Trenova** using the **Microsoft IPP Class Driver**,
pointed at the service's loopback IPP endpoint (`http://127.0.0.1:<port>/ipp/print`).

- **Why IPP.** Windows Protected Print Mode, rolling out on Windows 11, removes third-party print
  drivers and port monitors. An IPP printer on the inbox class driver keeps working under it.
  A v3/v4 driver or a port-monitor DLL loaded into `spoolsv.exe` would stop working, and it
  also runs our code inside the spooler as SYSTEM, which is exactly what we don't want.
- **What we accept.** The service advertises `document-format-supported` =
  `application/pdf`, `image/pwg-raster`. PDF keeps the text layer (so extraction needs no OCR);
  PWG raster is the universal fallback and is converted to per-page PDF in the service.
- **Attribution.** The IPP `requesting-user-name` is a hint, not proof, since any local process
  can connect. The service confirms the owner by matching the job against the local spooler
  queue (`EnumJobs` on the Trenova printer, `JOB_INFO_2.pUserName` plus the submitting
  session), and rejects a job it cannot attribute. The listener binds loopback only and caps
  job size.
- **Where a print lands.** In the intake queue, unless the user *armed* a print destination in
  the web app ("Print into this shipment", §8.2). The agent then shows a toast ("3 pages sent to
  intake — Open").

**Must verify before building (spike, §12 phase 0):** that the IPP Class Driver accepts a
loopback IPP URL (`Add-Printer -IppURL` or the equivalent) on Windows 10 22H2 and Windows 11 with
Protected Print Mode on, and which formats it sends to a printer advertising PDF. The fallback,
if loopback is refused, is a **Print Support App virtual printer** (Windows 11 22H2+), an
MSIX-packaged component Windows sanctions for exactly this. It needs a packaged WinRT
background task, which would be the only non-Rust piece.

### 5.5 Tray UI

- Sign in / sign out (pairing, §6.1), connection state, pending uploads, the last few batches
  with "Open in Trenova".
- "Scan to intake…": pick source and profile, then scan. This is the no-browser path.
- Toasts for completed batches, jams, and failed uploads.
- Nothing else. Destination choice, splitting and filing live in the web app, so there is one
  implementation of each and it is the one that gets the design system.

### 5.6 Repository layout

```
native/capture/                  Cargo workspace
├── crates/
│   ├── capture-protocol/        API + helper-pipe message types (serde)
│   ├── capture-client/          API client (reqwest, rustls + platform verifier), stream, upload queue, spool
│   ├── capture-twain/           twain.h bindings, state machine, capability negotiation
│   ├── capture-wia/             WIA 2.0 over windows-rs
│   ├── capture-imaging/         G4/JPEG encode, single-page PDF writer, PWG raster → PDF
│   ├── capture-ipp/             IPP/2.0 server subset (Print-Job, Validate-Job, Get-Printer-Attributes, Get-Jobs)
│   └── capture-platform/        DPAPI, Credential Manager, named pipes + ACLs, spooler queries, toasts
├── bins/
│   ├── trenova-capture/         per-user agent + tray
│   ├── trenova-capture-svc/     service
│   └── trenova-capture-scan/    scan helper (built x64 and x86)
└── installer/                   WiX v4 MSI
```

## 6. Identity and the channel to the server

### 6.1 Pairing

This is the OAuth 2.0 Device Authorization Grant shape (RFC 8628), implemented in `authservice`.

1. The agent calls `POST /capture/v1/pair` and receives a `device_code` and a short `user_code`
   (8 characters, 10-minute life).
2. It opens the browser to `/capture/pair?code=ABCD-EFGH`. The signed-in user sees the machine
   name, Windows user and agent version, and approves.
3. The agent, polling with the `device_code`, receives a **refresh token** (opaque, stored hashed
   the way `apikey` stores secrets, rotated on every use, reuse detection revokes the device)
   and a 15-minute **access token**.
4. Tokens are kept in Windows Credential Manager, DPAPI-protected to that Windows user.

Pairing polls and code attempts are rate-limited per IP and per code. A user can revoke a device
from their settings; an admin can revoke any device in the organisation.

### 6.2 The principal

A new principal type `capture_device` in `authctx`, which **carries the paired user's ID** as
well as org and BU. This matters: `documentservice.Upload` requires `UploadedByID` from
`TenantInfo.UserID`, which is why an `api_key` principal cannot upload documents today.

Device tokens are narrow. They reach only `/capture/v1/*`: pair, stream, sources, batches and
pages, and releases. They cannot call the rest of the API. Actions still run the **user's**
permission checks (document create, read on the target record), so a device can never do more
than its user.

### 6.3 Server → companion: the device stream

The agent keeps one SSE connection, `GET /capture/v1/stream`, carried on the existing realtime
broker with the audience set to the **device**. The browser asks for a scan by creating a
`capture.Request` through GraphQL; the server publishes it to that device's stream. Progress
flows back through the batch's status and is published to the user's browser as ordinary
resource invalidations.

Why not have the browser call the companion on `localhost`:

- Chrome's Local Network Access now prompts before a public origin calls loopback, and
  enterprise policies vary. A relay has no prompt and no policy dependency.
- A localhost HTTP API is a CSRF target for every web page the user visits. Having no
  listener removes it; the IPP listener takes no commands, only print data.
- It works identically when the browser and the companion are in different sessions (browser on
  a laptop, scanner on a shared workstation, a Citrix-published browser).

Presence: the agent heartbeats on the stream, and `capture.Device.LastSeenAt` plus the broker's
connection state give the web app an "online" dot per device.

## 7. Server domain (`internal/core/domain/capture`)

### 7.1 Device — `capture_devices`, prefix `cdev_`

The paired user, org and BU, display name, machine name, Windows user, agent version and
architecture, status (`Active`, `Revoked`), `LastSeenAt`, refresh-token hash and family ID
(for reuse detection), and a **sources snapshot** (JSONB: each scanner's name, protocol,
bitness and supported capabilities).

### 7.2 Request — `capture_requests`, prefix `creq_`

A browser-initiated intent: device, mode (`Scan` or `Print`), target (`ResourceType`,
`ResourceID`, optional `DocumentTypeID`), profile and source, status (`Pending`, `Delivered`,
`InProgress`, `Completed`, `Canceled`, `Expired`, `Failed`, with a failure code), and an expiry.
Scan requests expire after 2 minutes if the device never picks them up. An armed print expires
after 10 minutes, or on the next print job.

### 7.3 Batch — `capture_batches`, prefix `cbat_`

One acquisition. It records:

- source (`Scan` or `Print`), device, capturing user, optional request;
- scan settings as negotiated, or print job name and submitting application;
- expected page count and manifest digest (set when sealed);
- target, copied from the request, if any.

Statuses: `Receiving` → `Sealed` → `Processing` → `Ready` → `Filed` | `PartiallyFiled` |
`Discarded` | `Expired`, or `Failed`.

### 7.4 Page — `capture_pages`, prefix `cpg_`

Batch, sequence, storage path (envelope-encrypted with the same `CryptoMode` as documents),
checksum, size, pixel dimensions and dpi, rotation, thumbnail path, blank score, and **markers**
(JSONB: patch code, barcode payloads, verified cover-sheet reference, separator flag).

### 7.5 Item — `capture_items`, prefix `citm_`

A proposed document within a batch. It holds:

- an ordered list of page IDs;
- the suggested document type and target, with the **provenance** of each suggestion
  (`CoverSheet`, `Request`, `Classifier`, `Person`) and a confidence;
- status: `Proposed` → `Filed` | `Discarded`;
- once filed, the `DocumentID` and who filed it, when.

A page belongs to at most one live item. Edits are optimistic-locked on the batch `Version`, so
two people cannot split the same stack two ways.

### 7.6 Profile — `capture_profiles`, prefix `cprf_`

Organisation-level scan presets: dpi, pixel type, duplex, feeder, blank-page removal, JPEG
quality, whether to show the driver UI, **separator strategy**
(`None`, `PatchCode`, `CoverSheet`, `BlankPage`, `FixedPageCount(n)`, or a combination), and a
default flag. Users pick one; the admin page manages them.

### 7.7 Cover sheet — `capture_cover_sheets`, prefix `ccs_`

A printable separator that says where the following pages go: target record and document type,
issued by, issued at, and optional expiry. The QR code encodes
`TRNV1:<cover-sheet-id>:<hmac>`, with the HMAC keyed per organisation. A sheet scanned in
another tenant, or a forged one, fails verification and is treated as a plain separator. Filing
still re-checks the user's permission on the target, so a valid sheet never grants access.

### 7.8 Organisation settings

On a `CaptureControl`, alongside the existing per-tenant controls:

- capture enabled;
- **review policy** for cover-sheet-routed items: `AlwaysReview`, `ReviewBelowConfidence` or
  `AutoFile`. This is the same three-way choice `inboundmessage.ReviewPolicy` makes, so it is
  lifted into a shared type rather than copied.
- unfiled batch retention (default 30 days, with a reminder at 7 days left);
- minimum agent version;
- auto-update allowed.

## 8. Flows

### 8.1 Scan from the Shipment Documents tab

1. `DocumentsTab` gains a **Scan** split button: device (defaulting to the last used and
   online), source and profile. It also gains **Print cover sheets**.
2. The client calls `createCaptureRequest(target: shipment, documentTypeId?)`. The server
   checks `document:create` and read on the shipment, then publishes to the device.
3. The agent runs the scan and uploads pages. The tab shows the batch live, thumbnails appearing
   as pages land (realtime invalidation on the batch).
4. When processing finishes:
   - **One item and a known document type**: it is filed immediately. The person chose the
     target by clicking Scan on it, so no review is owed.
   - **Several items** (separators were found), or **no document type**: a review sheet opens
     in the tab. It uses the same page-strip editor as intake (§8.3), with the target
     pre-filled and document types suggested by the classifier.

### 8.2 Print into a shipment

**Print into this shipment** on the same tab creates a `Print` request with a 10-minute
expiry. The tab shows "Waiting for a print to Trenova…". The next job the user prints on that
device fulfils it and follows the same completion path as §8.1.4. With nothing armed, a print
goes to intake.

### 8.3 Intake queue

This is a new route, `/intake`, built on the `/inbox` pattern: `folder-rail`, list, reading
pane, triage keys.

- **Rail:** Mine, Everyone's (for users with the wider data scope on `capture_batch`), Filed,
  Discarded, and a filter by source (Scan or Print).
- **List:** one row per batch, with source, device, page count, age, and a
  "3 of 5 filed" progress count.
- **Reading pane:** the **page strip**. It shows thumbnails in order, with a split marker
  between items, and supports:
  - drag to reorder;
  - click a gap to split, or a marker to merge;
  - rotate, delete a page, move a page to another item;
  - a larger preview of the focused page.
- **Filing panel,** one per item:
  - record picker (shipment, worker, tractor, trailer, customer, carrier: whatever the owner
    model accepts), document type, and **File**;
  - suggestions (target and type from the classifier, pre-filled and marked with `AssistMark`);
  - cover-sheet routes, shown as pre-filled and labelled with where they came from.
- **Keyboard:** `s` split at focus, `m` merge, `r` rotate, `f` file, `x` discard, `j`/`k` move.
- **Bulk:** file every item whose suggestion meets the confidence bar, in one action, with a
  summary of what went where.

### 8.4 Cover sheets

"Print cover sheets" on a shipment builds a PDF with one sheet per **missing required document
type**, read from `documentpacketrule` and the packet summary the tab already shows. Each sheet
names the shipment, the document type and a QR code. Operators interleave them into a stack;
the server splits on them and routes each item, and the review policy decides whether that
filing waits for a person. Cover sheets can also be printed blank (split-only) from intake.

### 8.5 Filing (every path ends here)

`captureservice.FileItem(itemID, target, documentTypeID)`:

1. It checks, as the **acting person**, `document:create` and read on the target via
   `document.OwnerResource()`.
2. It assembles the item's pages into one PDF. An untouched print job reuses the original PDF,
   so the text layer survives.
3. It creates the document through `documentuploadservice` with a new `ProcessingProfile`,
   `capture`, which `SupportsIntelligence`. It gets the same encryption, checksum, thumbnail,
   extraction, packet-rule and auto-ready-to-invoice behaviour as any upload.
4. It marks the item `Filed`, rolls the batch status forward, publishes invalidations for the
   batch and the target's document list, and writes an audit entry.

Filing is idempotent per item: a retried file returns the existing document.

## 9. Processing (Temporal)

`ProcessCaptureBatchWorkflow`, on a `CaptureTaskQueue`, starts when a batch is sealed:

1. Per page, in parallel with a bound: rasterize (`go-fitz`), then compute:
   - the thumbnail;
   - the blank score (ink coverage after thresholding);
   - QR/barcode payloads (`gozxing`), with cover sheets verified by HMAC.
2. Propose items from the profile's separator strategy, the device-reported patch codes and the
   verified cover sheets. Separator pages are dropped from items but kept on the batch, so a
   wrong split can be undone.
3. Suggest a type and target per item. This reuses the document-intelligence analysis
   (text extraction, OCR, feature and provider detection, parsing rules, the kind → type
   mapping, and the reference matching inbound email uses to find a shipment from PRO/BOL
   numbers) through one analyzer port. Today that logic sits inside
   `documentintelligencejobs/activities.go` and only runs against a stored `Document`; it is
   extracted so both callers share it rather than copied.
4. Move the batch to `Ready`. Apply auto-filing where the request or review policy allows it.

Two scheduled workflows go with it:

- `ExpireCaptureWorkflow` expires stale requests, sends the retention reminder, and
  discards and purges batches past retention.
- `ReconcileCaptureWorkflow` catches batches whose processing was lost.

Both follow `ReconcileDocumentUploadsWorkflow`.

## 10. API surface

**Companion (REST, `capture_device` principal only),** under `/capture/v1/`:

| Method | Path | Purpose |
|---|---|---|
| `POST` | `pair`, `pair/token` | Device authorisation grant |
| `POST` | `token/refresh` | Rotate the refresh token, issue an access token |
| `GET` | `stream` | SSE: requests, revocation, config |
| `PUT` | `sources` | Sources snapshot |
| `POST` | `batches` | Open a batch (from a request or ad hoc) |
| `PUT` | `batches/:id/pages/:seq` | Upload one page (idempotent) |
| `POST` | `batches/:id/seal` | Seal with count + digest |
| `POST` | `requests/:id/status` | Delivered, in progress, failed (with code) |
| `GET` | `releases/latest` | Signed update manifest |

Page uploads are raw bodies, not multipart, capped per page (20 MB) and per batch (1,000 pages).
They are content-sniffed and must decode as a PDF with exactly one page, or a PDF/PWG print job.

**Web app (GraphQL),** next to `inboundmessage.graphqls`:

- queries `captureBatches` (a connection with the `IncludeTotalCount` gate), `captureBatch`,
  `myCaptureDevices`, `captureProfiles`;
- mutations `createCaptureRequest`, `cancelCaptureRequest`, `editCaptureItems` (split, merge,
  reorder, rotate, delete pages, with batch version), `fileCaptureItem`, `fileCaptureItems`,
  `discardCaptureBatch`, `createCoverSheets`, `revokeCaptureDevice`/`revokeMyCaptureDevice`,
  and profile and control CRUD.

Every root resolver reaches a permission check (authzlint). New resources: `capture_batch`,
`capture_device`, `capture_profile`, `capture_control`. Page and thumbnail bytes are served
through short-lived presigned URLs, as document previews are now.

## 11. Security summary

- **Device credentials:** hashed at rest, rotated on use, reuse detection, per-device
  revocation, 15-minute access tokens. On the client they live in Credential Manager under
  DPAPI.
- **Device principal:** narrow routes; all actions check the paired user's permissions.
- **No browser-to-localhost API.** The IPP listener is loopback-only, attributes every job
  through the spooler, and takes no commands.
- **Local isolation:** named pipes and spool directories are ACL'd per user SID. Scan helpers
  run in isolated processes with no network or credential.
- **Cover sheets:** HMAC'd per organisation, and never a grant of access.
- **At rest:** pages are envelope-encrypted like documents, and purged with the batch.
- **Updates:** the manifest is signed (ed25519, key pinned in the binary). The MSI is
  Authenticode-signed and checked with `WinVerifyTrust` before the service installs it.
  Organisations can pin a minimum version or turn auto-update off and deploy through Intune or
  GPO (the MSI takes `TRENOVAURL` and `AUTOUPDATE` properties for silent install).
- **Existing gap, not introduced here:** uploads are not virus-scanned
  (`StatusQuarantined` is never set). Capture widens intake, so this is worth closing, but it
  is its own change.

## 12. Build order

Every phase is shipped complete; the order only reflects dependencies.

0. **Spikes (must pass before phase 4):**
   - the loopback IPP printer under the IPP Class Driver on Windows 10 22H2 and Windows 11
     with Protected Print Mode, and which formats it sends;
   - TWAIN memory transfer in both bitnesses against a VRS-equipped scanner, plus one of each
     common family (Fujitsu/Ricoh fi, Canon DR, Epson DS, Kodak Alaris).
1. **Server domain and companion API:** migrations, domain, repositories (buncolgen),
   pairing and the `capture_device` principal, the device stream, batch and page upload, the
   processing workflow, filing, and the analyzer-port extraction.
2. **Web:**
   - the `/intake` route and page-strip editor;
   - Scan, Print-into and cover sheets on the Documents tab;
   - device settings and the capture admin (profiles, control, device fleet, installer
     download);
   - product guide regeneration.
3. **Companion core:** agent, tray, pairing, stream, spool/upload queue, TWAIN helper (x64 and
   x86) and WIA.
4. **Virtual printer:** service, IPP server, attribution, pipe handoff, printer installation.
5. **Distribution:**
   - MSI (WiX v4), signing, and the update manifest and flow;
   - a `native-capture.yml` workflow on `windows-latest`: `cargo fmt`/`clippy`/`test`, both
     targets, MSI build.

## 13. Open questions

1. **Code signing:** who owns the certificate? An EV certificate or Azure Trusted Signing
   avoids SmartScreen warnings on first run.
2. **Test hardware:** which scanners and VRS versions do the pilot customers run? That fixes
   the phase-0 test lab.
3. **RDS/Citrix:** do any customers publish Trenova from a terminal server? The design supports
   it; it only changes how much testing phase 4 needs.
4. **Shared intake:** is "Everyone's" (by data scope) enough, or do teams need named queues,
   such as a billing intake separate from a dispatch intake?
