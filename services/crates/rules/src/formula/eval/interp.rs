//! The interpreter: walks a parsed formula over one environment, metering every step.
//!
//! - **Scopes.** A `let` binds a name for the statements after it (at top level) or for the rest
//!   of its sequence (inside parentheses, a block or a predicate). Bindings live on one stack
//!   that a sequence truncates when it ends, so sibling scopes may reuse a name.
//! - **Predicates.** `map(list, ...)` and the other predicate functions push one frame per item:
//!   `#` is the item, `.field` its field, `#index` its position and `#acc` the accumulator of
//!   `reduce`.
//! - **Short circuits.** `and`, `or`, `??`, the ternary, `if` and chained comparisons evaluate
//!   only what decides them; `a?.b.c` stops at the first empty `?.` and is nil.
//! - **Empty values.** `??`, `coalesce`, `== nil` and optional chaining take nil without error.
//!   Anything else that needs a value and gets nil fails with `missingField`, naming the field
//!   path the nil was read from (also through a `let` bound to a field, and as `.field` inside a
//!   predicate) and the guard that keeps the formula pricing: `coalesce(weight, 0)`,
//!   `origin?.zip ?? ""`. The Go engine named every nil field the formula referenced whenever
//!   anything failed; this names the one that caused the failure.
//! - **Eager lets.** A `let` is evaluated where it is written. A `let` that fails fails the
//!   formula, used or not, as in the Go engine; only the statement trace goes on to evaluate the
//!   statements after it, so the Studio can show each one.
//!
//! The tree is at most `syntax::MAX_DEPTH` deep, and the interpreter also counts its own
//! recursion, so no formula can exhaust the WebAssembly stack.

use super::budget::Limits;
use super::budget::Meter;
use super::builtins;
use super::builtins::Context;
use super::builtins::RegexCache;
use super::error::Code;
use super::error::EvalError;
use super::error::expected_boolean;
use super::error::expected_list;
use super::error::expected_text;
use super::operators;
use super::operators::Arithmetic;
use super::operators::Comparison;
use super::operators::TextTest;
use super::operators::too_deep;
use super::tables::Matched;
use super::tables::Tables;
use super::value::List;
use super::value::Value;
use crate::formula::syntax::ast::BinaryOp;
use crate::formula::syntax::ast::CompareOp;
use crate::formula::syntax::ast::Expr;
use crate::formula::syntax::ast::ExprKind;
use crate::formula::syntax::ast::Pointer;
use crate::formula::syntax::ast::Program;
use crate::formula::syntax::ast::StatementKind;
use crate::formula::syntax::ast::UnaryOp;
use crate::message::Arg;
use crate::message::Message;
use crate::span::Span;
use rust_decimal::Decimal;
use std::collections::BTreeMap;
use std::rc::Rc;

/// How deeply the interpreter may recurse: well past what a tree of `syntax::MAX_DEPTH` levels
/// needs, well inside the WebAssembly stack.
const MAX_RECURSION: usize = 512;

/// Where lookups read from.
#[derive(Debug, Clone, Copy)]
pub enum LookupSource<'a> {
    /// The request's rate tables.
    Tables(&'a Tables),
    /// Every table exists and every lookup is zero: the Go engine's `StubLookup`, which yes-or-no
    /// predicates rate with, so a condition never loads a tenant's tables.
    Stub,
    /// The request carried no tables: every lookup fails with `lookupUnavailable`.
    Unavailable,
}

/// One lookup a formula made, for the receipt.
#[derive(Debug, Clone, PartialEq)]
pub struct LookupRecord {
    /// The function: `lookup`, `lookupOr`, `lookup2`, `lookup2Or`, `lookupInterp` or
    /// `deficitWeight`.
    pub function: &'static str,
    /// The table name.
    pub table: String,
    /// The keys as given: one, or the row and column.
    pub keys: Vec<Value>,
    /// What the table answered, when it answered.
    pub value: Option<Decimal>,
    /// Which entry, band or cell answered.
    pub matched: Option<Matched>,
    /// Why it did not answer. A miss a `*Or` function absorbed is recorded too.
    pub error: Option<EvalError>,
}

/// One top-level statement as the trace shows it.
#[derive(Debug, Clone)]
pub struct StatementOutcome {
    /// Its position among the statements.
    pub index: usize,
    /// The bound name of a `let`; `None` for a value.
    pub name: Option<String>,
    /// From its first token to the last token of its value.
    pub span: Span,
    /// Its value, or why it has none.
    pub outcome: Result<Value, EvalError>,
}

/// What a program came to.
#[derive(Debug)]
pub struct Evaluation {
    /// The value of the last statement, or the first failure.
    pub result: Result<Value, EvalError>,
    /// Each top-level statement, when traced.
    pub statements: Vec<StatementOutcome>,
    /// Each lookup, in the order made, when recorded.
    pub lookups: Vec<LookupRecord>,
}

