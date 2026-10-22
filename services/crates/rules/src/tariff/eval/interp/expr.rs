//! Expressions.

use super::{Env, Interp, refusal, unknown_name};
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::eval::error::{Code, EvalError, Stop};
use crate::tariff::eval::operators::{self, scales_a_percent};
use crate::tariff::eval::time::{Date, DateTime};
use crate::tariff::eval::trace::Event;
use crate::tariff::eval::value::{Value, dollars};
use crate::tariff::eval::{lists, methods, text};
use crate::tariff::syntax::ast::{
    BinaryOp, Condition, DateLit, Expr, ExprKind, Text, TextPart, UnaryOp, UnitExpr,
};
use crate::tariff::types::Type;
use crate::tariff::units::Unit;
use rust_decimal::Decimal;
use rust_decimal::prelude::ToPrimitive;
use std::rc::Rc;

/// The roots a file's own code may read.
const ROOTS: [&str; 5] = ["shipment", "lane", "customer", "fuel", "rating"];

pub(super) fn internal(template: &'static str, args: impl IntoIterator<Item = Arg>) -> Stop {
    Stop::Failed(EvalError::internal(Message::with(template, args)))
}

fn date(literal: DateLit) -> Option<Date> {
    Date::from_civil(
        i64::from(literal.year),
        i64::from(literal.month),
        i64::from(literal.day),
    )
}

/// A unit as written in the tree: `lb`, `$/mi`'s `mi`, `lb/ft3`.
pub(super) fn unit_of(unit: &UnitExpr) -> Result<Unit, Stop> {
    Unit::parse(&unit.text()).map_err(|error| Stop::Failed(operators::unit_error(&error)))
}

/// The unit an `.in(…)` argument names: `cwt` or `lb/ft3`, written as names.
pub(super) fn unit_argument(expr: &Expr) -> Option<Unit> {
    fn text(expr: &Expr) -> Option<String> {
        match &expr.kind {
            ExprKind::Name(name) => Some(name.text.clone()),
            ExprKind::Binary {
                op: BinaryOp::Div,
                left,
                right,
                ..
            } => Some(format!("{}/{}", text(left)?, text(right)?)),
            _ => None,
        }
    }
    Unit::parse(&text(expr)?).ok()
}

