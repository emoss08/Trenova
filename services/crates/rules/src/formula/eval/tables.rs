//! Rate tables and the lookups that read them.
//!
//! A formula's `lookup("fuel_tiers", x)` reads a rate matrix. The Go host loads a tenant's active
//! one- and two-axis matrices, encodes each once into the wire form below (and caches the bytes
//! with the matrix stamp), and sends the ones a request needs. Everything about how a key finds
//! its value happens here, in exact decimals, exactly as `matrix_lookup.go` did:
//!
//! - An **exact** axis compares normalised keys: `Trim`, `Upper` (trim then upper-case), `Zip3`
//!   (trim, upper-case, first three characters) or `None`. A `FreightClass` axis first reads a
//!   key as one of the eighteen NMFC classes (`85`, `"77.5"`, `"Class92_5"`), falling back to the
//!   normalisation. Stored keys and looked-up keys pass through the same rule. Numbers become
//!   keys in plain notation (`3`, `2.5`); `true` and `false` become `"true"` and `"false"`.
//! - A **range** axis matches bands half-open, `[min, max)`: a quantity on an upper bound belongs
//!   to the next band up. A band without a maximum is open-ended. A cell without a minimum on a
//!   range axis is not a band and is ignored.
//! - A quantity no band covers follows the axis's overflow policy: `Error` misses,
//!   `ClampToTopBand` prices anything at or past the top band's floor at the top band, `Nearest`
//!   also takes the bottom band below the first floor and the closest band in a gap.
//! - A two-axis lookup narrows to the cells whose row matches (every overlapping band is a
//!   candidate), takes the first whose column matches, and when none does on a range column,
//!   retries once with the column moved into a band by the column's overflow policy.
//! - `lookupInterp` reads a single-axis banded table as a curve through its floors;
//!   `deficitWeight` bills the next break's minimum when that is cheaper at the next break's rate.
//!
//! A miss (the table exists but has no entry, band or cell) is the only failure `lookupOr` and
//! `lookup2Or` absorb; a missing table, the wrong shape or an unusable key always surfaces.
//!
//! ## Wire form
//!
//! ```json
//! {"name": "class_rates",
//!  "axes": [{"match": "Exact", "kind": "Zone", "normalization": "Upper"},
//!           {"match": "Range", "kind": "WeightBreak", "overflow": "ClampToTopBand"}],
//!  "cells": [{"at": [{"key": "ZONE_5"}, {"min": "0", "max": "5000"}], "value": "18.50"}]}
//! ```
//!
//! The enum strings are Go's own (`ratematrix.MatchMode`, `DimensionKind`, `KeyNormalization`,
//! `RangeOverflow`), so the host encodes a stamped matrix field for field. `axes` is in position
//! order and has one or two entries; each cell's `at` has one entry per axis.

use super::error::Code;
use super::error::EvalError;
use super::number;
use super::value::Value;
use super::value::format_number;
use crate::message::Arg;
use crate::wire::Dec;
use rust_decimal::Decimal;
use serde::Deserialize;
use std::collections::BTreeMap;

/// One table as the host sends it.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct TableWire {
    /// The matrix code a formula names.
    pub name: String,
    /// The axes in position order: one or two.
    pub axes: Vec<AxisWire>,
    /// The cells.
    pub cells: Vec<CellWire>,
}

/// One axis of a table.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct AxisWire {
    /// `Exact` or `Range`.
    #[serde(rename = "match")]
    pub match_mode: MatchMode,
    /// What the axis varies by; only `FreightClass` changes how keys read.
    #[serde(default)]
    pub kind: DimensionKind,
    /// How an exact axis normalises keys; empty or absent is `None`.
    #[serde(default)]
    pub normalization: Normalization,
    /// What a range axis does with a quantity no band covers; empty or absent is `Error`.
    #[serde(default)]
    pub overflow: Overflow,
}

/// Where a cell sits on one axis.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct AtWire {
    /// The key on an exact axis.
    #[serde(default)]
    pub key: String,
    /// The band floor on a range axis.
    #[serde(default)]
    pub min: Option<Dec>,
    /// The band ceiling on a range axis; absent is open-ended.
    #[serde(default)]
    pub max: Option<Dec>,
}

