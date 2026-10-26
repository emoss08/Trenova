//! Evaluates expressions: names, X12 paths, literals, records, operators, crosswalks, lookups
//! and calls, within a budget of steps, recording lineage and what lookups a run still needs.

use std::collections::{BTreeMap, BTreeSet};
use std::rc::Rc;

use rust_decimal::Decimal;
use serde::Serialize;

use super::tree::{Node, Tree};
use super::value::{Reference, Unit, Value, compare, convert, display, equal};
use crate::lading::check::x12::{Reference as X12Ref, reference};
use crate::lading::check::{Decl, Program};
use crate::lading::schema;
use crate::lading::syntax::ast::{
    BinOp, Direction, Else, Expr, ExprKind, FieldInit, Item, MemberKind, OutcomeKind, TextPart,
    UnOp,
};
use crate::lading::types::{Dim, Type};
use crate::lading::values::{ZoneValue, parse_date, parse_instant, parse_time};
use crate::x12::guide::compiled::NodeKind;

/// Why evaluation stopped.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Halt {
    /// The budget of steps ran out.
    Budget,
}

/// The result of evaluating.
pub type Eval<T = Value> = Result<T, Halt>;

/// Where a value came from: an X12 element, a CSV cell, a JSON path or a document field.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Source {
    /// How it is named: `0300[2].0310.N401`, `rows[3].pickup`, `events[0].at`.
    pub path: String,
    /// The segment's ID, for an X12 element.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub segment: String,
    /// The segment's position in the transaction set, ST being 1.
    #[serde(skip_serializing_if = "is_zero")]
    pub position: u32,
    /// The element's position.
    #[serde(skip_serializing_if = "is_zero")]
    pub element: u32,
}

#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_zero(value: &u32) -> bool {
    *value == 0
}

/// Where one value of the output came from.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Lineage {
    /// The output's path: `stops[1].address.city`, or `0205[1].AT701`.
    pub path: String,
    /// What it was computed from.
    pub sources: Vec<Source>,
    /// The functions, crosswalks and fixes it passed through, in order.
    pub steps: Vec<String>,
    /// The value, as text.
    pub value: String,
}

#[derive(Debug, Default)]
struct TraceFrame {
    sources: Vec<Source>,
    steps: Vec<String>,
}

/// A lookup the host must answer.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Need {
    /// The lookup kind.
    pub kind: String,
    /// The partner's code.
    pub code: String,
}

/// A lookup the host answered with no mapping.
#[derive(Debug, Clone, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Unmapped {
    /// The lookup kind.
    pub kind: String,
    /// The partner's code.
    pub code: String,
    /// What the map gave to suggest a match from.
    #[serde(skip_serializing_if = "serde_json::Value::is_null")]
    pub hint: serde_json::Value,
    /// Where in the output it was asked for.
    pub path: String,
}

/// What an outcome is.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub enum OutcomeKind2 {
    /// The message is rejected.
    Reject,
    /// The message is declined.
    Decline,
    /// The document is held.
    Hold,
    /// A note.
    Warn,
}

/// One outcome a map's `else` or a rule produced.
#[derive(Debug, Clone, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Outcome {
    /// Which.
    pub kind: OutcomeKind2,
    /// The message.
    pub message: String,
    /// The rule's title, for a rule.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub rule: String,
    /// The document path it is about.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub path: String,
    /// The X12 element that path was read from.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub element: Option<Source>,
    /// A decline's code.
    #[serde(skip_serializing_if = "String::is_empty")]
    pub code: String,
    /// The file and the byte offset of the statement.
    #[serde(skip)]
    pub at: (usize, u32),
}

/// What a run is given besides its input.
#[derive(Debug, Clone, Default)]
pub struct Given {
    /// Setting values that differ from the partner file's defaults.
    pub settings: BTreeMap<String, Value>,
    /// Lookup answers by kind and code; none for a code with no mapping.
    pub lookups: BTreeMap<(String, String), Option<Reference>>,
    /// The time the run started.
    pub now: i64,
}

/// One scope's names.
type Scope = Vec<(String, Value)>;

/// What code being evaluated can name.
#[derive(Debug, Clone, Default)]
pub struct Frame {
    /// The file.
    pub unit: usize,
    /// Its scopes, innermost last.
    pub scopes: Vec<Scope>,
    /// The document whose fields are names.
    pub document: Option<Value>,
    /// The transaction bare X12 references read.
    pub root: Option<Rc<Tree>>,
    /// The segment or loop a `[cond]` is about.
    pub filter: Option<Value>,
    /// The output path assignments are made under (`stops[1]`).
    pub path: String,
    /// The document path a name was read from, for `for` items.
    pub origins: Vec<(String, String)>,
}

impl Frame {
    /// A frame for a file.
    #[must_use]
    pub fn new(unit: usize) -> Self {
        Self {
            unit,
            scopes: vec![Vec::new()],
            ..Self::default()
        }
    }

    /// Opens a scope.
    pub fn push(&mut self) {
        self.scopes.push(Vec::new());
    }

    /// Closes a scope.
    pub fn pop(&mut self) {
        self.scopes.pop();
        if self.scopes.is_empty() {
            self.scopes.push(Vec::new());
        }
    }

    /// Binds a name.
    pub fn bind(&mut self, name: &str, value: Value) {
        if let Some(scope) = self.scopes.last_mut() {
            scope.push((name.to_owned(), value));
        }
    }

    /// The innermost binding of a name.
    #[must_use]
    pub fn lookup(&self, name: &str) -> Option<&Value> {
        self.scopes
            .iter()
            .rev()
            .flat_map(|scope| scope.iter().rev())
            .find(|(bound, _)| bound == name)
            .map(|(_, value)| value)
    }

