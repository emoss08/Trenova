//! A property's axes: the values each `vary` takes, in order, as its cases use them.
//!
//! - **A range** takes its start and its last value (the end, or the last step before an
//!   excluded end); every landmark inside it with its neighbours one step either side; and
//!   evenly spread values, splitting the widest gap first, until it has [`POINTS`] values. The
//!   step is 1 in the last decimal place the range's ends are written with, a day for dates
//!   and a minute for date-times. At most [`MAX_POINTS`] values are kept, the ends and the
//!   lowest landmarks first.
//! - **A list** takes every item, in order.
//! - **Without `in`**: `false` and `true`, every variant in declaration order, then `none` for
//!   an optional; or an input's constraint, read from its declaration.
//!
//! A **landmark** ([`landmarks`]) is a value the file or its libraries write down: every
//! number, money, percent, date and date-time literal, every constant, every input default,
//! and every band edge of a range-keyed rate table axis the request carries. An axis uses the
//! landmarks of its own kind and dimension, converted into its start's unit.

use super::super::StaticTypes;
use super::super::budget::{Limits, Meter};
use super::super::error::EvalError;
use super::super::interp::link::Linked;
use super::super::interp::{Env, Interp, Setup, literal_value};
use super::super::operators;
use super::super::tables::Tables;
use super::super::time::{Date, DateTime};
use super::super::value::Value;
use crate::message::Message;
use crate::tariff::schema::Schema;
use crate::tariff::syntax::ast::{Expr, ExprKind, Input, ItemKind, Name, Stmt, StmtKind, UnaryOp};
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::syntax::walk;
use crate::tariff::types::Type;
use crate::tariff::units::Unit;
use rust_decimal::Decimal;
use rust_decimal::prelude::ToPrimitive;
use std::collections::BTreeMap;

/// The values a range axis is filled to.
pub const POINTS: usize = 24;

/// The most values one axis keeps.
pub const MAX_POINTS: usize = 256;

/// The most landmarks gathered from a program.
const MAX_LANDMARKS: usize = 4_096;

/// What a `vary` sets.
#[derive(Debug, Clone)]
pub enum Target<'a> {
    /// A record path: `shipment.weight`.
    Path(&'a [Name]),
    /// `input.name`.
    Input(String),
    /// A new name.
    Name(String),
}

/// One `vary`: its target and the values its cases take.
#[derive(Debug)]
pub struct Axis<'a> {
    /// What it sets.
    pub target: Target<'a>,
    /// The target as written: `shipment.weight`, `input.rate`, `w`.
    pub label: String,
    /// The type of one value, for writing it as source.
    pub ty: Type,
    /// The values, in order: ascending for a range, as written for a list.
    pub points: Vec<Value>,
    /// Whether each value is an end, a landmark or a listed item rather than a filler.
    pub marked: Vec<bool>,
    /// A range's number line, for searching between its values.
    pub line: Option<Line>,
}

/// A range axis's values as positions on a number line, in steps from its start.
#[derive(Debug, Clone)]
pub struct Line {
    model: Model,
    /// The start's position.
    pub start: Decimal,
    /// One step.
    pub step: Decimal,
}

/// How a range's values map to positions.
#[derive(Debug, Clone)]
enum Model {
    /// Money, a rate, a percent or a quantity, in this unit.
    Quantity(Unit),
    /// Whole numbers.
    Int,
    /// Numbers.
    Number,
    /// Dates, by day.
    Date,
    /// Date-times, by second since the epoch, shown at this UTC offset.
    DateTime(i32),
}

impl Model {
    fn of(value: &Value) -> Option<Self> {
        match value {
            Value::Quantity(quantity) => Some(Self::Quantity(quantity.unit.clone())),
            Value::Int(_) => Some(Self::Int),
            Value::Number(_) => Some(Self::Number),
            Value::Date(_) => Some(Self::Date),
            Value::DateTime(at) => Some(Self::DateTime(at.offset_minutes())),
            _ => None,
        }
    }

    fn position(&self, value: &Value) -> Option<Decimal> {
        match (self, value) {
            (Self::Quantity(unit), Value::Quantity(quantity)) => {
                if quantity.unit.dim() != unit.dim() {
                    return None;
                }
                quantity.to(unit).ok().map(|converted| converted.value)
            }
            (Self::Int, Value::Int(number))
            | (Self::Number, Value::Int(number) | Value::Number(number)) => Some(*number),
            (Self::Date, Value::Date(date)) => Some(Decimal::from(date.days())),
            (Self::DateTime(_), Value::DateTime(at)) => Some(Decimal::from(at.unix())),
            _ => None,
        }
    }

