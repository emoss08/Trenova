//! What runs: values with units, the metered interpreter, the bill and its total clauses, rate
//! tables, the test runner and the `tariff.evaluate`, `tariff.test` and `tariff.scaffoldTest`
//! operations. Exact decimals throughout, no clock, no randomness, maps in key order: the same
//! answer in the browser and on the server.
//!
//! - [`value`], [`operators`], [`methods`], [`lists`], [`text`], [`time`]: values and what can
//!   be done with them.
//! - [`interp`]: the interpreter over the syntax tree; [`run`]: one program from inputs to
//!   [`run::Outcome`], the step the hosts gate behind the checker ([`run::run_program`]).
//! - [`bill`]: charge lines, notes, total clauses, rounding and exchange.
//! - [`tables`]: rate table lookups, loaded on demand.
//! - [`record`]: the host's record and the sample record.
//! - [`budget`], [`trace`], [`error`]: budgets, the rate trace, refusals and errors.
//! - [`tests`]: the test runner, [`property`] its properties and [`coverage`] the regions they
//!   reach; [`scaffold`]: a test written from a real shipment.
//! - [`ops`]: the operations.

pub mod bill;
pub mod budget;
pub mod coverage;
pub mod error;
pub mod interp;
pub mod lists;
pub mod methods;
pub mod operators;
pub mod ops;
pub mod property;
pub mod record;
pub mod run;
pub mod scaffold;
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
/// it is: the `sum` of an empty list, a percent scaled by a number, the declared type of each
/// input, which the value a host sends for it is decoded against, and each input's constraint,
/// which that value must satisfy.
pub trait StaticTypes {
    /// The static type of expression `node` in module `module`, when known.
    fn type_of(&self, module: ModuleId, node: NodeId) -> Option<Type>;
    /// The declared type of the root file's input `name`, when known.
    fn input_type(&self, name: &str) -> Option<Type>;
    /// The folded `where` constraint of the root file's input `name`, when it has one.
    fn input_constraint(&self, _name: &str) -> Option<&crate::tariff::constraint::Constraint> {
        None
    }
}

/// No static types: every question gets the dynamic default.
#[derive(Debug, Clone, Copy, Default)]
pub struct NoTypes;

impl StaticTypes for crate::tariff::check::Analysis {
    fn type_of(&self, module: ModuleId, node: NodeId) -> Option<Type> {
        crate::tariff::check::Analysis::type_of(self, module, node).cloned()
    }

    fn input_type(&self, name: &str) -> Option<Type> {
        self.inputs
            .iter()
            .find(|input| input.name == name)
            .map(|input| input.ty.clone())
    }

    fn input_constraint(&self, name: &str) -> Option<&crate::tariff::constraint::Constraint> {
        self.inputs
            .iter()
            .find(|input| input.name == name)
            .and_then(|input| input.constraint.as_ref())
    }
}

impl StaticTypes for NoTypes {
    fn type_of(&self, _: ModuleId, _: NodeId) -> Option<Type> {
        None
    }

    fn input_type(&self, _: &str) -> Option<Type> {
        None
    }
}
