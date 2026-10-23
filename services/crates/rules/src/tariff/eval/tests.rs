//! The test runner: a file's `test` and `property` blocks, run against the sample record, in
//! source order.
//!
//! A test in a tariff or condition:
//!
//! 1. starts from the sample record and the inputs' defaults;
//! 2. applies its `given`s in order: a record path (`shipment.weight`, `lane.origin.zip3`) or
//!    `input.name` (a value outside the input's `where` constraint is an `input` error). The
//!    test constructors (`pickup(…)`, `delivery(…)`, `splitPickup(…)`, `splitDelivery(…)`,
//!    `commodity(…)`) build stops and commodities from the sample defaults.
//!    Afterwards each stop without a sequence takes its position, and `shipment.pickup` and
//!    `shipment.delivery` follow the stops unless a `given` set them. A `given tables.Code[k,
//!    …] = value` gives a cell: every table the test gives cells of holds only those cells for
//!    the rest of the test ([`Tables::with_given`]), so its lookups never read live cells or
//!    record needs;
//! 3. runs the file;
//! 4. runs the rest of its body in order, with `charge(CODE)` / `charge("Label")` (the matching
//!    lines' sum, or `none`), `chosen("Label")` (the option that choice last kept, or `none`),
//!    `total` and `lines` in scope, and in a condition `applies` (its result). `expect cond`
//!    must hold; `expect fails` passes when the rating was refused (with that text, when one is
//!    given).
//!
//! A test in a library runs its body in order with no rating: `expect`s over function calls.
//! There, `expect fails` passes when the statement before it refused (with that text, when one
//! is given).
//!
//! A failing `expect` reports the values of the operands in it, so the author sees what the
//! tariff computed. An evaluation error makes the test's status `error`.
//!
//! A **property** runs many cases ([`super::property`]); each case is a test with extra
//! `given`s, one per `vary`, and its `assume`s checked before the rating. The runner here is
//! shared: [`Runner::rating_case`] and [`Runner::library_case`] run one test or one case.
//!
//! Tests read rate tables under the same on-demand protocol as a rating
//! ([`super::tables`]): a lookup of cells the request did not carry is a provisional miss and
//! the test keeps running, so one pass finds every need it can, from its `given`s, the rating
//! and its `expect`s alike. A test that recorded needs ends with status `needs` whatever else
//! happened, since the provisional miss may have caused it; the host loads the cells and runs
//! the tests again.
//!
//! **Budgets.** Each test, and each case of a property, runs under the request's limits. The
//! whole run also shares one step meter of [`FILE_STEPS`] times the per-test step limit: once
//! it is spent, every test and case left ends in a `budget` error, so a file of many
//! properties still finishes in bounded time.
//!
//! **Coverage.** When every test runs (no `names`), each test and property case records the
//! root file's regions it entered ([`super::coverage`]); the run's total is [`Suite::coverage`].

use super::StaticTypes;
use super::bill::Finished;
use super::budget::{Limits, Meter};
use super::coverage::{Hits, Regions};
use super::error::{Code, EvalError, Refusal, Stop};
use super::interp::link::Linked;
use super::interp::{Env, Interp, Setup, TestView};
use super::property::Counterexample;
use super::record::{sample, settle};
use super::run::{Supplied, bind_inputs, enforce_constraint};
use super::tables::{GivenCell, Need, Tables};
use super::value::{Record, Value};
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::schema::Settings;
use crate::tariff::syntax::ast::{
    Block, Expect, Expr, ExprKind, FileKind, ItemKind, Name, Stmt, StmtKind, TextPart,
};
use crate::tariff::syntax::program::Program;
use std::collections::BTreeMap;
use std::rc::Rc;

/// The whole run's step meter, as a multiple of the per-test step limit.
pub const FILE_STEPS: u64 = 32;

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

/// What a result is for.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Kind {
    /// A `test` block.
    Test,
    /// A `property` block.
    Property,
}

