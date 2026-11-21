//! `lading.test`: every test and property of a project, the statements no test reached, and
//! the data in the project's samples that no map reads.

use std::collections::{BTreeMap, BTreeSet};
use std::rc::Rc;

use rust_decimal::Decimal;
use serde::Serialize;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::check::tests::{Subject, glob_matches, subject};
use super::check::{MapInfo, Program, Source};
use super::eval::interp::{Frame, Given, Interp, OutcomeKind2, RUN_STEPS};
use super::eval::json::{from_json, to_json};
use super::eval::run::{fill_value, read_x12, sendable, unread, write_message};
use super::eval::value::{Value, display, equal};
use super::flow::{FlowAction, FlowEvent, FlowState, outcome_inputs, route, step};
use super::syntax::ast::{
    BinOp, Expr, ExprKind, FlowDecl, Header, Item, Stmt, TestStep, VarySource,
};
use super::types::Type;
use super::values::parse_instant;
use crate::x12::finding::Severity;

/// The most cases a property runs.
pub const MAX_CASES: usize = 256;

/// Where tests start the clock unless they say otherwise: 2026-01-01T00:00Z.
const DEFAULT_NOW: i64 = 1_767_225_600;

/// Why a test failed, at a step.
#[derive(Debug, Clone, PartialEq)]
pub struct Failure {
    /// The step.
    pub span: Span,
    /// What went wrong.
    pub message: Message,
}

/// One test or property's result.
#[derive(Debug, Clone, PartialEq)]
pub struct TestResult {
    /// The file.
    pub unit: usize,
    /// Its name.
    pub name: String,
    /// `test` or `property`.
    pub kind: &'static str,
    /// Where its name is.
    pub span: Span,
    /// Whether it passed.
    pub passed: bool,
    /// What failed.
    pub failures: Vec<Failure>,
    /// The cases a property ran.
    pub cases: usize,
    /// The simplest failing case of a property, as its varied names and values.
    pub counterexample: Option<Vec<(String, String)>>,
}

/// The statements of one file and whether a test reached them.
#[derive(Debug, Clone, PartialEq)]
pub struct FileCoverage {
    /// The file.
    pub unit: usize,
    /// Statements a test reached.
    pub covered: Vec<Span>,
    /// Statements no test reached.
    pub uncovered: Vec<Span>,
}

/// Data the samples hold that no map reads.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UnreadData {
    /// The map that reads the samples.
    pub map: String,
    /// The element (`N902`).
    pub element: String,
    /// The segment's qualifier, when its first element is one (`PO` of `N9*PO`).
    #[serde(skip_serializing_if = "String::is_empty")]
    pub qualifier: String,
    /// In how many samples it appears unread.
    pub samples: usize,
    /// How many samples the map read.
    pub total: usize,
    /// Where it was in the first sample: `0300[1].N9[1]02`.
    pub example: String,
    /// How a map reads it, when it is at the top of the transaction.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub read: String,
}

/// Everything `lading.test` reports.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct TestReport {
    /// Every test and property.
    pub tests: Vec<TestResult>,
    /// Coverage by file.
    pub coverage: Vec<FileCoverage>,
    /// Unread data.
    pub unread: Vec<UnreadData>,
}

/// Runs every test and property of a program.
#[must_use]
pub fn run_tests(program: &Program) -> TestReport {
    let mut report = TestReport::default();
    let mut covered = BTreeSet::new();
    if program.has_errors() {
        return report;
    }
    for unit in 0..program.units.len() {
        let subject = subject(program, unit);
        for item in program.units[unit].items() {
            match item {
                Item::Test(test) => {
                    let mut runner = Runner::new(program, unit, subject);
                    runner.run(&test.steps, &[]);
                    covered.extend(runner.coverage.iter().copied());
                    report.tests.push(TestResult {
                        unit,
                        name: test.name.value.clone(),
                        kind: "test",
                        span: test.name.span,
                        passed: runner.failures.is_empty(),
                        failures: runner.failures,
                        cases: 1,
                        counterexample: None,
                    });
                }
                Item::Property(property) => {
                    let (result, hit) = run_property(program, unit, subject, property);
                    covered.extend(hit);
                    report.tests.push(result);
                }
                _ => {}
            }
        }
    }
    report.coverage = coverage(program, &covered);
    report.unread = unread_data(program);
    report
}

/// The values a property varies a name over.
fn cases_of(program: &Program, unit: usize, source: &VarySource) -> Vec<Value> {
    let given = Given::default();
    let mut interp = Interp::new(program, &given, RUN_STEPS);
    let mut frame = Frame::new(unit);
    match source {
        VarySource::Samples(glob) => program
            .samples
            .iter()
            .filter(|sample| glob_matches(&glob.value, &sample.path))
            .map(|sample| Value::text(&sample.path))
            .collect(),
        VarySource::List(list) => interp
            .eval(&mut frame, list)
            .map(|value| value.items().to_vec())
            .unwrap_or_default(),
        VarySource::Range(low, high) => {
            let low = interp.eval(&mut frame, low).unwrap_or(Value::None);
            let high = interp.eval(&mut frame, high).unwrap_or(Value::None);
            spread(&low, &high)
        }
        VarySource::Enum(name) => crate::lading::schema::enumeration(&name.name)
            .map(|item| {
                item.variants
                    .iter()
                    .map(|(variant, _)| Value::Enum(item.name, variant))
                    .collect()
            })
            .unwrap_or_default(),
    }
}

/// 24 values from `low` to `high`: both ends, the landmarks between (0, 1) and points spread
/// evenly, in order.
fn spread(low: &Value, high: &Value) -> Vec<Value> {
    let (Some(start), Some(end)) = (magnitude(low), magnitude(high)) else {
        return vec![low.clone(), high.clone()];
    };
    if end < start {
        return vec![low.clone()];
    }
    let mut points = vec![start, end];
    for landmark in [Decimal::ZERO, Decimal::ONE] {
        if landmark > start && landmark < end {
            points.push(landmark);
        }
    }
    let steps = Decimal::from(23 - i64::try_from(points.len()).unwrap_or(2) + 2);
    let width = end - start;
    let whole = matches!(low, Value::Int(_) | Value::Date(_) | Value::Duration(_));
    for index in 1..21 {
        let mut point = start + width * Decimal::from(index) / steps;
        if whole {
            point = point.round();
        } else {
            point = point.round_dp(2);
        }
        points.push(point);
    }
    points.sort();
    points.dedup();
    points.truncate(24);
    points
        .into_iter()
        .filter_map(|point| rebuild(low, point))
        .collect()
}

