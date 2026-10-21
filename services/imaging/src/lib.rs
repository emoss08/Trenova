//! The imaging worker: stateless, tenant-agnostic document and image compute
//! for the TMS, behind the `trenova.imaging.v1` gRPC contract.
//!
//! The binary is two programs. `serve` is the supervisor: tokio and tonic
//! through `trenova-service-kit`, admission control, streaming, metrics and
//! health; it never parses a document. `child` is what the supervisor's
//! `trenova-sandbox` pool runs: PDFium, the image codecs and libheif, in a
//! hardened process that handles one job at a time.

pub mod certs;
pub mod child;
pub mod config;
pub mod engine;
pub mod protocol;
pub mod server;
pub mod supervisor;