    fn value(&self, position: Decimal) -> Option<Value> {
        match self {
            Self::Quantity(unit) => Some(Value::quantity(position.normalize(), unit.clone())),
            Self::Int => position.fract().is_zero().then_some(Value::Int(position.trunc())),
            Self::Number => Some(Value::Number(position.normalize())),
            Self::Date => Date::from_civil(1970, 1, 1)?
                .add_days(position.to_i64()?)
                .map(Value::Date),
            Self::DateTime(offset) => {
                DateTime::from_instant(position.to_i64()?, *offset).map(Value::DateTime)
            }
        }
    }

    /// One step: 1 in the last decimal place either end is written with, a day, a minute.
    fn step(&self, low: &Value, high: &Value) -> Decimal {
        let scale = |value: &Value| match value {
            Value::Quantity(quantity) => quantity.value.scale(),
            Value::Number(number) | Value::Int(number) => number.scale(),
            _ => 0,
        };
        match self {
            Self::Date | Self::Int => Decimal::ONE,
            Self::DateTime(_) => Decimal::from(60),
            Self::Quantity(_) | Self::Number => {
                Decimal::new(1, scale(low).max(scale(high)).min(20))
            }
        }
    }
}

impl Line {
    /// The value at a position.
    #[must_use]
    pub fn value(&self, position: Decimal) -> Option<Value> {
        self.model.value(position)
    }

    /// A value's position, when it is of this line's kind.
    #[must_use]
    pub fn position(&self, value: &Value) -> Option<Decimal> {
        self.model.position(value)
    }

    /// The last step at or below `position`.
    fn floor(&self, position: Decimal) -> Option<Decimal> {
        let steps = position
            .checked_sub(self.start)?
            .checked_div(self.step)?
            .floor();
        self.start.checked_add(steps.checked_mul(self.step)?)
    }

    /// A step strictly between `low` and `high`, near their middle; `None` when they are
    /// neighbours.
    #[must_use]
    pub fn between(&self, low: Decimal, high: Decimal) -> Option<Decimal> {
        let middle = low.checked_add(high)?.checked_div(Decimal::TWO)?;
        let mut found = self.floor(middle)?;
        if found <= low {
            found = found.checked_add(self.step)?;
        }
        (found < high).then_some(found)
    }
}

/// Every landmark a program writes down: literals, constants and input defaults in every
/// module, and the request's table band edges.
#[must_use]
pub fn landmarks(linked: &Linked<'_>, types: &dyn StaticTypes, tables: &Tables) -> Vec<Value> {
    let program = linked.program();
    let mut out: Vec<Value> = Vec::new();
    for module in &program.modules {
        walk::exprs(&module.file, &mut |expr| {
            if out.len() < MAX_LANDMARKS
                && let Some(value) = written(expr)
            {
                out.push(value);
            }
        });
    }
    let mut interp = Interp::new(Setup {
        linked,
        types,
        tables,
        record: Value::None,
        meter: Meter::new(Limits::DEFAULT),
        trace: false,
    });
    for (index, module) in program.modules.iter().enumerate() {
        for item in &module.file.items {
            let value = match &item.kind {
                ItemKind::Const(decl) => &decl.value,
                ItemKind::Input(Input {
                    default: Some(default),
                    ..
                }) => default,
                _ => continue,
            };
            if out.len() >= MAX_LANDMARKS {
                break;
            }
            if let Ok(found) = interp.eval_in(&mut Env::new(index, true), value) {
                out.push(found);
            }
        }
    }
    out.extend(tables.band_edges());
    out
}

/// The value a literal (or a negated literal) writes.
fn written(expr: &Expr) -> Option<Value> {
    match &expr.kind {
        ExprKind::Unary {
            op: UnaryOp::Neg,
            operand,
            ..
        } => operators::neg(&literal_value(operand)?).ok(),
        _ => literal_value(expr),
    }
}

/// What building an axis reads.
pub struct Sources<'s, 'a> {
    /// The checker's types.
    pub types: &'s dyn StaticTypes,
    /// The schema, for a path's type.
    pub schema: &'s Schema,
    /// The root file's inputs, for an input's type and constraint.
    pub inputs: Vec<&'a Input>,
    /// The program's landmarks.
    pub landmarks: &'s [Value],
}

impl std::fmt::Debug for Sources<'_, '_> {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("Sources")
            .field("landmarks", &self.landmarks.len())
            .finish_non_exhaustive()
    }
}

