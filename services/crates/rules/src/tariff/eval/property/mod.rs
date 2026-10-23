//! Running a `property`: many cases, each a test with one extra `given` per `vary`.
//!
//! 1. **Axes** ([`axes`]): each `vary`'s values, landmarks and their neighbours first.
//! 2. **Cases** ([`cases`]): every combination when there are at most 256, else each axis's
//!    landmarks against the other axes' ends, filled with combinations a generator seeded by
//!    the property's name draws. The same file always runs the same cases.
//! 3. **Each case** runs like a test ([`super::tests::Runner`]): the property's `given`s and the
//!    case's values, its `assume`s (a case where one is false is skipped and not counted),
//!    the rating, then its other statements. A case runs under the per-test budget, and the
//!    run's shared step meter bounds them all.
//! 4. **The first failing case** (an `expect cond` that does not hold, or a refusal) stops the
//!    cases and is **shrunk** ([`search`]): for each axis in turn, the smallest value from the
//!    range's start that still fails, found by halving, the others fixed (a list keeps its
//!    first failing item).
//! 5. **Monotone expectations** are checked once every case has passed: along each target,
//!    with the other axes fixed, neighbouring cases whose results are out of order are
//!    narrowed by halving the gap until they are one step apart.
//!
//! A case that needs table cells the request did not carry makes the property `needs`; the
//! cases keep running so one pass finds every need. An evaluation error ends the property in
//! `error`, with the failing case as its counterexample. Shrinking and narrowing runs count
//! toward the step budget but not toward `cases` or coverage.

pub mod axes;
pub mod cases;
pub mod report;
mod search;

pub use report::{Counterexample, Monotone};

use super::bill::Finished;
use super::error::{EvalError, Refusal};
use super::interp::Env;
use super::record::sample;
use super::tables::Need;
use super::tests::{CaseSetup, Ended, Failure, Kind, Runner, Status, TestResult, stopped};
use super::value::Value;
use crate::message::{Arg, Message};
use crate::tariff::schema::Schema;
use crate::tariff::syntax::ast::{
    Direction, Expect, Expr, ExprKind, FileKind, Property, StmtKind,
};
use axes::{Axis, Sources, Target};
use report::Writing;
use std::rc::Rc;

/// How one case ended.
#[derive(Debug, Clone)]
enum End {
    /// Every `expect` held.
    Passed(Option<Finished>),
    /// An `expect` did not hold.
    Failed(Option<Finished>),
    /// The rating refused.
    Refused(Refusal),
    /// An `assume` did not hold.
    Skipped,
    /// It could not be evaluated.
    Error(EvalError),
    /// It read table cells the request did not carry.
    Needs,
}

/// One case's run.
#[derive(Debug, Clone)]
struct Probe {
    end: End,
    failures: Vec<Failure>,
    monotone: Vec<Option<Value>>,
}

impl Probe {
    const fn failed(&self) -> bool {
        matches!(self.end, End::Failed(_) | End::Refused(_))
    }
}

/// A monotone expectation of the property.
#[derive(Debug, Clone, Copy)]
struct Order<'a> {
    stmt: &'a crate::tariff::syntax::ast::Stmt,
    value: &'a Expr,
    direction: Direction,
    axis: usize,
}

/// Runs one property.
pub fn run<'a>(runner: &mut Runner<'_, '_, 'a>, property: &'a Property, library: bool) -> TestResult {
    let mut result = TestResult {
        name: property.name.text.clone(),
        span: property.name.span,
        kind: Kind::Property,
        status: Status::Passed,
        failures: Vec::new(),
        error: None,
        bill: None,
        needs: Vec::new(),
        cases: 0,
        skipped: 0,
        counterexample: None,
    };
    if let Some(error) = runner.ledger.exhausted() {
        result.status = Status::Error;
        result.error = Some(error);
        return result;
    }
    let axes = match build(runner, property) {
        Ok(axes) => axes,
        Err(error) => {
            result.status = if runner.needs.is_empty() {
                Status::Error
            } else {
                Status::Needs
            };
            result.needs = std::mem::take(&mut runner.needs);
            result.error = (result.status == Status::Error).then_some(error);
            return result;
        }
    };
    let mut driver = Driver {
        runner,
        property,
        library,
        axes,
        probes: 0,
        needs: Vec::new(),
    };
    driver.explore(&mut result);
    result
}

