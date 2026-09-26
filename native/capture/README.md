# Trenova Capture

The Windows companion that puts paper into Trenova: a tray agent that pairs a computer to a
person, takes scan requests from the web app, scans through TWAIN or WIA, and uploads every
page; and a print service behind the "Trenova" printer, which hands what each person prints
to their agent. The design, and why it is Rust, is in
[docs/design/document-capture.md](../../docs/design/document-capture.md).

## Layout

| Crate | What it is | Runs on |
|---|---|---|
| `crates/capture-protocol` | The server's capture JSON, the agent↔helper pipe, the seal digest, the print hand-off | anywhere |
| `crates/capture-imaging` | CCITT G4 and JPEG encoding, PDFs of one page or many, BMP and PWG raster decoding | anywhere |
| `crates/capture-ipp` | The IPP/2.0 codec and the Trenova printer's operations | anywhere |
| `crates/capture-twain` | TWAIN 2.5: bindings, session, negotiation, memory transfer | Windows (tested anywhere against a scripted DSM) |
| `crates/capture-wia` | WIA 2.0 scanning | Windows |
| `crates/capture-client` | API, pairing, device stream, encrypted spool, uploader | anywhere |
| `crates/capture-update` | The release decision and the verified installer download | anywhere |
| `crates/capture-platform` | DPAPI, Credential Manager, accounts and SIDs, registry, paths, logging, shell | Windows |
| `bins/trenova-capture-scan` | The per-scan helper, built x64 and x86 | Windows |
| `bins/trenova-capture` | The tray agent; its core is a library that runs anywhere | Windows |
| `bins/trenova-capture-svc` | The print service; its listener, attribution and hand-off are a library that runs anywhere | Windows |
| `bins/trenova-capture-update` | The updater service; its flow is a library that runs anywhere | Windows |
| `bins/trenova-capture-release` | The release build's tool: key pair, signed manifest, verification | anywhere |

## Building

Release builds are made on Windows with the MSVC toolchain (`rust-toolchain.toml` pins it and
both targets). The CRT is linked statically, so nothing else needs installing.

```powershell
cargo build --release --target x86_64-pc-windows-msvc -p trenova-capture -p trenova-capture-scan -p trenova-capture-svc -p trenova-capture-update
cargo build --release --target i686-pc-windows-msvc -p trenova-capture-scan
```

`TRENOVA_CAPTURE_PUBLIC_KEY` (the base64 ed25519 key releases are signed with) must be in the
environment when the agent and the updater are built; a build without it never updates itself.
CI takes it from a repository variable.

The installer ships the helper as `trenova-capture-scan-x64.exe` and
`trenova-capture-scan-x86.exe` beside `trenova-capture.exe`. A development build finds the
plain `trenova-capture-scan.exe` of its own bitness in the same folder.

## Testing

Everything that does not touch Windows runs anywhere:

```bash
cargo fmt --all -- --check
cargo clippy --workspace --all-targets
cargo test --workspace
```

That covers the TWAIN state machine against a scripted data source, the API client, stream
and uploader against a mock server, the spool, the tray's menu and status, and the agent end
to end (a request from the web app scanned, uploaded and sealed; a jam continued into the same
batch; a print taken from the inbox and sent whole). On the print side: the IPP operations,
PWG raster decoding, attribution against a scripted print queue, and the listener end to end
over HTTP. The Windows-only code is checked, and linted, for both targets:

```bash
cargo clippy --workspace --all-targets --target x86_64-pc-windows-msvc
cargo clippy --workspace --all-targets --target i686-pc-windows-msvc
```

It runs only on Windows: the DSM loader and message pump, WIA, DPAPI, Credential Manager, the
tray itself, and the print service's host, spooler queries, ACLs and installation. Scanner behaviour is verified against the reference lab in the design doc
(§5.7).

## Running a development build

```powershell
trenova-capture.exe --server http://localhost:8080 --sign-in
```

`--server` saves the address for this Windows user (`HKCU\SOFTWARE\Trenova\Capture\ServerUrl`);
an address under `HKLM\SOFTWARE\Policies\Trenova\Capture` overrides it, and the installer
writes one to `HKLM\SOFTWARE\Trenova\Capture`. `--sign-in` starts pairing at once. Plain HTTP
is accepted only for `localhost`.