/// Builds one axis from a `vary`, evaluating its values with `eval`.
///
/// # Errors
///
/// When a value cannot be evaluated, or the values are not what the checker promised.
pub fn axis<'a>(
    stmt: &'a Stmt,
    sources: &Sources<'_, 'a>,
    eval: &mut dyn FnMut(&'a Expr) -> Result<Value, EvalError>,
) -> Result<Axis<'a>, EvalError> {
    let StmtKind::Vary { target, values } = &stmt.kind else {
        return Err(not_varied());
    };
    let label = target
        .iter()
        .map(|part| part.text.as_str())
        .collect::<Vec<_>>()
        .join(".");
    let (target, declared) = match target.as_slice() {
        [head, name] if head.text == "input" => (
            Target::Input(name.text.clone()),
            sources.types.input_type(&name.text),
        ),
        [name] => (Target::Name(name.text.clone()), None),
        path => (Target::Path(path), path_type(sources.schema, path)),
    };
    let input = match &target {
        Target::Input(name) => sources
            .inputs
            .iter()
            .find(|input| input.name.text == *name)
            .copied(),
        _ => None,
    };
    let values = match (values, input.and_then(|input| input.constraint.as_ref())) {
        (Some(values), _) | (None, Some(values)) => Some(values),
        (None, None) => None,
    };
    let (points, marked, line, written) = match values.map(|values| &values.kind) {
        Some(ExprKind::Range {
            start: Some(start),
            end: Some(end),
            inclusive,
        }) => {
            let low = eval(start)?;
            let high = eval(end)?;
            let (points, marked, line) = range(&low, &high, *inclusive, sources.landmarks)?;
            (points, marked, Some(line), Some(&**start))
        }
        Some(ExprKind::List(items)) => {
            let mut points = Vec::with_capacity(items.len().min(MAX_POINTS));
            for item in items.iter().take(MAX_POINTS) {
                points.push(eval(item)?);
            }
            let marked = vec![true; points.len()];
            (points, marked, None, items.first())
        }
        Some(_) => return Err(not_varied()),
        None => {
            let points = implicit(declared.as_ref()).ok_or_else(not_varied)?;
            let marked = vec![true; points.len()];
            (points, marked, None, None)
        }
    };
    if points.is_empty() {
        return Err(EvalError::internal(Message::new(crate::message!(
            "This vary takes no values"
        ))));
    }
    let ty = declared
        .map(|ty| match (&line, ty) {
            (Some(_), Type::Optional(inner)) => (*inner).clone(),
            (_, ty) => ty,
        })
        .or_else(|| written.and_then(|expr| sources.types.type_of(ModuleId::ROOT, expr.id)))
        .unwrap_or(Type::Error);
    Ok(Axis {
        target,
        label,
        ty,
        points,
        marked,
        line,
    })
}

fn not_varied() -> EvalError {
    EvalError::internal(Message::new(crate::message!(
        "A vary takes a range with both ends, a list, or a Bool, enum or constrained input"
    )))
}

/// The declared type at a record path.
fn path_type(schema: &Schema, path: &[Name]) -> Option<Type> {
    let (head, rest) = path.split_first()?;
    let mut ty = schema.root(&head.text)?.clone();
    for part in rest {
        let Type::Record(record) = ty.unwrapped() else {
            return None;
        };
        ty = record.field(&part.text)?.ty.clone();
    }
    Some(ty)
}

/// The values of a `vary` without `in`: both `Bool`s or every variant, then `none` for an
/// optional.
fn implicit(declared: Option<&Type>) -> Option<Vec<Value>> {
    let declared = declared?;
    let mut out = match declared.unwrapped() {
        Type::Bool => vec![Value::Bool(false), Value::Bool(true)],
        Type::Enum(enumeration) => enumeration
            .variants
            .iter()
            .map(|(name, _)| Value::variant(name))
            .collect(),
        _ => return None,
    };
    if declared.is_optional() {
        out.push(Value::None);
    }
    Some(out)
}

/// A range's values: its ends, its landmarks with their neighbours, and evenly spread fill.
fn range(
    low: &Value,
    high: &Value,
    inclusive: bool,
    landmarks: &[Value],
) -> Result<(Vec<Value>, Vec<bool>, Line), EvalError> {
    let empty = || {
        EvalError::internal(Message::new(crate::message!(
            "This range holds no values a property can vary over"
        )))
    };
    let model = Model::of(low).ok_or_else(empty)?;
    let (Some(start), Some(end)) = (model.position(low), model.position(high)) else {
        return Err(empty());
    };
    let line = Line {
        step: model.step(low, high),
        model,
        start,
    };
    let last = if inclusive {
        end
    } else {
        let floor = line.floor(end).ok_or_else(empty)?;
        if floor >= end {
            floor.checked_sub(line.step).ok_or_else(empty)?
        } else {
            floor
        }
    };
    if last < start {
        return Err(empty());
    }
    let mut points: BTreeMap<Decimal, bool> = BTreeMap::new();
    points.insert(start, true);
    points.insert(last, true);
    for landmark in landmarks {
        let Some(at) = line.position(landmark) else {
            continue;
        };
        if at < start || at > last {
            continue;
        }
        for near in [at.checked_sub(line.step), Some(at), at.checked_add(line.step)]
            .into_iter()
            .flatten()
        {
            if near >= start && near <= last {
                points.insert(near, true);
            }
        }
    }
    let mut points = trim(points, start, last);
    fill(&mut points, &line);
    let mut values = Vec::with_capacity(points.len());
    let mut marked = Vec::with_capacity(points.len());
    for (position, mark) in points {
        if let Some(value) = line.value(position) {
            values.push(value);
            marked.push(mark);
        }
    }
    Ok((values, marked, line))
}