/// What one evaluation runs with.
#[derive(Debug, Clone, Copy)]
pub struct Setup<'a> {
    /// The environment: a map of fields and inputs.
    pub env: &'a Value,
    /// Where lookups read from.
    pub lookups: LookupSource<'a>,
    /// The budgets.
    pub limits: Limits,
    /// Record every top-level statement's value.
    pub trace_statements: bool,
    /// Record every lookup.
    pub record_lookups: bool,
}

/// Evaluates a parsed program. A program with syntax diagnostics must not reach here.
#[must_use]
pub fn evaluate(program: &Program, setup: Setup<'_>) -> Evaluation {
    let mut interpreter = Interpreter {
        env: setup.env,
        lookups: setup.lookups,
        meter: Meter::new(setup.limits),
        regexes: RegexCache::default(),
        lets: Vec::new(),
        frames: Vec::new(),
        records: Vec::new(),
        record: setup.record_lookups,
        depth: 0,
    };
    let mut statements = Vec::new();
    let result = interpreter.run(program, setup.trace_statements, &mut statements);
    Evaluation {
        result,
        statements,
        lookups: interpreter.records,
    }
}

/// Where a value was read from, for a missing-value error.
#[derive(Debug, Clone)]
struct Origin {
    path: String,
    /// Read from the current predicate item (`.weight`), not from the environment.
    relative: bool,
}

#[derive(Debug)]
struct Binding<'a> {
    name: &'a str,
    value: Result<Value, EvalError>,
    origin: Option<Origin>,
}

#[derive(Debug)]
struct Frame {
    element: Value,
    index: usize,
    accumulator: Value,
}

/// What an empty value should have been, for the guard a missing-value error suggests.
#[derive(Debug, Clone, Copy)]
enum Zero {
    Number,
    Text,
    Bool,
    Date,
    Duration,
    List,
    Map,
}

impl Zero {
    const fn literal(self) -> &'static str {
        match self {
            Self::Number => "0",
            Self::Text => "\"\"",
            Self::Bool => "false",
            Self::Date => "ratingTime",
            Self::Duration => "duration(\"0s\")",
            Self::List => "[]",
            Self::Map => "{}",
        }
    }

    fn like(value: &Value) -> Self {
        match value {
            Value::Text(_) => Self::Text,
            Value::Bool(_) => Self::Bool,
            Value::Date(_) => Self::Date,
            Value::Duration(_) => Self::Duration,
            Value::List(_) => Self::List,
            Value::Map(_) => Self::Map,
            Value::Number(_) | Value::Nil => Self::Number,
        }
    }
}

/// The guard that keeps a formula pricing when `origin` is empty, as Go's `guardFor`.
fn guard(origin: &Origin, zero: Zero) -> String {
    let zero = zero.literal();
    if origin.relative {
        format!("{} ?? {zero}", origin.path)
    } else if origin.path.contains('.') {
        format!("{} ?? {zero}", origin.path.replace('.', "?."))
    } else {
        format!("coalesce({}, {zero})", origin.path)
    }
}

struct Interpreter<'a> {
    env: &'a Value,
    lookups: LookupSource<'a>,
    meter: Meter,
    regexes: RegexCache,
    lets: Vec<Binding<'a>>,
    frames: Vec<Frame>,
    records: Vec<LookupRecord>,
    record: bool,
    depth: usize,
}

type Outcome = Result<Value, EvalError>;

fn syntax_error(span: Span) -> EvalError {
    EvalError::new(
        Code::Syntax,
        Message::new(crate::message!("the formula has a syntax error here")),
    )
    .at(span)
}

impl<'a> Interpreter<'a> {
    fn run(
        &mut self,
        program: &'a Program,
        trace: bool,
        statements: &mut Vec<StatementOutcome>,
    ) -> Outcome {
        if program.statements.is_empty() {
            return Err(EvalError::new(
                Code::Syntax,
                Message::new(crate::message!("expression has no statements")),
            ));
        }
        let mut first_error: Option<EvalError> = None;
        let mut last = Value::Nil;
        for (index, statement) in program.statements.iter().enumerate() {
            let outcome = self.eval(&statement.value);
            if let Err(error) = &outcome {
                if error.is_fatal() || !trace {
                    return Err(error.clone());
                }
                if first_error.is_none() {
                    first_error = Some(error.clone());
                }
            }
            if trace {
                statements.push(StatementOutcome {
                    index,
                    name: match &statement.kind {
                        StatementKind::Let(name) => Some(name.text.clone()),
                        StatementKind::Result => None,
                    },
                    span: statement.span,
                    outcome: outcome.clone(),
                });
            }
            match &statement.kind {
                StatementKind::Let(name) => {
                    let origin = self.origin(&statement.value);
                    self.lets.push(Binding {
                        name: &name.text,
                        value: outcome,
                        origin,
                    });
                }
                StatementKind::Result => {
                    if let Ok(value) = outcome {
                        last = value;
                    }
                }
            }
        }
        if let Some(error) = first_error {
            return Err(error);
        }
        if let Some(statement) = program.statements.last()
            && matches!(statement.kind, StatementKind::Let(_))
        {
            return Err(syntax_error(statement.span));
        }
        Ok(last)
    }

