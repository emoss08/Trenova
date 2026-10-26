//! CSV input: a header row, then rows whose cells are read by their column's type and format.

use std::collections::BTreeMap;
use std::rc::Rc;

use rust_decimal::Decimal;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::stdlib::parse_pattern;
use super::value::{Unit, Value};
use crate::lading::syntax::ast::{CsvColumn, ExprKind, TextPart};
use crate::lading::types::Type;
use crate::lading::values::ZoneValue;

/// The most rows a file may have.
pub const MAX_ROWS: usize = 100_000;

/// Splits CSV text into records, with quoted fields and doubled quotes.
#[must_use]
pub fn records(text: &str) -> Vec<Vec<String>> {
    let mut rows = Vec::new();
    let mut row = Vec::new();
    let mut field = String::new();
    let mut quoted = false;
    let mut chars = text.trim_start_matches('\u{feff}').chars().peekable();
    while let Some(c) = chars.next() {
        if quoted {
            match c {
                '"' if chars.peek() == Some(&'"') => {
                    field.push('"');
                    chars.next();
                }
                '"' => quoted = false,
                other => field.push(other),
            }
            continue;
        }
        match c {
            '"' if field.is_empty() => quoted = true,
            ',' => row.push(std::mem::take(&mut field)),
            '\r' => {}
            '\n' => {
                row.push(std::mem::take(&mut field));
                if !(row.len() == 1 && row[0].is_empty()) {
                    rows.push(std::mem::take(&mut row));
                }
                row.clear();
                if rows.len() > MAX_ROWS {
                    break;
                }
            }
            other => field.push(other),
        }
    }
    if !field.is_empty() || !row.is_empty() {
        row.push(field);
        rows.push(row);
    }
    rows
}

/// A column's format: a pattern, a unit or a currency, written as text or a name.
fn format_of(column: &CsvColumn) -> Option<String> {
    match &column.format.as_ref()?.kind {
        ExprKind::Text(parts) => match parts.as_slice() {
            [TextPart::Literal(text)] => Some(text.clone()),
            _ => None,
        },
        ExprKind::Name(name) | ExprKind::Raw(name) => Some(name.clone()),
        _ => None,
    }
}

/// Reads a CSV file as rows of typed cells; problems name the row and column.
pub fn rows(
    text: &str,
    columns: &[(CsvColumn, Type)],
    zone: ZoneValue,
) -> (Vec<Value>, Vec<Message>) {
    let mut problems = Vec::new();
    let mut all = records(text).into_iter();
    let Some(header) = all.next() else {
        return (
            Vec::new(),
            vec![Message::new(message!("The file has no header row"))],
        );
    };
    let positions: Vec<Option<usize>> = columns
        .iter()
        .map(|(column, _)| {
            header
                .iter()
                .position(|name| name.trim().eq_ignore_ascii_case(&column.name.name))
        })
        .collect();
    for ((column, _), position) in columns.iter().zip(&positions) {
        if position.is_none() {
            problems.push(Message::with(
                message!("The file has no column {0}"),
                [Arg::from(column.name.name.as_str())],
            ));
        }
    }
    let mut values = Vec::new();
    for (row_index, record) in all.enumerate() {
        let mut fields = BTreeMap::new();
        for ((column, ty), position) in columns.iter().zip(&positions) {
            let raw = position
                .and_then(|position| record.get(position))
                .map_or("", |cell| cell.trim());
            let value = cell(raw, ty, format_of(column).as_deref(), zone);
            if value.is_none() && !raw.is_empty() {
                problems.push(Message::with(
                    message!("Row {0}, column {1}: {2} does not read as {3}"),
                    [
                        Arg::from(row_index + 2),
                        Arg::from(column.name.name.as_str()),
                        Arg::from(raw),
                        Arg::from(ty.unwrapped().to_string()),
                    ],
                ));
            }
            fields.insert(column.name.name.clone(), value);
        }
        values.push(Value::Struct(Rc::new(fields)));
    }
    (values, problems)
}

fn cell(raw: &str, ty: &Type, format: Option<&str>, zone: ZoneValue) -> Value {
    if raw.is_empty() {
        return Value::None;
    }
    let number = || raw.replace([',', '$'], "").parse::<Decimal>().ok();
    match ty.unwrapped() {
        Type::Int => raw.replace(',', "").parse().map_or(Value::None, Value::Int),
        Type::Decimal => number().map_or(Value::None, Value::Decimal),
        Type::Bool => match raw.to_ascii_lowercase().as_str() {
            "true" | "yes" | "y" | "1" => Value::Bool(true),
            "false" | "no" | "n" | "0" => Value::Bool(false),
            _ => Value::None,
        },
        Type::Date => parse_pattern(raw, format.unwrap_or("yyyy-MM-dd"))
            .and_then(|parts| parts.date())
            .map_or(Value::None, Value::Date),
        Type::Time => parse_pattern(raw, format.unwrap_or("HH:mm"))
            .and_then(|parts| parts.time())
            .map_or(Value::None, Value::Time),
        Type::DateTime => parse_pattern(raw, format.unwrap_or("yyyy-MM-dd HH:mm"))
            .and_then(|parts| Some((parts.date()?, parts.time().unwrap_or(0))))
            .map_or(Value::None, |(days, seconds)| {
                Value::DateTime(zone.instant(days * 86_400 + i64::from(seconds)), zone)
            }),
        Type::Money => number().map_or(Value::None, |amount| {
            Value::Money(amount, Rc::from(format.unwrap_or("USD")))
        }),
        Type::Measure(dim) => {
            let unit = format
                .and_then(Unit::named)
                .filter(|unit| unit.dim == *dim)
                .unwrap_or(Unit::base(*dim));
            number().map_or(Value::None, |amount| Value::Measure(amount, unit))
        }
        Type::Percent => raw
            .trim_end_matches('%')
            .parse()
            .map_or(Value::None, Value::Percent),
        _ => Value::text(raw),
    }
}

#[cfg(test)]
mod tests {
    use super::records;

    #[test]
    fn quoted_fields_keep_commas_and_quotes() {
        let rows = records("order,origin\r\n\"A-1\",\"Dallas, TX\"\n\"say \"\"hi\"\"\",x\n");
        assert_eq!(rows.len(), 3);
        assert_eq!(rows[1][1], "Dallas, TX");
        assert_eq!(rows[2][0], "say \"hi\"");
    }
}