fn magnitude(value: &Value) -> Option<Decimal> {
    match value {
        Value::Int(number) => Some(Decimal::from(*number)),
        Value::Decimal(number)
        | Value::Percent(number)
        | Value::Money(number, _)
        | Value::Measure(number, _) => Some(*number),
        Value::Date(days) | Value::Duration(days) => Some(Decimal::from(*days)),
        Value::DateTime(instant, _) => Some(Decimal::from(*instant)),
        _ => None,
    }
}

fn rebuild(like: &Value, point: Decimal) -> Option<Value> {
    let whole = || i64::try_from(point.round()).ok();
    Some(match like {
        Value::Int(_) => Value::Int(whole()?),
        Value::Decimal(_) => Value::Decimal(point),
        Value::Percent(_) => Value::Percent(point),
        Value::Money(_, currency) => Value::Money(point, Rc::clone(currency)),
        Value::Measure(_, unit) => Value::Measure(point, *unit),
        Value::Date(_) => Value::Date(whole()?),
        Value::Duration(_) => Value::Duration(whole()?),
        Value::DateTime(_, zone) => Value::DateTime(whole()?, *zone),
        _ => return None,
    })
}

fn run_property(
    program: &Program,
    unit: usize,
    subject: Subject,
    property: &crate::lading::syntax::ast::PropertyDecl,
) -> (TestResult, BTreeSet<(usize, u32)>) {
    let mut axes: Vec<(String, Vec<Value>, bool)> = Vec::new();
    for vary in &property.varies {
        let values = cases_of(program, unit, &vary.source);
        axes.push((
            vary.name.name.clone(),
            values,
            matches!(vary.source, VarySource::Samples(_)),
        ));
    }
    let mut cases: Vec<Vec<(String, Value, bool)>> = vec![Vec::new()];
    for (name, values, sample) in &axes {
        let mut next = Vec::new();
        for case in &cases {
            for value in values {
                if next.len() >= MAX_CASES {
                    break;
                }
                let mut extended = case.clone();
                extended.push((name.clone(), value.clone(), *sample));
                next.push(extended);
            }
        }
        cases = next;
    }
    let mut result = TestResult {
        unit,
        name: property.name.value.clone(),
        kind: "property",
        span: property.name.span,
        passed: true,
        failures: Vec::new(),
        cases: 0,
        counterexample: None,
    };
    let mut covered = BTreeSet::new();
    if axes.iter().any(|(_, values, _)| values.is_empty()) {
        result.passed = false;
        result.failures.push(Failure {
            span: property.name.span,
            message: Message::new(message!(
                "A varied name has no values, so the property runs no case"
            )),
        });
        return (result, covered);
    }
    for case in cases {
        result.cases += 1;
        let mut runner = Runner::new(program, unit, subject);
        for (name, value, sample) in &case {
            if *sample {
                runner.input_sample(&display(value));
            }
            runner.bindings.push((name.clone(), value.clone()));
        }
        runner.run(&property.steps, &[]);
        covered.extend(runner.coverage.iter().copied());
        if !runner.failures.is_empty() {
            result.passed = false;
            result.failures = runner.failures;
            result.counterexample = Some(
                case.iter()
                    .map(|(name, value, _)| (name.clone(), display(value)))
                    .collect(),
            );
            break;
        }
    }
    (result, covered)
}

/// What a test was given before it runs.
#[derive(Debug, Clone)]
enum Input {
    X12(String),
    Text(String),
    Document(Value),
}

struct Runner<'p> {
    program: &'p Program,
    unit: usize,
    subject: Subject,
    given: Given,
    input: Option<Input>,
    bindings: Vec<(String, Value)>,
    failures: Vec<Failure>,
    coverage: BTreeSet<(usize, u32)>,
    transaction: Option<u32>,
}

/// What a map run in a test produced.
#[derive(Default)]
struct Produced {
    document: Option<Value>,
    outcomes: Vec<super::eval::interp::Outcome>,
    rejected: bool,
    warnings: Vec<String>,
    x12: String,
    /// Why the partner's guide or the writer would refuse what an outbound map wrote.
    invalid: Vec<String>,
    /// Whether an outbound map wrote nothing to send.
    not_sent: bool,
}

/// A send a flow test's flow asked for that did not go out, as the host would leave it.
struct Unsent {
    set: String,
    /// `held`, `rejected` or `not sent`.
    why: &'static str,
    message: String,
}

/// One instance of a flow test's flow: the conversation about one key.
#[derive(Default)]
struct Instance {
    key: Option<String>,
    state: FlowState,
    wake: Option<i64>,
}

/// What a flow test's flow did so far, across the instances its keys opened.
#[derive(Default)]
struct FlowRun {
    instances: Vec<Instance>,
    current: usize,
    actions: Vec<FlowAction>,
    unsent: Vec<Unsent>,
    control_numbers: u32,
}

impl FlowRun {
    /// Makes the instance for `key` current, opening it on its first event; an event without a
    /// key (a planner's decision, an acknowledgment) goes to the instance last used.
    fn select(&mut self, key: Option<String>) {
        if let Some(key) = key {
            if let Some(index) = self
                .instances
                .iter()
                .position(|instance| instance.key.as_deref() == Some(key.as_str()))
            {
                self.current = index;
                return;
            }
            if let Some(index) = self
                .instances
                .iter()
                .position(|instance| instance.key.is_none())
            {
                self.instances[index].key = Some(key);
                self.current = index;
                return;
            }
            self.instances.push(Instance {
                key: Some(key),
                ..Instance::default()
            });
            self.current = self.instances.len() - 1;
        } else if self.instances.is_empty() {
            self.instances.push(Instance::default());
            self.current = 0;
        }
    }