    fn eval(&mut self, expr: &'a Expr) -> Outcome {
        self.depth += 1;
        let outcome = if self.depth > MAX_RECURSION {
            Err(too_deep())
        } else {
            self.meter.step().and_then(|()| self.eval_kind(expr))
        };
        self.depth -= 1;
        outcome.map_err(|error| error.at(expr.span))
    }

    fn eval_kind(&mut self, expr: &'a Expr) -> Outcome {
        match &expr.kind {
            ExprKind::Number(number) => number
                .value
                .map(Value::Number)
                .ok_or_else(|| syntax_error(expr.span)),
            ExprKind::String(text) => {
                Ok(Value::Text(Rc::from(text.as_str())))
            }
            ExprKind::Bool(flag) => Ok(Value::Bool(*flag)),
            ExprKind::Nil => Ok(Value::Nil),
            ExprKind::Pointer(pointer) => self.pointer(pointer),
            ExprKind::Ident(name) => self.identifier(name),
            ExprKind::Member(_) | ExprKind::Index(_) | ExprKind::Slice(_) | ExprKind::Method(_) => {
                Ok(self.access(expr)?.unwrap_or(Value::Nil))
            }
            ExprKind::Call(call) => self.call(&call.function.text, &call.args, expr.span),
            ExprKind::Unary(unary) => self.unary(unary.op, &unary.operand),
            ExprKind::Binary(binary) => {
                let outcome = self.binary(&binary.op, &binary.left, &binary.right)?;
                match (binary.not, outcome) {
                    (Some(_), Value::Bool(flag)) => Ok(Value::Bool(!flag)),
                    (_, value) => Ok(value),
                }
            }
            ExprKind::Chain(chain) => self.chain(&chain.operands, &chain.operators),
            ExprKind::Ternary(ternary) => {
                if self.condition(&ternary.test)? {
                    match &ternary.then {
                        Some(then) => self.eval(then),
                        None => Ok(Value::Bool(true)),
                    }
                } else {
                    self.eval(&ternary.otherwise)
                }
            }
            ExprKind::If(branch) => {
                if self.condition(&branch.test)? {
                    self.eval(&branch.then)
                } else {
                    self.eval(&branch.otherwise)
                }
            }
            ExprKind::Array(items) => {
                let mut values = Vec::with_capacity(items.len());
                for item in items {
                    values.push(self.eval(item)?);
                }
                self.meter.items(values.len())?;
                Value::list(values).map_err(|_| too_deep())
            }
            ExprKind::Map(entries) => {
                let mut map: BTreeMap<Rc<str>, Value> = BTreeMap::new();
                for entry in entries {
                    let value = self.eval(&entry.value)?;
                    map.insert(Rc::from(entry.key.text.as_str()), value);
                }
                self.meter.items(map.len())?;
                Value::map(map).map_err(|_| too_deep())
            }
            ExprKind::Predicate(predicate) => self.eval(&predicate.body),
            ExprKind::Sequence(sequence) => {
                let mark = self.lets.len();
                let mut outcome = Ok(Value::Nil);
                for binding in &sequence.lets {
                    match self.eval(&binding.value) {
                        Ok(value) => {
                            let origin = self.origin(&binding.value);
                            self.lets.push(Binding {
                                name: &binding.name.text,
                                value: Ok(value),
                                origin,
                            });
                        }
                        Err(error) => {
                            outcome = Err(error);
                            break;
                        }
                    }
                }
                if outcome.is_ok() {
                    outcome = self.eval(&sequence.body);
                }
                self.lets.truncate(mark);
                outcome
            }
            ExprKind::Missing => Err(syntax_error(expr.span)),
        }
    }

    fn pointer(&self, pointer: &Pointer) -> Outcome {
        let Some(frame) = self.frames.last() else {
            return Err(EvalError::with(
                Code::Type,
                crate::message!("{0} is only defined inside a predicate"),
                [Arg::from(pointer.text())],
            ));
        };
        match pointer {
            Pointer::Element | Pointer::Implicit => Ok(frame.element.clone()),
            Pointer::Index => Ok(Value::int(i64::try_from(frame.index).unwrap_or(i64::MAX))),
            Pointer::Accumulator => Ok(frame.accumulator.clone()),
            Pointer::Unknown(name) => Err(unknown_name(&format!("#{name}"))),
        }
    }

    fn identifier(&self, name: &str) -> Outcome {
        if let Some(binding) = self.lets.iter().rev().find(|binding| binding.name == name) {
            return binding.value.clone();
        }
        match self.env {
            Value::Map(map) => Ok(map.get(name).cloned().unwrap_or(Value::Nil)),
            _ => Ok(Value::Nil),
        }
    }

