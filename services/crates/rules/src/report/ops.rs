//! The `report.*` operations.
//!
//! `report.format` works on columns, because the server formats up to 100,000 rows of an export
//! and the browser a virtualised grid: one call carries every column with its display and its
//! values. `report.resolve` resolves the displays a report builder previews before the server
//! has seen the report.

use crate::report::display::Display;
use crate::report::display::ResolveInput;
use crate::report::display::resolve as resolve_display;
use crate::report::format::Rendered;
use crate::report::format::Value;
use crate::report::format::render;
use crate::wire::Dec;
use crate::wire::parse_decimal;
use serde::Deserialize;
use serde::Serialize;
use serde_json::value::RawValue;
use std::borrow::Cow;
use trenova_module_kit::ops::OpResult;
use trenova_module_kit::ops::decode;
use trenova_module_kit::ops::encode;

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct Column<'a> {
    #[serde(default)]
    display: Display,
    #[serde(default)]
    resolve: Option<ResolveInput>,
    #[serde(default)]
    value_type: String,
    #[serde(default, borrow)]
    values: Vec<&'a RawValue>,
}

#[derive(Deserialize)]
struct FormatInput<'a> {
    #[serde(default, borrow)]
    columns: Vec<Column<'a>>,
}

#[derive(Serialize)]
struct Cell<'a> {
    #[serde(skip_serializing_if = "Option::is_none")]
    text: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    date: Option<Dec>,
    #[serde(skip_serializing_if = "Option::is_none")]
    tone: Option<&'a str>,
}

#[derive(Serialize)]
struct ColumnOut<'a> {
    values: Vec<Cell<'a>>,
}

#[derive(Serialize)]
struct FormatOut<'a> {
    columns: Vec<ColumnOut<'a>>,
}

/// Reads one wire value. A JSON number is a number. A JSON string is text, except in a column
/// whose `valueType` is `int`, `decimal` or `epoch`, where a string holding a decimal is that
/// number (decimals cross GraphQL as strings so they stay exact), and in a `bool` column, where
/// `"true"` and `"false"` are booleans.
fn classify<'a>(raw: &'a RawValue, value_type: &str) -> Value<'a> {
    let text = raw.get();
    match text.as_bytes().first() {
        None | Some(b'n') => Value::Null,
        Some(b't') => Value::Bool(true),
        Some(b'f') => Value::Bool(false),
        Some(b'"') => {
            let Ok(string) = serde_json::from_str::<Cow<'a, str>>(text) else {
                return Value::Other;
            };
            match value_type {
                "int" | "decimal" | "epoch" if !string.trim().is_empty() => {
                    match parse_decimal(&string) {
                        Ok(number) => Value::Number(number),
                        Err(_) => Value::Text(string),
                    }
                }
                "bool" if string == "true" => Value::Bool(true),
                "bool" if string == "false" => Value::Bool(false),
                _ => Value::Text(string),
            }
        }
        Some(b'[' | b'{') => Value::Other,
        Some(_) => match parse_decimal(text) {
            Ok(number) => Value::Number(number),
            Err(_) => Value::Text(Cow::Borrowed(text)),
        },
    }
}

/// `report.format`: renders every value of every column, with its tone.
pub fn format(input: &RawValue) -> OpResult {
    let input: FormatInput<'_> = decode(input)?;
    let displays: Vec<Cow<'_, Display>> = input
        .columns
        .iter()
        .map(|column| match &column.resolve {
            Some(resolve) => Cow::Owned(resolve_display(resolve)),
            None => Cow::Borrowed(&column.display),
        })
        .collect();
    let columns = input
        .columns
        .iter()
        .zip(&displays)
        .map(|(column, display)| ColumnOut {
            values: column
                .values
                .iter()
                .map(|raw| {
                    let value = classify(raw, &column.value_type);
                    let (rendered, tone) = render(display, &value);
                    match rendered {
                        Rendered::Text(text) => Cell {
                            text: Some(text),
                            date: None,
                            tone,
                        },
                        Rendered::Date(seconds) => Cell {
                            text: None,
                            date: Some(Dec(seconds)),
                            tone,
                        },
                    }
                })
                .collect(),
        })
        .collect();
    encode(&FormatOut { columns })
}

#[derive(Deserialize)]
struct ResolveRequest {
    #[serde(default)]
    columns: Vec<ResolveInput>,
}

#[derive(Serialize)]
struct ResolveOut {
    displays: Vec<Display>,
}

/// `report.resolve`: resolves each column's display from its type, hint, band and overrides.
pub fn resolve(input: &RawValue) -> OpResult {
    let input: ResolveRequest = decode(input)?;
    encode(&ResolveOut {
        displays: input.columns.iter().map(resolve_display).collect(),
    })
}