impl<'a> Interp<'_, 'a> {
    /// Evaluates an expression in `env`: what code outside the interpreter (inputs' defaults,
    /// the test runner's `given`s) runs through.
    pub fn eval_in(&mut self, env: &mut Env, expr: &'a Expr) -> Result<Value, Stop> {
        self.eval(env, expr)
    }

    /// A text literal with its interpolations filled in, in `env`.
    pub fn interpolate_in(&mut self, env: &mut Env, text: &'a Text) -> Result<String, Stop> {
        self.interpolate(env, text)
    }

    /// Evaluates an expression.
    pub(super) fn eval(&mut self, env: &mut Env, expr: &'a Expr) -> Result<Value, Stop> {
        self.meter
            .step()
            .map_err(|error| Stop::Failed(error.at(expr.span)))?;
        self.eval_kind(env, expr).map_err(|stop| stop.at(expr.span))
    }

    fn eval_kind(&mut self, env: &mut Env, expr: &'a Expr) -> Result<Value, Stop> {
        match &expr.kind {
            ExprKind::Number { .. }
            | ExprKind::Money { .. }
            | ExprKind::Percent(_)
            | ExprKind::Date(_)
            | ExprKind::DateTime(_) => literal(&expr.kind),
            ExprKind::Text(text) => self.interpolate(env, text).map(|text| Value::text(&text)),
            ExprKind::Bool(flag) => Ok(Value::Bool(*flag)),
            ExprKind::None => Ok(Value::None),
            ExprKind::Name(name) => self.name(env, &name.text),
            ExprKind::Path { head, member } => Ok(Value::variant(
                member
                    .as_ref()
                    .map_or(head.text.as_str(), |member| member.text.as_str()),
            )),
            ExprKind::Field {
                target,
                name,
                optional,
            } => self.field(env, target, &name.text, *optional),
            ExprKind::Call { callee, args } => self.call(env, expr, callee, args),
            ExprKind::Index { target, keys } => self.index(env, expr, target, keys),
            ExprKind::Unary { op, operand, .. } => {
                let value = self.eval(env, operand)?;
                match op {
                    UnaryOp::Neg => operators::neg(&value).map_err(Stop::Failed),
                    UnaryOp::Not => match value {
                        Value::Bool(flag) => Ok(Value::Bool(!flag)),
                        other => Err(not_bool(&other)),
                    },
                }
            }
            ExprKind::Binary {
                op, left, right, ..
            } => self.binary(env, expr, *op, left, right),
            ExprKind::Fail(text) => {
                let message = self.interpolate(env, text)?;
                Err(refusal(message, self.root_span(env, expr.span)))
            }
            ExprKind::List(items) => {
                self.meter.items(items.len()).map_err(Stop::Failed)?;
                let mut values = Vec::with_capacity(items.len());
                for item in items {
                    values.push(self.eval(env, item)?);
                }
                Ok(Value::list(values))
            }
            ExprKind::Record(fields) => {
                self.meter.items(fields.len()).map_err(Stop::Failed)?;
                let mut values = Vec::with_capacity(fields.len());
                for field in fields {
                    values.push((field.name.text.as_str(), self.eval(env, &field.value)?));
                }
                Ok(Value::record(values))
            }
            ExprKind::If {
                branches,
                otherwise,
            } => {
                for (condition, value) in branches {
                    env.push();
                    let holds = self.condition(env, condition);
                    let result = match holds {
                        Ok(true) => Some(self.eval(env, value)),
                        Ok(false) => None,
                        Err(stop) => Some(Err(stop)),
                    };
                    env.pop();
                    if let Some(result) = result {
                        return result;
                    }
                }
                self.eval(env, otherwise)
            }
            ExprKind::Match { scrutinee, arms } => {
                let value = self.eval(env, scrutinee)?;
                self.match_value(env, &value, arms, expr.span)
            }
            ExprKind::Range { .. } => Err(internal(
                crate::message!("A range can only be walked by for or tested with in"),
                [],
            )),
            ExprKind::Lambda { .. } => Err(internal(
                crate::message!("A lambda can only be given to a list method"),
                [],
            )),
            ExprKind::Missing => Err(internal(
                crate::message!("The tariff is incomplete here"),
                [],
            )),
        }
    }

    /// Evaluates a `Bool` expression.
    pub(super) fn truth(&mut self, env: &mut Env, expr: &'a Expr) -> Result<bool, Stop> {
        match self.eval(env, expr)? {
            Value::Bool(flag) => Ok(flag),
            other => Err(not_bool(&other).at(expr.span)),
        }
    }

    /// Evaluates an `if`/`require` condition, binding `if let`'s name in the current scope.
    pub(super) fn condition(
        &mut self,
        env: &mut Env,
        condition: &'a Condition,
    ) -> Result<bool, Stop> {
        match condition {
            Condition::Expr(expr) => self.truth(env, expr),
            Condition::Let { name, value } => {
                let value = self.eval(env, value)?;
                if value.is_none() {
                    return Ok(false);
                }
                if self.traced(env) {
                    self.trace.push(Event::Bind {
                        name: name.text.clone(),
                        value: value.clone(),
                        span: name.span,
                    });
                }
                env.bind(&name.text, value, false);
                Ok(true)
            }
        }
    }

    /// A text literal with its interpolations filled in.
    pub(super) fn interpolate(&mut self, env: &mut Env, text: &'a Text) -> Result<String, Stop> {
        let mut out = String::new();
        for part in &text.parts {
            match part {
                TextPart::Literal(literal, _) => out.push_str(literal),
                TextPart::Expr(expr) => {
                    let value = self.eval(env, expr)?;
                    out.push_str(&value.display());
                }
            }
        }
        self.meter
            .text(out.len())
            .map_err(|error| Stop::Failed(error.at(text.span)))?;
        Ok(out)
    }

    fn name(&mut self, env: &mut Env, name: &str) -> Result<Value, Stop> {
        if let Some(value) = env.get(name) {
            return Ok(value.clone());
        }
        if !env.function {
            if ROOTS.contains(&name) {
                return self
                    .root(name)
                    .ok_or_else(|| Stop::Failed(unknown_name(name)));
            }
            if let Some(view) = &self.view {
                match name {
                    "total" => return Ok(Value::money(view.total)),
                    "lines" => return Ok(view.lines_value()),
                    _ => {}
                }
            }
        }
        if let Some((module, decl)) = self.linked.constant(env.module, name) {
            return self.constant(module, decl);
        }
        Err(Stop::Failed(unknown_name(name)))
    }

    /// A constant's value, evaluated once per run in its own module.
    pub(super) fn constant(
        &mut self,
        module: usize,
        decl: &'a crate::tariff::syntax::ast::Const,
    ) -> Result<Value, Stop> {
        let key = (module, decl.name.text.clone());
        if let Some(value) = self.consts.get(&key) {
            return Ok(value.clone());
        }
        if !self.evaluating.insert(key.clone()) {
            return Err(internal(
                crate::message!("The constant {0} is defined in terms of itself"),
                [Arg::from(decl.name.text.clone())],
            ));
        }
        let mut scope = Env::new(module, true);
        let result = self.eval(&mut scope, &decl.value);
        self.evaluating.remove(&key);
        let value = result?;
        self.consts.insert(key, value.clone());
        Ok(value)
    }

    /// `module.name` through a module alias, or `module.Enum.Variant`.
    fn module_member(
        &mut self,
        env: &Env,
        target: &'a Expr,
        name: &str,
    ) -> Option<Result<Value, Stop>> {
        match &target.kind {
            ExprKind::Name(alias) if env.get(&alias.text).is_none() => {
                let module = self.linked.alias(env.module, &alias.text)?;
                let (at, decl) = self.linked.constant(module, name)?;
                Some(self.constant(at, decl))
            }
            ExprKind::Field {
                target: inner,
                optional: false,
                ..
            } => match &inner.kind {
                ExprKind::Name(alias)
                    if env.get(&alias.text).is_none()
                        && self.linked.alias(env.module, &alias.text).is_some() =>
                {
                    Some(Ok(Value::variant(name)))
                }
                _ => None,
            },
            _ => None,
        }
    }

    fn field(
        &mut self,
        env: &mut Env,
        target: &'a Expr,
        name: &str,
        optional: bool,
    ) -> Result<Value, Stop> {
        if let ExprKind::Name(root) = &target.kind
            && root.text == "tables"
            && env.get("tables").is_none()
        {
            return Ok(Value::Table(Rc::from(name)));
        }
        if let Some(member) = self.module_member(env, target, name) {
            return member;
        }
        let value = self.eval(env, target)?;
        if optional && value.is_none() {
            return Ok(Value::None);
        }
        property(&value, name)
    }

    fn index(
        &mut self,
        env: &mut Env,
        expr: &'a Expr,
        target: &'a Expr,
        keys: &'a [Expr],
    ) -> Result<Value, Stop> {
        let target_value = self.eval(env, target)?;
        let mut values = Vec::with_capacity(keys.len());
        for key in keys {
            values.push(self.eval(env, key)?);
        }
        match (&target_value, values.as_slice()) {
            (Value::Table(code), _) => {
                let read = self
                    .tables
                    .index(code, &values, &mut self.meter)
                    .map_err(Stop::Failed)?;
                Ok(self.answer(env, read, expr.span))
            }
            (Value::List(items), [Value::Int(at)]) => {
                match at.to_usize().and_then(|at| items.get(at)) {
                    Some(item) => Ok(item.clone()),
                    None => self.out_of_range(env, expr, at),
                }
            }
            (Value::Text(content), [Value::Int(at)]) => match text::index(content, *at) {
                Some(character) => Ok(character),
                None => self.out_of_range(env, expr, at),
            },
            _ => Err(internal(
                crate::message!("{0} cannot be indexed like this"),
                [Arg::from(target_value.kind())],
            )),
        }
    }

    fn out_of_range(&self, env: &Env, expr: &Expr, at: &Decimal) -> Result<Value, Stop> {
        if matches!(
            self.static_type(env.module, expr.id),
            Some(Type::Optional(_))
        ) {
            return Ok(Value::None);
        }
        Err(Stop::Failed(EvalError::with(
            Code::Index,
            crate::message!("There is no item at position {0}"),
            [Arg::from(at.to_string())],
        )))
    }

    /// Records a table read and gives its value.
    pub(super) fn answer(
        &mut self,
        env: &Env,
        read: crate::tariff::eval::tables::Read,
        span: Span,
    ) -> Value {
        let span = self.root_span(env, span);
        self.record_needs(read.needs);
        self.trace.lookup(read.lookup, span);
        read.value
    }

    fn binary(
        &mut self,
        env: &mut Env,
        expr: &'a Expr,
        op: BinaryOp,
        left: &'a Expr,
        right: &'a Expr,
    ) -> Result<Value, Stop> {
        match op {
            BinaryOp::And | BinaryOp::Or => {
                let first = match self.eval(env, left)? {
                    Value::Bool(flag) => flag,
                    other => return Err(not_bool(&other).at(left.span)),
                };
                if first == (op == BinaryOp::Or) {
                    return Ok(Value::Bool(first));
                }
                match self.eval(env, right)? {
                    Value::Bool(flag) => Ok(Value::Bool(flag)),
                    other => Err(not_bool(&other).at(right.span)),
                }
            }
            BinaryOp::Coalesce => {
                let value = self.eval(env, left)?;
                if value.is_none() {
                    self.eval(env, right)
                } else {
                    Ok(value)
                }
            }
            BinaryOp::In | BinaryOp::NotIn => {
                let value = self.eval(env, left)?;
                let inside = self.contains(env, right, &value)?;
                Ok(Value::Bool(inside == (op == BinaryOp::In)))
            }
            _ => {
                let a = self.eval(env, left)?;
                let b = self.eval(env, right)?;
                self.arithmetic(env, expr, op, &a, &b).map_err(Stop::Failed)
            }
        }
    }

    fn arithmetic(
        &self,
        env: &Env,
        expr: &Expr,
        op: BinaryOp,
        a: &Value,
        b: &Value,
    ) -> Result<Value, EvalError> {
        use std::cmp::Ordering;
        let order = |wanted: &dyn Fn(Ordering) -> bool| -> Result<Value, EvalError> {
            if a.is_none() || b.is_none() {
                return Err(EvalError::internal(Message::with(
                    crate::message!("{0} and {1} have no order between them"),
                    [Arg::from(a.kind()), Arg::from(b.kind())],
                )));
            }
            Ok(Value::Bool(wanted(a.compare(b)?)))
        };
        match op {
            BinaryOp::Eq => a.equals(b).map(Value::Bool),
            BinaryOp::Ne => a.equals(b).map(|equal| Value::Bool(!equal)),
            BinaryOp::Lt => order(&Ordering::is_lt),
            BinaryOp::Le => order(&Ordering::is_le),
            BinaryOp::Gt => order(&Ordering::is_gt),
            BinaryOp::Ge => order(&Ordering::is_ge),
            BinaryOp::Add => operators::add(a, b),
            BinaryOp::Sub => operators::sub(a, b),
            BinaryOp::Mul => operators::mul(a, b, self.keeps_percent(env, expr, a, b)),
            BinaryOp::Div => operators::div(a, b, self.keeps_percent(env, expr, a, b)),
            BinaryOp::Rem => operators::rem(a, b),
            BinaryOp::Coalesce | BinaryOp::Or | BinaryOp::And | BinaryOp::In | BinaryOp::NotIn => {
                Err(EvalError::internal(Message::new(crate::message!(
                    "This operator is not arithmetic"
                ))))
            }
        }
    }

    /// Whether a percent scaled by a number stays a percent: as the static type says, else yes.
    fn keeps_percent(&self, env: &Env, expr: &Expr, a: &Value, b: &Value) -> bool {
        if !scales_a_percent(a, b) {
            return true;
        }
        !matches!(
            self.static_type(env.module, expr.id),
            Some(Type::Number | Type::Int)
        )
    }

    /// `value in right`: membership in a list, or in a range written in place.
    fn contains(&mut self, env: &mut Env, right: &'a Expr, value: &Value) -> Result<bool, Stop> {
        if let ExprKind::Range {
            start,
            end,
            inclusive,
        } = &right.kind
        {
            let start = match start {
                Some(start) => Some(self.eval(env, start)?),
                None => None,
            };
            let end = match end {
                Some(end) => Some(self.eval(env, end)?),
                None => None,
            };
            return in_range(value, start.as_ref(), end.as_ref(), *inclusive)
                .map_err(|error| Stop::Failed(error.at(right.span)));
        }
        match self.eval(env, right)? {
            Value::List(items) => {
                self.meter.steps(items.len()).map_err(Stop::Failed)?;
                for item in items.iter() {
                    if item.equals(value).map_err(Stop::Failed)? {
                        return Ok(true);
                    }
                }
                Ok(false)
            }
            other => Err(internal(
                crate::message!("in needs a list or a range, not {0}"),
                [Arg::from(other.kind())],
            )),
        }
    }
}