    fn instance(&mut self) -> &mut Instance {
        self.select(None);
        &mut self.instances[self.current]
    }
}

impl<'p> Runner<'p> {
    fn new(program: &'p Program, unit: usize, subject: Subject) -> Self {
        Self {
            program,
            unit,
            subject,
            given: Given {
                now: DEFAULT_NOW,
                stand_in: true,
                ..Given::default()
            },
            input: None,
            bindings: Vec::new(),
            failures: Vec::new(),
            coverage: BTreeSet::new(),
            transaction: None,
        }
    }

    fn fail(&mut self, span: Span, message: Message) {
        self.failures.push(Failure { span, message });
    }

    fn input_sample(&mut self, path: &str) {
        if let Some(sample) = self
            .program
            .samples
            .iter()
            .find(|sample| sample.path == path)
        {
            self.input = Some(Input::X12(sample.content.clone()));
        }
    }

    fn frame(&self) -> Frame {
        let mut frame = Frame::new(self.unit);
        for (name, value) in &self.bindings {
            frame.bind(name, value.clone());
        }
        frame
    }

    fn eval(&mut self, expr: &Expr, document: Option<&Value>) -> Value {
        let given = self.given.clone();
        let mut interp = Interp::new(self.program, &given, RUN_STEPS);
        let mut frame = self.frame();
        frame.document = document.cloned();
        interp.eval(&mut frame, expr).unwrap_or(Value::None)
    }

    fn run(&mut self, steps: &[(TestStep, Span)], _: &[()]) {
        if self.subject == Subject::Flow {
            self.run_flow(steps);
            return;
        }
        for (step, span) in steps {
            self.given_step(step, *span);
        }
        let Some(produced) = self.produce() else {
            return;
        };
        let several = produced.len() > 1;
        for (index, one) in produced.iter().enumerate() {
            let before = self.failures.len();
            for (step, span) in steps {
                self.expect_step(step, *span, one);
            }
            if several {
                for failure in &mut self.failures[before..] {
                    failure.message = Message::with(
                        message!("In transaction {0}: {1}"),
                        [Arg::from(index + 1), Arg::from(failure.message.english())],
                    );
                }
            }
        }
    }

    fn given_step(&mut self, step: &TestStep, span: Span) {
        match step {
            TestStep::GivenFile(path) => {
                let content = self
                    .program
                    .samples
                    .iter()
                    .find(|sample| sample.path == path.value)
                    .map(|sample| sample.content.clone());
                match content {
                    Some(content) => self.input = Some(Input::X12(content)),
                    None => self.fail(
                        span,
                        Message::with(
                            message!("There is no sample {0}"),
                            [Arg::from(path.value.as_str())],
                        ),
                    ),
                }
            }
            TestStep::GivenText(text) => {
                let value = self.eval(text, None);
                self.input = Some(Input::Text(display(&value)));
            }
            TestStep::GivenTransaction(index, _) => self.transaction = Some(*index),
            TestStep::GivenDocument { document, value } => {
                let value = self.eval(value, None);
                let given = self.given.clone();
                let interp = Interp::new(self.program, &given, RUN_STEPS);
                let name = crate::lading::schema::document(&document.name)
                    .map_or("", |record| record.name);
                let coerced = interp.coerce(value, &Type::Record(name));
                if coerced.is_none() {
                    self.fail(
                        span,
                        Message::new(message!("The document is missing a required field")),
                    );
                }
                self.input = Some(Input::Document(coerced));
            }
            TestStep::GivenSetting { name, value } => {
                let value = self.eval(value, None);
                self.given.settings.insert(name.name.clone(), value);
            }
            TestStep::GivenLookup { kind, code, label } => {
                let code = display(&self.eval(code, None));
                let found = label.as_ref().map(|label| {
                    let label = display(&self.eval(label, None));
                    super::eval::value::Reference {
                        kind: crate::lading::schema::lookup(&kind.name)
                            .map_or("", |lookup| lookup.name),
                        id: format!("{}:{code}", kind.name),
                        code: code.clone(),
                        label,
                        zone: None,
                    }
                });
                self.given.lookups.insert((kind.name.clone(), code), found);
            }
            TestStep::GivenNow(value) => {
                if let Value::DateTime(instant, _) = self.eval(value, None) {
                    self.given.now = instant;
                }
            }
            _ => {}
        }
    }

    fn map_info(&self) -> Option<(String, &'p MapInfo, bool)> {
        let program = self.program;
        match program.units[self.unit].header() {
            Header::Map { name, .. } => program
                .maps
                .get(&name.name)
                .map(|info| (name.name.clone(), info, false)),
            Header::Rules { map, .. } => program
                .maps
                .get(&map.name)
                .map(|info| (map.name.clone(), info, true)),
            _ => None,
        }
    }