/// One cell.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct CellWire {
    /// One position per axis.
    pub at: Vec<AtWire>,
    /// The cell's value.
    pub value: Dec,
}

/// How an axis matches.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
pub enum MatchMode {
    /// By key equality.
    Exact,
    /// By the band a quantity falls in.
    Range,
}

/// What an axis varies by, as Go's `ratematrix.DimensionKind`. Only `FreightClass` changes how
/// keys read; the others are accepted so the host can send the matrix as it is stored.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Deserialize)]
pub enum DimensionKind {
    /// A zone code.
    Zone,
    /// A three-digit ZIP prefix.
    Zip3,
    /// A five-digit ZIP.
    Zip5,
    /// A state or province.
    State,
    /// A country.
    Country,
    /// A weight break.
    WeightBreak,
    /// A distance.
    Distance,
    /// A piece count.
    PieceCount,
    /// Linear feet.
    LinearFeet,
    /// An NMFC freight class: keys read as one of the eighteen classes.
    FreightClass,
    /// An equipment type.
    EquipmentType,
    /// A service type.
    ServiceType,
    /// Anything else.
    #[default]
    Custom,
    /// A bare quantity a formula supplies.
    Quantity,
}

/// How an exact axis reads keys, as Go's `ratematrix.KeyNormalization`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Deserialize)]
pub enum Normalization {
    /// Keys compare as written.
    #[default]
    #[serde(alias = "")]
    None,
    /// Surrounding whitespace is ignored.
    Trim,
    /// Trimmed and upper-cased.
    Upper,
    /// Trimmed, upper-cased and cut to three characters.
    Zip3,
}

/// What a range axis does with a quantity no band covers, as Go's `ratematrix.RangeOverflow`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Deserialize)]
pub enum Overflow {
    /// The lookup misses.
    #[default]
    #[serde(alias = "")]
    Error,
    /// At or past the top band's floor prices at the top band.
    ClampToTopBand,
    /// Below the first band prices at the first, past the top at the top, in a gap at the closest.
    Nearest,
}

/// What a lookup matched, for the receipt: Go's `formulatypes.LookupMatch`.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Matched {
    /// The normalised key, the helper's name, or `row × col` for a two-axis cell.
    pub key: Option<String>,
    /// The band floor on a single range axis.
    pub band_min: Option<Decimal>,
    /// The band ceiling, absent for an open-ended band.
    pub band_max: Option<Decimal>,
    /// The overflow policy moved the key into the band.
    pub adjusted: bool,
}

#[derive(Debug, Clone, Copy)]
struct AxisKey {
    freight_class: bool,
    normalization: Normalization,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
struct Bounds {
    min: Decimal,
    max: Option<Decimal>,
}

impl Bounds {
    fn contains(&self, key: Decimal) -> bool {
        key >= self.min && self.max.is_none_or(|max| key < max)
    }