    /// The document path a name was bound from.
    #[must_use]
    pub fn origin(&self, name: &str) -> Option<&str> {
        self.origins
            .iter()
            .rev()
            .find(|(bound, _)| bound == name)
            .map(|(_, path)| path.as_str())
    }
}

/// The evaluator's state for one run.
#[derive(Debug)]
pub struct Interp<'p> {
    /// The program.
    pub program: &'p Program,
    /// What the run was given.
    pub given: &'p Given,
    steps: u64,
    limit: u64,
    /// Lookups the host has not answered.
    pub needs: BTreeSet<Need>,
    /// Lookups the host answered with no mapping.
    pub unmapped: Vec<Unmapped>,
    /// What maps' elses and rules decided.
    pub outcomes: Vec<Outcome>,
    /// Statements that ran: file and byte offset.
    pub coverage: BTreeSet<(usize, u32)>,
    /// Where each output value came from.
    pub lineage: Vec<Lineage>,
    trace: Vec<TraceFrame>,
    consts: BTreeMap<(usize, String), Value>,
    partner: Value,
    org: Value,
}

/// The default step budget of a map run.
pub const RUN_STEPS: u64 = 10_000_000;

impl<'p> Interp<'p> {
    /// An evaluator for a program, with a step budget.
    #[must_use]
    pub fn new(program: &'p Program, given: &'p Given, limit: u64) -> Self {
        let mut interp = Self {
            program,
            given,
            steps: 0,
            limit,
            needs: BTreeSet::new(),
            unmapped: Vec::new(),
            outcomes: Vec::new(),
            coverage: BTreeSet::new(),
            lineage: Vec::new(),
            trace: Vec::new(),
            consts: BTreeMap::new(),
            partner: Value::None,
            org: Value::None,
        };
        interp.partner = interp.partner_value();
        interp.org = interp.org_value();
        interp
    }

    /// Steps used.
    #[must_use]
    pub fn steps(&self) -> u64 {
        self.steps
    }

    fn tick(&mut self) -> Eval<()> {
        self.steps += 1;
        if self.steps > self.limit {
            Err(Halt::Budget)
        } else {
            Ok(())
        }
    }

    fn partner_value(&mut self) -> Value {
        let program = self.program;
        let mut fields = BTreeMap::new();
        if let Some(partner) = &program.partner {
            fields.insert("id".into(), Value::text(&partner.name));
            fields.insert("name".into(), Value::text(&partner.label));
            fields.insert("code".into(), Value::text(&partner.name));
            fields.insert(
                "zone".into(),
                Value::Zone(
                    partner
                        .zone
                        .as_deref()
                        .and_then(ZoneValue::parse)
                        .unwrap_or(ZoneValue::UTC),
                ),
            );
            fields.insert(
                "currency".into(),
                Value::text(partner.currency.as_deref().unwrap_or("USD")),
            );
            for setting in &partner.settings {
                let value = match self.given.settings.get(&setting.name) {
                    Some(value) => value.clone(),
                    None => match program.units[setting.unit].items().get(setting.item) {
                        Some(Item::Setting(decl)) => match &decl.default {
                            Some(default) => {
                                let mut frame = Frame::new(setting.unit);
                                self.eval(&mut frame, default).unwrap_or(Value::None)
                            }
                            None => Value::None,
                        },
                        _ => Value::None,
                    },
                };
                fields.insert(setting.name.clone(), value);
            }
        }
        Value::Struct(Rc::new(fields))
    }

    fn org_value(&self) -> Value {
        let org = &self.program.context.org;
        let mut fields = BTreeMap::new();
        fields.insert("name".into(), Value::text(&org.name));
        fields.insert("scac".into(), Value::text(&org.scac));
        fields.insert(
            "zone".into(),
            Value::Zone(ZoneValue::parse(&org.zone).unwrap_or(ZoneValue::UTC)),
        );
        Value::Struct(Rc::new(fields))
    }

    /// The partner, as code reads it.
    #[must_use]
    pub fn partner(&self) -> &Value {
        &self.partner
    }

    /// Starts collecting where a value comes from.
    pub fn begin_trace(&mut self) {
        self.trace.push(TraceFrame::default());
    }

    /// Ends collecting, recording the lineage of `path`.
    pub fn end_trace(&mut self, path: &str, value: &Value) {
        let frame = self.trace.pop().unwrap_or_default();
        if value.is_none() || path.is_empty() {
            return;
        }
        let mut sources = frame.sources;
        sources.sort();
        sources.dedup();
        self.lineage.push(Lineage {
            path: path.to_owned(),
            sources,
            steps: frame.steps,
            value: display(value),
        });
    }

    /// Records a source of the value being computed.
    pub fn note_source(&mut self, source: Source) {
        if let Some(frame) = self.trace.last_mut() {
            frame.sources.push(source);
        }
    }

    /// Records a step the value being computed passed through.
    pub fn note_step(&mut self, step: &str) {
        if let Some(frame) = self.trace.last_mut()
            && frame.steps.last().is_none_or(|last| last != step)
        {
            frame.steps.push(step.to_owned());
        }
    }

    /// The first X12 element an output path was read from.
    #[must_use]
    pub fn element_of(&self, path: &str) -> Option<Source> {
        self.lineage
            .iter()
            .rev()
            .find(|entry| entry.path == path)
            .and_then(|entry| {
                entry
                    .sources
                    .iter()
                    .find(|source| source.element > 0)
                    .cloned()
            })
    }

    /// Marks a statement as run.
    pub fn cover(&mut self, unit: usize, offset: u32) {
        self.coverage.insert((unit, offset));
    }