    /// Runs what the test is about on its input: one result per transaction a message holds
    /// (or the one `given transaction` picks), or one for any other input.
    #[allow(clippy::too_many_lines)]
    fn produce(&mut self) -> Option<Vec<Produced>> {
        let program = self.program;
        let given = self.given.clone();
        match self.subject {
            Subject::Library | Subject::None | Subject::Flow => Some(vec![Produced::default()]),
            Subject::Guide => {
                let Header::Guide { name, .. } = program.units[self.unit].header() else {
                    return None;
                };
                let guide = *program.guide_names.get(&name.name)?;
                let text = match &self.input {
                    Some(Input::X12(text) | Input::Text(text)) => text.clone(),
                    _ => return Some(vec![Produced::default()]),
                };
                let mut produced = Produced::default();
                match read_x12(program, guide, &text) {
                    Ok(transactions) => {
                        for transaction in transactions {
                            produced.rejected |= transaction.rejected;
                            produced.warnings.extend(
                                transaction
                                    .findings
                                    .iter()
                                    .filter(|finding| finding.severity == Severity::Warning)
                                    .map(|finding| finding.message.english()),
                            );
                        }
                    }
                    Err(_) => produced.rejected = true,
                }
                Some(vec![produced])
            }
            Subject::Inbound(_) => {
                let (map, info, rules) = self.map_info()?;
                let mut frame = Frame::new(info.unit);
                match (&info.source, self.input.clone()) {
                    (Source::X12(guide), Some(Input::X12(text) | Input::Text(text))) => {
                        let transactions = match read_x12(program, *guide, &text) {
                            Ok(transactions) => transactions,
                            Err(message) => {
                                self.fail(Span::default(), message);
                                return None;
                            }
                        };
                        let count = transactions.len();
                        let picked: Vec<_> = transactions
                            .into_iter()
                            .enumerate()
                            .filter(|(index, _)| {
                                self.transaction
                                    .is_none_or(|wanted| wanted as usize == index + 1)
                            })
                            .map(|(_, transaction)| transaction)
                            .collect();
                        if let Some(wanted) = self.transaction
                            && picked.is_empty()
                        {
                            self.fail(
                                Span::default(),
                                Message::with(
                                    message!("The message holds {0} transactions, not {1}"),
                                    [Arg::from(count), Arg::from(i64::from(wanted))],
                                ),
                            );
                            return None;
                        }
                        let mut all = Vec::new();
                        for transaction in picked {
                            let mut produced = Produced {
                                rejected: transaction.rejected,
                                ..Produced::default()
                            };
                            if let Some(tree) = transaction.tree {
                                let mut frame = Frame::new(info.unit);
                                frame.root = Some(tree);
                                self.fill(&given, &map, rules, info, frame, &mut produced)?;
                            }
                            all.push(produced);
                        }
                        if all.is_empty() {
                            all.push(Produced::default());
                        }
                        return Some(all);
                    }
                    (Source::Json, Some(Input::X12(text) | Input::Text(text))) => {
                        match serde_json::from_str(&text) {
                            Ok(json) => frame.bind("input", Value::Json(Rc::new(json))),
                            Err(problem) => {
                                self.fail(
                                    Span::default(),
                                    Message::with(
                                        message!("The JSON does not read: {0}"),
                                        [Arg::from(problem.to_string())],
                                    ),
                                );
                                return None;
                            }
                        }
                    }
                    (Source::Csv(types), Some(Input::X12(text) | Input::Text(text))) => {
                        let columns = match program.units[info.unit].header() {
                            Header::Map {
                                from: crate::lading::syntax::ast::Endpoint::Csv { columns, .. },
                                ..
                            } => columns
                                .iter()
                                .cloned()
                                .zip(types.iter().map(|(_, ty)| ty.clone()))
                                .collect::<Vec<_>>(),
                            _ => Vec::new(),
                        };
                        let (rows, _) =
                            super::eval::csv::rows(&text, &columns, super::values::ZoneValue::UTC);
                        frame.bind("rows", Value::list(rows));
                    }
                    _ => {
                        self.fail(
                            Span::default(),
                            Message::new(message!("The test gives no input this map reads")),
                        );
                        return None;
                    }
                }
                let mut produced = Produced::default();
                self.fill(&given, &map, rules, info, frame, &mut produced)?;
                Some(vec![produced])
            }
            Subject::Outbound(_) => {
                let (map, info, rules) = self.map_info()?;
                let Some(Input::Document(document)) = self.input.clone() else {
                    return Some(vec![Produced::default()]);
                };
                let result = write_message(program, &given, &map, &to_json(&document), rules);
                self.coverage.extend(result.coverage.iter().copied());
                if let Some(error) = result.error {
                    self.fail(Span::default(), error);
                    return None;
                }
                if result.not_sent {
                    return Some(vec![Produced {
                        document: Some(document),
                        not_sent: true,
                        ..Produced::default()
                    }]);
                }
                let output = result.output?;
                let invalid: Vec<String> = output
                    .findings
                    .iter()
                    .filter(|finding| finding.severity == "error")
                    .map(|finding| finding.message.english())
                    .chain(
                        output
                            .validation
                            .iter()
                            .filter(|finding| finding.severity == Severity::Error)
                            .map(|finding| finding.message.english()),
                    )
                    .collect();
                let written = info
                    .document
                    .and_then(|name| {
                        from_json(program, &output.document, &Type::Record(name), "").ok()
                    })
                    .unwrap_or(document);
                Some(vec![Produced {
                    document: Some(written),
                    outcomes: output.outcomes,
                    rejected: !invalid.is_empty(),
                    warnings: output
                        .findings
                        .iter()
                        .filter(|finding| finding.severity == "warning")
                        .map(|finding| finding.message.english())
                        .collect(),
                    x12: output.transaction,
                    invalid,
                    not_sent: false,
                }])
            }
        }
    }

    /// Fills an inbound map's document and runs its rules into `produced`.
    fn fill(
        &mut self,
        given: &Given,
        map: &str,
        rules: bool,
        info: &MapInfo,
        frame: Frame,
        produced: &mut Produced,
    ) -> Option<()> {
        let mut interp = Interp::new(self.program, given, RUN_STEPS);
        if let Ok((document, _)) = fill_value(&mut interp, map, rules, info, frame) {
            produced.document = Some(document);
        } else {
            self.fail(
                Span::default(),
                Message::new(message!("The run used more than its budget of steps")),
            );
            return None;
        }
        produced.outcomes = std::mem::take(&mut interp.outcomes);
        self.coverage.extend(interp.coverage.iter().copied());
        Some(())
    }