    fn label(&self) -> String {
        match self.max {
            Some(max) => format!("{}\u{2013}{}", format_number(self.min), format_number(max)),
            None => format!("{}+", format_number(self.min)),
        }
    }
}

#[derive(Debug)]
struct ExactTable {
    entries: BTreeMap<String, Decimal>,
    key: AxisKey,
}

#[derive(Debug)]
struct RangeTable {
    bands: Vec<Bounds>,
    values: Vec<Decimal>,
    overflow: Overflow,
}

#[derive(Debug)]
enum Single {
    Exact(ExactTable),
    Range(RangeTable),
}

#[derive(Debug)]
struct Cell {
    row_key: String,
    row: Option<Bounds>,
    col_key: String,
    col: Option<Bounds>,
    value: Decimal,
}

#[derive(Debug, Clone, Copy)]
struct AxisSpec {
    mode: MatchMode,
    key: AxisKey,
    overflow: Overflow,
}

#[derive(Debug)]
struct TwoAxis {
    row: AxisSpec,
    col: AxisSpec,
    cells: Vec<Cell>,
    row_bands: Vec<Bounds>,
    col_bands: Vec<Bounds>,
    by_row: BTreeMap<String, Vec<usize>>,
    by_row_band: BTreeMap<Bounds, Vec<usize>>,
}

/// Every table a request carries, by name. A name may be both a single-axis and a two-axis
/// table, as two matrices may share a code in Go; each call family reads its own.
#[derive(Debug, Default)]
pub struct Tables {
    single: BTreeMap<String, Single>,
    two: BTreeMap<String, TwoAxis>,
}

const FREIGHT_CLASSES: [(i64, u32, &str); 18] = [
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

/// Reads a key as an NMFC freight class, as Go's `commodity.ParseFreightClass`: an optional
/// `Class` prefix in any case, `_` for the decimal point, and the class's number.
#[must_use]
pub fn parse_freight_class(text: &str) -> Option<&'static str> {
    let mut rest = text.trim();
    if rest.len() >= 5 && rest.is_char_boundary(5) && rest[..5].eq_ignore_ascii_case("class") {
        rest = rest[5..].trim();
    }
    if rest.is_empty() {
        return None;
    }
    let number = crate::wire::parse_decimal(&rest.replace('_', ".")).ok()?;
    FREIGHT_CLASSES
        .iter()
        .find(|(mantissa, scale, _)| Decimal::new(*mantissa, *scale) == number)
        .map(|(_, _, class)| *class)
}

impl Normalization {
    /// Normalises a key, as Go's `KeyNormalization.Apply`.
    #[must_use]
    pub fn apply(self, key: &str) -> String {
        match self {
            Self::None => key.to_owned(),
            Self::Trim => key.trim().to_owned(),
            Self::Upper => key.trim().to_uppercase(),
            Self::Zip3 => key.trim().to_uppercase().chars().take(3).collect(),
        }
    }
}

impl AxisKey {
    fn of(axis: &AxisWire) -> Self {
        Self {
            freight_class: axis.kind == DimensionKind::FreightClass,
            normalization: axis.normalization,
        }
    }

    fn apply(self, key: &str) -> String {
        if self.freight_class
            && let Some(class) = parse_freight_class(key)
        {
            return class.to_owned();
        }
        self.normalization.apply(key)
    }
}

impl Tables {
    /// Indexes the tables a request carries; refuses a table that is not one a host may send.
    pub fn build(tables: &[TableWire]) -> Result<Self, String> {
        let mut built = Self::default();
        for table in tables {
            if table
                .cells
                .iter()
                .any(|cell| cell.at.len() != table.axes.len())
            {
                return Err(format!(
                    "rate table {:?}: every cell needs one position per axis",
                    table.name
                ));
            }
            match table.axes.len() {
                1 => {
                    let name = table.name.clone();
                    if built
                        .single
                        .insert(name.clone(), index_single(table))
                        .is_some()
                    {
                        return Err(format!("rate table {name:?} is sent twice"));
                    }
                }
                2 => {
                    let name = table.name.clone();
                    if built.two.insert(name.clone(), index_two(table)).is_some() {
                        return Err(format!("two-axis rate table {name:?} is sent twice"));
                    }
                }
                count => {
                    return Err(format!(
                        "rate table {:?} has {count} axes; lookups read one or two",
                        table.name
                    ));
                }
            }
        }
        Ok(built)
    }

    /// `lookup(table, key)`.
    pub fn lookup(&self, table: &str, key: &Value) -> Result<(Decimal, Matched), EvalError> {
        match self.single.get(table) {
            Some(Single::Exact(exact)) => {
                let wanted = exact.key.apply(&key_text(table, key, KeyRole::Key)?);
                exact.entries.get(&wanted).map_or_else(
                    || {
                        Err(EvalError::with(
                            Code::LookupMiss,
                            crate::message!("rate table \"{0}\" has no entry for key \"{1}\""),
                            [Arg::from(table), Arg::from(wanted.clone())],
                        ))
                    },
                    |value| {
                        Ok((
                            *value,
                            Matched {
                                key: Some(wanted.clone()),
                                ..Matched::default()
                            },
                        ))
                    },
                )
            }
            Some(Single::Range(range)) => {
                let (index, adjusted) = resolve_band(table, range, key)?;
                let bounds = range.bands[index];
                Ok((
                    range.values[index],
                    Matched {
                        key: None,
                        band_min: Some(bounds.min),
                        band_max: bounds.max,
                        adjusted,
                    },
                ))
            }
            None if self.two.contains_key(table) => Err(EvalError::with(
                Code::TableShape,
                crate::message!(
                    "rate table \"{0}\" has two axes — use lookup2(table, rowKey, colKey)"
                ),
                [Arg::from(table)],
            )),
            None => Err(not_found(table)),
        }
    }