impl Kind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Test => "test",
            Self::Property => "property",
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

/// One test's or property's result.
#[derive(Debug, Clone)]
pub struct TestResult {
    /// The test's name.
    pub name: String,
    /// The test's name as written.
    pub span: Span,
    /// A test or a property.
    pub kind: Kind,
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
    /// How many cases rated: 1 for a test.
    pub cases: usize,
    /// How many cases an `assume` skipped: 0 for a test.
    pub skipped: usize,
    /// A property's failing case.
    pub counterexample: Option<Counterexample>,
}

/// A whole run: every result, and the coverage of the root file's regions.
#[derive(Debug, Clone)]
pub struct Suite {
    /// The results, in source order.
    pub results: Vec<TestResult>,
    /// How many times each region ran; `None` when only some tests ran (`names`) or a test
    /// ended `needs`.
    pub coverage: Option<Hits>,
}

/// Runs the root file's tests and properties.
#[must_use]
pub fn run_tests(request: &TestRequest<'_>) -> Vec<TestResult> {
    run_suite(request).results
}

/// Runs the root file's tests and properties, recording coverage when all of them run.
#[must_use]
pub fn run_suite(request: &TestRequest<'_>) -> Suite {
    let linked = Linked::new(request.program);
    let Some(root) = request.program.root() else {
        return Suite {
            results: Vec::new(),
            coverage: None,
        };
    };
    let library = root
        .file
        .header
        .as_ref()
        .is_some_and(|header| header.kind == FileKind::Library);
    let regions = request
        .names
        .is_none()
        .then(|| Rc::new(Regions::of(&root.file)));
    let mut ledger = Ledger::new(request.limits, regions);
    let selected = |name: &Name| request.names.is_none_or(|names| names.contains(&name.text));
    let mut results = Vec::new();
    for item in &root.file.items {
        let mut runner = Runner::new(request, &linked, &mut ledger);
        match &item.kind {
            ItemKind::Test(test) if selected(&test.name) => {
                results.push(runner.test(&test.name, &test.body, library));
            }
            ItemKind::Property(property) if selected(&property.name) => {
                results.push(super::property::run(&mut runner, property, library));
            }
            _ => {}
        }
    }
    let needs = results.iter().any(|result| result.status == Status::Needs);
    Suite {
        coverage: ledger.hits.filter(|_| !needs),
        results,
    }
}

/// What every test of a run shares: the file's step meter and the coverage counts.
#[derive(Debug)]
pub struct Ledger {
    spent: u64,
    limit: u64,
    regions: Option<Rc<Regions>>,
    hits: Option<Hits>,
    /// The program's landmarks, gathered for the first property that varies over a range.
    pub landmarks: Option<Rc<Vec<Value>>>,
    /// Whether the code running now counts toward coverage: tests and a property's generated
    /// cases do, a property's shrinking runs do not.
    pub counting: bool,
}

impl Ledger {
    /// A fresh ledger for a run under `limits`, counting `regions` when given.
    #[must_use]
    pub fn new(limits: Limits, regions: Option<Rc<Regions>>) -> Self {
        Self {
            spent: 0,
            limit: limits.steps.saturating_mul(FILE_STEPS),
            hits: regions
                .as_ref()
                .map(|regions| Hits::new(Rc::clone(regions))),
            regions,
            landmarks: None,
            counting: true,
        }
    }

    /// The `budget` error once the file's step meter is spent.
    #[must_use]
    pub fn exhausted(&self) -> Option<EvalError> {
        (self.spent >= self.limit).then(|| {
            EvalError::with(
                Code::Budget,
                crate::message!(
                    "The tests used up the file's step budget ({0, plural, one {# step} other {# steps}}) before they finished"
                ),
                [Arg::Count(i64::try_from(self.limit).unwrap_or(i64::MAX))],
            )
        })
    }
}

