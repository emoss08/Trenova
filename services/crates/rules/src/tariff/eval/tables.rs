//! Rate tables: `tables.Code[k, …]`, `.cell(…)`, `.band(x)` and `.interpolate(x)`.
//!
//! The host sends each active rate matrix as a [`TableShape`] under `context.tables`, with its
//! cells:
//!
//! ```json
//! {"code": "Zones", "valueType": "Money/cwt",
//!  "axes": [{"match": "Exact", "kind": "Zip3", "normalization": "Zip3"},
//!           {"match": "Range", "kind": "WeightBreak", "unit": "lb", "overflow": "ClampToTopBand"}],
//!  "cells": [{"at": [{"key": "752"}, {"min": "0", "max": "5000"}], "value": "18.50",
//!             "minimum": "75.00", "deficit": false}],
//!  "loaded": [["752"]]}
//! ```
//!
//! How a key finds its cell, as Go's `matrix_lookup.go` did, generalised to one to four axes:
//!
//! - An **exact** axis compares normalised keys: `Trim`, `Upper` (trim then upper-case), `Zip3`
//!   (trim, upper-case, first three characters) or `None`. A `FreightClass` axis first reads a
//!   key as one of the eighteen NMFC classes (`85`, `"77.5"`, `"Class92_5"`, the `FreightClass`
//!   variant), falling back to the normalisation. Stored keys and looked-up keys pass through
//!   the same rule. Numbers become keys in plain notation; `true` and `false` become text.
//! - A **range** axis matches bands half-open, `[min, max)`; a band without a maximum is
//!   open-ended. A quantity key is first converted into the axis's `unit` (`lb` for weight
//!   breaks, `mi` for distance and `ft` for linear feet when the axis names none), so a key of
//!   `5 cwt` matches the band holding 500 lb.
//! - A key no band covers follows the axis's overflow policy, over the bands still in play:
//!   `Error` misses, `ClampToTopBand` prices anything at or past the top band's floor at the top
//!   band, `Nearest` also takes the bottom band below the first floor and the closest band in a
//!   gap.
//! - Axes narrow the cells in position order. Cells are kept sorted by their band floors (then
//!   ceilings, open-ended last), stably, so when overlapping bands leave several cells the one
//!   with the lowest floors wins, then the one sent first: the tie-break is deterministic and the
//!   same everywhere.
//!
//! - An exact-axis key may be a list (`tables.Zones[lane.origin.zones, lane.destination.zones]`):
//!   the combinations are tried in list order, the first axis outermost, and the first that
//!   matches wins.
//!
//! **Cells load on demand.** A table can hold hundreds of thousands of cells, so the host sends
//! only what it expects a rating to read: `loaded` lists the exact-axis key combinations whose
//! cells `cells` holds (absent: `cells` is the whole table; a table with no exact axes is always
//! sent whole). A lookup of a combination the request did not carry is a [`Need`]: it gives a
//! provisional miss and the program keeps running, so one pass finds every need it can. A run
//! that recorded needs answers `needs` instead of a bill; the host loads those cells and calls
//! again.
//!
//! A malformed table (a cell without one position per axis, an exact position without a key, a
//! band without a floor, a value that does not fit the value type) fails only the programs that
//! read it, with code `table`. Every lookup is recorded for the trace: the table, the keys, the
//! cell or band matched (or the miss) and the value.

use super::budget::Meter;
use super::error::{Code, EvalError};
use super::value::{Value, format_decimal};
use crate::message::{Arg, Message};
use crate::tariff::schema::{Axis, TableShape};
use crate::tariff::types::Type;
use crate::tariff::units::{Base, Quantity, Unit};
use crate::wire::Dec;
use rust_decimal::Decimal;
use serde::Deserialize;
use serde_json::{Map, Value as Json};
use std::cmp::Ordering;
use std::collections::{BTreeMap, BTreeSet};
use std::rc::Rc;

/// Where a cell sits on one axis: `{"key": "752"}` on an exact axis, `{"min": "0", "max":
/// "5000"}` on a range axis (no `max` is open-ended).
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct AtWire {
    /// The key on an exact axis.
    #[serde(default)]
    pub key: Option<String>,
    /// The band floor on a range axis.
    #[serde(default)]
    pub min: Option<Dec>,
    /// The band ceiling on a range axis; absent is open-ended.
    #[serde(default)]
    pub max: Option<Dec>,
}