    /// Records an outcome.
    pub fn outcome(
        &mut self,
        frame: &mut Frame,
        outcome: &crate::lading::syntax::ast::Outcome,
        rule: &str,
        path: &str,
    ) -> Eval<()> {
        let message = self.eval(frame, &outcome.message)?;
        let (kind, code, at_path) = match &outcome.kind {
            OutcomeKind::Reject(at) => {
                let at_path = at
                    .as_ref()
                    .and_then(|at| Self::doc_path(frame, at))
                    .unwrap_or_else(|| path.to_owned());
                (OutcomeKind2::Reject, String::new(), at_path)
            }
            OutcomeKind::Decline(code) => {
                let code = match code {
                    Some(code) => display(&self.eval(frame, code)?),
                    None => String::new(),
                };
                (OutcomeKind2::Decline, code, path.to_owned())
            }
            OutcomeKind::Hold => (OutcomeKind2::Hold, String::new(), path.to_owned()),
            OutcomeKind::Warn => (OutcomeKind2::Warn, String::new(), path.to_owned()),
        };
        let element = if kind == OutcomeKind2::Reject {
            self.element_of(&at_path)
        } else {
            None
        };
        self.outcomes.push(Outcome {
            kind,
            message: display(&message),
            rule: rule.to_owned(),
            path: at_path,
            element,
            code,
            at: (frame.unit, outcome.span.start),
        });
        Ok(())
    }

    /// The document path an expression names: `stop.address.postal` inside `for stop in stops`
    /// is `stops[1].address.postal`.
    #[must_use]
    pub fn doc_path(frame: &Frame, expr: &Expr) -> Option<String> {
        match &expr.kind {
            ExprKind::Name(name) => frame.origin(name).map(str::to_owned).or_else(|| {
                let document = frame.document.as_ref()?;
                matches!(document, Value::Record(..)).then(|| join_path(&frame.path, name))
            }),
            ExprKind::Member { target, member, .. } => {
                let base = Self::doc_path(frame, target)?;
                Some(join_path(&base, &member.name))
            }
            _ => None,
        }
    }

    /// Evaluates an expression.
    #[allow(clippy::too_many_lines)]
    pub fn eval(&mut self, frame: &mut Frame, expr: &Expr) -> Eval {
        self.tick()?;
        Ok(match &expr.kind {
            ExprKind::Error | ExprKind::Lambda { .. } | ExprKind::Each { .. } | ExprKind::None => {
                Value::None
            }
            ExprKind::Name(name) => self.name(frame, name)?,
            ExprKind::Type(variant) => {
                schema::enum_of_variant(variant).map_or(Value::None, |owner| {
                    let name = owner
                        .variants
                        .iter()
                        .find(|(candidate, _)| candidate == variant)
                        .map_or("", |(name, _)| *name);
                    Value::Enum(owner.name, name)
                })
            }
            ExprKind::X12(text) => {
                let (tree, instance) = match &frame.filter {
                    Some(Value::Loop(tree, instance)) => (Rc::clone(tree), *instance),
                    _ => match &frame.root {
                        Some(tree) => (Rc::clone(tree), 0),
                        None => return Ok(Value::None),
                    },
                };
                self.x12(&tree, instance, text)
            }
            ExprKind::Digits(text) => match frame.filter.clone() {
                Some(Value::Loop(tree, instance)) => loops(&tree, instance, text),
                Some(Value::Segment(tree, index)) => {
                    let position = text.parse().unwrap_or(0);
                    self.read_element(&tree, index, position)
                }
                Some(Value::Struct(fields)) => {
                    fields.get(text.as_str()).cloned().unwrap_or(Value::None)
                }
                _ => match &frame.root {
                    Some(tree) => loops(tree, 0, text),
                    None => Value::None,
                },
            },
            ExprKind::Number(text) => {
                if text.contains('.') {
                    text.parse().map_or(Value::None, Value::Decimal)
                } else {
                    text.parse().map_or_else(
                        |_| text.parse().map_or(Value::None, Value::Decimal),
                        Value::Int,
                    )
                }
            }
            ExprKind::Quantity(number, unit) => {
                let amount: Decimal = number.parse().unwrap_or_default();
                match unit.as_str() {
                    "s" | "min" | "h" | "d" => {
                        let seconds = match unit.as_str() {
                            "s" => 1,
                            "min" => 60,
                            "h" => 3600,
                            _ => 86_400,
                        };
                        let total = (amount * Decimal::from(seconds)).round();
                        Value::Duration(i64::try_from(total).unwrap_or(0))
                    }
                    _ => Unit::named(unit).map_or(Value::None, |unit| Value::Measure(amount, unit)),
                }
            }
            ExprKind::Percent(number) => number.parse().map_or(Value::None, Value::Percent),
            ExprKind::Money(number) => number
                .parse()
                .map_or(Value::None, |amount| Value::Money(amount, Rc::from("USD"))),
            ExprKind::Text(parts) => {
                let mut text = String::new();
                for part in parts {
                    match part {
                        TextPart::Literal(literal) => text.push_str(literal),
                        TextPart::Expr(inner) => {
                            let value = self.eval(frame, inner)?;
                            text.push_str(&display(&value));
                        }
                    }
                }
                Value::text(text)
            }
            ExprKind::Raw(text) => Value::text(text),
            ExprKind::Date(text) => parse_date(text).map_or(Value::None, Value::Date),
            ExprKind::Time(text) => parse_time(text).map_or(Value::None, Value::Time),
            ExprKind::Instant(text) => {
                parse_instant(text).map_or(Value::None, |(at, zone)| Value::DateTime(at, zone))
            }
            ExprKind::Bool(value) => Value::Bool(*value),
            ExprKind::List(items) => {
                let mut values = Vec::with_capacity(items.len());
                for item in items {
                    values.push(self.eval(frame, item)?);
                }
                Value::list(values)
            }
            ExprKind::Record { ty, fields } => {
                self.record(frame, ty.as_ref().map(|ident| ident.name.as_str()), fields)?
            }
            ExprKind::Member {
                target,
                member,
                kind,
            } => {
                let value = self.eval(frame, target)?;
                self.member(frame, &value, &member.name, *kind)?
            }
            ExprKind::Index { target, index } => {
                let value = self.eval(frame, target)?;
                self.index(frame, &value, index)?
            }
            ExprKind::Call { callee, args } => self.call(frame, callee, args)?,
            ExprKind::Unary(op, inner) => {
                let value = self.eval(frame, inner)?;
                match (op, value) {
                    (UnOp::Not, Value::Bool(value)) => Value::Bool(!value),
                    (UnOp::Not, Value::None) => Value::Bool(true),
                    (UnOp::Neg, value) => negate(&value),
                    _ => Value::None,
                }
            }
            ExprKind::Binary(op, lhs, rhs) => self.binary(frame, *op, lhs, rhs)?,
            ExprKind::Via {
                value,
                crosswalk,
                back,
            } => {
                let value = self.eval(frame, value)?;
                self.via(frame, &value, &crosswalk.name, *back)?
            }
            ExprKind::Exists(inner) => {
                let value = self.eval(frame, inner)?;
                Value::Bool(!value.is_none())
            }
            ExprKind::Lookup { kind, code, learn } => {
                let code_value = self.eval(frame, code)?;
                let hint = match learn {
                    Some(learn) => self.eval(frame, learn)?,
                    None => Value::None,
                };
                self.lookup(frame, &kind.name, &code_value, &hint)
            }
            ExprKind::Match {
                value,
                arms,
                otherwise,
            } => {
                let value = self.eval(frame, value)?;
                for arm in arms {
                    for pattern in &arm.patterns {
                        let pattern_value = self.eval(frame, pattern)?;
                        if equal(&value, &pattern_value) {
                            return self.eval(frame, &arm.value);
                        }
                    }
                }
                match otherwise {
                    Some(otherwise) => self.eval(frame, otherwise)?,
                    None => Value::None,
                }
            }
            ExprKind::If {
                cond,
                then,
                otherwise,
            } => {
                if self.eval(frame, cond)?.truthy() {
                    self.eval(frame, then)?
                } else {
                    self.eval(frame, otherwise)?
                }
            }
        })
    }

