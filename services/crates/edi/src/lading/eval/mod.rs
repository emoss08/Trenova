//! Runs a checked program: maps on messages, CSV, JSON and documents, rules on documents,
//! with lineage, coverage, unread data and the lookups a run still needs.

pub mod csv;
pub mod inbound;
pub mod interp;
pub mod json;
pub mod outbound;
pub mod rules;
pub mod run;
pub mod stdlib;
pub mod tree;
pub mod value;
