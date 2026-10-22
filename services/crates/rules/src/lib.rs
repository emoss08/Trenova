//! The rules the browser and the server must agree on, written once.
//!
//! Trenova prices shipments with Tariff, its rating language, splits charges between payers,
//! formats report numbers, ranks rate agreement lanes and picks rate matrix cells. Each of these used to
//! live twice, once in Go and once in TypeScript, and drifted. This crate is the only
//! implementation: it compiles to one WebAssembly module (`task rust:wasm-rules`) that the TMS
//! runs in wazero and the web app loads lazily, so the editor's diagnostics, the server's
//! validation and the evaluated price come from the same bytes.
//!
//! The module is pure computation: no clock, no randomness, no I/O and no imports. Every call is
//! one JSON request in and one JSON response out through `abi` (on `wasm32`); [`ops`] routes it by name.
//! Numbers are exact decimals throughout, never `f64`. Text positions on the wire are UTF-16
//! code units, which is what both CodeMirror and the server's statement traces count in.
//!
//! See `docs/engineering/rules-engine.md` for the operations, their wire shapes and budgets.

#[cfg(target_arch = "wasm32")]
pub mod abi;
pub mod allocation;
pub mod lane;
pub mod matrix;
pub mod message;
pub mod ops;
pub mod report;
pub mod span;
pub mod tariff;
pub mod wire;