/// Whether `value` is in `start..end` (or `..=`); either end may be open. `none` is in no range.
pub(super) fn in_range(
    value: &Value,
    start: Option<&Value>,
    end: Option<&Value>,
    inclusive: bool,
) -> Result<bool, EvalError> {
    if value.is_none() {
        return Ok(false);
    }
    if let Some(start) = start
        && value.compare(start)?.is_lt()
    {
        return Ok(false);
    }
    if let Some(end) = end {
        let order = value.compare(end)?;
        if order.is_gt() || (order.is_eq() && !inclusive) {
            return Ok(false);
        }
    }
    Ok(true)
}

/// A number literal's value: an `Int` when no point was written and its unit only counts
/// (`3 pallet`), a percent, or a quantity.
fn literal_number(value: Decimal, integer: bool, unit: Unit) -> Value {
    if unit.dim().is_none() && !crate::tariff::eval::value::is_percent_unit(&unit) {
        return if integer {
            Value::Int(value)
        } else {
            Value::Number(value)
        };
    }
    if unit == dollars() {
        return Value::money(value);
    }
    Value::quantity(value, unit)
}

/// The value of a number, money, percent, date or date-time literal.
fn literal(kind: &ExprKind) -> Result<Value, Stop> {
    match kind {
        ExprKind::Number {
            value,
            integer,
            unit,
        } => {
            let unit = match unit {
                Some(unit) => unit_of(unit)?,
                None => Unit::none(),
            };
            Ok(literal_number(*value, *integer, unit))
        }
        ExprKind::Money { value, per } => Ok(match per {
            None => Value::money(*value),
            Some(per) => Value::quantity(
                *value,
                Unit::parse(&format!("$/{}", per.text()))
                    .map_err(|error| Stop::Failed(operators::unit_error(&error)))?,
            ),
        }),
        ExprKind::Percent(value) => Ok(Value::percent(*value)),
        ExprKind::Date(literal) => date(*literal)
            .map(Value::Date)
            .ok_or_else(|| Stop::Failed(EvalError::overflow())),
        ExprKind::DateTime(literal) => date(literal.date)
            .and_then(|day| {
                DateTime::from_local(
                    day,
                    i64::from(literal.hour),
                    i64::from(literal.minute),
                    i64::from(literal.second),
                    i32::from(literal.offset_minutes),
                )
            })
            .map(Value::DateTime)
            .ok_or_else(|| Stop::Failed(EvalError::overflow())),
        _ => Err(internal(crate::message!("This is not a literal"), [])),
    }
}

