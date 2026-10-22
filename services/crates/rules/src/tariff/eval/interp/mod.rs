//! The metered interpreter: runs a loaded program over a record and produces a bill, a
//! condition's value, a refusal or an error.
//!
//! The interpreter is dynamically typed over the syntax tree. It trusts that the program
//! checked clean, and never panics or returns a wrong number when it did not: an operation on
//! values of the wrong kind is an [`Code::Internal`] error. Where only the static type can say
//! what a value is (the `sum` of an empty list, a percent scaled by a number), it asks the
//! [`StaticTypes`] oracle.
//!
//! Scopes follow the language: a block is a scope; a root file's `let`s, `var`s and inputs live
//! in its top scope; a function body sees only its parameters and its module's declarations and
//! imports (functions are closed), so it cannot read the record, an input or a `let` from
//! outside. Library constants are evaluated once per run, on first use.
//!
//! - [`link`]: the program's modules, declarations and imports by name.
//! - `expr`, `stmt`, `call`, `pattern`: expressions, statements, calls and `match` patterns.

pub mod link;

mod call;
mod expr;
mod pattern;
mod stmt;

use super::StaticTypes;
use super::bill::{Bill, Clauses, Line};
use super::budget::Meter;
use super::error::{EvalError, Refusal, RefusalText, Stop};
use super::tables::{Need, Tables};
use super::text;
use super::trace::Trace;
use super::value::Value;
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::syntax::ast::NodeId;
use crate::tariff::types::Type;
use link::Linked;
use regex_lite::Regex;
use rust_decimal::Decimal;
use std::collections::{BTreeMap, BTreeSet};
use std::rc::Rc;

/// How a statement ended.
#[derive(Debug, Clone)]
pub enum Flow {
    /// It ran to its end.
    Normal,
    /// `break`.
    Break,
    /// `continue`.
    Continue,
    /// `return value`.
    Return(Value),
}

/// One name in scope.
#[derive(Debug, Clone)]
struct Slot {
    name: Rc<str>,
    value: Value,
    mutable: bool,
}

/// The scopes code runs in.
#[derive(Debug, Clone)]
pub struct Env {
    /// The module the code belongs to.
    pub module: usize,
    /// Whether this is a function body (or a constant or default), which cannot see the record.
    pub function: bool,
    frames: Vec<Vec<Slot>>,
}

impl Env {
    /// A fresh scope for code in `module`.
    #[must_use]
    pub fn new(module: usize, function: bool) -> Self {
        Self {
            module,
            function,
            frames: vec![Vec::new()],
        }
    }

    /// Opens a block's scope.
    pub fn push(&mut self) {
        self.frames.push(Vec::new());
    }

    /// Closes the innermost scope.
    pub fn pop(&mut self) {
        if self.frames.len() > 1 {
            self.frames.pop();
        }
    }

    /// Binds a name in the innermost scope.
    pub fn bind(&mut self, name: &str, value: Value, mutable: bool) {
        if let Some(frame) = self.frames.last_mut() {
            frame.push(Slot {
                name: Rc::from(name),
                value,
                mutable,
            });
        }
    }

    /// The value a name has in scope.
    #[must_use]
    pub fn get(&self, name: &str) -> Option<&Value> {
        self.frames
            .iter()
            .rev()
            .flat_map(|frame| frame.iter().rev())
            .find(|slot| &*slot.name == name)
            .map(|slot| &slot.value)
    }

    fn slot_mut(&mut self, name: &str) -> Option<&mut Slot> {
        self.frames
            .iter_mut()
            .rev()
            .flat_map(|frame| frame.iter_mut().rev())
            .find(|slot| &*slot.name == name)
    }
}

/// What a test's `expect`s read after the rating: `total` and `lines`, and `charge(…)`.
#[derive(Debug, Clone)]
pub struct TestView {
    /// The finished bill's lines.
    pub lines: Vec<Line>,
    /// The finished bill's total.
    pub total: Decimal,
}

