//! The test runner: a file's `test` blocks, run against the sample record.
//!
//! A test in a tariff or condition:
//!
//! 1. starts from the sample record and the inputs' defaults;
//! 2. applies its `given`s in order: a record path (`shipment.weight`, `lane.origin.zip3`) or
//!    `input.name` (a value outside the input's `where` constraint is an `input` error). The
//!    test constructors (`pickup(…)`, `delivery(…)`, `splitPickup(…)`, `splitDelivery(…)`,
//!    `commodity(…)`) build stops and commodities from the sample defaults.
//!    Afterwards each stop without a sequence takes its position, and `shipment.pickup` and
//!    `shipment.delivery` follow the stops unless a `given` set them;
//! 3. runs the file;
//! 4. runs the rest of its body in order, with `charge(CODE)` / `charge("Label")` (the matching
//!    lines' sum, or `none`), `total` and `lines` in scope. `expect cond` must hold;
//!    `expect fails` passes when the rating was refused (with that text, when one is given).
//!
//! A test in a library runs its body in order with no rating: `expect`s over function calls.
//! There, `expect fails` passes when the statement before it refused (with that text, when one
//! is given).
//!
//! A failing `expect` reports the values of the operands in it, so the author sees what the
//! tariff computed. An evaluation error makes the test's status `error`.
//!
//! Tests read rate tables under the same on-demand protocol as a rating
//! ([`super::tables`]): a lookup of cells the request did not carry is a provisional miss and
//! the test keeps running, so one pass finds every need it can, from its `given`s, the rating
//! and its `expect`s alike. A test that recorded needs ends with status `needs` whatever else
//! happened, since the provisional miss may have caused it; the host loads the cells and runs
//! the tests again.

use super::StaticTypes;
use super::bill::Finished;
use super::budget::{Limits, Meter};
use super::error::{EvalError, Refusal, Stop};
use super::interp::link::Linked;
use super::interp::{Env, Interp, Setup, TestView};
use super::record::{sample, settle};
use super::run::{Supplied, bind_inputs, enforce_constraint};
use super::tables::{Need, Tables};
use super::value::{Record, Value};
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::schema::Settings;
use crate::tariff::syntax::ast::{
    Expect, Expr, ExprKind, FileKind, Name, Stmt, StmtKind, Test, TextPart,
};
use crate::tariff::syntax::program::Program;
use std::collections::BTreeMap;
use std::rc::Rc;

/// What a test run is given.
pub struct TestRequest<'p> {
    /// The loaded program; its root's tests run.
    pub program: &'p Program,
    /// The checker's types, or [`super::NoTypes`].
    pub types: &'p dyn StaticTypes,
    /// The organisation's settings, for the sample record.
    pub settings: &'p Settings,
    /// The request's tables.
    pub tables: &'p Tables,
    /// The budgets, per test.
    pub limits: Limits,
    /// Only these tests, when given.
    pub names: Option<&'p [String]>,
}

impl std::fmt::Debug for TestRequest<'_> {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("TestRequest")
            .field("names", &self.names)
            .finish_non_exhaustive()
    }
}

/// How a test ended.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Status {
    /// Every `expect` held.
    Passed,
    /// An `expect` did not hold.
    Failed,
    /// The test could not be evaluated.
    Error,
    /// The test read table cells the request did not carry, so its result is not final.
    Needs,
}

impl Status {
    /// The status's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Passed => "passed",
            Self::Failed => "failed",
            Self::Error => "error",
            Self::Needs => "needs",
        }
    }
}

/// An `expect` that did not hold.
#[derive(Debug, Clone)]
pub struct Failure {
    /// The `expect`.
    pub span: Span,
    /// Why it failed.
    pub message: Message,
    /// The operands in it: where each is, its value as text, and its value on the wire when it
    /// has one (an operand that refused or failed has only the text).
    pub values: Vec<(Span, String, Option<serde_json::Value>)>,
}

/// One test's result.
#[derive(Debug, Clone)]
pub struct TestResult {
    /// The test's name.
    pub name: String,
    /// The test's name as written.
    pub span: Span,
    /// How it ended.
    pub status: Status,
    /// The `expect`s that did not hold.
    pub failures: Vec<Failure>,
    /// The error that stopped it, for status `error`.
    pub error: Option<EvalError>,
    /// The bill the rating produced, for a tariff that rated.
    pub bill: Option<Finished>,
    /// The table cells the test needs, for status `needs`.
    pub needs: Vec<Need>,
}