    fn expect_step(&mut self, step: &TestStep, span: Span, produced: &Produced) {
        match step {
            TestStep::Expect(cond) => self.expect_true(cond, span, produced.document.as_ref()),
            TestStep::ExpectX12 { contains, text } => {
                let wanted = display(&self.eval(text, None));
                let holds = if *contains {
                    produced.x12.contains(&wanted)
                } else {
                    produced.x12 == wanted
                };
                if !produced.invalid.is_empty() {
                    self.fail(
                        span,
                        Message::with(
                            message!("The map wrote {0}, and the partner's guide refuses it: {1}"),
                            [
                                Arg::from(produced.x12.as_str()),
                                Arg::from(produced.invalid.join("; ")),
                            ],
                        ),
                    );
                } else if !holds {
                    self.fail(
                        span,
                        Message::with(
                            message!("The map wrote {0}"),
                            [Arg::from(produced.x12.as_str())],
                        ),
                    );
                }
            }
            TestStep::ExpectOutcome {
                negated,
                what,
                detail,
                ..
            } => {
                let detail = detail
                    .as_ref()
                    .map(|detail| display(&self.eval(detail, None)));
                let kind = match what.name.as_str() {
                    "held" | "holds" => Some(OutcomeKind2::Hold),
                    "rejected" | "rejects" => Some(OutcomeKind2::Reject),
                    "declined" | "declines" => Some(OutcomeKind2::Decline),
                    "warned" | "warnings" => Some(OutcomeKind2::Warn),
                    _ => None,
                };
                let found = match kind {
                    None if what.name == "sent" => !produced.not_sent && !produced.x12.is_empty(),
                    Some(OutcomeKind2::Reject) if produced.rejected => true,
                    Some(OutcomeKind2::Warn)
                        if !produced.warnings.is_empty() && detail.is_none() =>
                    {
                        true
                    }
                    Some(kind) => {
                        produced.outcomes.iter().any(|outcome| {
                            outcome.kind == kind
                                && detail
                                    .as_ref()
                                    .is_none_or(|detail| outcome.message.contains(detail.as_str()))
                        }) || (kind == OutcomeKind2::Warn
                            && detail.as_ref().is_some_and(|detail| {
                                produced
                                    .warnings
                                    .iter()
                                    .any(|warning| warning.contains(detail.as_str()))
                            }))
                    }
                    None => false,
                };
                if found == *negated {
                    let listed: Vec<String> = produced
                        .outcomes
                        .iter()
                        .map(|outcome| outcome.message.clone())
                        .chain(produced.invalid.iter().cloned())
                        .chain(produced.not_sent.then(|| {
                            Message::new(message!("Nothing in it is sent to this partner"))
                                .english()
                        }))
                        .collect();
                    self.fail(
                        span,
                        if *negated {
                            Message::with(
                                message!("Expected no {0}, and there was: {1}"),
                                [Arg::from(what.name.as_str()), Arg::from(listed.join("; "))],
                            )
                        } else {
                            Message::with(
                                message!("Expected {0}, and the outcomes were: {1}"),
                                [Arg::from(what.name.as_str()), Arg::from(listed.join("; "))],
                            )
                        },
                    );
                }
            }
            _ => {}
        }
    }

    fn expect_true(&mut self, cond: &Expr, span: Span, document: Option<&Value>) {
        let value = self.eval(cond, document);
        if value.truthy() {
            return;
        }
        let source = &self.program.units[self.unit].source;
        let text = source
            .get(cond.span.start as usize..cond.span.end as usize)
            .unwrap_or_default()
            .to_owned();
        if let ExprKind::Binary(BinOp::Eq, lhs, rhs) = &cond.kind {
            let left = self.eval(lhs, document);
            let right = self.eval(rhs, document);
            if !equal(&left, &right) {
                self.fail(
                    span,
                    Message::with(
                        message!("{0}: the left side is {1}, the right {2}"),
                        [
                            Arg::from(text),
                            Arg::from(shown(&left)),
                            Arg::from(shown(&right)),
                        ],
                    ),
                );
                return;
            }
        }
        self.fail(
            span,
            Message::with(message!("This is not true: {0}"), [Arg::from(text)]),
        );
    }

    #[allow(clippy::too_many_lines)]
    fn run_flow(&mut self, steps: &[(TestStep, Span)]) {
        let program = self.program;
        let Header::Flow { name: flow, .. } = program.units[self.unit].header() else {
            return;
        };
        let flow = flow.name.clone();
        let mut run = FlowRun::default();
        for (step_kind, span) in steps {
            self.given_step(step_kind, *span);
            let mut moves_load: Option<&'static str> = None;
            let events = match step_kind {
                TestStep::Receive { file, source } => {
                    let text = if *file {
                        let path = display(&self.eval(source, None));
                        if let Some(sample) =
                            program.samples.iter().find(|sample| sample.path == path)
                        {
                            sample.content.clone()
                        } else {
                            self.fail(
                                *span,
                                Message::with(
                                    message!("There is no sample {0}"),
                                    [Arg::from(path)],
                                ),
                            );
                            continue;
                        }
                    } else {
                        display(&self.eval(source, None))
                    };
                    if let Some(events) = self.receive(&flow, &text) {
                        events
                    } else {
                        self.fail(
                            *span,
                            Message::new(message!(
                                "No receive clause of the flow reads this message"
                            )),
                        );
                        continue;
                    }
                }
                TestStep::Respond { accept, reason } => vec![FlowEvent::Planner {
                    decision: if *accept {
                        "accept".into()
                    } else {
                        "decline".into()
                    },
                    reason: reason
                        .as_ref()
                        .map(|reason| display(&self.eval(reason, None)))
                        .unwrap_or_default(),
                }],
                TestStep::Event { kind, value } => {
                    let document = value
                        .as_ref()
                        .map_or(Value::None, |value| self.eval(value, None));
                    let name =
                        crate::lading::schema::event(&kind.name).map_or("", |event| event.document);
                    let given = self.given.clone();
                    let interp = Interp::new(program, &given, RUN_STEPS);
                    let document = interp.coerce(document, &Type::Record(name));
                    moves_load = load_state_after(&kind.name);
                    vec![FlowEvent::Trenova {
                        event: kind.name.clone(),
                        document: to_json(&document),
                    }]
                }
                TestStep::Ack(accepted) => {
                    let control = run
                        .instance()
                        .state
                        .pending
                        .first()
                        .map(|pending| pending.control.clone())
                        .unwrap_or_default();
                    vec![FlowEvent::Ack {
                        control,
                        accepted: *accepted,
                    }]
                }
                TestStep::Advance(duration) => {
                    if let Value::Duration(seconds) = self.eval(duration, None) {
                        self.given.now += seconds;
                    }
                    for index in 0..run.instances.len() {
                        run.current = index;
                        let mut fired = 0;
                        while run.instances[index]
                            .wake
                            .is_some_and(|at| at <= self.given.now)
                            && fired < 64
                        {
                            fired += 1;
                            self.feed(&flow, &mut run, &FlowEvent::Timer);
                        }
                    }
                    Vec::new()
                }
                TestStep::ExpectOutcome {
                    negated,
                    what,
                    set,
                    detail,
                } => {
                    let detail = detail.as_ref().map(|detail| match &detail.kind {
                        ExprKind::Name(word) => word.clone(),
                        _ => display(&self.eval(detail, None)),
                    });
                    let set = set.as_ref().map(|set| set.name.as_str());
                    let found = run
                        .actions
                        .iter()
                        .any(|action| flow_matches(action, &what.name, set, detail.as_deref()))
                        || run.unsent.iter().any(|unsent| {
                            unsent.why == what.name
                                && set.is_none_or(|wanted| wanted == unsent.set)
                                && detail
                                    .as_deref()
                                    .is_none_or(|detail| unsent.message.contains(detail))
                        });
                    if found == *negated {
                        let listed: Vec<String> = run
                            .actions
                            .iter()
                            .map(describe)
                            .chain(run.unsent.iter().map(|unsent| {
                                Message::with(
                                    message!("{0} not sent, {1}: {2}"),
                                    [
                                        Arg::from(unsent.set.as_str()),
                                        Arg::from(unsent.why),
                                        Arg::from(unsent.message.as_str()),
                                    ],
                                )
                                .english()
                            }))
                            .collect();
                        self.fail(
                            *span,
                            if *negated {
                                Message::with(
                                    message!("Expected the flow not to have {0}; it did: {1}"),
                                    [Arg::from(what.name.as_str()), Arg::from(listed.join("; "))],
                                )
                            } else {
                                Message::with(
                                    message!("Expected the flow to have {0}; it did: {1}"),
                                    [Arg::from(what.name.as_str()), Arg::from(listed.join("; "))],
                                )
                            },
                        );
                    }
                    Vec::new()
                }
                _ => Vec::new(),
            };
            for event in &events {
                run.select(route(program, &flow, event, self.given.now));
                self.feed(&flow, &mut run, event);
                let state = &mut run.instance().state;
                if let Some(next) = moves_load
                    && state.load.exists
                {
                    next.clone_into(&mut state.load.state);
                }
            }
        }
    }