    /// `lookup2(table, rowKey, colKey)`.
    pub fn lookup2(
        &self,
        table: &str,
        row: &Value,
        col: &Value,
    ) -> Result<(Decimal, Matched), EvalError> {
        let Some(grid) = self.two.get(table) else {
            if self.single.contains_key(table) {
                return Err(EvalError::with(
                    Code::TableShape,
                    crate::message!(
                        "rate table \"{0}\" has a single axis — use lookup(table, key)"
                    ),
                    [Arg::from(table)],
                ));
            }
            return Err(EvalError::with(
                Code::TableNotFound,
                crate::message!("two-axis rate table \"{0}\" not found"),
                [Arg::from(table)],
            ));
        };
        let (cell, adjusted) = grid.find(table, row, col)?;
        let cell = &grid.cells[cell];
        Ok((
            cell.value,
            Matched {
                key: Some(describe_cell(cell)),
                adjusted,
                ..Matched::default()
            },
        ))
    }

    /// `lookupInterp(table, key)`: the straight line between the two band floors around the key;
    /// the edge band's value below the first floor or past the last.
    pub fn interpolate(&self, table: &str, key: &Value) -> Result<Decimal, EvalError> {
        let range = self.banded(table)?;
        let key = key_number(table, key, KeyRole::Key)?;
        let (Some(first), Some(last)) = (range.bands.first(), range.bands.last()) else {
            return Err(no_bands(table));
        };
        if key <= first.min {
            return Ok(range.values[0]);
        }
        if key >= last.min {
            return Ok(range.values[range.values.len() - 1]);
        }
        for index in 0..range.bands.len() - 1 {
            let (low, high) = (range.bands[index], range.bands[index + 1]);
            if key < low.min || key >= high.min {
                continue;
            }
            let span = number::sub(high.min, low.min)?;
            if span.is_zero() {
                return Ok(range.values[index + 1]);
            }
            let ratio = number::div(number::sub(key, low.min)?, span)?;
            let rise = number::sub(range.values[index + 1], range.values[index])?;
            return number::add(range.values[index], number::mul(rise, ratio)?);
        }
        Ok(range.values[range.values.len() - 1])
    }

    /// `deficitWeight(table, weight)`: the actual weight, unless the next break's minimum at the
    /// next break's rate costs less, in which case that minimum.
    pub fn deficit_weight(&self, table: &str, weight: &Value) -> Result<Decimal, EvalError> {
        let range = self.banded(table)?;
        let quantity = key_number(table, weight, KeyRole::Key)?;
        let (index, _) = resolve_band(table, range, weight)?;
        let current = range.bands[index];
        let Some(next) = range.bands.iter().position(|band| band.min > current.min) else {
            return Ok(quantity);
        };
        let at_next = number::mul(range.bands[next].min, range.values[next])?;
        let at_actual = number::mul(quantity, range.values[index])?;
        Ok(if at_next < at_actual {
            range.bands[next].min
        } else {
            quantity
        })
    }

