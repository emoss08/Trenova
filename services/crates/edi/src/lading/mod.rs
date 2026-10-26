//! Lading, the language an organisation brings its own EDI to Trenova in.
//!
//! See `docs/engineering/lading.md`, which is normative: a project of partner, guide, map,
//! rules, flow and library files; maps that turn X12, CSV or JSON into Trenova documents and
//! documents into X12; rules with outcomes; flows Trenova runs per conversation; tests,
//! properties, coverage, lineage and replay; and the editor's operations.

pub mod check;
pub mod eval;
pub mod facts;
pub mod flow;
pub mod project;
pub mod replay;
pub mod schema;
pub mod stdlib;
pub mod syntax;
pub mod testing;
pub mod types;
pub mod tz;
pub mod values;