/// At most [`MAX_POINTS`] values: both ends, then the lowest of the others.
fn trim(points: BTreeMap<Decimal, bool>, start: Decimal, last: Decimal) -> BTreeMap<Decimal, bool> {
    if points.len() <= MAX_POINTS {
        return points;
    }
    let mut kept: BTreeMap<Decimal, bool> = BTreeMap::new();
    kept.insert(start, true);
    kept.insert(last, true);
    for (position, mark) in points {
        if kept.len() >= MAX_POINTS {
            break;
        }
        kept.insert(position, mark);
    }
    kept
}

/// Adds evenly spread values until there are [`POINTS`], splitting the widest gap first (the
/// lowest of equal gaps), on the line's steps.
fn fill(points: &mut BTreeMap<Decimal, bool>, line: &Line) {
    for _ in 0..MAX_POINTS {
        if points.len() >= POINTS {
            return;
        }
        let positions: Vec<Decimal> = points.keys().copied().collect();
        let mut widest: Option<(Decimal, Decimal)> = None;
        for pair in positions.windows(2) {
            let [low, high] = pair else {
                continue;
            };
            let Some(inside) = line.between(*low, *high) else {
                continue;
            };
            let width = *high - *low;
            if widest.is_none_or(|(best, _)| width > best) {
                widest = Some((width, inside));
            }
        }
        let Some((_, inside)) = widest else {
            return;
        };
        points.insert(inside, false);
    }
}

#[cfg(test)]
mod tests {
    use super::{Line, Model, range};
    use crate::tariff::eval::value::Value;
    use crate::tariff::units::Unit;
    use rust_decimal::Decimal;
    use std::str::FromStr;

    fn q(value: &str, unit: &str) -> Value {
        Value::quantity(
            Decimal::from_str(value).unwrap_or_default(),
            Unit::parse(unit).unwrap_or_default(),
        )
    }

    fn shown(values: &[Value]) -> Vec<String> {
        values.iter().map(Value::display).collect()
    }

    #[test]
    fn a_range_with_no_landmarks_spreads_evenly() {
        let (points, marked, _) =
            range(&q("1", "lb"), &q("101", "lb"), false, &[]).unwrap_or_default_parts();
        assert_eq!(points.len(), 24);
        assert_eq!(shown(&points).first().map(String::as_str), Some("1 lb"));
        assert_eq!(shown(&points).last().map(String::as_str), Some("100 lb"));
        assert_eq!(marked.iter().filter(|mark| **mark).count(), 2);
    }

    #[test]
    fn a_short_range_takes_every_step() {
        let (points, _, _) =
            range(&q("1", "lb"), &q("5", "lb"), true, &[]).unwrap_or_default_parts();
        assert_eq!(shown(&points), ["1 lb", "2 lb", "3 lb", "4 lb", "5 lb"]);
    }

    #[test]
    fn steps_follow_the_written_precision() {
        let line = Line {
            model: Model::Quantity(Unit::parse("$/mi").unwrap_or_default()),
            start: Decimal::from_str("0.50").unwrap_or_default(),
            step: Decimal::new(1, 2),
        };
        assert_eq!(
            line.between(Decimal::new(50, 2), Decimal::new(51, 2)),
            None
        );
        assert_eq!(
            line.between(Decimal::new(50, 2), Decimal::new(60, 2)),
            Some(Decimal::new(55, 2))
        );
    }

    trait Parts {
        fn unwrap_or_default_parts(self) -> (Vec<Value>, Vec<bool>, Option<Line>);
    }

    impl Parts for Result<(Vec<Value>, Vec<bool>, Line), crate::tariff::eval::error::EvalError> {
        fn unwrap_or_default_parts(self) -> (Vec<Value>, Vec<bool>, Option<Line>) {
            match self {
                Ok((points, marked, line)) => (points, marked, Some(line)),
                Err(_) => (Vec::new(), Vec::new(), None),
            }
        }
    }
}
