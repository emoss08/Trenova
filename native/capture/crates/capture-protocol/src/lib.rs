//! What the Trenova Capture companion says, to the API and to itself.
//!
//! [`api`] mirrors the server's JSON (`captureservice` and the `capture`
//! domain in `services/tms`) field for field. [`helper`] is the framed pipe
//! between the per-user agent and the short-lived scan helper it starts for
//! each scan. [`manifest`] is the digest a batch is sealed with.

pub mod api;
pub mod helper;
pub mod manifest;

pub use manifest::{ManifestBuilder, page_checksum};