/// Runs the root file's tests.
#[must_use]
pub fn run_tests(request: &TestRequest<'_>) -> Vec<TestResult> {
    let linked = Linked::new(request.program);
    let kind = request
        .program
        .root()
        .and_then(|root| root.file.header.as_ref())
        .map(|header| header.kind);
    linked
        .tests(0)
        .into_iter()
        .filter(|test| {
            request
                .names
                .is_none_or(|names| names.contains(&test.name.text))
        })
        .map(|test| {
            let mut runner = Runner {
                request,
                linked: &linked,
                failures: Vec::new(),
                needs: Vec::new(),
            };
            let ended = if kind == Some(FileKind::Library) {
                runner.library(test).map(|()| None)
            } else {
                runner.rating(test)
            };
            let (status, error, bill) = match ended {
                _ if !runner.needs.is_empty() => {
                    runner.failures.clear();
                    (Status::Needs, None, None)
                }
                Ok(bill) if runner.failures.is_empty() => (Status::Passed, None, bill),
                Ok(bill) => (Status::Failed, None, bill),
                Err(error) => (Status::Error, Some(error), None),
            };
            TestResult {
                name: test.name.text.clone(),
                span: test.name.span,
                status,
                failures: runner.failures,
                error,
                bill,
                needs: runner.needs,
            }
        })
        .collect()
}

struct Runner<'r, 'p, 'a> {
    request: &'r TestRequest<'p>,
    linked: &'r Linked<'a>,
    failures: Vec<Failure>,
    needs: Vec<Need>,
}

/// Every table need of a set of results, each once, in the order first met.
#[must_use]
pub fn needs_of(results: &[TestResult]) -> Vec<Need> {
    let mut needs: Vec<Need> = Vec::new();
    for need in results.iter().flat_map(|result| &result.needs) {
        if !needs.contains(need) {
            needs.push(need.clone());
        }
    }
    needs
}

/// How the tested rating ended.
enum Rating {
    Rated(Finished),
    Refused(Refusal),
}