    /// Where a node's value was read from, when it is a field path.
    fn origin(&self, expr: &Expr) -> Option<Origin> {
        match &expr.kind {
            ExprKind::Ident(name) => {
                match self.lets.iter().rev().find(|binding| binding.name == name) {
                    Some(binding) => binding.origin.clone(),
                    None => Some(Origin {
                        path: name.clone(),
                        relative: false,
                    }),
                }
            }
            ExprKind::Pointer(Pointer::Element | Pointer::Implicit) => {
                self.frames.last().map(|_| Origin {
                    path: String::from("#"),
                    relative: true,
                })
            }
            ExprKind::Member(member) => {
                match member.object.kind {
                    ExprKind::Pointer(Pointer::Implicit) => {
                        return Some(Origin {
                            path: format!(".{}", member.property.text),
                            relative: true,
                        });
                    }
                    ExprKind::Pointer(Pointer::Element) => {
                        return Some(Origin {
                            path: format!("#.{}", member.property.text),
                            relative: true,
                        });
                    }
                    _ => {}
                }
                self.origin(&member.object).map(|object| Origin {
                    path: format!("{}.{}", object.path, member.property.text),
                    relative: object.relative,
                })
            }
            _ => None,
        }
    }

    /// Turns an operation's empty-operand error into a missing-value error naming the field.
    fn explain(&self, mut error: EvalError, operands: &[&Expr], zero: Zero) -> EvalError {
        let Some(index) = error.empty_argument.take() else {
            return error;
        };
        let Some(operand) = operands.get(index) else {
            return error;
        };
        match self.origin(operand) {
            Some(origin) => {
                let guard = guard(&origin, zero);
                EvalError::missing(vec![origin.path], vec![guard]).at(operand.span)
            }
            None => error,
        }
    }

    fn condition(&mut self, test: &'a Expr) -> Result<bool, EvalError> {
        match self.eval(test)? {
            Value::Bool(flag) => Ok(flag),
            Value::Nil => Err(self.explain(
                expected_boolean("nil").empty_argument(0),
                &[test],
                Zero::Bool,
            )),
            other => Err(expected_boolean(other.kind_key()).at(test.span)),
        }
    }

    /// A member, index, slice or method, or `None` when an optional link met an empty value.
    fn access(&mut self, expr: &'a Expr) -> Result<Option<Value>, EvalError> {
        let (object, optional) = match &expr.kind {
            ExprKind::Member(member) => (&member.object, member.optional),
            ExprKind::Index(index) => (&index.object, index.optional),
            ExprKind::Slice(slice) => (&slice.object, slice.optional),
            ExprKind::Method(method) => (&method.receiver, method.optional),
            _ => return self.eval(expr).map(Some),
        };
        let chained = object.parens == 0
            && matches!(
                object.kind,
                ExprKind::Member(_) | ExprKind::Index(_) | ExprKind::Slice(_) | ExprKind::Method(_)
            );
        let base = if chained {
            self.depth += 1;
            let base = if self.depth > MAX_RECURSION {
                Err(too_deep())
            } else {
                self.meter.step().and_then(|()| self.access(object))
            };
            self.depth -= 1;
            match base.map_err(|error| error.at(object.span))? {
                Some(value) => value,
                None => return Ok(None),
            }
        } else {
            self.eval(object)?
        };
        if optional && base.is_nil() {
            return Ok(None);
        }
        let outcome = match &expr.kind {
            ExprKind::Member(member) => operators::member(&base, &member.property.text)
                .map_err(|error| self.missing_member(error, object, &member.property.text)),
            ExprKind::Index(index) => {
                let key = self.eval(&index.index)?;
                operators::index(&base, &key, &mut self.meter)
                    .map_err(|error| self.explain(error, &[object, &index.index], Zero::List))
            }
            ExprKind::Slice(slice) => {
                let start = match &slice.start {
                    Some(start) => Some(self.position(start)?),
                    None => None,
                };
                let end = match &slice.end {
                    Some(end) => Some(self.position(end)?),
                    None => None,
                };
                operators::slice(&base, start, end, &mut self.meter)
                    .map_err(|error| self.explain(error, &[object], Zero::Text))
            }
            ExprKind::Method(method) => {
                if !method.args.is_empty() {
                    let placeholders = vec![Value::Nil; method.args.len()];
                    return Err(builtins::arity(&method.method.text, &placeholders, 0, 0)
                        .err()
                        .unwrap_or_else(|| unknown_name(&method.method.text)));
                }
                let zero = if matches!(
                    method.method.text.as_str(),
                    "Hours" | "Minutes" | "Seconds"
                ) {
                    Zero::Duration
                } else {
                    Zero::Date
                };
                operators::method(&base, &method.method.text)
                    .map_err(|error| self.explain(error, &[object], zero))
            }
            _ => return Ok(None),
        };
        outcome.map(Some)
    }

    /// Reading `.property` of an empty object names the path that was read: `origin.zip`.
    fn missing_member(&self, mut error: EvalError, object: &Expr, property: &str) -> EvalError {
        if error.empty_argument.take().is_none() {
            return error;
        }
        match self.origin(object) {
            Some(origin) => {
                let read = Origin {
                    path: format!("{}.{property}", origin.path),
                    relative: origin.relative,
                };
                let guard = guard(&read, Zero::Number);
                EvalError::missing(vec![read.path], vec![guard]).at(object.span)
            }
            None => error,
        }
    }