/// Runs tests and property cases, collecting what each one found.
pub struct Runner<'r, 'p, 'a> {
    /// The request.
    pub request: &'r TestRequest<'p>,
    /// The program.
    pub linked: &'r Linked<'a>,
    /// What the run shares.
    pub ledger: &'r mut Ledger,
    /// The `expect`s that did not hold in the test or case being run.
    pub failures: Vec<Failure>,
    /// The table cells it needed.
    pub needs: Vec<Need>,
    /// The value of each monotone `expect`, in order: `None` when the rating refused.
    pub monotone: Vec<Option<Value>>,
    /// The tables the test or case being run sees, once it gives table cells.
    given_tables: Option<Rc<Tables>>,
}

impl std::fmt::Debug for Runner<'_, '_, '_> {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("Runner")
            .field("failures", &self.failures.len())
            .finish_non_exhaustive()
    }
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

/// What a property case adds to a test: a value for each varied path, input and name.
#[derive(Debug, Default)]
pub struct CaseSetup<'a> {
    /// Record paths and their values, applied after the `given`s.
    pub paths: Vec<(&'a [Name], Value)>,
    /// Input values, applied after the `given`s.
    pub inputs: Vec<(String, Value)>,
    /// Names bound for the `assume`s and the statements after the rating.
    pub names: Vec<(String, Value)>,
}

/// How one test or case ended, when it could be evaluated.
#[derive(Debug, Clone)]
pub enum Ended {
    /// It ran to its end: the bill, for a tariff (the `expect`s' failures are the runner's).
    Rated(Option<Finished>),
    /// The rating (or, in a library, a statement) refused.
    Refused(Refusal),
    /// An `assume` did not hold, so nothing was rated.
    Skipped,
}

/// How the tested rating ended.
enum Rating {
    Rated(Finished),
    Refused(Refusal),
}

impl<'r, 'p, 'a> Runner<'r, 'p, 'a> {
    /// A runner for one test or property.
    pub fn new(
        request: &'r TestRequest<'p>,
        linked: &'r Linked<'a>,
        ledger: &'r mut Ledger,
    ) -> Self {
        Self {
            request,
            linked,
            ledger,
            failures: Vec::new(),
            needs: Vec::new(),
            monotone: Vec::new(),
            given_tables: None,
        }
    }

    /// An interpreter over `record`, with the test constructors in scope.
    pub fn interp(&self, record: Value) -> Interp<'r, 'a> {
        let mut interp = Interp::new(Setup {
            linked: self.linked,
            types: self.request.types,
            tables: self.request.tables,
            record,
            meter: Meter::new(self.request.limits),
            trace: false,
        });
        interp.constructors = true;
        interp.given_tables.clone_from(&self.given_tables);
        if self.ledger.counting {
            interp.coverage = self
                .ledger
                .regions
                .as_ref()
                .map(|regions| Hits::new(Rc::clone(regions)));
        }
        interp
    }

    /// Moves what an interpreter recorded onto the run: its needs, its steps and the regions
    /// it entered.
    pub fn gather(&mut self, interp: &mut Interp<'_, '_>) {
        for need in std::mem::take(&mut interp.needs) {
            if !self.needs.contains(&need) {
                self.needs.push(need);
            }
        }
        self.ledger.spent = self.ledger.spent.saturating_add(interp.meter.steps_used());
        if let (Some(hits), Some(total)) = (interp.coverage.take(), self.ledger.hits.as_mut()) {
            total.merge(&hits);
        }
    }

