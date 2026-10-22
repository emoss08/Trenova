//! Running one loaded program: the step the hosts gate behind the checker.
//!
//! [`run_program`] takes a program the loader built and the checker passed, the record, the
//! input values the host sent, the tables and the budgets, and gives back the [`Outcome`] and
//! the trace:
//!
//! 1. **Inputs.** Each input takes the value the host sent, decoded against the type the
//!    checker lowered from its annotation ([`super::StaticTypes::input_type`]; without the
//!    checker's types a sent value is an `internal` error), else its default. An input value
//!    that does not fit is an error with code `input`; so is a value for an input the file does
//!    not declare. An input with neither a value nor a default refuses to rate, naming the
//!    input.
//! 2. **The body.** The root file's statements run top to bottom.
//! 3. **The result.** A tariff's bill is finished by its total clauses; a condition's last
//!    expression is its value. When a table lookup needed cells the request did not carry, the
//!    outcome is [`Outcome::Needs`] whatever else happened, since a provisional miss may have
//!    caused it.

use super::StaticTypes;
use super::bill::{Clauses, Finished};
use super::budget::{Limits, Meter};
use super::error::{Code, EvalError, Refusal, RefusalText, Stop};
use super::interp::link::Linked;
use super::interp::{Env, Interp, Setup};
use super::tables::{Need, Tables};
use super::trace::Trace;
use super::value::Value;
use crate::message::{Arg, Message};
use crate::tariff::syntax::ast::FileKind;
use crate::tariff::syntax::program::Program;
use crate::tariff::wire::decode;
use serde_json::Value as Json;
use std::collections::BTreeMap;

/// What a run is given.
pub struct Request<'p> {
    /// The loaded program; its root is what runs.
    pub program: &'p Program,
    /// The checker's types, or [`super::NoTypes`].
    pub types: &'p dyn StaticTypes,
    /// The request's tables.
    pub tables: &'p Tables,
    /// The record, already decoded.
    pub record: Value,
    /// The input values the host sent, by name, in the wire form.
    pub inputs: &'p BTreeMap<String, Json>,
    /// The budgets.
    pub limits: Limits,
    /// Whether to record a trace.
    pub trace: bool,
}

impl std::fmt::Debug for Request<'_> {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("Request")
            .field("inputs", &self.inputs)
            .field("limits", &self.limits)
            .field("trace", &self.trace)
            .finish_non_exhaustive()
    }
}

/// How a run ended.
#[derive(Debug, Clone)]
pub enum Outcome {
    /// A tariff rated: its finished bill and the clauses that finished it.
    Rated {
        /// The bill.
        bill: Finished,
        /// The total clauses, for converting the bill into the billing currency.
        clauses: Clauses,
    },
    /// A condition's value.
    Condition(bool),
    /// The program refused to rate.
    Refused(Refusal),
    /// The program could not be evaluated.
    Failed(EvalError),
    /// Table cells the request did not carry; the host loads them and runs again.
    Needs(Vec<Need>),
}

/// A finished run.
#[derive(Debug, Clone)]
pub struct Run {
    /// How it ended.
    pub outcome: Outcome,
    /// What it did.
    pub trace: Trace,
}

/// Runs a loaded program.
#[must_use]
pub fn run_program(request: Request<'_>) -> Run {
    let linked = Linked::new(request.program);
    let mut interp = Interp::new(Setup {
        linked: &linked,
        types: request.types,
        tables: request.tables,
        record: request.record,
        meter: Meter::new(request.limits),
        trace: request.trace,
    });
    let kind = request
        .program
        .root()
        .and_then(|root| root.file.header.as_ref())
        .map(|header| header.kind);
    let mut env = Env::new(0, false);
    let ran = bind_inputs(&mut interp, &mut env, &Supplied::Wire(request.inputs))
        .and_then(|()| interp.run_file(&mut env));
    let outcome = finish(&mut interp, kind, ran);
    let outcome = if interp.needs.is_empty() {
        outcome
    } else {
        Outcome::Needs(std::mem::take(&mut interp.needs))
    };
    Run {
        outcome,
        trace: interp.trace,
    }
}