    fn position(&mut self, expr: &'a Expr) -> Result<i64, EvalError> {
        let value = self.eval(expr)?;
        operators::position(&value, 0).map_err(|error| self.explain(error, &[expr], Zero::Number))
    }

    fn unary(&mut self, op: UnaryOp, operand: &'a Expr) -> Outcome {
        let value = self.eval(operand)?;
        match op {
            UnaryOp::Not { .. } => operators::not(&value)
                .map_err(|error| self.explain(error, &[operand], Zero::Bool)),
            UnaryOp::Negate => operators::sign(true, &value)
                .map_err(|error| self.explain(error, &[operand], Zero::Number)),
            UnaryOp::Plus => operators::sign(false, &value)
                .map_err(|error| self.explain(error, &[operand], Zero::Number)),
        }
    }

    fn binary(&mut self, op: &BinaryOp, left: &'a Expr, right: &'a Expr) -> Outcome {
        match op {
            BinaryOp::Or { .. } => {
                if self.condition(left)? {
                    return Ok(Value::Bool(true));
                }
                return self.condition(right).map(Value::Bool);
            }
            BinaryOp::And { .. } => {
                if !self.condition(left)? {
                    return Ok(Value::Bool(false));
                }
                return self.condition(right).map(Value::Bool);
            }
            BinaryOp::Coalesce => {
                let value = self.eval(left)?;
                return if value.is_nil() {
                    self.eval(right)
                } else {
                    Ok(value)
                };
            }
            _ => {}
        }
        let left_value = self.eval(left)?;
        let right_value = self.eval(right)?;
        let operands = [left, right];
        let zero_left = Zero::like(&right_value);
        let zero_right = Zero::like(&left_value);
        let both_empty = left_value.is_nil() && right_value.is_nil();
        let explain = |interpreter: &Self, error: EvalError| {
            if both_empty
                && error.empty_argument.is_some()
                && let (Some(first), Some(second)) =
                    (interpreter.origin(left), interpreter.origin(right))
            {
                let guards = vec![guard(&first, Zero::Number), guard(&second, Zero::Number)];
                return EvalError::missing(vec![first.path, second.path], guards)
                    .at(left.span.cover(right.span));
            }
            let zero = if error.empty_argument == Some(1) {
                zero_right
            } else {
                zero_left
            };
            interpreter.explain(error, &operands, zero)
        };
        let arithmetic = |kind: Arithmetic, interpreter: &mut Self| {
            operators::arithmetic(kind, &left_value, &right_value, &mut interpreter.meter)
                .map_err(|error| explain(interpreter, error))
        };
        match op {
            BinaryOp::Equal => operators::equal(&left_value, &right_value).map(Value::Bool),
            BinaryOp::NotEqual => operators::equal(&left_value, &right_value)
                .map(|equal| Value::Bool(!equal)),
            BinaryOp::Compare(compare) => {
                operators::compare(comparison(*compare), &left_value, &right_value)
                    .map(Value::Bool)
                    .map_err(|error| explain(self, error))
            }
            BinaryOp::In => operators::membership(&left_value, &right_value).map(Value::Bool),
            BinaryOp::Matches => {
                let text = text_value(&left_value, 0).map_err(|error| explain(self, error))?;
                let pattern = text_value(&right_value, 1).map_err(|error| explain(self, error))?;
                self.regexes
                    .matches(text, pattern, &mut self.meter)
                    .map(Value::Bool)
            }
            BinaryOp::Contains => self.text_test(TextTest::Contains, &left_value, &right_value, explain),
            BinaryOp::StartsWith => {
                self.text_test(TextTest::StartsWith, &left_value, &right_value, explain)
            }
            BinaryOp::EndsWith => {
                self.text_test(TextTest::EndsWith, &left_value, &right_value, explain)
            }
            BinaryOp::Word(word) => {
                let args = [left_value.clone(), right_value.clone()];
                let mut context = Context {
                    meter: &mut self.meter,
                    regexes: &mut self.regexes,
                };
                builtins::call(word, &args, &mut context)
                    .unwrap_or_else(|| Err(unknown_name(word)))
                    .map_err(|error| explain(self, error))
            }
            BinaryOp::Range => operators::range(&left_value, &right_value, &mut self.meter)
                .map_err(|error| explain(self, error)),
            BinaryOp::Add => arithmetic(Arithmetic::Add, self),
            BinaryOp::Subtract => arithmetic(Arithmetic::Sub, self),
            BinaryOp::Multiply => arithmetic(Arithmetic::Mul, self),
            BinaryOp::Divide => arithmetic(Arithmetic::Div, self),
            BinaryOp::Modulo => arithmetic(Arithmetic::Rem, self),
            BinaryOp::Power { .. } => arithmetic(Arithmetic::Pow, self),
            BinaryOp::Or { .. } | BinaryOp::And { .. } | BinaryOp::Coalesce => Ok(Value::Nil),
        }
    }

