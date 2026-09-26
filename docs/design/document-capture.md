# Document Capture — Scanning and Virtual Printing

> Status: all five phases complete; the Windows-only code runs for the first time in the phase-5 CI, and the phase-0 lab gates a release. Purpose: let a person put paper or
> another program's output into Trenova without first producing a file on their own disk, from a
> scanner (including one behind Kofax VRS) or from the Windows print dialog of any application.

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

These follow from the product answers: Windows only, our own companion, both landing options,
batch scanning with assignment, one intake queue with filters rather than named queues, Trenova
owns the code-signing certificate, and customers run the cloud product (terminal-server
deployments are not a target). Everything below is built on them.

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
7. **x64 only.** Scanner vendors do not ship TWAIN drivers for Windows on ARM, Ricoh's fi series
   does not run there, and Kofax VRS has known ARM issues, so an ARM build would pair and find
   nothing to scan with. `capture.Architecture` accepts `x64` alone.

## 4. Why Rust for the companion

The companion's job is mostly FFI against C and COM ABIs, running in the same process as
third-party scanner drivers, while parsing input from other local processes and holding a
credential.

- **The ABIs.** TWAIN is a C ABI (`DSM_Entry` in `twaindsm.dll`); WIA 2.0, the print spooler,
  DPAPI and Credential Manager are Win32/COM. Rust reaches TWAIN through declarations
  `bindgen` generates from the TWAIN Working Group's `twain.h` (checked in for both bitnesses by
  `capture-twain/tools/generate-bindings.sh`, from a pinned `twain-dsm` commit verified by
  SHA-256, with every structure's size and offsets asserted at compile time), and everything
  else through `windows-rs`, which Microsoft maintains and generates from the Windows metadata.
  This is the same surface C++ would use, with no wrapper layer, and no libclang in the build.
- **Memory safety where it matters.** The companion parses IPP requests from any local process,
  PDF and raster print data, and image buffers handed over by driver code we do not control. A
  memory bug there is a local privilege boundary problem in a process holding a Trenova
  credential. Rust removes that class of bug from our side of the boundary.
- **Deployment.** One statically linked `.exe` per architecture, static CRT, no runtime or VC++
  redistributable to install. That matters for GPO/Intune rollouts.
- **The 32-bit problem.** Many scanner TWAIN drivers are 32-bit only, and Kofax VRS 5.3's
  TWAIN source appears to be too: it installs under `Program Files (x86)` and requires only the
  x86 VC++ runtime, and no Tungsten document mentions a 64-bit source (§5.7). A 64-bit process cannot load them. `cargo build --target i686-pc-windows-msvc` produces
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
│  TrenovaCaptureSvc (Windows service, LocalService + service SID) │
│   └─ IPP listener 127.0.0.1:<port>  ◄── Windows spooler          │
│        (IPP Class Driver, "Trenova" printer)                     │
│   └─ per-user inbox  %ProgramData%\Trenova\Capture\spool\<SID>\  │
│        (ACL: that SID, the service, SYSTEM, Administrators)      │
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
- **The service** exists because the spooler, not the user, connects to the printer, and a
  print must be accepted even before the tray agent has started. (It also keeps a shared
  workstation correct, although terminal servers are not a target.) The service receives the job, attributes it
  to the submitting user (§5.4), and leaves it in that user's **inbox**, a directory whose ACL
  admits only that user's SID and the service. The agent looks every two seconds; a job
  printed while it is not running waits there until the next logon. The hand-off is a file
  contract (`capture_protocol::handoff`): the PDF is written under a `.part` name and renamed,
  then its JSON description the same way, so a description on disk means its PDF is whole.
  The agent verifies the PDF against the description's SHA-256, moves it into its own
  DPAPI spool under a key derived from the job's name, and only then deletes it, so a crash in
  between sends the job once. A job that cannot be read is renamed `.rejected`, not deleted.
- **Scan helpers** are short-lived child processes, one per scan, in the bitness of the chosen
  data source. A driver that crashes takes down the helper, not the agent. The helper speaks a
  length-prefixed protocol on an anonymous pipe (page metadata + page bytes) and knows nothing
  about the network.

### 5.2 Scanning

