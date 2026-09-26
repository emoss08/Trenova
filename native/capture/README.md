# Trenova Capture

The Windows companion that puts paper into Trenova: a tray agent that pairs a computer to a
person, takes scan requests from the web app, scans through TWAIN or WIA, and uploads every
page. The design, and why it is Rust, is in
[docs/design/document-capture.md](../../docs/design/document-capture.md).

## Layout

| Crate | What it is | Runs on |
|---|---|---|
| `crates/capture-protocol` | The server's capture JSON, the agent↔helper pipe, the seal digest | anywhere |
| `crates/capture-imaging` | CCITT G4 and JPEG encoding, one-page PDFs, BMP decoding | anywhere |
| `crates/capture-twain` | TWAIN 2.5: bindings, session, negotiation, memory transfer | Windows (tested anywhere against a scripted DSM) |
| `crates/capture-wia` | WIA 2.0 scanning | Windows |
| `crates/capture-client` | API, pairing, device stream, encrypted spool, uploader | anywhere |
| `crates/capture-platform` | DPAPI, Credential Manager, machine identity, registry, shell | Windows |
| `bins/trenova-capture-scan` | The per-scan helper, built x64 and x86 | Windows |
| `bins/trenova-capture` | The tray agent; its core is a library that runs anywhere | Windows |

## Building

Release builds are made on Windows with the MSVC toolchain (`rust-toolchain.toml` pins it and
both targets). The CRT is linked statically, so nothing else needs installing.

```powershell
cargo build --release --target x86_64-pc-windows-msvc -p trenova-capture -p trenova-capture-scan
cargo build --release --target i686-pc-windows-msvc -p trenova-capture-scan
```

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
batch). The Windows-only code is checked, and linted, for both targets:

```bash
cargo clippy --workspace --all-targets --target x86_64-pc-windows-msvc
cargo clippy --workspace --all-targets --target i686-pc-windows-msvc
```

It runs only on Windows: the DSM loader and message pump, WIA, DPAPI, Credential Manager, and
the tray itself. Scanner behaviour is verified against the reference lab in the design doc
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