/// One cell as the host sends it.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct CellWire {
    /// One position per axis.
    pub at: Vec<AtWire>,
    /// The cell's value, in the table's value type.
    pub value: Dec,
    /// The cell's minimum charge.
    #[serde(default)]
    pub minimum: Option<Dec>,
    /// Whether the cell allows deficit rating.
    #[serde(default)]
    pub deficit: bool,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Normalization {
    None,
    Trim,
    Upper,
    Zip3,
}

impl Normalization {
    fn apply(self, key: &str) -> String {
        match self {
            Self::None => key.to_owned(),
            Self::Trim => key.trim().to_owned(),
            Self::Upper => key.trim().to_uppercase(),
            Self::Zip3 => key.trim().to_uppercase().chars().take(3).collect(),
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Overflow {
    Error,
    ClampToTopBand,
    Nearest,
}

/// What a range axis's bounds are.
#[derive(Debug, Clone)]
enum BoundKind {
    /// A quantity in this unit.
    Quantity(Unit),
    /// A whole number (pieces).
    Int,
    /// A plain number.
    Number,
}

#[derive(Debug, Clone)]
enum AxisRule {
    Exact {
        normalization: Normalization,
        freight_class: bool,
    },
    Range {
        bound: BoundKind,
        overflow: Overflow,
    },
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct Bounds {
    min: Decimal,
    max: Option<Decimal>,
}

impl Bounds {
    fn contains(self, key: Decimal) -> bool {
        key >= self.min && self.max.is_none_or(|max| key < max)
    }

    fn order(self, other: Self) -> Ordering {
        self.min.cmp(&other.min).then(match (self.max, other.max) {
            (Some(left), Some(right)) => left.cmp(&right),
            (Some(_), None) => Ordering::Less,
            (None, Some(_)) => Ordering::Greater,
            (None, None) => Ordering::Equal,
        })
    }
}

#[derive(Debug, Clone)]
enum Position {
    Key(String),
    Band(Bounds),
}

#[derive(Debug, Clone)]
struct Cell {
    at: Vec<Position>,
    value: Value,
    minimum: Option<Decimal>,
    deficit: bool,
}

/// One table, indexed.
#[derive(Debug)]
struct Table {
    axes: Vec<AxisRule>,
    cells: Vec<Cell>,
    /// The cells by their exact-axis keys in axis order, each list in cell order.
    by_exact: BTreeMap<Vec<String>, Vec<usize>>,
    /// Whether the request carried cells at all.
    sent: bool,
    /// The exact-key combinations the cells cover; `None` when they are the whole table.
    loaded: Option<BTreeSet<Vec<String>>>,
}

/// Cells a lookup needs that the request did not carry: the table and one normalised key per
/// exact axis, in axis order.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord)]
pub struct Need {
    /// The table's code.
    pub table: String,
    /// The exact-axis keys.
    pub keys: Vec<String>,
}

/// What a lookup gives back.
#[derive(Debug, Clone)]
pub struct Read {
    /// The value; `none` on a miss, and on a provisional miss when `needs` is not empty.
    pub value: Value,
    /// The lookup, for the trace.
    pub lookup: Lookup,
    /// Cells the lookup needs before its answer is final.
    pub needs: Vec<Need>,
}

struct Found {
    cell: Option<usize>,
    adjusted: bool,
    needs: Vec<Vec<String>>,
}

/// One lookup, for the trace.
#[derive(Debug, Clone)]
pub struct Lookup {
    /// The table's code.
    pub table: String,
    /// How it was read: `index`, `cell`, `band` or `interpolate`.
    pub op: &'static str,
    /// The keys given.
    pub keys: Vec<Value>,
    /// Each axis's matched key or band; `None` on a miss.
    pub matched: Option<Vec<Json>>,
    /// Whether an overflow policy moved a key into its band.
    pub adjusted: bool,
    /// Whether the answer waits for cells the request did not carry.
    pub pending: bool,
    /// The value given back, `none` on a miss.
    pub value: Value,
}

/// Every table a request carries, by code.
#[derive(Debug, Default)]
pub struct Tables {
    tables: BTreeMap<String, Result<Rc<Table>, EvalError>>,
}

impl Tables {
    /// Indexes the context's tables. A table that does not fit its shape is kept as the error
    /// a program reading it will get.
    #[must_use]
    pub fn build(shapes: &[TableShape]) -> Self {
        let mut tables = BTreeMap::new();
        for shape in shapes {
            let built = if tables.contains_key(&shape.code) {
                Err(table_error(
                    crate::message!("The rate table {0} is sent twice"),
                    &shape.code,
                ))
            } else {
                index(shape).map(Rc::new)
            };
            tables.insert(shape.code.clone(), built);
        }
        Self { tables }
    }

    fn table(&self, code: &str) -> Result<Rc<Table>, EvalError> {
        match self.tables.get(code) {
            Some(Ok(table)) => Ok(Rc::clone(table)),
            Some(Err(error)) => Err(error.clone()),
            None => Err(table_error(
                crate::message!("No rate table named {0} was sent with the request"),
                code,
            )),
        }
    }

    /// `tables.Code[keys…]`: the value, or `none` on a miss.
    ///
    /// # Errors
    ///
    /// When the table is missing or malformed, a key cannot be read, or the budget runs out.
    pub fn index(&self, code: &str, keys: &[Value], meter: &mut Meter) -> Result<Read, EvalError> {
        self.read(code, "index", keys, meter, |_, cell| cell.value.clone())
    }

    /// `tables.Code.cell(keys…)`: `{ value, minimum, deficit }`, or `none` on a miss.
    ///
    /// # Errors
    ///
    /// As [`Tables::index`].
    pub fn cell(&self, code: &str, keys: &[Value], meter: &mut Meter) -> Result<Read, EvalError> {
        self.read(code, "cell", keys, meter, |_, cell| {
            Value::record(vec![
                ("value", cell.value.clone()),
                ("minimum", cell.minimum.map_or(Value::None, Value::money)),
                ("deficit", Value::Bool(cell.deficit)),
            ])
        })
    }

    /// `tables.Code.band(x)` on a single-axis range table: `{ min, max, value }`, or `none`.
    ///
    /// # Errors
    ///
    /// When the table is not a single-axis range table, or as [`Tables::index`].
    pub fn band(&self, code: &str, key: &Value, meter: &mut Meter) -> Result<Read, EvalError> {
        let bound = self.table(code)?.banded(code)?;
        self.read(
            code,
            "band",
            std::slice::from_ref(key),
            meter,
            |_, cell| match cell.at.first() {
                Some(Position::Band(bounds)) => Value::record(vec![
                    ("min", bound_value(&bound, bounds.min)),
                    (
                        "max",
                        bounds
                            .max
                            .map_or(Value::None, |max| bound_value(&bound, max)),
                    ),
                    ("value", cell.value.clone()),
                ]),
                _ => Value::None,
            },
        )
    }

    fn read(
        &self,
        code: &str,
        op: &'static str,
        keys: &[Value],
        meter: &mut Meter,
        give: impl Fn(&Table, &Cell) -> Value,
    ) -> Result<Read, EvalError> {
        let table = self.table(code)?;
        let found = table.find(code, keys, meter)?;
        let value = found
            .cell
            .and_then(|index| table.cells.get(index))
            .map_or(Value::None, |cell| give(&table, cell));
        Ok(table.answer(code, op, keys, &found, value))
    }

    /// `tables.Code.interpolate(x)` on a single-axis range table: the straight line between the
    /// two band floors around the key, and the edge band's value below the first floor or past
    /// the last. `none` when the table has no bands.
    ///
    /// # Errors
    ///
    /// When the table is not a single-axis range table, its values are not numeric, or a step
    /// overflows.
    pub fn interpolate(
        &self,
        code: &str,
        key: &Value,
        meter: &mut Meter,
    ) -> Result<Read, EvalError> {
        let table = self.table(code)?;
        let bound = table.banded(code)?;
        let at = Table::range_key(code, &bound, key)?;
        let keys = std::slice::from_ref(key);
        if !table.sent {
            let found = Found {
                cell: None,
                adjusted: false,
                needs: vec![Vec::new()],
            };
            return Ok(table.answer(code, "interpolate", keys, &found, Value::None));
        }
        meter.steps(table.cells.len())?;
        let points: Vec<(Decimal, &Value)> = table
            .cells
            .iter()
            .filter_map(|cell| match cell.at.first() {
                Some(Position::Band(bounds)) => Some((bounds.min, &cell.value)),
                _ => None,
            })
            .collect();
        let missing = Found {
            cell: None,
            adjusted: false,
            needs: Vec::new(),
        };
        let (Some(first), Some(last)) = (points.first(), points.last()) else {
            return Ok(table.answer(code, "interpolate", keys, &missing, Value::None));
        };
        let value = if at <= first.0 {
            first.1.clone()
        } else if at >= last.0 {
            last.1.clone()
        } else {
            let mut result = last.1.clone();
            for pair in points.windows(2) {
                let [(low, low_value), (high, high_value)] = pair else {
                    continue;
                };
                if at >= *low && at < *high {
                    result = between(at, (*low, low_value), (*high, high_value))?;
                    break;
                }
            }
            result
        };
        let mut read = table.answer(code, "interpolate", keys, &missing, value);
        read.lookup.matched = Some(vec![Json::String(format_decimal(at))]);
        Ok(read)
    }
}

/// The point at `at` on the line from `(low, low_value)` to `(high, high_value)`.
fn between(
    at: Decimal,
    low: (Decimal, &Value),
    high: (Decimal, &Value),
) -> Result<Value, EvalError> {
    let (low_at, low_value) = low;
    let (high_at, high_value) = high;
    let span = high_at
        .checked_sub(low_at)
        .ok_or_else(EvalError::overflow)?;
    if span.is_zero() {
        return Ok(high_value.clone());
    }
    let ratio = at
        .checked_sub(low_at)
        .and_then(|offset| offset.checked_div(span))
        .ok_or_else(EvalError::overflow)?;
    let (low_number, unit) = numeric(low_value)?;
    let (high_number, _) = numeric(high_value)?;
    let value = high_number
        .checked_sub(low_number)
        .and_then(|rise| rise.checked_mul(ratio))
        .and_then(|offset| low_number.checked_add(offset))
        .ok_or_else(EvalError::overflow)?
        .normalize();
    Ok(match unit {
        Some(unit) => Value::Quantity(Rc::new(Quantity::new(value, unit))),
        None => Value::Number(value),
    })
}

fn numeric(value: &Value) -> Result<(Decimal, Option<Unit>), EvalError> {
    match value {
        Value::Int(number) | Value::Number(number) => Ok((*number, None)),
        Value::Quantity(quantity) => Ok((quantity.value, Some(quantity.unit.clone()))),
        other => Err(EvalError::internal(Message::with(
            crate::message!("{0} cannot be interpolated"),
            [Arg::from(other.kind())],
        ))),
    }
}

fn bound_value(bound: &BoundKind, value: Decimal) -> Value {
    match bound {
        BoundKind::Quantity(unit) => Value::Quantity(Rc::new(Quantity::new(value, unit.clone()))),
        BoundKind::Int => Value::Int(value),
        BoundKind::Number => Value::Number(value),
    }
}

fn table_error(template: &'static str, code: &str) -> EvalError {
    EvalError::with(Code::Table, template, [Arg::from(code)])
}

fn axis_rule(code: &str, axis: &Axis) -> Result<AxisRule, EvalError> {
    match axis.match_mode.as_str() {
        "Exact" => Ok(AxisRule::Exact {
            normalization: match axis.normalization.as_str() {
                "" | "None" => Normalization::None,
                "Trim" => Normalization::Trim,
                "Upper" => Normalization::Upper,
                "Zip3" => Normalization::Zip3,
                _ => {
                    return Err(table_error(
                        crate::message!(
                            "The rate table {0} has an axis with an unknown key normalisation"
                        ),
                        code,
                    ));
                }
            },
            freight_class: axis.kind == "FreightClass",
        }),
        "Range" => Ok(AxisRule::Range {
            bound: range_bound(code, axis)?,
            overflow: match axis.overflow.as_str() {
                "" | "Error" => Overflow::Error,
                "ClampToTopBand" => Overflow::ClampToTopBand,
                "Nearest" => Overflow::Nearest,
                _ => {
                    return Err(table_error(
                        crate::message!(
                            "The rate table {0} has an axis with an unknown overflow policy"
                        ),
                        code,
                    ));
                }
            },
        }),
        _ => Err(table_error(
            crate::message!("The rate table {0} has an axis that is neither Exact nor Range"),
            code,
        )),
    }
}

fn range_bound(code: &str, axis: &Axis) -> Result<BoundKind, EvalError> {
    let default = match axis.kind.as_str() {
        "WeightBreak" => Some("lb"),
        "Distance" => Some("mi"),
        "LinearFeet" => Some("ft"),
        "PieceCount" => return Ok(BoundKind::Int),
        _ => None,
    };
    match axis.unit.as_deref().or(default) {
        Some(text) => {
            let unit = Unit::parse(text).map_err(|_| {
                table_error(
                    crate::message!("The rate table {0} has an axis whose unit is not a unit"),
                    code,
                )
            })?;
            if unit.dim().is_none() {
                Ok(BoundKind::Number)
            } else {
                Ok(BoundKind::Quantity(unit))
            }
        }
        None => Ok(BoundKind::Number),
    }
}

fn index(shape: &TableShape) -> Result<Table, EvalError> {
    let code = shape.code.as_str();
    if shape.axes.is_empty() || shape.axes.len() > 4 {
        return Err(table_error(
            crate::message!("The rate table {0} must have one to four axes"),
            code,
        ));
    }
    let value_type = shape.value().ok_or_else(|| {
        table_error(
            crate::message!("The rate table {0} has a value type a table cannot hold"),
            code,
        )
    })?;
    let axes = shape
        .axes
        .iter()
        .map(|axis| axis_rule(code, axis))
        .collect::<Result<Vec<_>, _>>()?;
    let mut cells = Vec::with_capacity(shape.cells.as_ref().map_or(0, Vec::len));
    for wire in shape.cells.iter().flatten() {
        cells.push(cell(code, &axes, &value_type, wire)?);
    }
    cells.sort_by(|left, right| {
        for (left, right) in left.at.iter().zip(&right.at) {
            if let (Position::Band(left), Position::Band(right)) = (left, right) {
                let order = left.order(*right);
                if order != Ordering::Equal {
                    return order;
                }
            }
        }
        Ordering::Equal
    });
    let exact: Vec<usize> = axes
        .iter()
        .enumerate()
        .filter(|(_, rule)| matches!(rule, AxisRule::Exact { .. }))
        .map(|(index, _)| index)
        .collect();
    let mut by_exact: BTreeMap<Vec<String>, Vec<usize>> = BTreeMap::new();
    for (index, cell) in cells.iter().enumerate() {
        let keys = exact
            .iter()
            .filter_map(|axis| match cell.at.get(*axis) {
                Some(Position::Key(key)) => Some(key.clone()),
                _ => None,
            })
            .collect();
        by_exact.entry(keys).or_default().push(index);
    }
    Ok(Table {
        axes,
        cells,
        by_exact,
        sent: shape.cells.is_some(),
        loaded: shape
            .loaded
            .as_ref()
            .map(|loaded| loaded.iter().cloned().collect()),
    })
}

fn cell(
    code: &str,
    axes: &[AxisRule],
    value_type: &Type,
    wire: &CellWire,
) -> Result<Cell, EvalError> {
    if wire.at.len() != axes.len() {
        return Err(table_error(
            crate::message!("The rate table {0} has a cell without one position per axis"),
            code,
        ));
    }
    let mut at = Vec::with_capacity(axes.len());
    for (rule, position) in axes.iter().zip(&wire.at) {
        at.push(match rule {
            AxisRule::Exact {
                normalization,
                freight_class,
            } => {
                let Some(key) = &position.key else {
                    return Err(table_error(
                        crate::message!(
                            "The rate table {0} has a cell without a key on an exact axis"
                        ),
                        code,
                    ));
                };
                Position::Key(normalise(key, *normalization, *freight_class))
            }
            AxisRule::Range { .. } => {
                let Some(min) = position.min else {
                    return Err(table_error(
                        crate::message!(
                            "The rate table {0} has a cell without a band floor on a range axis"
                        ),
                        code,
                    ));
                };
                Position::Band(Bounds {
                    min: min.get(),
                    max: position.max.map(Dec::get),
                })
            }
        });
    }
    Ok(Cell {
        at,
        value: cell_value(code, value_type, wire.value.get())?,
        minimum: wire.minimum.map(Dec::get),
        deficit: wire.deficit,
    })
}

fn cell_value(code: &str, value_type: &Type, value: Decimal) -> Result<Value, EvalError> {
    Ok(match value_type {
        Type::Int if value.fract().is_zero() => Value::Int(value.trunc().normalize()),
        Type::Int => {
            return Err(table_error(
                crate::message!(
                    "The rate table {0} holds whole numbers, but a cell has a fraction"
                ),
                code,
            ));
        }
        Type::Number => Value::Number(value),
        Type::Percent => Value::percent(value),
        Type::Quantity(quantity) if quantity.dim.exponent(Base::Money) == 1 => {
            let unit = quantity
                .display
                .clone()
                .unwrap_or_else(super::value::dollars);
            Value::Quantity(Rc::new(Quantity::new(value, unit)))
        }
        _ => {
            return Err(table_error(
                crate::message!("The rate table {0} has a value type a table cannot hold"),
                code,
            ));
        }
    })
}

/// Reads a key as an NMFC freight class: an optional `Class` prefix in any case, `_` for the
/// decimal point, and the class's number. Gives the `FreightClass` variant name.
#[must_use]
pub fn parse_freight_class(text: &str) -> Option<&'static str> {
    const CLASSES: [(i64, u32, &str); 18] = [
        (50, 0, "Class50"),
        (55, 0, "Class55"),
        (60, 0, "Class60"),
        (65, 0, "Class65"),
        (70, 0, "Class70"),
        (775, 1, "Class77_5"),
        (85, 0, "Class85"),
        (925, 1, "Class92_5"),
        (100, 0, "Class100"),
        (110, 0, "Class110"),
        (125, 0, "Class125"),
        (150, 0, "Class150"),
        (175, 0, "Class175"),
        (200, 0, "Class200"),
        (250, 0, "Class250"),
        (300, 0, "Class300"),
        (400, 0, "Class400"),
        (500, 0, "Class500"),
    ];
    let mut rest = text.trim();
    if rest.len() >= 5 && rest.is_char_boundary(5) && rest[..5].eq_ignore_ascii_case("class") {
        rest = rest[5..].trim();
    }
    if rest.is_empty() {
        return None;
    }
    let number = crate::wire::parse_decimal(&rest.replace('_', ".")).ok()?;
    CLASSES
        .iter()
        .find(|(mantissa, scale, _)| Decimal::new(*mantissa, *scale) == number)
        .map(|(_, _, class)| *class)
}

fn normalise(key: &str, normalization: Normalization, freight_class: bool) -> String {
    if freight_class && let Some(class) = parse_freight_class(key) {
        return class.to_owned();
    }
    normalization.apply(key)
}

fn unusable_key(code: &str, key: &Value) -> EvalError {
    EvalError::internal(Message::with(
        crate::message!("The rate table {0} cannot be read with {1} as a key"),
        [Arg::from(code), Arg::from(key.kind())],
    ))
}

impl Table {
    /// The bound kind of a single-axis range table.
    fn banded(&self, code: &str) -> Result<BoundKind, EvalError> {
        match self.axes.as_slice() {
            [AxisRule::Range { bound, .. }] => Ok(bound.clone()),
            _ => Err(EvalError::internal(Message::with(
                crate::message!("The rate table {0} is not a single-axis range table"),
                [Arg::from(code)],
            ))),
        }
    }

    /// The normalised keys an exact axis tries, in order: one, or each item of a list key.
    fn exact_keys(
        code: &str,
        key: &Value,
        normalization: Normalization,
        freight_class: bool,
    ) -> Result<Vec<String>, EvalError> {
        let one = |key: &Value| -> Result<String, EvalError> {
            let text = match key {
                Value::Text(text) => text.to_string(),
                Value::Enum(variant) => variant.to_string(),
                Value::Int(number) | Value::Number(number) => format_decimal(*number),
                Value::Bool(flag) => String::from(if *flag { "true" } else { "false" }),
                other => return Err(unusable_key(code, other)),
            };
            Ok(normalise(&text, normalization, freight_class))
        };
        match key {
            Value::List(items) => items.iter().map(one).collect(),
            other => Ok(vec![one(other)?]),
        }
    }

    fn range_key(code: &str, bound: &BoundKind, key: &Value) -> Result<Decimal, EvalError> {
        match (bound, key) {
            (BoundKind::Quantity(unit), Value::Quantity(quantity)) => quantity
                .to(unit)
                .map(|converted| converted.value)
                .map_err(|_| unusable_key(code, key)),
            (_, Value::Int(number) | Value::Number(number)) => Ok(*number),
            _ => Err(unusable_key(code, key)),
        }
    }

    /// Whether the request carried the cells for one combination of exact keys.
    fn covers(&self, combination: &[String]) -> bool {
        self.sent
            && self
                .loaded
                .as_ref()
                .is_none_or(|loaded| combination.is_empty() || loaded.contains(combination))
    }

    /// The cell the keys find, whether an overflow policy moved a key, and the exact-key
    /// combinations that were needed but not carried. Combinations of list keys are tried in
    /// order, the first axis outermost; the first that matches wins, unless one before it was
    /// not carried, in which case the answer is a provisional miss.
    fn find(&self, code: &str, keys: &[Value], meter: &mut Meter) -> Result<Found, EvalError> {
        if keys.len() != self.axes.len() {
            return Err(EvalError::internal(Message::with(
                crate::message!("The rate table {0} takes {1} keys, not {2}"),
                [
                    Arg::from(code),
                    Arg::from(self.axes.len()),
                    Arg::from(keys.len()),
                ],
            )));
        }
        let mut options: Vec<Vec<String>> = Vec::new();
        let mut ranges: Vec<(usize, Decimal, Overflow)> = Vec::new();
        for (axis, (rule, key)) in self.axes.iter().zip(keys).enumerate() {
            match rule {
                AxisRule::Exact {
                    normalization,
                    freight_class,
                } => options.push(Self::exact_keys(code, key, *normalization, *freight_class)?),
                AxisRule::Range { bound, overflow } => {
                    ranges.push((axis, Self::range_key(code, bound, key)?, *overflow));
                }
            }
        }
        let count = options
            .iter()
            .try_fold(1usize, |total, keys| total.checked_mul(keys.len()));
        let count = count.unwrap_or(usize::MAX);
        meter.items_would_fit(u64::try_from(count).unwrap_or(u64::MAX))?;
        meter.steps(count)?;
        let mut needs = Vec::new();
        for combination in combinations(&options) {
            if !self.covers(&combination) {
                needs.push(combination);
                continue;
            }
            let Some(candidates) = self.by_exact.get(&combination) else {
                continue;
            };
            meter.steps(candidates.len())?;
            let mut narrowed = candidates.clone();
            let mut adjusted = false;
            for (axis, at, overflow) in &ranges {
                let (next, moved) = self.narrow_band(&narrowed, *axis, *at, *overflow);
                adjusted |= moved;
                narrowed = next;
                if narrowed.is_empty() {
                    break;
                }
            }
            if let Some(first) = narrowed.first() {
                return Ok(Found {
                    cell: needs.is_empty().then_some(*first),
                    adjusted,
                    needs,
                });
            }
        }
        Ok(Found {
            cell: None,
            adjusted: false,
            needs,
        })
    }

    fn position(&self, cell: usize, axis: usize) -> Option<&Position> {
        self.cells.get(cell).and_then(|cell| cell.at.get(axis))
    }

    fn band_of(&self, cell: usize, axis: usize) -> Option<Bounds> {
        match self.position(cell, axis) {
            Some(Position::Band(bounds)) => Some(*bounds),
            _ => None,
        }
    }

    fn narrow_band(
        &self,
        candidates: &[usize],
        axis: usize,
        at: Decimal,
        overflow: Overflow,
    ) -> (Vec<usize>, bool) {
        let containing: Vec<usize> = candidates
            .iter()
            .copied()
            .filter(|index| {
                self.band_of(*index, axis)
                    .is_some_and(|bounds| bounds.contains(at))
            })
            .collect();
        if !containing.is_empty() {
            return (containing, false);
        }
        let mut bands: Vec<Bounds> = candidates
            .iter()
            .filter_map(|index| self.band_of(*index, axis))
            .collect();
        bands.sort_by(|left, right| left.order(*right));
        bands.dedup();
        let Some(chosen) = overflow_band(&bands, at, overflow) else {
            return (Vec::new(), false);
        };
        (
            candidates
                .iter()
                .copied()
                .filter(|index| self.band_of(*index, axis) == Some(chosen))
                .collect(),
            true,
        )
    }

    fn answer(
        &self,
        code: &str,
        op: &'static str,
        keys: &[Value],
        found: &Found,
        value: Value,
    ) -> Read {
        let matched = found
            .cell
            .and_then(|index| self.cells.get(index))
            .map(|cell| {
                cell.at
                    .iter()
                    .map(|position| match position {
                        Position::Key(key) => {
                            let mut object = Map::new();
                            object.insert(String::from("key"), Json::String(key.clone()));
                            Json::Object(object)
                        }
                        Position::Band(bounds) => {
                            let mut object = Map::new();
                            object.insert(
                                String::from("min"),
                                Json::String(format_decimal(bounds.min)),
                            );
                            object.insert(
                                String::from("max"),
                                bounds
                                    .max
                                    .map_or(Json::Null, |max| Json::String(format_decimal(max))),
                            );
                            Json::Object(object)
                        }
                    })
                    .collect()
            });
        Read {
            value: value.clone(),
            lookup: Lookup {
                table: code.to_owned(),
                op,
                keys: keys.to_vec(),
                matched,
                adjusted: found.cell.is_some() && found.adjusted,
                pending: !found.needs.is_empty(),
                value,
            },
            needs: found
                .needs
                .iter()
                .map(|keys| Need {
                    table: code.to_owned(),
                    keys: keys.clone(),
                })
                .collect(),
        }
    }
}

/// Every combination of one key per exact axis, the first axis outermost.
fn combinations(options: &[Vec<String>]) -> Vec<Vec<String>> {
    let mut out: Vec<Vec<String>> = vec![Vec::new()];
    for keys in options {
        let mut next = Vec::with_capacity(out.len().saturating_mul(keys.len()));
        for prefix in &out {
            for key in keys {
                let mut combination = prefix.clone();
                combination.push(key.clone());
                next.push(combination);
            }
        }
        out = next;
    }
    out
}

/// The band an uncovered key prices at under `overflow`, if any. Bands are sorted by floor.
fn overflow_band(bands: &[Bounds], key: Decimal, overflow: Overflow) -> Option<Bounds> {
    let (bottom, top) = (bands.first()?, bands.last()?);
    match overflow {
        Overflow::Error => None,
        Overflow::ClampToTopBand => (key >= top.min).then_some(*top),
        Overflow::Nearest => {
            if key < bottom.min {
                return Some(*bottom);
            }
            if key >= top.min {
                return Some(*top);
            }
            let distance = |band: &Bounds| {
                if key < band.min {
                    band.min - key
                } else {
                    match band.max {
                        Some(max) if key >= max => key - max,
                        _ => Decimal::ZERO,
                    }
                }
            };
            let mut best: Option<(Bounds, Decimal)> = None;
            for band in bands {
                let gap = distance(band);
                if best.is_none_or(|(_, closest)| gap < closest) {
                    best = Some((*band, gap));
                }
            }
            best.map(|(band, _)| band)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::parse_freight_class;

    #[test]
    fn freight_classes_read_every_spelling() {
        assert_eq!(parse_freight_class("85"), Some("Class85"));
        assert_eq!(parse_freight_class(" class 77.5 "), Some("Class77_5"));
        assert_eq!(parse_freight_class("Class92_5"), Some("Class92_5"));
        assert_eq!(parse_freight_class("CLASS500"), Some("Class500"));
        assert_eq!(parse_freight_class("86"), None);
        assert_eq!(parse_freight_class(""), None);
    }
}