    /// Takes one flow step, then writes each message it sends with its map, rules and the
    /// partner's guide as the host does: one that writes is reported sent with the next control
    /// number; one held or refused stays unsent, as the host leaves it; one that holds nothing
    /// the partner takes is reported as nothing sent.
    fn feed(&mut self, flow: &str, run: &mut FlowRun, event: &FlowEvent) {
        let program = self.program;
        let instance = run.instance();
        let result = step(
            program,
            flow,
            instance.state.clone(),
            event,
            self.given.now,
            &self.given,
        );
        self.coverage.extend(result.coverage.iter().copied());
        instance.state = result.state;
        instance.wake = result.wake;
        for action in result.actions {
            let FlowAction::Send {
                id,
                set,
                map,
                document,
            } = &action
            else {
                run.actions.push(action);
                continue;
            };
            let written = write_message(program, &self.given, map, document, true);
            self.coverage.extend(written.coverage.iter().copied());
            let control = if written.not_sent {
                run.unsent.push(Unsent {
                    set: set.clone(),
                    why: "not sent",
                    message: Message::new(message!("Nothing in it is sent to this partner"))
                        .english(),
                });
                Some(String::new())
            } else if let Some(error) = &written.error {
                run.unsent.push(Unsent {
                    set: set.clone(),
                    why: "rejected",
                    message: error.english(),
                });
                None
            } else if let Some(output) = written.output.as_ref().filter(|output| !sendable(output))
            {
                run.unsent.push(blocked(set, output));
                None
            } else {
                run.control_numbers += 1;
                Some(format!("{:04}", run.control_numbers))
            };
            let id = id.clone();
            if control
                .as_deref()
                .is_some_and(|control| !control.is_empty())
            {
                run.actions.push(action);
            }
            if let Some(control) = control {
                let instance = run.instance();
                let report = step(
                    program,
                    flow,
                    instance.state.clone(),
                    &FlowEvent::Sent { id, control },
                    self.given.now,
                    &self.given,
                );
                instance.state = report.state;
                instance.wake = report.wake;
            }
        }
    }

    /// A message received: each transaction of each set it holds, read by the receive clause's
    /// map and rules, in order.
    fn receive(&mut self, flow: &str, text: &str) -> Option<Vec<FlowEvent>> {
        let program = self.program;
        let unit = *program.flows.get(flow)?;
        let mut events = Vec::new();
        for set in transaction_sets(text) {
            let receive = program.units[unit]
                .items()
                .iter()
                .find_map(|item| match item {
                    Item::Flow(FlowDecl::Receive(receive)) if receive.set.name == set => {
                        Some(receive)
                    }
                    _ => None,
                })?;
            let map = receive.map.as_ref()?.name.clone();
            let info = program.maps.get(&map)?;
            let Source::X12(guide) = info.source else {
                return None;
            };
            for transaction in read_x12(program, guide, text).ok()? {
                let given = self.given.clone();
                let mut interp = Interp::new(program, &given, RUN_STEPS);
                let mut frame = Frame::new(info.unit);
                frame.root = Some(transaction.tree?);
                let (document, _) =
                    fill_value(&mut interp, &map, receive.rules.is_some(), info, frame).ok()?;
                self.coverage.extend(interp.coverage.iter().copied());
                events.push(FlowEvent::Received {
                    set: set.clone(),
                    document: to_json(&document),
                    outcomes: outcome_inputs(&interp.outcomes),
                    current: None,
                    load: None,
                });
            }
        }
        (!events.is_empty()).then_some(events)
    }
}