    /// Runs one `test` block.
    fn test(&mut self, name: &Name, body: &'a Block, library: bool) -> TestResult {
        let setup = CaseSetup::default();
        let ended = match self.ledger.exhausted() {
            Some(error) => Err(error),
            None if library => self.library_case(body, &setup),
            None => self.rating_case(body, &setup),
        };
        let (status, error, bill) = match ended {
            _ if !self.needs.is_empty() => {
                self.failures.clear();
                (Status::Needs, None, None)
            }
            Ok(Ended::Rated(bill)) if self.failures.is_empty() => (Status::Passed, None, bill),
            Ok(Ended::Rated(bill)) => (Status::Failed, None, bill),
            Ok(Ended::Refused(_) | Ended::Skipped) if self.failures.is_empty() => {
                (Status::Passed, None, None)
            }
            Ok(Ended::Refused(_) | Ended::Skipped) => (Status::Failed, None, None),
            Err(error) => (Status::Error, Some(error), None),
        };
        TestResult {
            name: name.text.clone(),
            span: name.span,
            kind: Kind::Test,
            status,
            failures: std::mem::take(&mut self.failures),
            error,
            bill,
            needs: std::mem::take(&mut self.needs),
            cases: 1,
            skipped: 0,
            counterexample: None,
        }
    }

