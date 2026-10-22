//! What runs: values with units, the metered interpreter, the bill and its total clauses, rate
//! tables, the test runner and the `tariff.evaluate` and `tariff.test`
//! operations. Exact decimals throughout, no clock, no randomness, maps in key order: the same
//! answer in the browser and on the server.
//!
//! - [`value`], [`operators`], [`methods`], [`lists`], [`text`], [`time`]: values and what can
//!   be done with them.
//! - [`interp`]: the interpreter over the syntax tree; [`run`]: one program from inputs to
//!   [`run::Outcome`], the step the hosts gate behind the checker ([`run::run_program`]).
//! - [`bill`]: charge lines, notes, total clauses, rounding and exchange.
//! - [`tables`]: rate table lookups, loaded on demand.
//! - [`record`]: the host's record and the sample record; [`declared`]: declared types, for
//!   decoding inputs.
//! - [`budget`], [`trace`], [`error`]: budgets, the rate trace, refusals and errors.
//! - [`tests`]: the test runner.
//! - [`ops`]: the operations.

pub mod bill;
pub mod budget;
pub mod declared;
pub mod error;
pub mod interp;
pub mod lists;
pub mod methods;
pub mod operators;
pub mod ops;
pub mod record;
pub mod run;
pub mod tables;
pub mod tests;
pub mod text;
pub mod time;
pub mod trace;
pub mod value;

use crate::tariff::syntax::ast::NodeId;
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::types::Type;

/// What the checker knows about a program's types, for the few places a value cannot say what
/// it is: the `sum` of an empty list, a percent scaled by a number, whether `min`/`max` give an
/// item or a selected value, and whether an index outside a list is `none`.
pub trait StaticTypes {
    /// The static type of expression `node` in module `module`, when known.
    fn type_of(&self, module: ModuleId, node: NodeId) -> Option<Type>;
}

/// No static types: every question gets the dynamic default.
#[derive(Debug, Clone, Copy, Default)]
pub struct NoTypes;

impl StaticTypes for NoTypes {
    fn type_of(&self, _: ModuleId, _: NodeId) -> Option<Type> {
        None
    }
}
