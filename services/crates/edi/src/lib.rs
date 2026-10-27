//! Trenova's EDI core, written once for the browser and the server.
//!
//! An organisation sets up its own trading partners in Trenova, without an EDI specialist. That
//! takes three things the product must get exactly right and show as a person works: reading a
//! partner's file, checking it against the partner's implementation guide, and answering it with
//! the acknowledgment the standard prescribes. This crate does all three for X12. It compiles to
//! one WebAssembly module (`task rust:wasm-edi`) that the TMS runs in wazero, so the designer,
//! the inspector and production read a file with the same bytes.
//!
//! The module is pure computation: no clock, no randomness, no I/O and no imports. Every call is
//! one JSON request in and one JSON response out through `abi` (on `wasm32`); [`ops`] routes it
//! by name. Text positions on the wire are byte offsets into the request's text.
//!
//! See `docs/engineering/edi-x12.md` for the operations, their wire shapes and limits, and
//! `docs/notes/edi/self-serve-edi.md` for the design this is the first part of.

#[cfg(target_arch = "wasm32")]
pub mod abi;
pub mod lading;
pub mod ops;
pub mod x12;