    fn banded(&self, table: &str) -> Result<&RangeTable, EvalError> {
        match self.single.get(table) {
            Some(Single::Range(range)) if range.bands.is_empty() => Err(no_bands(table)),
            Some(Single::Range(range)) => Ok(range),
            Some(Single::Exact(_)) => Err(EvalError::with(
                Code::TableShape,
                crate::message!(
                    "rate table \"{0}\" matches exact keys, not bands — use lookup(table, key)"
                ),
                [Arg::from(table)],
            )),
            None if self.two.contains_key(table) => Err(EvalError::with(
                Code::TableShape,
                crate::message!(
                    "rate table \"{0}\" has two axes — use lookup2(table, rowKey, colKey)"
                ),
                [Arg::from(table)],
            )),
            None => Err(not_found(table)),
        }
    }
}

fn not_found(table: &str) -> EvalError {
    EvalError::with(
        Code::TableNotFound,
        crate::message!("rate table \"{0}\" not found"),
        [Arg::from(table)],
    )
}

fn no_bands(table: &str) -> EvalError {
    EvalError::with(
        Code::TableShape,
        crate::message!("rate table \"{0}\" has no bands"),
        [Arg::from(table)],
    )
}

fn index_single(table: &TableWire) -> Single {
    let Some(axis) = table.axes.first() else {
        return Single::Range(RangeTable {
            bands: Vec::new(),
            values: Vec::new(),
            overflow: Overflow::Error,
        });
    };
    match axis.match_mode {
        MatchMode::Exact => {
            let key = AxisKey::of(axis);
            let mut entries = BTreeMap::new();
            for cell in &table.cells {
                let position = cell.at.first().cloned().unwrap_or_default();
                entries.insert(key.apply(&position.key), cell.value.get());
            }
            Single::Exact(ExactTable { entries, key })
        }
        MatchMode::Range => {
            let mut bands: Vec<(Bounds, Decimal)> = table
                .cells
                .iter()
                .filter_map(|cell| {
                    let position = cell.at.first()?;
                    Some((
                        Bounds {
                            min: position.min?.get(),
                            max: position.max.map(Dec::get),
                        },
                        cell.value.get(),
                    ))
                })
                .collect();
            bands.sort_by(|left, right| left.0.min.cmp(&right.0.min));
            Single::Range(RangeTable {
                bands: bands.iter().map(|(bounds, _)| *bounds).collect(),
                values: bands.iter().map(|(_, value)| *value).collect(),
                overflow: axis.overflow,
            })
        }
    }
}

fn index_two(table: &TableWire) -> TwoAxis {
    let spec = |axis: Option<&AxisWire>| {
        axis.map_or(
            AxisSpec {
                mode: MatchMode::Exact,
                key: AxisKey {
                    freight_class: false,
                    normalization: Normalization::None,
                },
                overflow: Overflow::Error,
            },
            |axis| AxisSpec {
                mode: axis.match_mode,
                key: AxisKey::of(axis),
                overflow: axis.overflow,
            },
        )
    };
    let row = spec(table.axes.first());
    let col = spec(table.axes.get(1));
    let bounds = |at: &AtWire| {
        at.min.map(|min| Bounds {
            min: min.get(),
            max: at.max.map(Dec::get),
        })
    };
    let mut cells = Vec::with_capacity(table.cells.len());
    for cell in &table.cells {
        let (Some(row_at), Some(col_at)) = (cell.at.first(), cell.at.get(1)) else {
            continue;
        };
        let row_bounds = bounds(row_at);
        let col_bounds = bounds(col_at);
        if (row.mode == MatchMode::Range && row_bounds.is_none())
            || (col.mode == MatchMode::Range && col_bounds.is_none())
        {
            continue;
        }
        cells.push(Cell {
            row_key: row.key.apply(&row_at.key),
            row: row_bounds,
            col_key: col.key.apply(&col_at.key),
            col: col_bounds,
            value: cell.value.get(),
        });
    }
    let distinct = |pick: fn(&Cell) -> Option<Bounds>| {
        let mut seen: Vec<Bounds> = Vec::new();
        for cell in &cells {
            if let Some(bounds) = pick(cell)
                && !seen.contains(&bounds)
            {
                seen.push(bounds);
            }
        }
        seen.sort_by(|left, right| left.min.cmp(&right.min));
        seen
    };
    let row_bands = if row.mode == MatchMode::Range {
        distinct(|cell| cell.row)
    } else {
        Vec::new()
    };
    let col_bands = if col.mode == MatchMode::Range {
        distinct(|cell| cell.col)
    } else {
        Vec::new()
    };
    let mut by_row: BTreeMap<String, Vec<usize>> = BTreeMap::new();
    let mut by_row_band: BTreeMap<Bounds, Vec<usize>> = BTreeMap::new();
    for (index, cell) in cells.iter().enumerate() {
        if row.mode == MatchMode::Exact {
            by_row.entry(cell.row_key.clone()).or_default().push(index);
        } else if let Some(bounds) = cell.row {
            by_row_band.entry(bounds).or_default().push(index);
        }
    }
    TwoAxis {
        row,
        col,
        cells,
        row_bands,
        col_bands,
        by_row,
        by_row_band,
    }
}

impl TwoAxis {
    fn find(&self, table: &str, row: &Value, col: &Value) -> Result<(usize, bool), EvalError> {
        let (candidates, row_adjusted) = self.row_candidates(table, row)?;
        let mut cell = self.first_column_match(table, &candidates, col)?;
        let mut col_adjusted = false;
        if cell.is_none() && self.col.mode == MatchMode::Range && !candidates.is_empty() {
            let quantity = key_number(table, col, KeyRole::Column)?;
            if let Some(band) = overflow_band(&self.col_bands, quantity, self.col.overflow) {
                cell = self.first_column_match(
                    table,
                    &candidates,
                    &Value::Number(self.col_bands[band].min),
                )?;
                col_adjusted = cell.is_some();
            }
        }
        cell.map(|cell| (cell, row_adjusted || col_adjusted))
            .ok_or_else(|| {
                EvalError::with(
                    Code::LookupMiss,
                    crate::message!(
                        "rate table \"{0}\" has no cell matching row {1} and column {2}"
                    ),
                    [
                        Arg::from(table),
                        Arg::from(row.display()),
                        Arg::from(col.display()),
                    ],
                )
            })
    }