impl<'r, 'a> Runner<'r, '_, 'a> {
    fn interp(&self, record: Value) -> Interp<'r, 'a> {
        let mut interp = Interp::new(Setup {
            linked: self.linked,
            types: self.request.types,
            tables: self.request.tables,
            record,
            meter: Meter::new(self.request.limits),
            trace: false,
        });
        interp.constructors = true;
        interp
    }

    /// Moves the needs an interpreter recorded onto the test.
    fn gather(&mut self, interp: &mut Interp<'_, '_>) {
        for need in std::mem::take(&mut interp.needs) {
            if !self.needs.contains(&need) {
                self.needs.push(need);
            }
        }
    }

    /// A test of a tariff or condition.
    fn rating(&mut self, test: &'a Test) -> Result<Option<Finished>, EvalError> {
        let (record, inputs) = self.givens(test)?;
        let mut interp = self.interp(record.clone());
        let mut env = Env::new(0, false);
        let ran = bind_inputs(&mut interp, &mut env, &Supplied::Values(&inputs))
            .and_then(|()| interp.run_file(&mut env));
        self.gather(&mut interp);
        let rating = match ran {
            Err(Stop::Failed(error)) => return Err(error),
            Err(Stop::Refused(refusal)) => Rating::Refused(refusal),
            Ok(()) => {
                let clauses = interp.clauses.clone();
                Rating::Rated(std::mem::take(&mut interp.bill).finish(&clauses)?)
            }
        };
        let mut checker = self.interp(record);
        if let Rating::Rated(bill) = &rating {
            checker.view = Some(TestView {
                lines: bill.lines.clone(),
                total: bill.total,
            });
        }
        let outcome = self.expectations(&mut checker, test, &rating);
        self.gather(&mut checker);
        outcome?;
        Ok(match rating {
            Rating::Rated(bill) => Some(bill),
            Rating::Refused(_) => None,
        })
    }

    /// A rating test's body after the rating: its `expect`s and the statements between them.
    fn expectations(
        &mut self,
        checker: &mut Interp<'r, 'a>,
        test: &'a Test,
        rating: &Rating,
    ) -> Result<(), EvalError> {
        let mut scope = Env::new(0, false);
        for stmt in &test.body.stmts {
            match &stmt.kind {
                StmtKind::Given { .. } => {}
                StmtKind::Expect(Expect::Fails(message)) => {
                    let wanted = match message {
                        Some(message) => Some(
                            checker
                                .interpolate_in(&mut scope, message)
                                .map_err(stopped)?,
                        ),
                        None => None,
                    };
                    let refusal = match rating {
                        Rating::Refused(refusal) => Some(refusal),
                        Rating::Rated(_) => None,
                    };
                    self.expect_refused(stmt.span, refusal, wanted.as_deref());
                }
                StmtKind::Expect(Expect::Holds(expr)) => {
                    if let Rating::Refused(refusal) = rating {
                        self.failures.push(Failure {
                            span: stmt.span,
                            message: Message::with(
                                crate::message!("The rating was refused: {0}"),
                                [Arg::from(refusal.message.text())],
                            ),
                            values: Vec::new(),
                        });
                        continue;
                    }
                    self.expect_holds(checker, &mut scope, stmt, expr)?;
                }
                _ => {
                    checker.exec(&mut scope, stmt).map_err(stopped)?;
                }
            }
        }
        Ok(())
    }

    /// The record and input values after a test's `given`s.
    fn givens(&mut self, test: &'a Test) -> Result<(Value, BTreeMap<String, Value>), EvalError> {
        let mut interp = self.interp(sample(self.request.settings));
        let applied = apply_givens(&mut interp, test);
        self.gather(&mut interp);
        let (inputs, pickup, delivery) = applied?;
        Ok((settle(&interp.record, pickup, delivery), inputs))
    }

    fn expect_refused(&mut self, span: Span, refusal: Option<&Refusal>, wanted: Option<&str>) {
        let message = match (refusal, wanted) {
            (None, _) => Message::new(crate::message!("Expected a refusal, but nothing refused")),
            (Some(refusal), Some(wanted)) if refusal.message.text() != wanted => Message::with(
                crate::message!(
                    "Expected the rating to be refused with \"{0}\", but it was refused with \"{1}\""
                ),
                [Arg::from(wanted), Arg::from(refusal.message.text())],
            ),
            (Some(_), _) => return,
        };
        self.failures.push(Failure {
            span,
            message,
            values: Vec::new(),
        });
    }

    fn expect_holds(
        &mut self,
        interp: &mut Interp<'r, 'a>,
        scope: &mut Env,
        stmt: &Stmt,
        expr: &'a Expr,
    ) -> Result<(), EvalError> {
        match interp.eval_in(scope, expr) {
            Ok(Value::Bool(true)) => Ok(()),
            Ok(Value::Bool(false)) => {
                let mut values = Vec::new();
                for operand in operands(expr) {
                    let (shown, wire) = match interp.eval_in(scope, operand) {
                        Ok(value) => (value.display(), Some(crate::tariff::wire::encode(&value))),
                        Err(Stop::Refused(refusal)) => (refusal.message.text(), None),
                        Err(Stop::Failed(error)) => (error.message.english(), None),
                    };
                    values.push((operand.span, shown, wire));
                }
                self.failures.push(Failure {
                    span: stmt.span,
                    message: Message::new(crate::message!("This expectation does not hold")),
                    values,
                });
                Ok(())
            }
            Ok(other) => Err(EvalError::internal(Message::with(
                crate::message!("An expectation must be true or false, not {0}"),
                [Arg::from(other.kind())],
            ))
            .at(expr.span)),
            Err(stop) => Err(stopped(stop)),
        }
    }

    /// A test of a library: its body, in order.
    fn library(&mut self, test: &'a Test) -> Result<(), EvalError> {
        let mut interp = self.interp(sample(self.request.settings));
        let ran = self.library_body(&mut interp, test);
        self.gather(&mut interp);
        ran
    }

    fn library_body(
        &mut self,
        interp: &mut Interp<'r, 'a>,
        test: &'a Test,
    ) -> Result<(), EvalError> {
        let mut scope = Env::new(0, false);
        let mut refused: Option<Refusal> = None;
        for stmt in &test.body.stmts {
            match &stmt.kind {
                StmtKind::Expect(Expect::Fails(message)) => {
                    let wanted = match message {
                        Some(message) => Some(
                            interp
                                .interpolate_in(&mut scope, message)
                                .map_err(stopped)?,
                        ),
                        None => None,
                    };
                    let refusal = refused.take();
                    self.expect_refused(stmt.span, refusal.as_ref(), wanted.as_deref());
                }
                _ if refused.is_some() => {}
                StmtKind::Expect(Expect::Holds(expr)) => {
                    self.expect_holds(interp, &mut scope, stmt, expr)?;
                }
                StmtKind::Given { .. } => {
                    return Err(EvalError::internal(Message::new(crate::message!(
                        "A library's tests have no record to give values to"
                    )))
                    .at(stmt.span));
                }
                _ => match interp.exec(&mut scope, stmt) {
                    Ok(_) => {}
                    Err(Stop::Refused(refusal)) => refused = Some(refusal),
                    Err(Stop::Failed(error)) => return Err(error),
                },
            }
        }
        if let Some(refusal) = refused {
            self.failures.push(Failure {
                span: refusal.span,
                message: Message::with(
                    crate::message!("The test was refused: {0}"),
                    [Arg::from(refusal.message.text())],
                ),
                values: Vec::new(),
            });
        }
        Ok(())
    }
}