/// The interpreter's state for one run of one program.
pub struct Interp<'p, 'a> {
    /// The program.
    pub linked: &'p Linked<'a>,
    types: &'p dyn StaticTypes,
    tables: &'p Tables,
    /// The record the schema roots read.
    pub record: Value,
    /// The budgets.
    pub meter: Meter,
    /// The trace.
    pub trace: Trace,
    /// The bill being built.
    pub bill: Bill,
    /// The total clauses written.
    pub clauses: Clauses,
    /// Table cells needed and not carried.
    pub needs: Vec<Need>,
    /// What `expect`s read, in a test after the rating.
    pub view: Option<TestView>,
    /// Whether the test constructors (`pickup(…)`, `commodity(…)`) are in scope.
    pub constructors: bool,
    /// The value of the last expression statement at file level: a condition's result.
    pub last_value: Option<Value>,
    consts: BTreeMap<(usize, String), Value>,
    evaluating: BTreeSet<(usize, String)>,
    regexes: BTreeMap<String, Regex>,
    anchor: Option<Span>,
}

impl std::fmt::Debug for Interp<'_, '_> {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("Interp")
            .field("steps", &self.meter.steps_used())
            .finish_non_exhaustive()
    }
}

/// What an interpreter starts from.
pub struct Setup<'p, 'a> {
    /// The program.
    pub linked: &'p Linked<'a>,
    /// The static type oracle.
    pub types: &'p dyn StaticTypes,
    /// The request's tables.
    pub tables: &'p Tables,
    /// The record.
    pub record: Value,
    /// The budgets.
    pub meter: Meter,
    /// Whether to record a trace.
    pub trace: bool,
}

impl std::fmt::Debug for Setup<'_, '_> {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.debug_struct("Setup").finish_non_exhaustive()
    }
}

impl<'p, 'a> Interp<'p, 'a> {
    /// An interpreter ready to run.
    #[must_use]
    pub fn new(setup: Setup<'p, 'a>) -> Self {
        Self {
            linked: setup.linked,
            types: setup.types,
            tables: setup.tables,
            record: setup.record,
            meter: setup.meter,
            trace: Trace::new(setup.trace),
            bill: Bill::default(),
            clauses: Clauses::default(),
            needs: Vec::new(),
            view: None,
            constructors: false,
            last_value: None,
            consts: BTreeMap::new(),
            evaluating: BTreeSet::new(),
            regexes: BTreeMap::new(),
            anchor: None,
        }
    }

    /// The static type of a node, when the oracle knows it.
    fn static_type(&self, module: usize, node: NodeId) -> Option<Type> {
        self.types.type_of(link::id(module), node)
    }

    /// Whether events of code in `env` belong on the trace: the root file's own statements.
    fn traced(&self, env: &Env) -> bool {
        self.trace.enabled() && env.module == 0 && !env.function
    }

    /// A span for something that happened in `env`: its own in the root file, else the call in
    /// the root file that reached it.
    fn root_span(&self, env: &Env, span: Span) -> Span {
        if env.module == 0 {
            span
        } else {
            self.anchor.unwrap_or(span)
        }
    }

    fn compile(&mut self, pattern: &str) -> Result<Regex, EvalError> {
        if let Some(regex) = self.regexes.get(pattern) {
            return Ok(regex.clone());
        }
        let regex = text::compile(pattern)?;
        self.regexes.insert(pattern.to_owned(), regex.clone());
        Ok(regex)
    }

    fn record_needs(&mut self, needs: Vec<Need>) {
        for need in needs {
            if !self.needs.contains(&need) {
                self.needs.push(need);
            }
        }
    }

    /// The value of a schema root.
    fn root(&self, name: &str) -> Option<Value> {
        match &self.record {
            Value::Record(record) => record.get(name).cloned(),
            _ => None,
        }
    }
}

/// An internal error about a name nothing declares.
fn unknown_name(name: &str) -> EvalError {
    EvalError::internal(Message::with(
        crate::message!("Nothing named {0} is in scope here"),
        [Arg::from(name)],
    ))
}

/// A refusal with the author's text.
fn refusal(message: String, span: Span) -> Stop {
    Stop::Refused(Refusal {
        message: RefusalText::Author(message),
        span,
    })
}