fn not_bool(value: &Value) -> Stop {
    internal(
        crate::message!("A condition must be true or false, not {0}"),
        [Arg::from(value.kind())],
    )
}

/// A field of a record, or a property of a list, text, quantity, date or date-time.
pub(super) fn property(value: &Value, name: &str) -> Result<Value, Stop> {
    let found = match value {
        Value::Record(record) => record.get(name).cloned(),
        Value::List(items) => lists::property(items, name),
        Value::Text(content) => text::property(content, name),
        other => methods::property(other, name),
    };
    found.ok_or_else(|| {
        internal(
            crate::message!("{0} has no field or property named {1}"),
            [Arg::from(value.kind()), Arg::from(name)],
        )
    })
}

impl super::TestView {
    /// `lines`: every line as `{ label, code, amount, kind }`.
    #[must_use]
    pub fn lines_value(&self) -> Value {
        Value::list(
            self.lines
                .iter()
                .map(|line| {
                    Value::record(vec![
                        (
                            "label",
                            Value::text(line.label.as_deref().unwrap_or_default()),
                        ),
                        (
                            "code",
                            line.code.as_deref().map_or(Value::None, Value::text),
                        ),
                        ("amount", Value::money(line.amount)),
                        ("kind", Value::variant(line.kind.variant())),
                    ])
                })
                .collect(),
        )
    }
}
