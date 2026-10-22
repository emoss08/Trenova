//! The metered interpreter and everything it runs on.
//!
//! The server prices every shipment with this, so its semantics are exact and its failures
//! loud: one exact decimal number type, text in characters, maps in key order, every step and
//! every byte counted, and every failure an [`EvalError`] with a stable code rather than a
//! number that happens to come out.
//!
//! - [`interp`]: the interpreter over the syntax tree; lets, predicates, short circuits,
//!   optional chains, missing-value errors, lookups and statement traces.
//! - [`value`]: run-time values, shared and cheap to clone, maps in key order.
//! - [`tree`]: value trees on the wire, numbers read exactly, `$date` and `$duration` tags.
//! - [`number`], [`time`]: exact decimal arithmetic, the civil calendar and Go durations.
//! - [`operators`], [`builtins`]: what operators and functions do to values.
//! - [`budget`]: the step, text and item budgets every program runs under.
//! - [`tables`]: rate tables and the lookups that read them, with what each lookup matched.
//! - [`trace`]: values as statement traces show them.
//! - [`error`]: evaluation errors with stable codes.
//! - [`ops`]: the `formula.evaluate` operation.

pub mod budget;
pub mod builtins;
pub mod error;
pub mod interp;
pub mod number;
pub mod operators;
pub mod ops;
pub mod tables;
pub mod time;
pub mod trace;
pub mod tree;
pub mod value;

pub use budget::Limits;
pub use error::Code;
pub use error::EvalError;
pub use tables::Tables;
pub use tree::parse_value;
pub use tree::write_value;
pub use value::Kind;
pub use value::Value;