**Source enumeration.** The agent asks both helpers (`MSG_GETFIRST`/`MSG_GETNEXT` against
`twaindsm.dll` in each bitness, and WIA's device manager in the 64-bit one) and merges the
lists, keeping the 64-bit driver when a TWAIN source is installed in both. Each source is
reported to the API with its bitness and protocol. A TWAIN source is listed without being
opened, because opening one whose scanner is off often shows the driver's own error dialog;
what it can do (duplex, feeder, patch codes, resolutions, pixel types) is read when it is
first opened for a scan and reported then. WIA opens devices without UI, so WIA sources are
described at once.

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
  `application/pdf`, `image/pwg-raster` (black_1, sgray_8 and srgb_8 at 300 and 600 DPI;
  letter, legal and A4), and implements Print-Job, Validate-Job, Create-Job with
  Send-Document, Cancel-Job, Get-Job-Attributes, Get-Jobs and Get-Printer-Attributes
  (`capture-ipp`). PDF keeps the text layer (so extraction needs no OCR) and is passed through.
  PWG raster is the universal fallback: the service decodes it a page at a time and writes one
  PDF, bilevel pages as G4 and the rest as JPEG at quality 85, and a colour page with no colour
  in it as gray. The server splits either kind into pages (`PUT …/print-job/`).
- **Attribution.** The IPP `requesting-user-name` is a hint, not proof, since any local process
  can connect. The service reads the Trenova queue (`EnumJobs`, `JOB_INFO_2`): the document
  must match a queued job by name (exactly, or by prefix for names of 32 characters or more,
  which a driver may cut short; failing any match, the only unclaimed job on the queue). Each
  queued job can be claimed by **one** document, so a process that copies a name it can see on
  the queue takes that job's place rather than adding a document to someone's intake. A name
  that several people are printing at once, a claim that contradicts the queue, or a user
  name signed in under two domains is refused. The domain comes from the signed-in sessions
  (`WTSEnumerateSessions`), and `LookupAccountName` gives the SID that names the inbox.
- **The listener** binds 127.0.0.1 only (default port 8631; `PrintPort` under
  `HKLM\SOFTWARE\Policies\Trenova\Capture` or `HKLM\SOFTWARE\Trenova\Capture` moves it). It
  refuses, before reading a body, any request a web page could send: one with an `Origin`
  header, a `Host` other than this loopback address (DNS rebinding), or a content type other
  than `application/ipp`. Requests are capped at 512 MB, and at most two documents are
  converted at once.
- **Isolation.** The service runs as LocalService with an unrestricted service SID, and every
  directory it creates carries a protected DACL naming `NT SERVICE\TrenovaCaptureSvc` rather
  than LocalService, which other services share. The spool root lets signed-in users traverse
  but not list it.
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

The tray icon is the Trenova logo, `client/apps/web/public/logo.ico` (it already carries the 16,
24 and 32 px sizes a tray needs). The agent's build script embeds that file as its Win32 icon
resource, so there is one source for the mark rather than a copy that drifts.

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
│   ├── capture-protocol/        API + helper-pipe message types (serde), manifest digest, print hand-off
│   ├── capture-client/          API client (reqwest over SChannel), pairing, stream, spool, upload queue
│   ├── capture-twain/           twain.h bindings, state machine, capability negotiation, memory transfer
│   ├── capture-wia/             WIA 2.0 over windows-rs
│   ├── capture-imaging/         G4/JPEG encode, PDF writer (one page or many), BMP and PWG raster decode
│   ├── capture-ipp/             IPP/2.0 codec and the Trenova printer's operations
│   ├── capture-update/          release decision and the verified download, for the agent and the updater
│   └── capture-platform/        DPAPI, Credential Manager, accounts and SIDs, ACLs, settings, paths, logging, shell
├── bins/
│   ├── trenova-capture/         per-user agent + tray
│   ├── trenova-capture-svc/     print service: listener, attribution, inboxes, install
│   ├── trenova-capture-update/  updater service: verify, download, WinVerifyTrust, msiexec, relaunch
│   ├── trenova-capture-release/ the release build's signing tool (keygen, sign, verify)
│   └── trenova-capture-scan/    scan helper (built x64 and x86)
└── installer/                   WiX v4 MSI (Package.wxs), build.ps1, sign.ps1
```

TLS is the platform's own (SChannel, through `native-tls`) rather than rustls: it trusts what
the Windows certificate store trusts, which is what an organization that inspects TLS has
already configured, and it needs no C toolchain for either target. The agent's core
(`trenova-capture`'s library) and `capture-client` know nothing of Windows, so the whole flow
from a web-app request to a sealed batch runs in tests on Linux against a mock server and a
scripted scanner; `native/capture/README.md` says what runs where.

### 5.7 Supported scanners

Trucking imaging vendors do not publish certified lists: Transflo supports "any TWAIN
compatible scanner", and McLeod and Trimble name no models publicly. Trenova Capture also works
with any TWAIN or WIA source; the list below is what we test against and recommend. Every model
listed is on Tungsten's Scanner Configurator as VRS-certified.

| Tier | Model | TWAIN 64 | TWAIN 32 | WIA | Patch codes | Notes |
|---|---|---|---|---|---|---|
| Desk | Ricoh fi-8040 | Yes | Yes | Yes | Yes (PaperStream IP) | 40 ppm, 50-sheet ADF |
| Desk | Canon DR-C225 II | Unverified | Yes | Yes | No | 25 ppm, 45-sheet ADF |
| Desk | Epson DS-530 II | Yes (Epson Scan 2) | Yes | Unverified | Unverified | 35 ppm, 50-sheet ADF |
| Workgroup | **Ricoh fi-8170** (reference) | Yes | Yes | Yes | Yes | 70 ppm, 100-sheet ADF; replaces fi-7160 |
| Workgroup | Canon DR-M260 | Unverified | Yes | Yes | Unverified | 60 ppm, 80-sheet ADF, ships with VRS Pro |
| Workgroup | Epson DS-790WN | Yes (Epson Scan 2) | Yes | Yes | Unverified | 45 ppm, network |
| Workgroup | Brother ADS-4900W | Unverified | Yes | Yes | Unverified | 60 ppm, 100-sheet ADF |
| Workgroup | Alaris S2080w | Yes | Yes | Yes | Unverified (barcodes yes) | 80 ppm, 80-sheet ADF |
| Production | Ricoh fi-8190 | Yes | Yes | Yes | Yes | 90 ppm, 100-sheet ADF |
| Production | Canon DR-G2110 | Yes | Yes | Yes | Yes | 110 ppm, 500-sheet ADF |

Legacy, supported but not recommended: Fujitsu fi-7160 (replaced by the fi-8170) and Panasonic
KV-S (Panasonic left the market in 2023; support ends 2029).

Two consequences for the build:

- **Patch-code separation is a TWAIN feature.** Ricoh notes PaperStream IP features, patch codes
  included, may be unavailable over WIA, so profiles offer patch codes only for TWAIN sources.
  Cover sheets work on every source, because the server reads them.
- **Epson Scan and Epson Scan 2 cannot be installed together**, and only Epson Scan 2 is 64-bit.
  The companion lists both bitnesses, so either works, but the support note says which to use.

The phase-0 lab is the reference fi-8170 with VRS 5.3, one Canon DR, one Epson DS and one Alaris.
The 64-bit VRS question is settled there with the free VRS Elite trial.

## 6. Identity and the channel to the server

### 6.1 Pairing

This is the OAuth 2.0 Device Authorization Grant (RFC 8628), in `captureservice/pairing.go`.

1. The agent calls `POST /api/v1/capture/pair/` and receives a `deviceCode` and an
   eight-letter `userCode` shown as `XXXX-XXXX` (consonants only, per RFC 8628 §6.1; 10-minute
   life; unique among open codes by a partial index).
2. It opens the browser to `/capture/pair?code=XXXX-XXXX`. The signed-in person sees the
   machine name, Windows user, agent version and IP (the `captureDevicePairing` query) and
   approves or denies (`approveCaptureDevicePairing` / `denyCaptureDevicePairing`). Approval binds the grant to
   that person and the organization they are signed in to, and needs `capture_batch:create`.
3. The agent polls `POST /api/v1/capture/pair/token/` every 5 seconds and gets RFC 8628 error
   bodies (`authorization_pending`, `slow_down`, `access_denied`, `expired_token`,
   `invalid_grant`) until approval, then its credential, exactly once. A second exchange of the
   same grant is `invalid_grant`.
4. The credential is a 15-minute access token (`tcd_at_…`) and a refresh token (`tcd_rt_…`),
   both 32 random bytes stored as SHA-256 only. `POST /api/v1/capture/token/refresh/` rotates
   both; the replaced refresh hash is kept, and presenting it again revokes the device, since
   it means the credential exists in two places. A device idle for 90 days is revoked on its
   next refresh. The refresh is also where the person's current standing is checked: capture
   still enabled, `capture_batch:create` still held, and the agent at least the organization's
   minimum version (answered `426` with `minimumVersion` so the tray can offer to update).
5. On Windows the tokens live in Credential Manager, DPAPI-protected to that Windows user.

A person can revoke their own device; revoking anybody else's takes `capture_device:update`.
Revocation clears both hashes and closes the device's stream.

### 6.2 The principal

A new principal type `capture_device` in `authctx`, which **carries the paired user's ID** as
well as org and BU. This matters: `documentservice.Upload` requires `UploadedByID` from
`TenantInfo.UserID`, which is why an `api_key` principal cannot upload documents today.

Device tokens are narrow: `capturehandler.RequireDevice` guards `/api/v1/capture/device/*` and
nothing else, and the session middleware never accepts a device token. The device group has no
CSRF check, because nothing there is reachable with a cookie, but it keeps the per-person rate
limit and the control-plane entitlement check. Every action runs the **person's** permission
checks through the permission engine, so a device can never do more than its person. It shares
the person's rate budget (600 requests a minute by default), which a companion honouring
`Retry-After` stays well inside.

### 6.3 Server → companion: the device stream

The agent keeps one SSE connection, `GET /api/v1/capture/device/stream/`. It is opened on the
existing realtime gateway as the device's person, and the handler forwards only what the device
needs:

- `ready`, `heartbeat`, `reset` and `close`, as the web app's stream has them;
- `capture.request`, when an invalidation addressed to the person names this device: fetch
  `GET /api/v1/capture/device/requests/`;
- `capture.revoked`, after which the stream ends.

Frames are filtered on raw bytes before any decoding, so a busy tenant costs a device nothing.
The device fetches its requests on `capture.request`, on `reset` and on every reconnect, so a
lost event never loses a request. Heartbeats keep `LastSeenAt` current (at most one write per 30
seconds). The stream is recycled on the same jittered lifetime as the web app's, so a revoked
credential is noticed at the next reconnect even without the event.

Why not have the browser call the companion on `localhost`:

- Chrome's Local Network Access now prompts before a public origin calls loopback, and
  enterprise policies vary. A relay has no prompt and no policy dependency.
- A localhost HTTP API is a CSRF target for every web page the user visits. Having no
  listener removes it; the IPP listener takes no commands, only print data.
- It works identically when the browser and the companion are in different sessions (browser on
  a laptop, scanner on a shared workstation, a Citrix-published browser).

Presence: `CaptureDevice.IsOnline` is true when the device was heard from in the last 90
seconds, which the web app shows as an "online" dot per device.

## 7. Server domain (`internal/core/domain/capture`)

### 7.1 Device — `capture_devices`, prefix `cdev_`

The paired user, org and BU, display name, machine name, Windows user, agent version and
architecture, status (`Active`, `Revoked`), `LastSeenAt` and IP, the access and refresh token
hashes plus the replaced refresh hash (for reuse detection), and a **sources snapshot** (JSONB:
each scanner's name, protocol, bitness, duplex, feeder, patch codes, barcodes, blank discard,
resolutions and pixel types).

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
- status: `Proposed` → `Filing` → `Filed`, or `Discarded`; a failed filing is `Failed` and
  can be filed again;
- once filed, the `DocumentID` and who filed it, when.

A page belongs to at most one live item. Edits are optimistic-locked on the batch `Version`, so
two people cannot split the same stack two ways.

### 7.6 Profile — `capture_profiles`, prefix `cprf_`

Organization-level scan presets: DPI (100–600), pixel type, duplex, feeder, blank-page removal,
JPEG quality, whether to show the driver UI, **separator strategies** (`PatchCode`,
`CoverSheet`, `BlankPage`, `FixedPageCount` with a page count, in any combination), and a
default flag (at most one per tenant, enforced by a partial unique index; setting a new default
clears the old one in the same transaction). Patch sheets and cover sheets divide a stack
whatever the profile says, because a person put them there on purpose. The default is 300 DPI
black and white, duplex, blank pages dropped.

### 7.7 Cover sheet — `capture_cover_sheets`, prefix `ccs_`

A printable separator that says where the following pages go: target record and document type,
issued by, and an expiry (180 days). The QR code encodes `TRNV-CS1:<token>`, where the token is
32 random bytes stored only as its SHA-256, looked up inside the tenant that scanned it. That
replaced the per-organization HMAC in the first draft: a random token needs no key management,
names no record on paper, and a sheet copied into another tenant's stack finds no row. An
unknown, foreign or expired sheet still divides the stack and routes nothing. Filing re-checks
the person's permission on the target, so a valid sheet never grants access.

### 7.8 Organisation settings

On the existing `DocumentControl` rather than a new control, since these are document settings
and the document-intelligence admin page already edits that record:

- `enableCapture`;
- `captureAutoFileCoverSheets`. The first draft had a three-way review policy borrowed from
  inbound mail, but a verified cover sheet is not a guess with a confidence: somebody printed
  it for that record and that document type. The only question is whether the organization
  trusts that without a second look, which is a yes or no. Classifier suggestions always wait
  for a person.
- `captureRetentionDays` (1–365, default 30);
- `captureMinAgentVersion` (`MAJOR.MINOR.PATCH`, compared by `shared/versionutils`);
- `captureAllowAutoUpdate`.

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
2. It assembles the item's pages into one PDF (`pdfcpu`, in `infrastructure/pdfassembly`),
   applying each page's rotation. A print job was split into pages by page tree, not
   rasterized, so each printed page keeps its own text layer when it is joined again.
3. It creates the document through `documentuploadservice` with a new `ProcessingProfile`,
   `capture`, which `SupportsIntelligence`. It gets the same encryption, checksum, thumbnail,
   extraction, packet-rule and auto-ready-to-invoice behaviour as any upload.
4. With a worker, `FileCaptureItemWorkflow` runs the upload pipeline's own finalize workflow as
   a child and records the document it returns; without one, the upload service finalizes
   inline. Either way the item becomes `Filed` with its `DocumentID`, the batch is recounted
   (`PartiallyFiled`, then `Filed`), the cover sheet's use is recorded, an audit entry is written
   and the batch and the target record are invalidated.

Filing is idempotent per item: an item already filing or filed returns as it is, and recording
the same document twice changes nothing. A filing that fails returns the item to the person as
`Failed` with the reason, pages intact.

## 9. Processing (Temporal)

`ProcessCaptureBatchWorkflow`, on the new `capture-queue`, starts when a batch is sealed. Its
activity heartbeats per page, so a worker lost halfway through a long stack is replaced in
minutes.

1. Each page not yet read is rendered at 150 DPI (`go-fitz`, in `infrastructure/captureimaging`)
   and measured: a 240 px JPEG thumbnail (stored encrypted beside the page), ink coverage inside
   a 6% margin (blank below 0.4%), and every QR code on it (`gozxing`). Cover-sheet codes are
   looked up in the tenant. A page that cannot be read is kept and marked `Failed`, never
   dropped. In a `nofitz` build inspection is skipped and pages are still kept and filed.
2. `capture.SplitPages` divides the stack: patch sheets and cover sheets always, blank pages and
   fixed page counts when the profile says so. A cover sheet routes what follows it until the
   next division; a page-count division keeps the route. Separator pages are flagged but kept.
3. Each item gets a suggestion from the most trusted source available: its cover sheet, then
   the record the scan was started from, then its content. Content is read by
   `documentintelligencejobs.CaptureAnalyzer`, which runs the same extraction, OCR,
   classification and parsing rules as a stored document, without storing or recording
   anything, and maps the kind to a document type code. Reference numbers (the extracted
   reference field, then numbers from the text, via `shared/referenceutils`, which inbound mail
   now uses too) are looked up with the inbound shipment finder, which refuses a reference
   naming two shipments.
4. The batch becomes `Ready` (or `Discarded` if nothing survived). Items that need no person are
   filed as the capturing person: the only document of a scan started from a record with a
   document type, and cover-sheet routes when `captureAutoFileCoverSheets` is on. An automatic
   filing the person could not have made stays in intake.

`CaptureMaintenanceWorkflow` runs every five minutes: it expires pairings and requests nobody
picked up, restarts sealed or processing batches untouched for 30 minutes, fails or seals
uploads abandoned for 7 days, expires unfiled batches past retention, and deletes the stored
pages and thumbnails of every batch past retention, filed or not (a filed document has its own
copy).

## 10. API surface

All paths are under `/api/v1/capture/` and documented in the OpenAPI spec (tag `Capture`).

**Public (rate-limited per IP):**

| Method | Path | Purpose |
|---|---|---|
| `POST` | `pair/` | Start a device authorization grant |
| `POST` | `pair/token/` | Poll it; returns the credential once |
| `POST` | `token/refresh/` | Rotate the credential |

**Device (`Authorization: Bearer tcd_at_…` only):**

| Method | Path | Purpose |
|---|---|---|
| `GET` | `device/` | The device record, with the name of the person and organization it acts for |
| `DELETE` | `device/` | Sign out: revoke the device's own credential |
| `GET` | `device/profiles/` | The profiles the person may scan with, for scans started from the tray |
| `PUT` | `device/sources/` | Report reachable scanners |
| `GET` | `device/stream/` | SSE stream (§6.3) |
| `GET` | `device/requests/` | Open requests, oldest first |
| `POST` | `device/requests/:id/status/` | Delivered, in progress, failed or cancelled, with a code |
| `POST` | `device/batches/` | Open a batch; idempotent on `clientKey` |
| `PUT` | `device/batches/:id/pages/:seq/` | Upload one page |
| `PUT` | `device/batches/:id/print-job/` | Upload a whole print job; splits and seals |
| `POST` | `device/batches/:id/seal/` | Seal with page count and manifest digest |

An organization that turned capture off answers `422` with `params.reason` =
`capture_disabled`, so the companion holds every page until capture is back on instead of
treating the refusal as a bad upload.

A page is a raw `application/pdf` body of exactly one page, at most 20 MB, sniffed rather than
trusted, with scanner markers in `X-Capture-Dpi`, `X-Capture-Patch-Code` and repeatable
`X-Capture-Barcode` headers. Uploads are idempotent on `(batch, sequence)`: the same page again
returns the stored page, and a different page at a taken sequence is `409`. A print job is at
most 200 MB. The manifest digest is SHA-256 over each page's SHA-256 hex digest followed by a
newline, in sequence order. A batch holds at most 1,000 pages. The update manifest endpoint
(`releases/latest`) is part of phase 5.

**Signed-in (session):** one REST route, because it serves bytes a browser shows straight from
an `<img>` or a PDF viewer.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `pages/:id/content/` | A page as PDF, or `?kind=thumbnail` |

Page and thumbnail bytes are served through the API because they are encrypted at rest; a
presigned link would hand out ciphertext. `CapturePage.contentPath` and `thumbnailPath` carry the
paths, so the web app never builds them.

**Web app (GraphQL),** in `schema/capture.graphqls`, owned by the document-management feature.
Everything else a signed-in person does is here; the REST routes phase 1 had for pairing,
devices and requests were removed so there is one way to do each thing.

- Queries: `captureBatches` (the intake queue, a cursor connection that counts only when
  `totalCount` is selected), `captureBatch` (pages, items and device), `myCaptureDevices`,
  `captureDevices`, `availableCaptureProfiles`, `captureProfiles`, `captureProfile`,
  `captureRequestsForTarget`, `captureDevicePairing`.
- Mutations: `editCaptureItems` (split, merge, reorder, drop, rotate, against the batch version),
  `fileCaptureItem`, `fileCaptureItems` (a whole stack at once; each document is filed as it
  would be alone, and one that cannot be comes back with its reason while the rest file),
  `discardCaptureItem` (returns the batch with its new counts), `discardCaptureBatch`,
  `createCaptureRequest`, `cancelCaptureRequest`, `createCaptureCoverSheets` (the QR payload is
  returned once), `approveCaptureDevicePairing`, `denyCaptureDevicePairing`,
  `revokeMyCaptureDevice`, `revokeCaptureDevice`, and profile create, update and delete.
- Where an item or batch points at a record, `suggestedRecord`, `filedRecord` and `target` name
  it (a PRO and BOL, a worker's name, a unit number and plate), through one dataloader that asks
  once per record kind per request, and only for a reader who may read that kind of record.

Permission resources: `capture_batch` (read; create = capture, pair, request, print cover
sheets; update = file; delete = discard; data scope decides own versus everyone's intake),
`capture_device` (read and revoke across the organization; a person revokes their own with
`revokeMyCaptureDevice`) and `capture_profile` (administration; choosing one when scanning
needs only `capture_batch:create`). Capture settings live on `document_control`. Every root
resolver reaches a permission check (authzlint).

## 11. Security summary

- **Device credentials:** hashed at rest, rotated on use, reuse detection, per-device
  revocation, 15-minute access tokens. On the client they live in Credential Manager under
  DPAPI.
- **Device principal:** narrow routes; all actions check the paired user's permissions.
- **No browser-to-localhost API.** The IPP listener is loopback-only, attributes every job
  through the spooler, and takes no commands.
- **Local isolation:** each person's print inbox is ACL'd to their SID and the service's own
  SID; the agent's spool is encrypted to the user with DPAPI. Scan helpers
  run in isolated processes with no network or credential.
- **Cover sheets:** random tokens, hashed at rest and resolved only within the scanning tenant;
  never a grant of access.
- **At rest:** pages are envelope-encrypted like documents, and purged with the batch.
- **Updates:** the manifest is signed (ed25519; the public key is built into the agent and the
  updater from `TRENOVA_CAPTURE_PUBLIC_KEY`, and configured on the server as
  `update.capturePublicKey`). The server mirrors the manifest at
  `GET /api/v1/capture/releases/latest/` after verifying it, and the agent verifies it again.
  Installing is the updater service's job (`TrenovaCaptureUpdater`, LocalSystem, demand start,
  startable by signed-in users and nothing more): it re-reads the manifest over HTTPS, verifies
  the signature, refuses anything not newer than itself or needing a newer Windows, downloads
  the MSI checking size and SHA-256, has Windows verify its Authenticode signature
  (`WinVerifyTrust`) and requires the signer to be the same publisher as the running updater,
  and only then runs `msiexec /qn`. An unprivileged process can thus cause a genuine, newer
  Trenova Capture to be installed and nothing else. Organisations pin a minimum version and
  allow or refuse self-update on the server (`captureMinAgentVersion`,
  `captureAllowAutoUpdate`); IT can also refuse it per computer (`AutoUpdate` under
  `HKLM\SOFTWARE\Policies\Trenova\Capture`) and deploy through Intune or GPO (the MSI takes
  `TRENOVAURL`, `AUTOUPDATE` and `PRINTPORT` for a silent install).
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
1. **Server domain and companion API — complete.** Migration `20261231006880_document_capture`
   (and its generated SQLite twin); the `capture` domain with the split rules; repositories on
   buncolgen; pairing, credentials and the `capture_device` principal; the device stream; batch,
   page and print-job intake; processing, auto-filing and filing workflows on `capture-queue`;
   the capture analyzer; maintenance; permissions; OpenAPI. Tested by domain and service unit
   tests over in-memory stores and real PDFs, adapter tests, handler tests, and a repository
   integration test (`-tags integration`) that ran green against Postgres 16.
2. **Web — complete.**
   - the GraphQL schema and resolvers over the phase 1 service, bulk filing, record names for
     intake rows (one dataloader per request), the profile administration service, and removal
     of the REST routes GraphQL replaces;
   - the `/intake` route and page-strip editor, and `capture_batch`, `capture_device` and
     `capture_profile` in the realtime `RESOURCE_QUERY_KEY_MAP`;
   - Scan, Print-into and cover sheets on the Documents tab, including the cover-sheet PDF
     (QR code and label per sheet);
   - the `/capture/pair` approval page, `/capture/devices`, and the capture section of the
     admin (settings, profiles, device fleet);
   - the retention reminder: a week before a stack still waiting on a person loses its unfiled
     pages, its owner is notified once (the stack is claimed through `retention_reminded_at`
     before the notification is sent, so concurrent sweeps notify once);
   - product guides for Intake, scanners and the admin page, i18n in all four catalogs.

   Tested by service tests, loader and mapping tests, authzlint, the projection and
   schema-diff checks, the label and reminder queries against Postgres 16, and web unit tests
   for the page-layout editor, queue filters, destinations, cover-sheet PDF, profile schema and
   realtime keys.
3. **Companion core — complete.** The tray agent (pairing, sign-out, the device stream, the
   encrypted spool and upload queue, scan requests, "Scan to intake", continuing a jammed
   batch, notifications), the scan helper (TWAIN in both bitnesses, WIA), page encoding, and the
   platform services (DPAPI, Credential Manager, registry settings). The server gained the
   device identity and profile routes, device sign-out, the `capture_disabled` reason, and the
   write-coverage entries for every capture write. Tested on Linux by 103 unit and integration
   tests: the TWAIN session against a scripted DSM (enumeration, negotiation and refusals,
   strips of unknown length, extended image info, jams, cancel, teardown order, no leaked
   container), the API client, stream and uploader against a mock server, the spool, and the
   agent end to end; sample pages read back pixel-exact in MuPDF and pass the server's page
   check. The Windows-only code (the DSM loader and message pump, WIA, DPAPI, Credential
   Manager, the tray) is compile-checked and linted for x64 and x86 but has not run: that needs
   the phase-0 lab and the phase-5 Windows CI.
4. **Virtual printer — complete.** `capture-ipp` (the IPP/2.0 codec, bounded in depth and
   attribute count, and the printer's operations over a job handler); PWG raster decoding and
   the multi-page PDF writer in `capture-imaging`; `trenova-capture-svc` (the loopback listener,
   spooler attribution, raster conversion, the ACL'd per-user inbox, the Windows service host,
   and `install`/`uninstall` and `install-printer`/`uninstall-printer`, which runs
   `Add-Printer -IppURL` from the system PowerShell); and the agent's side, which takes jobs
   from the inbox into its spool and sends each whole to `print-job`, where the server attaches
   the armed destination. The service needs no running agent.

   **Changed from the plan:** the hand-off is the per-user inbox directory, not a named pipe.
   The directory was needed anyway for jobs printed while the agent is not running, and with
   the agent polling it, a pipe would have been a second path to the same place: another
   listener, another ACL and another protocol, for a two-second saving on a print.

   Tested on Linux by 141 tests across the workspace, including the IPP operations and their
   refusals, every raster layout and hostile raster streams, the attribution rules, the
   hand-off contract, the listener end to end over HTTP (a raster job to a G4 PDF in the owner's
   inbox; a replay refused; each browser-reachable request refused before its body), and the
   agent end to end (a job in the inbox sent whole to the armed destination). A converted
   job reads back in MuPDF as two letter pages, the bilevel one pixel-exact. The Windows-only
   code (the service host, spooler and session queries, SID lookups, directory ACLs and the
   install commands) is compile-checked and linted for x64 and x86. It has not run, and the
   phase-0 spike on the IPP Class Driver still gates a release.
5. **Distribution — complete.**
   - The signed release manifest (`capture_protocol::release`, `capturereleaseservice` on the
     server; a fixture signed by the Rust tool is verified by the Go tests, so the two
     languages are held to the same bytes), the `trenova-capture-release` tool, the server's
     `releases/latest` mirror and the `captureAgentRelease` query, and the update policy a
     device learns when it describes itself (and on a 426).
   - The updater service and the agent's update flow: a check at sign-in and every six hours,
     the updater started when both the organization and the computer allow it, a note for IT
     when they do not, and never during a scan.
   - The MSI (`installer/Package.wxs`): agent, helpers, both services, the printer, the
     machine settings, a Start menu shortcut; major upgrades; `configure` commands for what
     Windows Installer cannot declare. `build.ps1` and `sign.ps1` (Azure Trusted Signing, a
     store certificate or a PFX).
   - The installer download on `/capture/devices` and the admin Computers tab, with the
     version, size, Windows requirement and SHA-256 from the manifest.
   - `native-capture.yml`: Linux (fmt, clippy for every target, tests, the sample PDFs read
     back in MuPDF) and Windows (tests, both targets, the MSI); a `capture-vX.Y.Z` tag signs
     and publishes the MSI and manifest and moves the rolling `capture-stable` manifest.

   **Changed from the plan:** agents are closed and restarted around an install by the
   updater (and by the MSI's final `relaunch` step for IT-deployed upgrades) rather than left
   to Windows Installer's restart manager, which under `/qn` would have scheduled a reboot
   instead. The `install`/`uninstall` service commands remain for development; the MSI owns
   the services and calls `configure` for the ACLs and the printer.

   Tested on Linux by 159 tests across the workspace and the server's Go tests. The Windows
   code (services, WinVerifyTrust and the publisher check, the session relaunch, the DACLs, the
   MSI itself) runs first in the Windows CI job, and against real hardware in the phase-0 lab.

## 13. Answered questions

1. **Code signing:** Trenova owns the certificate. An EV certificate or Azure Trusted Signing
   avoids SmartScreen warnings on first run.
2. **Test hardware:** no customer list exists, so §5.7 sets the supported list and the lab.
3. **Terminal servers:** not a target; customers use the cloud product, and self-hosters run
   the documented Docker deployment.
4. **Queues:** one intake queue with filters and sorting; no named queues.