    fn row_candidates(&self, table: &str, row: &Value) -> Result<(Vec<usize>, bool), EvalError> {
        if self.row.mode == MatchMode::Exact {
            let wanted = self.row.key.apply(&key_text(table, row, KeyRole::Row)?);
            return Ok((self.by_row.get(&wanted).cloned().unwrap_or_default(), false));
        }
        let quantity = key_number(table, row, KeyRole::Row)?;
        let candidates = self.rows_containing(quantity);
        if candidates.is_empty()
            && let Some(band) = overflow_band(&self.row_bands, quantity, self.row.overflow)
        {
            return Ok((self.rows_containing(self.row_bands[band].min), true));
        }
        Ok((candidates, false))
    }

    fn rows_containing(&self, quantity: Decimal) -> Vec<usize> {
        let mut candidates = Vec::new();
        for band in &self.row_bands {
            if quantity < band.min {
                break;
            }
            if band.max.is_some_and(|max| quantity >= max) {
                continue;
            }
            if let Some(cells) = self.by_row_band.get(band) {
                candidates.extend_from_slice(cells);
            }
        }
        candidates
    }

    fn first_column_match(
        &self,
        table: &str,
        candidates: &[usize],
        col: &Value,
    ) -> Result<Option<usize>, EvalError> {
        if candidates.is_empty() {
            return Ok(None);
        }
        if self.col.mode == MatchMode::Exact {
            let wanted = self.col.key.apply(&key_text(table, col, KeyRole::Column)?);
            return Ok(candidates
                .iter()
                .copied()
                .find(|index| self.cells[*index].col_key == wanted));
        }
        let quantity = key_number(table, col, KeyRole::Column)?;
        Ok(candidates.iter().copied().find(|index| {
            self.cells[*index]
                .col
                .is_some_and(|bounds| bounds.contains(quantity))
        }))
    }
}

fn describe_cell(cell: &Cell) -> String {
    let side = |key: &str, bounds: Option<Bounds>| {
        if key.is_empty() {
            bounds.map(|bounds| bounds.label()).unwrap_or_default()
        } else {
            key.to_owned()
        }
    };
    format!(
        "{} \u{d7} {}",
        side(&cell.row_key, cell.row),
        side(&cell.col_key, cell.col)
    )
}

fn resolve_band(table: &str, range: &RangeTable, key: &Value) -> Result<(usize, bool), EvalError> {
    let quantity = key_number(table, key, KeyRole::Key)?;
    if let Some(index) = range.bands.iter().position(|band| band.contains(quantity)) {
        return Ok((index, false));
    }
    if let Some(index) = overflow_band(&range.bands, quantity, range.overflow) {
        return Ok((index, true));
    }
    Err(EvalError::with(
        Code::LookupMiss,
        crate::message!("rate table \"{0}\" has no band matching {1}"),
        [Arg::from(table), Arg::from(format_number(quantity))],
    ))
}

/// The band an uncovered quantity prices at under `overflow`, if any. Bands are sorted by floor.
fn overflow_band(bands: &[Bounds], key: Decimal, overflow: Overflow) -> Option<usize> {
    let (bottom, top) = (bands.first()?, bands.last()?);
    match overflow {
        Overflow::Error => None,
        Overflow::ClampToTopBand => (key >= top.min).then(|| bands.len() - 1),
        Overflow::Nearest => {
            if key < bottom.min {
                return Some(0);
            }
            if key >= top.min {
                return Some(bands.len() - 1);
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
            let mut best: Option<(usize, Decimal)> = None;
            for (index, band) in bands.iter().enumerate() {
                let gap = distance(band);
                if best.is_none_or(|(_, closest)| gap < closest) {
                    best = Some((index, gap));
                }
            }
            best.map(|(index, _)| index)
        }
    }
}

#[derive(Debug, Clone, Copy)]
enum KeyRole {
    Key,
    Row,
    Column,
}

/// A key as an exact axis reads it.
fn key_text(table: &str, key: &Value, role: KeyRole) -> Result<String, EvalError> {
    match key {
        Value::Text(text) => Ok(text.to_string()),
        Value::Number(number) => Ok(format_number(*number)),
        Value::Bool(flag) => Ok(String::from(if *flag { "true" } else { "false" })),
        other => Err(unsupported_key(table, other.kind_key(), role)),
    }
}

/// A key as a range axis reads it: a number, or text holding one.
fn key_number(table: &str, key: &Value, role: KeyRole) -> Result<Decimal, EvalError> {
    match key {
        Value::Number(number) => Ok(*number),
        Value::Text(text) => {
            let parsed = if text.trim() == &**text {
                crate::wire::parse_decimal(text).ok()
            } else {
                None
            };
            parsed.ok_or_else(|| {
                let args = [Arg::from(table), Arg::from(text.to_string())];
                match role {
                    KeyRole::Key => EvalError::with(
                        Code::LookupKey,
                        crate::message!("rate table \"{0}\": lookup key \"{1}\" is not numeric"),
                        args,
                    ),
                    KeyRole::Row => EvalError::with(
                        Code::LookupKey,
                        crate::message!(
                            "rate table \"{0}\" row key: lookup key \"{1}\" is not numeric"
                        ),
                        args,
                    ),
                    KeyRole::Column => EvalError::with(
                        Code::LookupKey,
                        crate::message!(
                            "rate table \"{0}\" column key: lookup key \"{1}\" is not numeric"
                        ),
                        args,
                    ),
                }
            })
        }
        other => Err(unsupported_key(table, other.kind_key(), role)),
    }
}

fn unsupported_key(table: &str, kind: &'static str, role: KeyRole) -> EvalError {
    let args = [Arg::from(table), Arg::from(kind)];
    let error = match role {
        KeyRole::Key => EvalError::with(
            Code::LookupKey,
            crate::message!("rate table \"{0}\": unsupported lookup key type {1}"),
            args,
        ),
        KeyRole::Row => EvalError::with(
            Code::LookupKey,
            crate::message!("rate table \"{0}\" row key: unsupported lookup key type {1}"),
            args,
        ),
        KeyRole::Column => EvalError::with(
            Code::LookupKey,
            crate::message!("rate table \"{0}\" column key: unsupported lookup key type {1}"),
            args,
        ),
    };
    if kind == "nil" {
        error.empty_argument(match role {
            KeyRole::Key | KeyRole::Row => 1,
            KeyRole::Column => 2,
        })
    } else {
        error
    }
}

#[cfg(test)]
mod tests {
    use super::{Normalization, parse_freight_class};

    #[test]
    fn freight_classes_read_every_spelling() {
        assert_eq!(parse_freight_class("85"), Some("Class85"));
        assert_eq!(parse_freight_class(" class 77.5 "), Some("Class77_5"));
        assert_eq!(parse_freight_class("Class92_5"), Some("Class92_5"));
        assert_eq!(parse_freight_class("CLASS500"), Some("Class500"));
        assert_eq!(parse_freight_class("86"), None);
        assert_eq!(parse_freight_class("heavy"), None);
        assert_eq!(parse_freight_class(""), None);
    }

    #[test]
    fn normalisation_matches_go() {
        assert_eq!(Normalization::Zip3.apply(" 75201-1234 "), "752");
        assert_eq!(Normalization::Upper.apply(" atl-mia "), "ATL-MIA");
        assert_eq!(Normalization::Trim.apply(" ATL-JAX "), "ATL-JAX");
        assert_eq!(Normalization::None.apply(" x "), " x ");
    }
}
