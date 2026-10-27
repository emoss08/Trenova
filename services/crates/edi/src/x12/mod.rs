//! X12: reading an interchange against its guides, acknowledging it, and writing segments.
//!
//! [`read`] is the whole reading of one text: separators, segments, envelopes, the loop tree of
//! each transaction set and every finding. [`ack`] turns a reading's groups into 997s or 999s,
//! and [`write`] turns segments into text. The rest are the parts those are built from.

pub mod ack;
pub mod element;
pub mod envelope;
pub mod finding;
pub mod guide;
pub mod lexer;
pub mod ops;
pub mod read;
pub mod separators;
pub mod structure;
pub mod write;