    fn text_test(
        &self,
        test: TextTest,
        left: &Value,
        right: &Value,
        explain: impl Fn(&Self, EvalError) -> EvalError,
    ) -> Outcome {
        operators::text_test(test, left, right)
            .map(Value::Bool)
            .map_err(|error| explain(self, error))
    }

    fn chain(&mut self, operands: &'a [Expr], links: &[(CompareOp, Span)]) -> Outcome {
        let Some(first) = operands.first() else {
            return Ok(Value::Bool(true));
        };
        let mut left_node = first;
        let mut left = self.eval(first)?;
        for (index, (op, _)) in links.iter().enumerate() {
            let Some(right_node) = operands.get(index + 1) else {
                break;
            };
            let right = self.eval(right_node)?;
            let holds = operators::compare(comparison(*op), &left, &right).map_err(|error| {
                let zero = if error.empty_argument == Some(1) {
                    Zero::like(&left)
                } else {
                    Zero::like(&right)
                };
                self.explain(error, &[left_node, right_node], zero)
            })?;
            if !holds {
                return Ok(Value::Bool(false));
            }
            left = right;
            left_node = right_node;
        }
        Ok(Value::Bool(true))
    }

    fn arguments(&mut self, args: &'a [Expr]) -> Result<Vec<Value>, EvalError> {
        let mut values = Vec::with_capacity(args.len());
        for arg in args {
            values.push(self.eval(arg)?);
        }
        Ok(values)
    }

    fn call(&mut self, name: &str, args: &'a [Expr], span: Span) -> Outcome {
        if let Some(outcome) = self.predicate_call(name, args) {
            return outcome;
        }
        if is_lookup(name) {
            let values = self.arguments(args)?;
            let nodes: Vec<&Expr> = args.iter().collect();
            return self
                .lookup(name, &values)
                .map_err(|error| self.explain(error, &nodes, Zero::Number));
        }
        if !builtins::is_builtin(name) {
            return Err(unknown_name(name).at(span));
        }
        let values = self.arguments(args)?;
        let mut context = Context {
            meter: &mut self.meter,
            regexes: &mut self.regexes,
        };
        let outcome = builtins::call(name, &values, &mut context)
            .unwrap_or_else(|| Err(unknown_name(name)));
        outcome.map_err(|error| {
            let nodes: Vec<&Expr> = args.iter().collect();
            let zero = argument_zero(name, error.empty_argument);
            self.explain(error, &nodes, zero)
        })
    }

    /// The list a predicate function walks.
    fn items(&mut self, name: &str, args: &'a [Expr]) -> Result<Rc<List>, EvalError> {
        let Some(list) = args.first() else {
            return Err(builtins::arity(name, &[], 2, 2)
                .err()
                .unwrap_or_else(|| unknown_name(name)));
        };
        match self.eval(list)? {
            Value::List(items) => Ok(items),
            Value::Nil => Err(self.explain(
                expected_list("nil").empty_argument(0),
                &[list],
                Zero::List,
            )),
            other => Err(expected_list(other.kind_key()).at(list.span)),
        }
    }

    fn with_frame<T>(
        &mut self,
        frame: Frame,
        run: impl FnOnce(&mut Self) -> Result<T, EvalError>,
    ) -> Result<T, EvalError> {
        self.meter.step()?;
        self.frames.push(frame);
        let outcome = run(self);
        self.frames.pop();
        outcome
    }