fn finish(interp: &mut Interp<'_, '_>, kind: Option<FileKind>, ran: Result<(), Stop>) -> Outcome {
    if let Err(stop) = ran {
        return match stop {
            Stop::Refused(refusal) => Outcome::Refused(refusal),
            Stop::Failed(error) => Outcome::Failed(error),
        };
    }
    match kind {
        Some(FileKind::Tariff) => {
            let clauses = interp.clauses.clone();
            match std::mem::take(&mut interp.bill).finish(&clauses) {
                Ok(bill) => {
                    record_dropped(interp, &bill);
                    Outcome::Rated { bill, clauses }
                }
                Err(error) => Outcome::Failed(error),
            }
        }
        Some(FileKind::Condition) => match interp.last_value.take() {
            Some(Value::Bool(holds)) => Outcome::Condition(holds),
            _ => Outcome::Failed(EvalError::internal(Message::new(crate::message!(
                "A condition must end with an expression that is true or false"
            )))),
        },
        Some(FileKind::Library) | None => Outcome::Failed(EvalError::internal(Message::new(
            crate::message!("Only a tariff or a condition can be evaluated"),
        ))),
    }
}

fn record_dropped(interp: &mut Interp<'_, '_>, bill: &Finished) {
    for line in &bill.dropped {
        interp.trace.push(super::trace::Event::Dropped {
            label: line.label.clone().unwrap_or_default(),
            code: line.code.clone(),
            span: line.span,
        });
    }
}

/// The input values a run starts from.
#[derive(Debug, Clone, Copy)]
pub enum Supplied<'s> {
    /// What the host sent, in the wire form.
    Wire(&'s BTreeMap<String, Json>),
    /// What a test's `given input.x = …` set.
    Values(&'s BTreeMap<String, Value>),
}

impl Supplied<'_> {
    fn names(&self) -> Vec<&str> {
        match self {
            Self::Wire(values) => values.keys().map(String::as_str).collect(),
            Self::Values(values) => values.keys().map(String::as_str).collect(),
        }
    }
}

/// One input's supplied value.
enum Given<'s> {
    Wire(&'s Json),
    Value(&'s Value),
}

/// Binds every input of the root file in its top scope: the value supplied, else the default.
///
/// # Errors
///
/// An `input` error for a value that does not fit or names no input; a refusal for an input
/// with neither a value nor a default.
pub fn bind_inputs(
    interp: &mut Interp<'_, '_>,
    env: &mut Env,
    provided: &Supplied<'_>,
) -> Result<(), Stop> {
    let linked = *interp.linked;
    let declared = linked.inputs(0);
    if let Some(unknown) = provided
        .names()
        .into_iter()
        .find(|name| !declared.iter().any(|input| input.name.text == *name))
    {
        return Err(Stop::Failed(EvalError::with(
            Code::Input,
            crate::message!("This tariff has no input named {0}"),
            [Arg::from(unknown)],
        )));
    }
    for input in declared {
        let name = input.name.text.as_str();
        let supplied = match provided {
            Supplied::Wire(values) => values.get(name).map(Given::Wire),
            Supplied::Values(values) => values.get(name).map(Given::Value),
        };
        let value = match (supplied, &input.default) {
            (Some(Given::Value(value)), _) => value.clone(),
            (Some(Given::Wire(json)), _) => {
                let ty = interp.input_type(name).ok_or_else(|| {
                    Stop::Failed(
                        EvalError::internal(Message::with(
                            crate::message!(
                                "The input {0} has no checked type, so its value can't be read"
                            ),
                            [Arg::from(name)],
                        ))
                        .at(input.name.span),
                    )
                })?;
                decode(json, &ty, name).map_err(|error| {
                    Stop::Failed(
                        EvalError::with(
                            Code::Input,
                            crate::message!(
                                "The value given for the input {0} does not fit it: {1}"
                            ),
                            [Arg::from(name), Arg::from(error.message)],
                        )
                        .at(input.name.span),
                    )
                })?
            }
            (None, Some(default)) => {
                let mut scope = Env::new(0, true);
                interp.eval_in(&mut scope, default)?
            }
            (None, None) => {
                return Err(Stop::Refused(Refusal {
                    message: RefusalText::Crate(Message::with(
                        crate::message!("The input {0} has no value and no default"),
                        [Arg::from(name)],
                    )),
                    span: input.name.span,
                }));
            }
        };
        env.bind(name, value, false);
    }
    Ok(())
}