/// Every axis of a property, its values worked out once.
fn build<'a>(runner: &mut Runner<'_, '_, 'a>, property: &'a Property) -> Result<Vec<Axis<'a>>, EvalError> {
    let landmarks = landmarks_for(runner, property);
    let schema = Schema::new(runner.request.settings);
    let sources = Sources {
        types: runner.request.types,
        schema: &schema,
        inputs: runner.linked.inputs(0),
        landmarks: &landmarks,
    };
    let mut interp = runner.interp(sample(runner.request.settings));
    let mut env = Env::new(0, false);
    let mut out = Vec::new();
    let mut failed = None;
    for stmt in &property.body.stmts {
        if !matches!(stmt.kind, StmtKind::Vary { .. }) {
            continue;
        }
        let mut eval = |expr: &'a Expr| interp.eval_in(&mut env, expr).map_err(stopped);
        match axes::axis(stmt, &sources, &mut eval) {
            Ok(axis) => out.push(axis),
            Err(error) => {
                failed = Some(error.at(stmt.span));
                break;
            }
        }
    }
    runner.gather(&mut interp);
    match failed {
        Some(error) => Err(error),
        None => Ok(out),
    }
}

/// The program's landmarks, gathered once per run and only when a property varies over a
/// range.
fn landmarks_for(runner: &mut Runner<'_, '_, '_>, property: &Property) -> Rc<Vec<Value>> {
    let ranged = property.body.stmts.iter().any(|stmt| {
        matches!(
            &stmt.kind,
            StmtKind::Vary { values: Some(values), .. } if matches!(values.kind, ExprKind::Range { .. })
        ) || matches!(
            &stmt.kind,
            StmtKind::Vary { target, values: None } if target.first().is_some_and(|head| head.text == "input")
        )
    });
    if !ranged {
        return Rc::new(Vec::new());
    }
    if let Some(known) = &runner.ledger.landmarks {
        return Rc::clone(known);
    }
    let found = Rc::new(axes::landmarks(
        runner.linked,
        runner.request.types,
        runner.request.tables,
    ));
    runner.ledger.landmarks = Some(Rc::clone(&found));
    found
}

/// What one property's run holds.
struct Driver<'d, 'r, 'p, 'a> {
    runner: &'d mut Runner<'r, 'p, 'a>,
    property: &'a Property,
    library: bool,
    axes: Vec<Axis<'a>>,
    probes: usize,
    needs: Vec<Need>,
}

/// One generated case that passed: its value indexes and its monotone values.
type Record = (Vec<usize>, Vec<Option<Value>>);

/// What exploring found.
enum Found {
    /// Every case passed, with what each gave.
    Passed(Vec<Record>),
    /// A case could not be evaluated.
    Error(Vec<Value>, EvalError),
    /// Some case needs table cells.
    Needs,
    /// A case failed.
    Failed(Vec<Value>, Probe),
    /// A monotone expectation does not hold.
    Unordered(Box<Unordered>),
}

/// A monotone failure, narrowed.
struct Unordered {
    order: usize,
    values: Vec<Value>,
    before: (Value, Value),
    after: (Value, Value),
    bill: Option<Finished>,
}