    fn predicate_body(args: &'a [Expr], index: usize) -> Option<&'a Expr> {
        args.get(index).map(|arg| match &arg.kind {
            ExprKind::Predicate(predicate) => &*predicate.body,
            _ => arg,
        })
    }

    /// Runs `body` for one item and reads its answer as true or false.
    fn test(&mut self, body: &'a Expr, frame: Frame) -> Result<bool, EvalError> {
        self.with_frame(frame, |interpreter| interpreter.condition(body))
    }

    fn predicate_call(&mut self, name: &str, args: &'a [Expr]) -> Option<Outcome> {
        let (min, max) = match name {
            "all" | "none" | "any" | "one" | "filter" | "map" | "find" | "findIndex"
            | "findLast" | "findLastIndex" | "groupBy" => (2, 2),
            "count" => (1, 2),
            "sortBy" | "reduce" => (2, 3),
            _ => return None,
        };
        if !(min..=max).contains(&args.len()) {
            let placeholders = vec![Value::Nil; args.len()];
            return Some(builtins::arity(name, &placeholders, min, max).map(|()| Value::Nil));
        }
        Some(self.predicate(name, args))
    }

    fn frame(element: &Value, index: usize, accumulator: Value) -> Frame {
        Frame {
            element: element.clone(),
            index,
            accumulator,
        }
    }

    fn predicate(&mut self, name: &str, args: &'a [Expr]) -> Outcome {
        let list = self.items(name, args)?;
        let items = list.items();
        let body = Self::predicate_body(args, 1);
        match (name, body) {
            ("count", None) => {
                let mut count = 0i64;
                for item in items {
                    self.meter.step()?;
                    match item {
                        Value::Bool(true) => count += 1,
                        Value::Bool(false) => {}
                        other => return Err(expected_boolean(other.kind_key())),
                    }
                }
                Ok(Value::int(count))
            }
            (_, None) => Err(unknown_name(name)),
            ("all" | "none" | "any" | "one" | "count", Some(body)) => {
                let mut count = 0i64;
                for (index, item) in items.iter().enumerate() {
                    let holds = self.test(body, Self::frame(item, index, Value::Nil))?;
                    match (name, holds) {
                        ("all", false) => return Ok(Value::Bool(false)),
                        ("none" | "any", true) => return Ok(Value::Bool(name == "any")),
                        ("one", true) if count == 1 => return Ok(Value::Bool(false)),
                        (_, true) => count += 1,
                        _ => {}
                    }
                }
                Ok(match name {
                    "all" | "none" => Value::Bool(true),
                    "any" => Value::Bool(false),
                    "one" => Value::Bool(count == 1),
                    _ => Value::int(count),
                })
            }
            ("filter", Some(body)) => {
                let mut kept = Vec::new();
                for (index, item) in items.iter().enumerate() {
                    if self.test(body, Self::frame(item, index, Value::Nil))? {
                        kept.push(item.clone());
                    }
                }
                self.meter.items(kept.len())?;
                Value::list(kept).map_err(|_| too_deep())
            }
            ("map", Some(body)) => {
                let mut mapped = Vec::with_capacity(items.len());
                for (index, item) in items.iter().enumerate() {
                    let frame = Self::frame(item, index, Value::Nil);
                    mapped.push(self.with_frame(frame, |interpreter| interpreter.eval(body))?);
                }
                self.meter.items(mapped.len())?;
                Value::list(mapped).map_err(|_| too_deep())
            }
            ("find" | "findIndex", Some(body)) => {
                for (index, item) in items.iter().enumerate() {
                    if self.test(body, Self::frame(item, index, Value::Nil))? {
                        return Ok(if name == "find" {
                            item.clone()
                        } else {
                            Value::int(i64::try_from(index).unwrap_or(i64::MAX))
                        });
                    }
                }
                Ok(if name == "find" {
                    Value::Nil
                } else {
                    Value::int(-1)
                })
            }
            ("findLast" | "findLastIndex", Some(body)) => {
                for (index, item) in items.iter().enumerate().rev() {
                    if self.test(body, Self::frame(item, index, Value::Nil))? {
                        return Ok(if name == "findLast" {
                            item.clone()
                        } else {
                            Value::int(i64::try_from(index).unwrap_or(i64::MAX))
                        });
                    }
                }
                Ok(if name == "findLast" {
                    Value::Nil
                } else {
                    Value::int(-1)
                })
            }
            ("groupBy", Some(body)) => {
                let mut groups: BTreeMap<Rc<str>, Vec<Value>> = BTreeMap::new();
                for (index, item) in items.iter().enumerate() {
                    let frame = Self::frame(item, index, Value::Nil);
                    let key = self.with_frame(frame, |interpreter| interpreter.eval(body))?;
                    let key = operators::map_key(&key, 0).map_err(|mut error| {
                        error.empty_argument = None;
                        error.at(body.span)
                    })?;
                    groups.entry(Rc::from(key)).or_default().push(item.clone());
                }
                self.meter.items(items.len() + groups.len())?;
                let mut entries = BTreeMap::new();
                for (key, members) in groups {
                    entries.insert(key, Value::list(members).map_err(|_| too_deep())?);
                }
                Value::map(entries).map_err(|_| too_deep())
            }
            ("sortBy", Some(body)) => {
                let order = match args.get(2) {
                    Some(order) => Some(self.eval(order)?),
                    None => None,
                };
                let descending = builtins::sort_order(order.as_ref())?;
                let mut keys = Vec::with_capacity(items.len());
                for (index, item) in items.iter().enumerate() {
                    let frame = Self::frame(item, index, Value::Nil);
                    keys.push(self.with_frame(frame, |interpreter| interpreter.eval(body))?);
                }
                let mut context = Context {
                    meter: &mut self.meter,
                    regexes: &mut self.regexes,
                };
                builtins::sort_by_keys(items, &keys, descending, &mut context)
                    .map_err(|error| error.at(body.span))
            }
            ("reduce", Some(body)) => {
                let (mut accumulator, start) = match args.get(2) {
                    Some(initial) => (self.eval(initial)?, 0),
                    None => match items.first() {
                        Some(first) => (first.clone(), 1),
                        None => {
                            return Err(EvalError::new(
                                Code::Argument,
                                Message::new(crate::message!(
                                    "reduce of empty array with no initial value"
                                )),
                            ));
                        }
                    },
                };
                for (index, item) in items.iter().enumerate().skip(start) {
                    let frame = Self::frame(item, index, accumulator);
                    accumulator = self.with_frame(frame, |interpreter| interpreter.eval(body))?;
                }
                Ok(accumulator)
            }
            _ => Err(unknown_name(name)),
        }
    }

    fn lookup(&mut self, name: &str, args: &[Value]) -> Outcome {
        let (function, arity): (&'static str, usize) = match name {
            "lookup" => ("lookup", 2),
            "lookupOr" => ("lookupOr", 3),
            "lookup2" => ("lookup2", 3),
            "lookup2Or" => ("lookup2Or", 4),
            "lookupInterp" => ("lookupInterp", 2),
            _ => ("deficitWeight", 2),
        };
        builtins::arity(function, args, arity, arity)?;
        let table = match &args[0] {
            Value::Text(table) => Rc::clone(table),
            other => return Err(expected_text(other.kind_key())),
        };
        let keys: Vec<Value> = match function {
            "lookup2" | "lookup2Or" => args[1..3].to_vec(),
            _ => args[1..2].to_vec(),
        };
        let fallback = match function {
            "lookupOr" => Some(builtins::number_arg(args, 2)?),
            "lookup2Or" => Some(builtins::number_arg(args, 3)?),
            _ => None,
        };
        let answer: Result<(Decimal, Option<Matched>), EvalError> = match self.lookups {
            LookupSource::Unavailable => Err(EvalError::with(
                Code::LookupUnavailable,
                crate::message!("rate table lookups are not available in this context: \"{0}\""),
                [Arg::from(&*table)],
            )),
            LookupSource::Stub => Ok((Decimal::ZERO, None)),
            LookupSource::Tables(tables) => match function {
                "lookup" | "lookupOr" => tables
                    .lookup(&table, &keys[0])
                    .map(|(value, matched)| (value, Some(matched))),
                "lookup2" | "lookup2Or" => tables
                    .lookup2(&table, &keys[0], &keys[1])
                    .map(|(value, matched)| (value, Some(matched))),
                "lookupInterp" => tables.interpolate(&table, &keys[0]).map(|value| {
                    (
                        value,
                        Some(Matched {
                            key: Some(String::from("lookupInterp")),
                            ..Matched::default()
                        }),
                    )
                }),
                _ => tables.deficit_weight(&table, &keys[0]).map(|value| {
                    (
                        value,
                        Some(Matched {
                            key: Some(String::from("deficitWeight")),
                            ..Matched::default()
                        }),
                    )
                }),
            },
        };
        if self.record {
            let (value, matched, error) = match &answer {
                Ok((value, matched)) => (Some(*value), matched.clone(), None),
                Err(error) => {
                    let mut recorded = error.clone();
                    recorded.empty_argument = None;
                    (None, None, Some(recorded))
                }
            };
            self.records.push(LookupRecord {
                function,
                table: table.to_string(),
                keys,
                value,
                matched,
                error,
            });
        }
        match (answer, fallback) {
            (Ok((value, _)), _) => Ok(Value::Number(value)),
            (Err(error), Some(fallback)) if error.code == Code::LookupMiss => {
                Ok(Value::Number(fallback))
            }
            (Err(error), _) => Err(error),
        }
    }
}

fn is_lookup(name: &str) -> bool {
    matches!(
        name,
        "lookup" | "lookupOr" | "lookup2" | "lookup2Or" | "lookupInterp" | "deficitWeight"
    )
}

fn comparison(op: CompareOp) -> Comparison {
    match op {
        CompareOp::Less => Comparison::Less,
        CompareOp::LessEqual => Comparison::LessOrEqual,
        CompareOp::Greater => Comparison::Greater,
        CompareOp::GreaterEqual => Comparison::GreaterOrEqual,
    }
}

fn text_value(value: &Value, side: usize) -> Result<&str, EvalError> {
    match value {
        Value::Text(text) => Ok(text),
        Value::Nil => Err(expected_text("nil").empty_argument(side)),
        other => Err(expected_text(other.kind_key())),
    }
}

fn unknown_name(name: &str) -> EvalError {
    EvalError::with(
        Code::UnknownName,
        crate::message!("unknown name {0}"),
        [Arg::from(name)],
    )
}

/// What an empty argument of `function` should have been.
fn argument_zero(function: &str, argument: Option<usize>) -> Zero {
    match (function, argument) {
        (
            "upper" | "lower" | "trim" | "trimPrefix" | "trimSuffix" | "hasPrefix" | "hasSuffix"
            | "indexOf" | "lastIndexOf" | "replace" | "split" | "splitAfter" | "date" | "duration",
            _,
        )
        | ("join", Some(1)) => Zero::Text,
        ("first" | "last" | "take" | "reverse" | "uniq" | "concat" | "flatten" | "sort"
        | "join", _) => Zero::List,
        ("keys" | "values" | "toPairs", _) => Zero::Map,
        _ => Zero::Number,
    }
}