/// Applies a test's `given`s to the interpreter's record: the input values they set, and
/// whether they set `shipment.pickup` and `shipment.delivery`.
fn apply_givens<'a>(
    interp: &mut Interp<'_, 'a>,
    test: &'a Test,
) -> Result<(BTreeMap<String, Value>, bool, bool), EvalError> {
    let mut inputs = BTreeMap::new();
    let (mut pickup, mut delivery) = (false, false);
    for stmt in &test.body.stmts {
        let StmtKind::Given { target, value } = &stmt.kind else {
            continue;
        };
        let mut scope = Env::new(0, false);
        let value = interp.eval_in(&mut scope, value).map_err(stopped)?;
        match target.as_slice() {
            [head, name] if head.text == "input" => {
                enforce_constraint(interp, &name.text, &value)
                    .map_err(|error| error.at(stmt.span))?;
                inputs.insert(name.text.clone(), value);
            }
            path => {
                pickup |= matches!(path, [root, field] if root.text == "shipment" && field.text == "pickup");
                delivery |= matches!(path, [root, field] if root.text == "shipment" && field.text == "delivery");
                interp.record =
                    set_path(&interp.record, path, value).map_err(|error| error.at(stmt.span))?;
            }
        }
    }
    Ok((inputs, pickup, delivery))
}

fn stopped(stop: Stop) -> EvalError {
    match stop {
        Stop::Failed(error) => error,
        Stop::Refused(refusal) => EvalError::internal(Message::with(
            crate::message!("A test's own statements refused: {0}"),
            [Arg::from(refusal.message.text())],
        ))
        .at(refusal.span),
    }
}

/// `record` with the value at `path` replaced.
fn set_path(record: &Value, path: &[Name], value: Value) -> Result<Value, EvalError> {
    let Some((first, rest)) = path.split_first() else {
        return Ok(value);
    };
    let Value::Record(fields) = record else {
        return Err(not_a_field(&first.text));
    };
    let Some(current) = fields.get(&first.text) else {
        return Err(not_a_field(&first.text));
    };
    let replaced = set_path(current, rest, value)?;
    let mut changed: Record = (**fields).clone();
    changed.set(&first.text, replaced);
    Ok(Value::Record(Rc::new(changed)))
}

fn not_a_field(name: &str) -> EvalError {
    EvalError::internal(Message::with(
        crate::message!("{0} is not a field a test can give a value to"),
        [Arg::from(name)],
    ))
}

/// The operands a failing `expect` reports: each side of a comparison or arithmetic, through
/// `and`, `or` and `not`, leaving out literals.
fn operands(expr: &Expr) -> Vec<&Expr> {
    let mut out = Vec::new();
    collect(expr, &mut out, 0);
    out
}

fn collect<'e>(expr: &'e Expr, out: &mut Vec<&'e Expr>, depth: usize) {
    use crate::tariff::syntax::ast::{BinaryOp, UnaryOp};
    if depth > 8 || out.len() >= 16 {
        return;
    }
    match &expr.kind {
        ExprKind::Binary {
            op: BinaryOp::And | BinaryOp::Or,
            left,
            right,
            ..
        } => {
            collect(left, out, depth + 1);
            collect(right, out, depth + 1);
        }
        ExprKind::Unary {
            op: UnaryOp::Not,
            operand,
            ..
        } => collect(operand, out, depth + 1),
        ExprKind::Binary { left, right, .. } => {
            for side in [left, right] {
                if !is_literal(side) {
                    out.push(side);
                }
            }
        }
        _ if !is_literal(expr) => out.push(expr),
        _ => {}
    }
}

fn is_literal(expr: &Expr) -> bool {
    match &expr.kind {
        ExprKind::Number { .. }
        | ExprKind::Money { .. }
        | ExprKind::Percent(_)
        | ExprKind::Date(_)
        | ExprKind::DateTime(_)
        | ExprKind::Bool(_)
        | ExprKind::None
        | ExprKind::Path { .. } => true,
        ExprKind::Text(text) => text
            .parts
            .iter()
            .all(|part| matches!(part, TextPart::Literal(..))),
        ExprKind::Unary { operand, .. } => is_literal(operand),
        _ => false,
    }
}