The agent keeps its files in `%LOCALAPPDATA%\Trenova\Capture`: `logs\agent.<date>.log` (fourteen
days; `TRENOVA_CAPTURE_LOG=debug` for more), and `spool\`, where every page waits, encrypted
with DPAPI, until the server has it. Batches the server refused are kept in `spool\failed\`
with the reason. The device credential is in Credential Manager as
`Trenova Capture/<server host>`.

## Regenerating the TWAIN bindings

`crates/capture-twain/src/sys/{x86_64,x86}.rs` are generated from `twain.h` at a pinned
commit of [twain/twain-dsm](https://github.com/twain/twain-dsm), checked by SHA-256. To move
the pin, change it in the script and run it (it needs `bindgen-cli` and libclang):

```bash
crates/capture-twain/tools/generate-bindings.sh
```

## Running the print service

From an elevated prompt:

```powershell
trenova-capture-svc.exe install          # the service (LocalService, its own SID), started
trenova-capture-svc.exe install-printer  # the "Trenova" printer on the IPP Class Driver
```

`uninstall` and `uninstall-printer` undo them; each can be run again safely. `run` serves in
the console instead of as a service, for development. The printer listens on
`http://127.0.0.1:8631/ipp/print`; `PrintPort` (a DWORD under
`HKLM\SOFTWARE\Policies\Trenova\Capture` or `HKLM\SOFTWARE\Trenova\Capture`) moves it, and
`install-printer` must be run again after changing it.

The service keeps its logs in `%ProgramData%\Trenova\Capture\logs\service.<date>.log`, and
leaves each print in `%ProgramData%\Trenova\Capture\spool\<SID of the person who printed>\`,
which only that person, the service, SYSTEM and Administrators can open. The agent takes it
from there within two seconds, or at its next start. A job the agent could not read is renamed
`.rejected` there, not deleted.

## The installer

`installer/Package.wxs` is the MSI (WiX v4): the agent, both helpers, the two services, the
Trenova printer, the machine settings and a Start menu shortcut, as one per-machine package
that upgrades in place. Build it on Windows after both release builds above:

```powershell
dotnet tool install --global wix
wix extension add -g WixToolset.Util.wixext
./installer/build.ps1            # installer\out\TrenovaCapture-<version>-x64.msi
./installer/build.ps1 -Sign      # also signs the executables and the MSI; see sign.ps1
```

A silent install takes the server address, whether computers update themselves, and the
printer's port:

```powershell
msiexec /i TrenovaCapture-1.0.0-x64.msi /qn TRENOVAURL=https://app.example.com AUTOUPDATE=0 PRINTPORT=8631
```

## Releases and updates

A release is a `capture-vX.Y.Z` tag on a commit whose `Cargo.toml` says `X.Y.Z`. The
`native-capture.yml` workflow builds and signs the MSI, signs the release manifest with
`trenova-capture-release` (the private key is the `TRENOVA_CAPTURE_SIGNING_KEY` secret),
publishes both on that release, and copies the manifest to the rolling `capture-stable`
release, which servers read (`update.captureManifestUrl`, verified with
`update.capturePublicKey`) and mirror at `/api/v1/capture/releases/latest/`.

The agent checks that address at sign-in and every six hours. When a newer release is
published, the organization allows self-update and the computer's policy does not forbid it,
the agent starts the updater service, which verifies everything itself (the manifest's
signature, the download's size and SHA-256, the MSI's Authenticode signature and that its
signer is the same publisher as the running updater) before running the installer. Otherwise
the tray says a version is available for IT to install.

To make a key pair once:

```powershell
trenova-capture-release keygen
```

Keep the private half in the CI secret and nowhere else; the public half goes in the
`TRENOVA_CAPTURE_PUBLIC_KEY` repository variable and in each server's configuration. The
updater keeps its logs in `%ProgramData%\Trenova\Capture\logs\updater.<date>.log` and
Windows Installer's in `install.log` beside them.