    fn name(&mut self, frame: &mut Frame, name: &str) -> Eval {
        if let Some(value) = frame.lookup(name) {
            return Ok(value.clone());
        }
        match name {
            "partner" => return Ok(self.partner.clone()),
            "org" => return Ok(self.org.clone()),
            _ => {}
        }
        if let Some(document) = &frame.document
            && let Value::Record(record, fields) = document
            && schema::record(record)
                .and_then(|owner| owner.field(name))
                .is_some()
        {
            let value = fields.get(name).cloned().unwrap_or(Value::None);
            if let Some(path) = Some(join_path(&frame.path, name)) {
                self.note_source(Source {
                    path,
                    segment: String::new(),
                    position: 0,
                    element: 0,
                });
            }
            return Ok(read_default(record, name, value));
        }
        if let Some((unit, Decl::Const(index))) = self.program.decl(frame.unit, name) {
            let key = (unit, name.to_owned());
            if let Some(value) = self.consts.get(&key) {
                return Ok(value.clone());
            }
            let Some(Item::Const(decl)) = self.program.units[unit].items().get(index) else {
                return Ok(Value::None);
            };
            let mut const_frame = Frame::new(unit);
            let value = self.eval(&mut const_frame, &decl.value)?;
            self.consts.insert(key, value.clone());
            return Ok(value);
        }
        Ok(Value::None)
    }

    /// A segment or element at a node of a transaction.
    fn x12(&mut self, tree: &Rc<Tree>, instance: Node, text: &str) -> Value {
        match reference(text) {
            Some(X12Ref::Segment(id)) => segments(tree, instance, id),
            Some(X12Ref::Element(id, position)) => match segments(tree, instance, id) {
                Value::Segment(_, index) => self.read_element(tree, index, position),
                Value::List(items) => match items.first() {
                    Some(Value::Segment(_, index)) => self.read_element(tree, *index, position),
                    _ => Value::None,
                },
                _ => Value::None,
            },
            None => Value::None,
        }
    }

    /// An element of a segment, recorded as a source.
    pub fn read_element(&mut self, tree: &Rc<Tree>, index: usize, position: u32) -> Value {
        let value = tree.element(index, position);
        if !value.is_none() {
            let segment = &tree.segments[index];
            self.note_source(Source {
                path: format!("{}{position:02}", segment.path),
                segment: segment.id.clone(),
                position: segment.position,
                element: position,
            });
        }
        value
    }

    #[allow(clippy::too_many_lines)]
    fn member(&mut self, frame: &mut Frame, value: &Value, name: &str, kind: MemberKind) -> Eval {
        Ok(match (value, kind) {
            (Value::None, _) => Value::None,
            (Value::Transaction(tree), MemberKind::X12) => self.x12(tree, 0, name),
            (Value::Loop(tree, instance), MemberKind::X12) => self.x12(tree, *instance, name),
            (Value::Transaction(tree), MemberKind::Digits) => loops(tree, 0, name),
            (Value::Loop(tree, instance), MemberKind::Digits) => loops(tree, *instance, name),
            (Value::Segment(tree, index), MemberKind::Digits) => {
                let position = name.parse().unwrap_or(0);
                self.read_element(tree, *index, position)
            }
            (Value::Struct(fields), MemberKind::Digits) => {
                fields.get(name).cloned().unwrap_or(Value::None)
            }
            (Value::Record(record, fields), MemberKind::Name) => {
                if schema::record(record)
                    .and_then(|owner| owner.field(name))
                    .is_some()
                {
                    read_default(
                        record,
                        name,
                        fields.get(name).cloned().unwrap_or(Value::None),
                    )
                } else {
                    self.property(frame, value, name)?
                }
            }
            (Value::Struct(fields) | Value::Map(fields), MemberKind::Name) => {
                match fields.get(name) {
                    Some(found) => found.clone(),
                    None => self.property(frame, value, name)?,
                }
            }
            (Value::Ref(reference), MemberKind::Name) => match name {
                "id" => Value::text(&reference.id),
                "code" => Value::text(&reference.code),
                "label" => Value::text(&reference.label),
                "zone" => reference.zone.map_or(Value::None, Value::Zone),
                _ => self.property(frame, value, name)?,
            },
            (Value::Json(json), MemberKind::Name) => match json.get(name) {
                Some(found) if !found.is_null() => Value::Json(Rc::new(found.clone())),
                _ => Value::None,
            },
            (_, MemberKind::Name) => self.property(frame, value, name)?,
            _ => Value::None,
        })
    }

