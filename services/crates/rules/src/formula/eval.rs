//! The metered interpreter and everything it runs on.
//!
//! - [`value`]: run-time values, shared and cheap to clone, maps in key order.
//! - [`tree`]: value trees on the wire, numbers read exactly, `$date` and `$duration` tags.
//! - [`number`], [`time`]: exact decimal arithmetic, the civil calendar and Go durations.
//! - [`operators`], [`builtins`]: what operators and functions do to values.
//! - [`budget`]: the step, text and item budgets every program runs under.
//! - [`tables`]: rate tables and the lookups that read them, with what each lookup matched.
//! - [`trace`]: values as statement traces show them.
//! - [`error`]: evaluation errors with stable codes.
//! - `ops`: the `formula.evaluate` operation.

pub mod budget;
pub mod builtins;
pub mod error;
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