/// What a send the host would refuse says: held when only the rules held it, rejected when the
/// rules rejected it or the writer or the partner's guide found an error.
fn blocked(set: &str, output: &super::eval::run::WriteResult) -> Unsent {
    let outcome = |kind: OutcomeKind2| output.outcomes.iter().any(|found| found.kind == kind);
    let why = if outcome(OutcomeKind2::Hold) && !outcome(OutcomeKind2::Reject) {
        "held"
    } else {
        "rejected"
    };
    let message = output
        .outcomes
        .iter()
        .filter(|found| matches!(found.kind, OutcomeKind2::Reject | OutcomeKind2::Hold))
        .map(|found| found.message.clone())
        .chain(
            output
                .findings
                .iter()
                .filter(|finding| finding.severity == "error")
                .map(|finding| finding.message.english()),
        )
        .chain(
            output
                .validation
                .iter()
                .filter(|finding| finding.severity == Severity::Error)
                .map(|finding| finding.message.english()),
        )
        .collect::<Vec<_>>()
        .join("; ");
    Unsent {
        set: set.to_owned(),
        why,
        message,
    }
}

/// The state a test's load moves to on one of Trenova's events, as the shipment's would: in
/// production the host reports the shipment's state with every message.
fn load_state_after(event: &str) -> Option<&'static str> {
    match event {
        "Dispatched" => Some("dispatched"),
        "Delivered" => Some("delivered"),
        "InvoicePosted" => Some("invoiced"),
        _ => None,
    }
}

/// The sets of the transactions in an interchange, each once, in the order they first appear:
/// ST01, read with the separators ISA declares.
fn transaction_sets(text: &str) -> Vec<String> {
    let trimmed = text.trim_start();
    let element = trimmed.chars().nth(3).unwrap_or('*');
    let terminator = trimmed
        .char_indices()
        .filter(|(_, c)| *c == element)
        .nth(15)
        .and_then(|(at, _)| trimmed[at..].chars().nth(2))
        .unwrap_or('~');
    let mut sets: Vec<String> = Vec::new();
    for segment in trimmed.split(terminator).map(str::trim) {
        let mut elements = segment.split(element);
        if elements.next() == Some("ST") {
            let set = elements.next().unwrap_or_default().to_owned();
            if !sets.contains(&set) {
                sets.push(set);
            }
        }
    }
    sets
}

/// Whether a flow action is what a test expects.
fn flow_matches(action: &FlowAction, what: &str, set: Option<&str>, detail: Option<&str>) -> bool {
    let text_matches =
        |message: &Message| detail.is_none_or(|detail| message.english().contains(detail));
    match (what, action) {
        (
            "sent",
            FlowAction::Acknowledge {
                set: acknowledged, ..
            },
        ) => set.is_none_or(|wanted| wanted == acknowledged),
        (
            "sent",
            FlowAction::Send {
                set: written,
                document,
                ..
            },
        ) => {
            set.is_none_or(|wanted| wanted == written)
                && detail.is_none_or(|detail| {
                    document
                        .get("decision")
                        .and_then(serde_json::Value::as_str)
                        .is_some_and(|decision| decision.eq_ignore_ascii_case(detail))
                })
        }
        ("held", FlowAction::Hold { message })
        | ("escalated", FlowAction::Escalate { message }) => text_matches(message),
        ("created", FlowAction::Create { .. })
        | ("updated", FlowAction::Update { .. })
        | ("cancelled", FlowAction::Cancel)
        | ("applied", FlowAction::Apply { .. })
        | ("ignored", FlowAction::Ignore) => true,
        (
            "rejected",
            FlowAction::Acknowledge {
                rejected, message, ..
            },
        ) => *rejected && message.as_ref().map_or(detail.is_none(), text_matches),
        _ => false,
    }
}

fn describe(action: &FlowAction) -> String {
    match action {
        FlowAction::Acknowledge {
            set,
            rejected,
            message,
        } => format!(
            "acknowledge {set}{}{}",
            if *rejected { " rejected" } else { "" },
            message
                .as_ref()
                .map(|message| format!(" \"{}\"", message.english()))
                .unwrap_or_default()
        ),
        FlowAction::Create { .. } => "create".into(),
        FlowAction::Update { fields, .. } => format!("update {}", fields.join(", ")),
        FlowAction::Cancel => "cancel".into(),
        FlowAction::Apply { set, .. } => format!("apply {set}"),
        FlowAction::Hold { message } => format!("hold \"{}\"", message.english()),
        FlowAction::Ignore => "ignore".into(),
        FlowAction::AskPlanner { .. } => "ask a planner".into(),
        FlowAction::Send { set, .. } => format!("send {set}"),
        FlowAction::Resend { .. } => "resend".into(),
        FlowAction::Escalate { message } => format!("escalate \"{}\"", message.english()),
    }
}

fn shown(value: &Value) -> String {
    if value.is_none() {
        "none".to_owned()
    } else {
        display(value)
    }
}

/// Every statement, rule and flow clause a test can reach, and whether one did.
fn coverage(program: &Program, covered: &BTreeSet<(usize, u32)>) -> Vec<FileCoverage> {
    let mut files = Vec::new();
    for unit in 0..program.units.len() {
        let mut spans = Vec::new();
        for item in program.units[unit].items() {
            match item {
                Item::Stmt(stmt) => coverable(stmt, &mut spans),
                Item::Rule(rule) => {
                    spans.push(rule.span);
                    for stmt in &rule.body.stmts {
                        coverable(stmt, &mut spans);
                    }
                }
                Item::Flow(FlowDecl::Receive(receive)) => {
                    spans.push(receive.span);
                    spans.extend(receive.arms.iter().map(|arm| arm.span));
                }
                Item::Flow(FlowDecl::Respond(respond)) => spans.push(respond.span),
                Item::Flow(FlowDecl::Send(send)) => spans.push(send.span),
                Item::Flow(FlowDecl::Await(answer)) => spans.push(answer.span),
                _ => {}
            }
        }
        if spans.is_empty() {
            continue;
        }
        let (hit, missed): (Vec<Span>, Vec<Span>) = spans
            .into_iter()
            .partition(|span| covered.contains(&(unit, span.start)));
        files.push(FileCoverage {
            unit,
            covered: hit,
            uncovered: missed,
        });
    }
    files
}