    /// A zero-argument standard function used as a property.
    fn property(&mut self, frame: &mut Frame, value: &Value, name: &str) -> Eval {
        super::stdlib::call(
            self,
            frame,
            name,
            &[super::stdlib::Input::Value(value.clone())],
        )
    }

    fn index(&mut self, frame: &mut Frame, value: &Value, index: &Expr) -> Eval {
        match value {
            Value::List(items) => {
                let Some(first) = items.first() else {
                    return Ok(Value::None);
                };
                let saved = frame.filter.replace(first.clone());
                let probe = self.eval(frame, index);
                frame.filter = saved;
                match probe? {
                    Value::Int(position) => Ok(usize::try_from(position)
                        .ok()
                        .and_then(|position| items.get(position))
                        .cloned()
                        .unwrap_or(Value::None)),
                    Value::Bool(_) | Value::None => {
                        for item in items.iter() {
                            let saved = frame.filter.replace(item.clone());
                            let matched = self.eval(frame, index);
                            frame.filter = saved;
                            if matched?.truthy() {
                                return Ok(item.clone());
                            }
                        }
                        Ok(Value::None)
                    }
                    _ => Ok(Value::None),
                }
            }
            Value::Map(fields) => {
                let key = self.eval(frame, index)?;
                Ok(fields.get(&display(&key)).cloned().unwrap_or(Value::None))
            }
            Value::Segment(..) | Value::Loop(..) => {
                let saved = frame.filter.replace(value.clone());
                let matched = self.eval(frame, index);
                frame.filter = saved;
                Ok(if matched?.truthy() {
                    value.clone()
                } else {
                    Value::None
                })
            }
            Value::Json(json) => {
                let key = self.eval(frame, index)?;
                let found = match &key {
                    Value::Int(position) => usize::try_from(*position)
                        .ok()
                        .and_then(|position| json.get(position)),
                    other => json.get(display(other)),
                };
                Ok(found
                    .filter(|found| !found.is_null())
                    .map_or(Value::None, |found| Value::Json(Rc::new(found.clone()))))
            }
            _ => Ok(Value::None),
        }
    }

    /// A record literal; none when the type's required fields are not all there.
    fn record(&mut self, frame: &mut Frame, ty: Option<&str>, fields: &[FieldInit]) -> Eval {
        let mut values = BTreeMap::new();
        for field in fields {
            let mut value = self.eval(frame, &field.value)?;
            if value.is_none() {
                match &field.otherwise {
                    Some(Else::Value(default)) => value = self.eval(frame, default)?,
                    Some(Else::Outcome(outcome)) => {
                        let path = join_path(&frame.path, &field.name.name);
                        self.outcome(frame, outcome, "", &path)?;
                    }
                    None => {}
                }
            }
            values.insert(field.name.name.clone(), value);
        }
        let Some(record) = ty.and_then(schema::record) else {
            return Ok(Value::Struct(Rc::new(values)));
        };
        Ok(self.as_record(record.name, values))
    }

    /// Fields as a schema record: converted to the fields' types, none when a required field
    /// is none.
    #[must_use]
    pub fn as_record(&self, name: &'static str, values: BTreeMap<String, Value>) -> Value {
        let Some(record) = schema::record(name) else {
            return Value::Struct(Rc::new(values));
        };
        let mut fields = BTreeMap::new();
        for (key, value) in values {
            let converted = match record.field(&key) {
                Some(field) => self.coerce(value, &self.program.field_type(field)),
                None => value,
            };
            if !converted.is_none() {
                fields.insert(key, converted);
            }
        }
        if record
            .fields
            .iter()
            .any(|field| field.required && !fields.contains_key(field.name))
        {
            return Value::None;
        }
        Value::Record(record.name, Rc::new(fields))
    }

    /// A value made to fit a type: written-in-place records become schema records, whole
    /// numbers decimals, and so through lists.
    #[must_use]
    pub fn coerce(&self, value: Value, ty: &Type) -> Value {
        match (value, ty) {
            (value, Type::Optional(inner)) => self.coerce(value, inner),
            (Value::Struct(fields), Type::Record(name)) => {
                self.as_record(name, fields.as_ref().clone())
            }
            (Value::Int(number), Type::Decimal) => Value::Decimal(Decimal::from(number)),
            (Value::List(items), Type::List(item)) => Value::list(
                items
                    .iter()
                    .map(|value| self.coerce(value.clone(), item))
                    .filter(|value| !value.is_none())
                    .collect(),
            ),
            (Value::Struct(fields), Type::Map(_)) => Value::Map(fields),
            (value, _) => value,
        }
    }