impl<'a> Driver<'_, '_, '_, 'a> {
    fn explore(&mut self, result: &mut TestResult) {
        let found = self.cases(result);
        self.runner.ledger.counting = false;
        let found = match found {
            Found::Failed(values, probe) => {
                let (values, probe) = self.shrink(values, probe);
                Found::Failed(values, probe)
            }
            Found::Passed(records) => self.ordered(&records),
            other => other,
        };
        self.runner.ledger.counting = true;
        if !self.needs.is_empty() {
            result.status = Status::Needs;
            result.needs = std::mem::take(&mut self.needs);
            return;
        }
        match found {
            Found::Passed(_) | Found::Needs => {}
            Found::Error(values, error) => {
                result.status = Status::Error;
                result.error = Some(error);
                result.counterexample = Some(self.counterexample(&values, None, None));
            }
            Found::Failed(values, probe) => self.failed(result, &values, probe),
            Found::Unordered(unordered) => self.unordered(result, &unordered),
        }
    }

    /// Runs the generated cases, stopping at the first failure.
    fn cases(&mut self, result: &mut TestResult) -> Found {
        let marked: Vec<Vec<bool>> = self.axes.iter().map(|axis| axis.marked.clone()).collect();
        let combinations = cases::combinations(&marked, cases::seed(&self.property.name.text));
        let mut records: Vec<Record> = Vec::new();
        for combination in combinations {
            let values = self.values_of(&combination);
            let probe = self.case(&values);
            if !matches!(probe.end, End::Skipped) {
                result.cases += 1;
            }
            match probe.end {
                End::Skipped => result.skipped += 1,
                End::Error(error) => return Found::Error(values, error),
                End::Passed(_) => records.push((combination, probe.monotone)),
                End::Failed(_) | End::Refused(_) if self.needs.is_empty() => {
                    return Found::Failed(values, probe);
                }
                End::Needs | End::Failed(_) | End::Refused(_) => {}
            }
        }
        if self.needs.is_empty() {
            Found::Passed(records)
        } else {
            Found::Needs
        }
    }

    /// The values of a combination of value indexes.
    fn values_of(&self, combination: &[usize]) -> Vec<Value> {
        self.axes
            .iter()
            .zip(combination)
            .map(|(axis, index)| axis.points.get(*index).cloned().unwrap_or(Value::None))
            .collect()
    }

    /// Runs one case.
    fn case(&mut self, values: &[Value]) -> Probe {
        let ended = if let Some(error) = self.runner.ledger.exhausted() {
            Err(error)
        } else {
            let setup = self.setup(values);
            let body = &self.property.body;
            if self.library {
                self.runner.library_case(body, &setup)
            } else {
                self.runner.rating_case(body, &setup)
            }
        };
        let failures = std::mem::take(&mut self.runner.failures);
        let monotone = std::mem::take(&mut self.runner.monotone);
        let needs = std::mem::take(&mut self.runner.needs);
        let end = if needs.is_empty() {
            match ended {
                Err(error) => End::Error(error),
                Ok(Ended::Skipped) => End::Skipped,
                Ok(Ended::Refused(refusal)) => End::Refused(refusal),
                Ok(Ended::Rated(bill)) if failures.is_empty() => End::Passed(bill),
                Ok(Ended::Rated(bill)) => End::Failed(bill),
            }
        } else {
            for need in needs {
                if !self.needs.contains(&need) {
                    self.needs.push(need);
                }
            }
            End::Needs
        };
        Probe {
            end,
            failures,
            monotone,
        }
    }

    /// What a case adds to the property's body: each axis's value, where its target is.
    fn setup(&self, values: &[Value]) -> CaseSetup<'a> {
        let mut setup = CaseSetup::default();
        for (axis, value) in self.axes.iter().zip(values) {
            match &axis.target {
                Target::Path(path) => setup.paths.push((path, value.clone())),
                Target::Input(name) => setup.inputs.push((name.clone(), value.clone())),
                Target::Name(name) => setup.names.push((name.clone(), value.clone())),
            }
        }
        setup
    }

    /// The property's monotone expectations, each with the axis it compares along.
    fn orders(&self) -> Vec<Order<'a>> {
        let property: &'a Property = self.property;
        property
            .body
            .stmts
            .iter()
            .filter_map(|stmt| {
                let StmtKind::Expect(Expect::Monotone {
                    value,
                    direction,
                    target,
                    ..
                }) = &stmt.kind
                else {
                    return None;
                };
                let label = target
                    .iter()
                    .map(|part| part.text.as_str())
                    .collect::<Vec<_>>()
                    .join(".");
                let axis = self.axes.iter().position(|axis| axis.label == label)?;
                Some(Order {
                    stmt,
                    value,
                    direction: *direction,
                    axis,
                })
            })
            .collect()
    }

    /// A failed case's result: its failures, its bill and its counterexample.
    fn failed(&self, result: &mut TestResult, values: &[Value], probe: Probe) {
        result.status = Status::Failed;
        let mut failures = probe.failures;
        let refusal = match probe.end {
            End::Refused(refusal) => Some(refusal),
            End::Failed(bill) | End::Passed(bill) => {
                result.bill = bill;
                None
            }
            End::Skipped | End::Error(_) | End::Needs => None,
        };
        if let (Some(refusal), true) = (&refusal, failures.is_empty()) {
            failures.push(Failure {
                span: self.property.name.span,
                message: Message::with(
                    crate::message!("The rating was refused: {0}"),
                    [Arg::from(refusal.message.text())],
                ),
                values: Vec::new(),
            });
        }
        result.failures = failures;
        result.counterexample = Some(self.counterexample(values, None, refusal));
    }

    /// A monotone failure's result.
    fn unordered(&self, result: &mut TestResult, unordered: &Unordered) {
        let orders = self.orders();
        let Some(order) = orders.get(unordered.order) else {
            return;
        };
        let source = self.source();
        let expression = source
            .get(span_range(order.value.span))
            .unwrap_or_default()
            .to_owned();
        let label = self
            .axes
            .get(order.axis)
            .map(|axis| axis.label.clone())
            .unwrap_or_default();
        let template = match order.direction {
            Direction::Nondecreasing => crate::message!(
                "{0} falls from {1} at {2} to {3} at {4}: it must not fall as {5} grows"
            ),
            Direction::Nonincreasing => crate::message!(
                "{0} rises from {1} at {2} to {3} at {4}: it must not rise as {5} grows"
            ),
        };
        let (before_at, before) = &unordered.before;
        let (after_at, after) = &unordered.after;
        result.status = Status::Failed;
        result.bill.clone_from(&unordered.bill);
        result.failures = vec![Failure {
            span: order.stmt.span,
            message: Message::with(
                template,
                [
                    Arg::from(format!("`{expression}`")),
                    Arg::from(before.display()),
                    Arg::from(before_at.display()),
                    Arg::from(after.display()),
                    Arg::from(after_at.display()),
                    Arg::from(format!("`{label}`")),
                ],
            ),
            values: [before, after]
                .into_iter()
                .map(|value| {
                    (
                        order.value.span,
                        value.display(),
                        Some(crate::tariff::wire::encode(value)),
                    )
                })
                .collect(),
        }];
        let monotone = Monotone {
            target: label,
            expression,
            direction: order.direction,
            before: unordered.before.clone(),
            after: unordered.after.clone(),
        };
        result.counterexample = Some(self.counterexample(
            &unordered.values,
            Some((order.value, monotone)),
            None,
        ));
    }

    /// The counterexample of a case.
    fn counterexample(
        &self,
        values: &[Value],
        monotone: Option<(&Expr, Monotone)>,
        refusal: Option<Refusal>,
    ) -> Counterexample {
        let schema = Schema::new(self.runner.request.settings);
        let writing = Writing {
            source: self.source(),
            schema: &schema,
            types: self.runner.request.types,
            kind: if self.library {
                FileKind::Library
            } else {
                FileKind::Tariff
            },
        };
        let pinned = monotone
            .as_ref()
            .map(|(expression, monotone)| (*expression, &monotone.before.1));
        let test = writing.test(self.property, &self.axes, values, pinned);
        Counterexample {
            values: self
                .axes
                .iter()
                .zip(values)
                .map(|(axis, value)| (axis.label.clone(), value.clone()))
                .collect(),
            monotone: monotone.map(|(_, monotone)| monotone),
            refusal,
            test,
        }
    }

    /// The root file's source.
    fn source(&self) -> &'a str {
        let linked = *self.runner.linked;
        linked
            .program()
            .root()
            .map_or("", |root| root.source.as_str())
    }
}

/// A span as a byte range.
fn span_range(span: crate::span::Span) -> std::ops::Range<usize> {
    let start = usize::try_from(span.start).unwrap_or(usize::MAX);
    let end = usize::try_from(span.end).unwrap_or(usize::MAX);
    start..end.max(start)
}