fn coverable(stmt: &Stmt, spans: &mut Vec<Span>) {
    match stmt {
        Stmt::Assign { value, span, .. } => {
            spans.push(*span);
            if let ExprKind::Each { body, .. } = &value.kind {
                for inner in &body.stmts {
                    coverable(inner, spans);
                }
            }
        }
        Stmt::When { body, span, .. }
        | Stmt::Loop { body, span, .. }
        | Stmt::Each { body, span, .. }
        | Stmt::For { body, span, .. } => {
            spans.push(*span);
            for inner in &body.stmts {
                coverable(inner, spans);
            }
        }
        Stmt::Segment { span, .. }
        | Stmt::Require { span, .. }
        | Stmt::OutcomeWhen { span, .. }
        | Stmt::Fix { span, .. } => spans.push(*span),
        Stmt::Let(_) | Stmt::Error(_) => {}
    }
}

/// Data the X12 samples hold that no map reads.
fn unread_data(program: &Program) -> Vec<UnreadData> {
    let mut found: BTreeMap<(String, String, String), (usize, String, String)> = BTreeMap::new();
    let mut totals: BTreeMap<String, usize> = BTreeMap::new();
    let given = Given {
        now: DEFAULT_NOW,
        stand_in: true,
        ..Given::default()
    };
    for (name, info) in &program.maps {
        let Source::X12(guide) = info.source else {
            continue;
        };
        for sample in &program.samples {
            let Ok(transactions) = read_x12(program, guide, &sample.content) else {
                continue;
            };
            let Some(tree) = transactions
                .into_iter()
                .find_map(|transaction| transaction.tree)
            else {
                continue;
            };
            let mut interp = Interp::new(program, &given, RUN_STEPS);
            let mut frame = Frame::new(info.unit);
            frame.root = Some(Rc::clone(&tree));
            if fill_value(&mut interp, name, false, info, frame).is_err() {
                continue;
            }
            *totals.entry(name.clone()).or_default() += 1;
            let mut seen = BTreeSet::new();
            for (segment_index, position) in unread(&tree) {
                let segment = &tree.segments[segment_index];
                let marker = qualifier_of(&tree, segment_index);
                if marker.as_ref().is_some_and(|(at, _)| *at == position) {
                    continue;
                }
                let (qualifier_at, qualifier) = marker.unwrap_or((0, String::new()));
                let element = format!("{}{position:02}", segment.id);
                let key = (name.clone(), element.clone(), qualifier.clone());
                if !seen.insert(key.clone()) {
                    continue;
                }
                let read = if segment.instance == 0 {
                    if qualifier.is_empty() {
                        element.clone()
                    } else {
                        format!(
                            "{}[{qualifier_at:02} == \"{qualifier}\"].{position:02}",
                            segment.id
                        )
                    }
                } else {
                    String::new()
                };
                let entry =
                    found
                        .entry(key)
                        .or_insert((0, format!("{}{position:02}", segment.path), read));
                entry.0 += 1;
            }
        }
    }
    let mut items: Vec<UnreadData> = found
        .into_iter()
        .map(
            |((map, element, qualifier), (samples, example, read))| UnreadData {
                total: totals.get(&map).copied().unwrap_or(0),
                map,
                element,
                qualifier,
                samples,
                example,
                read,
            },
        )
        .collect();
    items.sort_by(|a, b| {
        b.samples
            .cmp(&a.samples)
            .then_with(|| a.element.cmp(&b.element))
    });
    items
}

/// The qualifier of a segment, when one of its elements is one (`PO` of `N9*PO`, `BM` of
/// `L11*123*BM`): its position and value.
fn qualifier_of(tree: &super::eval::tree::Tree, index: usize) -> Option<(u32, String)> {
    let segment = &tree.segments[index];
    let definition = tree.compiled.definition(segment.node, &segment.id)?;
    let (offset, _) = definition
        .elements
        .iter()
        .enumerate()
        .take(3)
        .find(|(_, element)| {
            element.kind == crate::x12::guide::model::ElementType::Identifier
                && (element.name.contains("Qualifier") || element.name == "Entity Identifier Code")
        })?;
    let value = segment.elements.get(offset)?.clone();
    (!value.is_empty()).then(|| (u32::try_from(offset + 1).unwrap_or(0), value))
}

/// Reads a test's clock from an instant literal.
#[must_use]
pub fn instant(text: &str) -> Option<i64> {
    parse_instant(text).map(|(at, _)| at)
}

/// The JSON of a test's document input, for the studio.
#[must_use]
pub fn document_value(
    program: &Program,
    json: &serde_json::Value,
    document: &'static str,
) -> Option<Value> {
    from_json(program, json, &Type::Record(document), "").ok()
}

#[cfg(test)]
mod tests {
    use super::run_tests;
    use crate::lading::check::check;
    use crate::lading::check::corpus::{acme, context};

    #[test]
    fn the_acme_tests_pass() {
        let project = acme();
        let program = check(&project, &context());
        let report = run_tests(&program);
        let failures: Vec<String> = report
            .tests
            .iter()
            .filter(|test| !test.passed)
            .map(|test| {
                format!(
                    "{} / {}: {:?}",
                    program.units[test.unit].path,
                    test.name,
                    test.failures
                        .iter()
                        .map(|failure| failure.message.english())
                        .collect::<Vec<_>>()
                )
            })
            .collect();
        assert!(failures.is_empty(), "{}", failures.join("\n"));
        assert!(report.tests.len() >= 8, "{}", report.tests.len());
        let property = report
            .tests
            .iter()
            .find(|test| test.kind == "property" && test.name.contains("reservation"))
            .expect("property");
        assert_eq!(property.cases, 2);
        assert!(report.coverage.iter().any(|file| !file.covered.is_empty()));
        assert!(
            report
                .unread
                .iter()
                .any(|item| item.element == "B206" && item.samples == 3),
            "{:?}",
            report.unread
        );
        assert!(
            report
                .unread
                .iter()
                .any(|item| item.element == "G6203" && item.qualifier == "69"),
            "{:?}",
            report.unread
        );
    }
}