    #[allow(clippy::too_many_lines)]
    fn binary(&mut self, frame: &mut Frame, op: BinOp, lhs: &Expr, rhs: &Expr) -> Eval {
        match op {
            BinOp::And => {
                let left = self.eval(frame, lhs)?;
                if !left.truthy() {
                    return Ok(Value::Bool(false));
                }
                Ok(Value::Bool(self.eval(frame, rhs)?.truthy()))
            }
            BinOp::Or => {
                let left = self.eval(frame, lhs)?;
                if left.truthy() {
                    return Ok(Value::Bool(true));
                }
                Ok(Value::Bool(self.eval(frame, rhs)?.truthy()))
            }
            BinOp::Default => {
                let left = self.eval(frame, lhs)?;
                if left.is_none() {
                    self.eval(frame, rhs)
                } else {
                    Ok(left)
                }
            }
            _ => {
                let left = self.eval(frame, lhs)?;
                let right = self.eval(frame, rhs)?;
                Ok(match op {
                    BinOp::Eq => Value::Bool(equal(&left, &right)),
                    BinOp::Ne => Value::Bool(!equal(&left, &right)),
                    BinOp::Lt | BinOp::Le | BinOp::Gt | BinOp::Ge => {
                        if left.is_none() || right.is_none() {
                            return Ok(Value::Bool(false));
                        }
                        let Some(order) = compare(&left, &right) else {
                            return Ok(Value::Bool(false));
                        };
                        Value::Bool(match op {
                            BinOp::Lt => order.is_lt(),
                            BinOp::Le => order.is_le(),
                            BinOp::Gt => order.is_gt(),
                            _ => order.is_ge(),
                        })
                    }
                    BinOp::In => Value::Bool(right.items().iter().any(|item| equal(&left, item))),
                    _ => arithmetic(op, &left, &right),
                })
            }
        }
    }

    /// `value via crosswalk`.
    fn via(&mut self, frame: &mut Frame, value: &Value, name: &str, back: bool) -> Eval {
        let program = self.program;
        let Some((unit, Decl::Crosswalk(index))) = program.decl(frame.unit, name) else {
            return Ok(Value::None);
        };
        let Some(Item::Crosswalk(decl)) = program.units[unit].items().get(index) else {
            return Ok(Value::None);
        };
        if value.is_none() {
            return Ok(Value::None);
        }
        self.note_step(name);
        let (left_ty, _) = program
            .crosswalks
            .get(&(unit, name.to_owned()))
            .cloned()
            .unwrap_or((Type::Any, Type::Any));
        let forward = !back && fits_value(value, &left_ty);
        let mut crosswalk_frame = Frame::new(unit);
        for pair in &decl.pairs {
            let usable = match pair.direction {
                Direction::Both => true,
                Direction::Forward => forward,
                Direction::Backward => !forward,
            };
            if !usable {
                continue;
            }
            let (from, to) = if forward {
                (&pair.left, &pair.right)
            } else {
                (&pair.right, &pair.left)
            };
            for candidate in from {
                let candidate_value = self.eval(&mut crosswalk_frame, candidate)?;
                if equal(&candidate_value, value) {
                    return match to.first() {
                        Some(first) => self.eval(&mut crosswalk_frame, first),
                        None => Ok(Value::None),
                    };
                }
            }
        }
        for (direction, otherwise) in &decl.otherwise {
            let applies = match direction {
                Direction::Backward => !forward,
                _ => forward,
            };
            if applies {
                return self.eval(&mut crosswalk_frame, otherwise);
            }
        }
        Ok(Value::None)
    }

    /// `lookup Kind(code)`: the host's answer, or a need.
    fn lookup(&mut self, frame: &Frame, kind: &str, code: &Value, hint: &Value) -> Value {
        if code.is_none() {
            return Value::None;
        }
        let code_text = display(code).trim().to_owned();
        if code_text.is_empty() {
            return Value::None;
        }
        self.note_step(&format!("lookup {kind}"));
        let key = (kind.to_owned(), code_text.clone());
        match self.given.lookups.get(&key) {
            Some(Some(found)) => Value::Ref(Rc::new(found.clone())),
            Some(None) => {
                if !self
                    .unmapped
                    .iter()
                    .any(|item| item.kind == kind && item.code == code_text)
                {
                    self.unmapped.push(Unmapped {
                        kind: kind.to_owned(),
                        code: code_text,
                        hint: super::json::to_json(hint),
                        path: frame.path.clone(),
                    });
                }
                Value::None
            }
            None => {
                self.needs.insert(Need {
                    kind: kind.to_owned(),
                    code: code_text,
                });
                Value::None
            }
        }
    }

    /// Calls a function, standard or the project's.
    fn call(
        &mut self,
        frame: &mut Frame,
        callee: &Expr,
        args: &[crate::lading::syntax::ast::Arg],
    ) -> Eval {
        let (name, receiver) = match &callee.kind {
            ExprKind::Name(name) => (name.as_str(), None),
            ExprKind::Member {
                target,
                member,
                kind: MemberKind::Name,
            } => (member.name.as_str(), Some(self.eval(frame, target)?)),
            _ => return Ok(Value::None),
        };
        if let Some((unit, Decl::Fn(index))) = self.program.decl(frame.unit, name) {
            return self.user_call(frame, unit, index, receiver, args);
        }
        let mut inputs = Vec::with_capacity(args.len() + 1);
        if let Some(receiver) = receiver {
            inputs.push(super::stdlib::Input::Value(receiver));
        }
        for (position, arg) in args.iter().enumerate() {
            match &arg.value.kind {
                ExprKind::Lambda { param, body } => {
                    inputs.push(super::stdlib::Input::Lambda(
                        param.name.clone(),
                        body.as_ref().clone(),
                    ));
                }
                ExprKind::Name(unit_name)
                    if matches!(name, "in" | "value")
                        && position == 0
                        && Unit::named(unit_name).is_some() =>
                {
                    inputs.push(super::stdlib::Input::Value(Value::text(unit_name)));
                }
                _ => inputs.push(super::stdlib::Input::Value(self.eval(frame, &arg.value)?)),
            }
        }
        self.note_step(name);
        super::stdlib::call(self, frame, name, &inputs)
    }