    /// One test (or property case) of a tariff or condition: the `given`s and the case's
    /// values, its `assume`s, the rating, then the body's other statements.
    ///
    /// # Errors
    ///
    /// When the test or the rating could not be evaluated.
    pub fn rating_case(
        &mut self,
        body: &'a Block,
        setup: &CaseSetup<'a>,
    ) -> Result<Ended, EvalError> {
        let (record, inputs) = self.givens(body, setup)?;
        if !self.assumed(body, setup, &record, &inputs)? {
            return Ok(Ended::Skipped);
        }
        let mut interp = self.interp(record.clone());
        let mut env = Env::new(0, false);
        let ran = bind_inputs(&mut interp, &mut env, &Supplied::Values(&inputs))
            .and_then(|()| interp.run_file(&mut env));
        let applies = match (&ran, self.condition(), &interp.last_value) {
            (Ok(()), true, Some(Value::Bool(holds))) => Some(*holds),
            _ => None,
        };
        if let Some(holds) = applies {
            interp.enter_outcome(holds);
        }
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
                chosen: interp.choices.last().clone(),
                applies,
            });
        }
        let outcome = self.expectations(&mut checker, body, &rating, setup);
        self.gather(&mut checker);
        outcome?;
        Ok(match rating {
            Rating::Rated(bill) => Ended::Rated(Some(bill)),
            Rating::Refused(refusal) => Ended::Refused(refusal),
        })
    }

    /// Whether the file under test is a condition, whose tests read `applies`.
    fn condition(&self) -> bool {
        self.request
            .program
            .root()
            .and_then(|root| root.file.header.as_ref())
            .is_some_and(|header| header.kind == FileKind::Condition)
    }

    /// Whether every `assume` of a case holds, over its record, inputs and varied names.
    fn assumed(
        &mut self,
        body: &'a Block,
        setup: &CaseSetup<'a>,
        record: &Value,
        inputs: &BTreeMap<String, Value>,
    ) -> Result<bool, EvalError> {
        if !body
            .stmts
            .iter()
            .any(|stmt| matches!(stmt.kind, StmtKind::Assume(_)))
        {
            return Ok(true);
        }
        let mut interp = self.interp(record.clone());
        let mut env = Env::new(0, false);
        let held = match bind_inputs(&mut interp, &mut env, &Supplied::Values(inputs)) {
            Err(Stop::Failed(error)) => Err(error),
            Err(Stop::Refused(_)) => Ok(true),
            Ok(()) => {
                bind_names(&mut env, setup);
                assumptions(&mut interp, &mut env, body)
            }
        };
        self.gather(&mut interp);
        held
    }

    /// A rating test's body after the rating: its `expect`s and the statements between them.
    fn expectations(
        &mut self,
        checker: &mut Interp<'r, 'a>,
        body: &'a Block,
        rating: &Rating,
        setup: &CaseSetup<'a>,
    ) -> Result<(), EvalError> {
        let mut scope = Env::new(0, false);
        bind_names(&mut scope, setup);
        for stmt in &body.stmts {
            match &stmt.kind {
                StmtKind::Given { .. } | StmtKind::Vary { .. } | StmtKind::Assume(_) => {}
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
                        self.rating_refused(stmt.span, refusal);
                        continue;
                    }
                    self.expect_holds(checker, &mut scope, stmt, expr)?;
                }
                StmtKind::Expect(Expect::Monotone { value, .. }) => {
                    if let Rating::Refused(refusal) = rating {
                        self.rating_refused(stmt.span, refusal);
                        self.monotone.push(None);
                        continue;
                    }
                    let found = checker.eval_in(&mut scope, value).map_err(stopped)?;
                    self.monotone.push(Some(found));
                }
                _ => {
                    checker.exec(&mut scope, stmt).map_err(stopped)?;
                }
            }
        }
        Ok(())
    }

    fn rating_refused(&mut self, span: Span, refusal: &Refusal) {
        self.failures.push(Failure {
            span,
            message: Message::with(
                crate::message!("The rating was refused: {0}"),
                [Arg::from(refusal.message.text())],
            ),
            values: Vec::new(),
        });
    }

    /// The record and input values after a test's `given`s and a case's values.
    fn givens(
        &mut self,
        body: &'a Block,
        setup: &CaseSetup<'a>,
    ) -> Result<(Value, BTreeMap<String, Value>), EvalError> {
        self.given_tables = None;
        let mut interp = self.interp(sample(self.request.settings));
        let applied = apply_givens(&mut interp, body, setup).and_then(|applied| {
            if applied.cells.is_empty() {
                return Ok(applied);
            }
            let tables = self
                .request
                .tables
                .with_given(&applied.cells, &mut interp.meter)?;
            self.given_tables = Some(Rc::new(tables));
            Ok(applied)
        });
        self.gather(&mut interp);
        let applied = applied?;
        Ok((
            settle(&interp.record, applied.pickup, applied.delivery),
            applied.inputs,
        ))
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

    /// One test (or property case) of a library: its body, in order, with no rating.
    ///
    /// # Errors
    ///
    /// When a statement could not be evaluated.
    pub fn library_case(
        &mut self,
        body: &'a Block,
        setup: &CaseSetup<'a>,
    ) -> Result<Ended, EvalError> {
        let mut interp = self.interp(sample(self.request.settings));
        let ran = self.library_body(&mut interp, body, setup);
        self.gather(&mut interp);
        ran
    }

    fn library_body(
        &mut self,
        interp: &mut Interp<'r, 'a>,
        body: &'a Block,
        setup: &CaseSetup<'a>,
    ) -> Result<Ended, EvalError> {
        let mut scope = Env::new(0, false);
        bind_names(&mut scope, setup);
        if !assumptions(interp, &mut scope, body)? {
            return Ok(Ended::Skipped);
        }
        let mut refused: Option<Refusal> = None;
        for stmt in &body.stmts {
            match &stmt.kind {
                StmtKind::Vary { .. } | StmtKind::Assume(_) => {}
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
                StmtKind::Expect(Expect::Monotone { .. }) if refused.is_some() => {
                    self.monotone.push(None);
                }
                _ if refused.is_some() => {}
                StmtKind::Expect(Expect::Holds(expr)) => {
                    self.expect_holds(interp, &mut scope, stmt, expr)?;
                }
                StmtKind::Expect(Expect::Monotone { value, .. }) => {
                    let found = interp.eval_in(&mut scope, value).map_err(stopped)?;
                    self.monotone.push(Some(found));
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
        let Some(refusal) = refused else {
            return Ok(Ended::Rated(None));
        };
        self.failures.push(Failure {
            span: refusal.span,
            message: Message::with(
                crate::message!("The test was refused: {0}"),
                [Arg::from(refusal.message.text())],
            ),
            values: Vec::new(),
        });
        Ok(Ended::Refused(refusal))
    }
}

/// Binds a case's varied names in a scope.
fn bind_names(scope: &mut Env, setup: &CaseSetup<'_>) {
    for (name, value) in &setup.names {
        scope.bind(name, value.clone(), false);
    }
}

/// Whether every `assume` of a body holds in `env`.
fn assumptions<'a>(
    interp: &mut Interp<'_, 'a>,
    env: &mut Env,
    body: &'a Block,
) -> Result<bool, EvalError> {
    for stmt in &body.stmts {
        let StmtKind::Assume(condition) = &stmt.kind else {
            continue;
        };
        match interp.eval_in(env, condition).map_err(stopped)? {
            Value::Bool(true) => {}
            Value::Bool(false) => return Ok(false),
            other => {
                return Err(EvalError::internal(Message::with(
                    crate::message!("An assumption must be true or false, not {0}"),
                    [Arg::from(other.kind())],
                ))
                .at(condition.span));
            }
        }
    }
    Ok(true)
}

/// What a test's `given`s set besides the record.
struct Applied {
    /// The input values.
    inputs: BTreeMap<String, Value>,
    /// Whether `shipment.pickup` was given.
    pickup: bool,
    /// Whether `shipment.delivery` was given.
    delivery: bool,
    /// The table cells, in order.
    cells: Vec<GivenCell>,
}

/// Applies a test's `given`s and a case's values to the interpreter's record: the input values
/// they set, whether they set `shipment.pickup` and `shipment.delivery`, and the table cells
/// they give.
fn apply_givens<'a>(
    interp: &mut Interp<'_, 'a>,
    body: &'a Block,
    setup: &CaseSetup<'a>,
) -> Result<Applied, EvalError> {
    let mut inputs = BTreeMap::new();
    let mut cells = Vec::new();
    let (mut pickup, mut delivery) = (false, false);
    let mut set = |interp: &mut Interp<'_, 'a>, path: &[Name], value: Value, span: Option<Span>| {
        pickup |=
            matches!(path, [root, field] if root.text == "shipment" && field.text == "pickup");
        delivery |=
            matches!(path, [root, field] if root.text == "shipment" && field.text == "delivery");
        interp.record = set_path(&interp.record, path, value).map_err(|error| match span {
            Some(span) => error.at(span),
            None => error,
        })?;
        Ok::<(), EvalError>(())
    };
    for stmt in &body.stmts {
        let StmtKind::Given {
            target,
            keys,
            value,
        } = &stmt.kind
        else {
            continue;
        };
        let mut scope = Env::new(0, false);
        if let ([head, code], Some(keys)) = (target.as_slice(), keys)
            && head.text == "tables"
        {
            let mut values = Vec::with_capacity(keys.len());
            for key in keys {
                values.push(interp.eval_in(&mut scope, key).map_err(stopped)?);
            }
            cells.push(GivenCell {
                code: code.text.clone(),
                keys: values,
                value: interp.eval_in(&mut scope, value).map_err(stopped)?,
                span: stmt.span,
            });
            continue;
        }
        let value = interp.eval_in(&mut scope, value).map_err(stopped)?;
        match target.as_slice() {
            [head, name] if head.text == "input" => {
                enforce_constraint(interp, &name.text, &value)
                    .map_err(|error| error.at(stmt.span))?;
                inputs.insert(name.text.clone(), value);
            }
            path => set(interp, path, value, Some(stmt.span))?,
        }
    }
    for (path, value) in &setup.paths {
        set(interp, path, value.clone(), None)?;
    }
    for (name, value) in &setup.inputs {
        enforce_constraint(interp, name, value)?;
        inputs.insert(name.clone(), value.clone());
    }
    Ok(Applied {
        inputs,
        pickup,
        delivery,
        cells,
    })
}

/// A refusal or error from a test's own statements, as the error that stops the test.
#[must_use]
pub fn stopped(stop: Stop) -> EvalError {
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
