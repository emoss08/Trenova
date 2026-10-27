//! Tariff, Trenova's rating language: a small, statically typed, pure scripting language whose
//! types are freight (money, rates, weights, distances, durations, percents, lanes and stops).
//! `docs/engineering/formula-language.md` is the language reference; this module implements it.
//!
//! - [`units`]: exact unit algebra and dimensions.
//! - [`types`]: the types the checker infers and input values are decoded against.
//! - [`schema`]: the data roots, the schema enums, and the table and charge code shapes.
//! - [`context`]: the libraries, tables, charge codes and settings a file is read with.
//! - [`docs`]: the docs the editor, the catalog and the inputs form show: the author's words or
//!   translated text.
//! - [`syntax`]: the lexer, the error-tolerant parser, the syntax tree, the formatter, tokens
//!   for highlighting, and loading a file with the libraries it imports.
//! - [`check`]: types, units, inference, flow narrowing, exhaustiveness, modules, inputs and
//!   diagnostics with fixes.
//! - [`eval`]: values with units, the metered interpreter, the bill and its total clauses, rate
//!   tables and the test runner.
//! - [`editor`]: completion, hover, signature help, go to definition, rename and inlay hints.
//! - [`stdlib`]: the standard library, written in Tariff and compiled into the module.
//! - [`wire`]: values on the wire, decoded against their types and encoded canonically.
//!
//! Each area's `ops` module holds the `tariff.*` operations it answers.

pub mod check;
pub mod constraint;
pub mod context;
pub mod docs;
pub mod editor;
pub mod eval;
pub mod schema;
pub mod stdlib;
pub mod syntax;
pub mod types;
pub mod units;
pub mod wire;