    fn user_call(
        &mut self,
        frame: &mut Frame,
        unit: usize,
        index: usize,
        receiver: Option<Value>,
        args: &[crate::lading::syntax::ast::Arg],
    ) -> Eval {
        let program = self.program;
        let Some(Item::Fn(decl)) = program.units[unit].items().get(index) else {
            return Ok(Value::None);
        };
        self.note_step(&decl.name.name);
        let signature = program.fns.get(&(unit, decl.name.name.clone())).cloned();
        let mut values: Vec<Option<Value>> = vec![None; decl.params.len()];
        let mut position = 0;
        if let Some(receiver) = receiver {
            if let Some(slot) = values.first_mut() {
                *slot = Some(receiver);
            }
            position = 1;
        }
        for arg in args {
            let slot = if let Some(named) = &arg.name {
                decl.params
                    .iter()
                    .position(|param| param.name.name == named.name)
            } else {
                let slot = position;
                position += 1;
                Some(slot)
            };
            let value = self.eval(frame, &arg.value)?;
            if let Some(slot) = slot.and_then(|slot| values.get_mut(slot)) {
                *slot = Some(value);
            }
        }
        let mut body_frame = Frame::new(unit);
        for (offset, param) in decl.params.iter().enumerate() {
            let value = match values[offset].take() {
                Some(value) => value,
                None => match &param.default {
                    Some(default) => self.eval(&mut body_frame, default)?,
                    None => Value::None,
                },
            };
            let param_ty = signature
                .as_ref()
                .and_then(|signature| signature.params.get(offset))
                .map(|(_, ty, _)| ty);
            if value.is_none() && param_ty.is_some_and(|ty| !ty.is_optional()) {
                return Ok(Value::None);
            }
            body_frame.bind(&param.name.name, value);
        }
        for binding in &decl.body.lets {
            let value = self.eval(&mut body_frame, &binding.value)?;
            body_frame.bind(&binding.name.name, value);
        }
        let result = self.eval(&mut body_frame, &decl.body.value)?;
        Ok(match signature {
            Some(signature) => self.coerce(result, &signature.ret),
            None => result,
        })
    }

    /// Applies a lambda to a value.
    pub fn apply(&mut self, frame: &mut Frame, param: &str, body: &Expr, value: Value) -> Eval {
        frame.push();
        frame.bind(param, value);
        let result = self.eval(frame, body);
        frame.pop();
        result
    }
}

/// A field as read: lists and references by qualifier are empty rather than none.
fn read_default(record: &str, name: &str, value: Value) -> Value {
    if !value.is_none() {
        return value;
    }
    match schema::record(record)
        .and_then(|owner| owner.field(name))
        .map(|field| field.spec)
    {
        Some(schema::Spec::Records(_) | schema::Spec::Texts) => Value::list(Vec::new()),
        Some(schema::Spec::Refs | schema::Spec::Custom(_)) => Value::Map(Rc::new(BTreeMap::new())),
        _ => Value::None,
    }
}

/// `base.name`, or `name` at the top.
#[must_use]
pub fn join_path(base: &str, name: &str) -> String {
    if base.is_empty() {
        name.to_owned()
    } else {
        format!("{base}.{name}")
    }
}

/// The segments with an ID under an instance: a list when the guide lets it repeat.
fn segments(tree: &Rc<Tree>, instance: Node, id: &str) -> Value {
    let found = tree.segments_in(instance, id);
    let repeats = found.len() > 1
        || found
            .first()
            .and_then(|index| tree.segments[*index].node)
            .or_else(|| {
                let parent = tree.instances[instance].node;
                crate::lading::check::x12::child_segment(&tree.compiled, parent, id)
            })
            .is_some_and(|node| tree.compiled.node(node).max > 1);
    if repeats {
        Value::list(
            found
                .into_iter()
                .map(|index| Value::Segment(Rc::clone(tree), index))
                .collect(),
        )
    } else {
        found
            .first()
            .map_or(Value::None, |index| Value::Segment(Rc::clone(tree), *index))
    }
}

/// The instances of a loop under an instance: a list when it repeats.
fn loops(tree: &Rc<Tree>, instance: Node, id: &str) -> Value {
    let found = tree.loops_in(instance, id);
    let parent = tree.instances[instance].node;
    let repeats =
        crate::lading::check::x12::child_loop(&tree.compiled, parent, id).is_some_and(|node| {
            tree.compiled.node(node).max > 1
                && matches!(tree.compiled.node(node).kind, NodeKind::Loop { .. })
        });
    if repeats || found.len() > 1 {
        Value::list(
            found
                .into_iter()
                .map(|child| Value::Loop(Rc::clone(tree), child))
                .collect(),
        )
    } else {
        found
            .first()
            .map_or(Value::None, |child| Value::Loop(Rc::clone(tree), *child))
    }
}

/// Whether a value is of a type, for choosing a crosswalk's direction.
fn fits_value(value: &Value, ty: &Type) -> bool {
    match (value, ty.unwrapped()) {
        (Value::Enum(owner, _), Type::Enum(name)) => owner == name,
        (Value::Enum(..), _) | (_, Type::Enum(_)) => false,
        (Value::Text(_), Type::Text | Type::Code(_))
        | (Value::Int(_) | Value::Decimal(_), Type::Int | Type::Decimal)
        | (Value::Ref(_), Type::Ref(_))
        | (_, Type::Any) => true,
        _ => false,
    }
}

fn negate(value: &Value) -> Value {
    match value {
        Value::Int(number) => number.checked_neg().map_or(Value::None, Value::Int),
        Value::Decimal(number) => Value::Decimal(-number),
        Value::Percent(number) => Value::Percent(-number),
        Value::Money(amount, currency) => Value::Money(-amount, Rc::clone(currency)),
        Value::Measure(amount, unit) => Value::Measure(-amount, *unit),
        Value::Duration(seconds) => Value::Duration(-seconds),
        _ => Value::None,
    }
}

