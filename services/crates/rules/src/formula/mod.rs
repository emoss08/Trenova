//! The formula language: one grammar, checker, formatter and metered evaluator for the Formula
//! Studio editor and for pricing on the server.
//!
//! - `syntax`: the lexer, the parser (error-tolerant, never panics) and the formatter.
//! - `catalog`: every function and operator the language has, with its shape and docs.
//! - `check`: types, diagnostics with fixes, `let` rules, nullable guards, the analysis the
//!   Studio shows.
//! - `eval`: values, the metered interpreter, the built-in functions, rate table lookups and
//!   statement traces.
//! - `ops`: the static operations (`formula.catalog`, `formula.check`, `formula.format`,
//!   `formula.signature`, `formula.identifier`); `eval::ops` holds `formula.evaluate`.

pub mod catalog;
pub mod check;
pub mod eval;
pub mod ops;
pub mod syntax;