/// Exact arithmetic; none when either side is none, the units differ in dimension, or a
/// division is by zero.
#[must_use]
#[allow(clippy::too_many_lines)]
pub fn arithmetic(op: BinOp, left: &Value, right: &Value) -> Value {
    let checked = |a: Decimal, b: Decimal| -> Option<Decimal> {
        match op {
            BinOp::Add => a.checked_add(b),
            BinOp::Sub => a.checked_sub(b),
            BinOp::Mul => a.checked_mul(b),
            BinOp::Div => a.checked_div(b),
            _ => None,
        }
    };
    let hundred = Decimal::from(100);
    match (left, right) {
        (Value::Int(a), Value::Int(b)) => match op {
            BinOp::Add => a.checked_add(*b).map_or(Value::None, Value::Int),
            BinOp::Sub => a.checked_sub(*b).map_or(Value::None, Value::Int),
            BinOp::Mul => a.checked_mul(*b).map_or(Value::None, Value::Int),
            _ => checked(Decimal::from(*a), Decimal::from(*b)).map_or(Value::None, Value::Decimal),
        },
        (Value::Text(a), Value::Text(b)) if op == BinOp::Add => Value::text(format!("{a}{b}")),
        (Value::Money(a, x), Value::Money(b, y)) => match op {
            BinOp::Add | BinOp::Sub if x == y => {
                checked(*a, *b).map_or(Value::None, |amount| Value::Money(amount, Rc::clone(x)))
            }
            BinOp::Div if x == y => checked(*a, *b).map_or(Value::None, Value::Decimal),
            _ => Value::None,
        },
        (Value::Money(amount, currency), Value::Percent(percent))
        | (Value::Percent(percent), Value::Money(amount, currency))
            if op == BinOp::Mul =>
        {
            amount
                .checked_mul(*percent)
                .and_then(|value| value.checked_div(hundred))
                .map_or(Value::None, |amount| {
                    Value::Money(amount, Rc::clone(currency))
                })
        }
        (Value::Money(amount, currency), number) if matches!(op, BinOp::Mul | BinOp::Div) => number
            .as_decimal()
            .and_then(|number| checked(*amount, number))
            .map_or(Value::None, |amount| {
                Value::Money(amount, Rc::clone(currency))
            }),
        (number, Value::Money(amount, currency)) if op == BinOp::Mul => number
            .as_decimal()
            .and_then(|number| number.checked_mul(*amount))
            .map_or(Value::None, |amount| {
                Value::Money(amount, Rc::clone(currency))
            }),
        (Value::Measure(a, x), Value::Measure(b, y)) if x.dim == y.dim => match op {
            BinOp::Add | BinOp::Sub if x.dim != Dim::Temperature || op == BinOp::Sub => {
                let b_in_x = convert(*b, *y, *x);
                checked(*a, b_in_x).map_or(Value::None, |amount| Value::Measure(amount, *x))
            }
            BinOp::Div => checked(*a, convert(*b, *y, *x)).map_or(Value::None, Value::Decimal),
            _ => Value::None,
        },
        (Value::Measure(amount, unit), number) if matches!(op, BinOp::Mul | BinOp::Div) => number
            .as_decimal()
            .and_then(|number| checked(*amount, number))
            .map_or(Value::None, |amount| Value::Measure(amount, *unit)),
        (number, Value::Measure(amount, unit)) if op == BinOp::Mul => number
            .as_decimal()
            .and_then(|number| number.checked_mul(*amount))
            .map_or(Value::None, |amount| Value::Measure(amount, *unit)),
        (Value::DateTime(at, zone), Value::Duration(seconds)) => match op {
            BinOp::Add => Value::DateTime(at + seconds, *zone),
            BinOp::Sub => Value::DateTime(at - seconds, *zone),
            _ => Value::None,
        },
        (Value::Duration(seconds), Value::DateTime(at, zone)) if op == BinOp::Add => {
            Value::DateTime(at + seconds, *zone)
        }
        (Value::DateTime(a, _), Value::DateTime(b, _)) if op == BinOp::Sub => {
            Value::Duration(a - b)
        }
        (Value::Date(days), Value::Duration(seconds)) => match op {
            BinOp::Add => Value::Date(days + seconds.div_euclid(86_400)),
            BinOp::Sub => Value::Date(days - seconds.div_euclid(86_400)),
            _ => Value::None,
        },
        (Value::Duration(a), Value::Duration(b)) => match op {
            BinOp::Add => Value::Duration(a + b),
            BinOp::Sub => Value::Duration(a - b),
            _ => Value::None,
        },
        (Value::Duration(seconds), number) if matches!(op, BinOp::Mul | BinOp::Div) => number
            .as_decimal()
            .and_then(|number| checked(Decimal::from(*seconds), number))
            .and_then(|value| i64::try_from(value.round()).ok())
            .map_or(Value::None, Value::Duration),
        (Value::Percent(a), Value::Percent(b)) if matches!(op, BinOp::Add | BinOp::Sub) => {
            checked(*a, *b).map_or(Value::None, Value::Percent)
        }
        (Value::Percent(percent), number) | (number, Value::Percent(percent))
            if op == BinOp::Mul =>
        {
            number
                .as_decimal()
                .and_then(|number| number.checked_mul(*percent))
                .and_then(|value| value.checked_div(hundred))
                .map_or(Value::None, Value::Decimal)
        }
        _ => match (left.as_decimal(), right.as_decimal()) {
            (Some(a), Some(b)) => checked(a, b).map_or(Value::None, Value::Decimal),
            _ => Value::None,
        },
    }
}
