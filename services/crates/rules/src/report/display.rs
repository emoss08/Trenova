//! The display contract of a report column and how it is resolved (`reportfmt.Resolve`).
//!
//! A column's display starts from a baseline its catalog type and format hint imply (a money
//! column is currency with two places and parentheses for negatives), then takes the author's
//! overrides. Restyling to another style resets what that style implies before the overrides
//! apply, so "money shown as a plain number" drops the parentheses but keeps an explicit
//! suffix. The resolved display is what the server sends with every report column and what
//! `report.format` renders.
//!
//! Every field is a string or number on the wire with Go's zero value as its default, so a
//! display that is missing a field reads the way Go would decode it. A zero display (style
//! `""`) renders values raw.

use crate::allocation::input::nullable;
use crate::report::band::Band;
use crate::report::rules::Rule;
use serde::Deserialize;
use serde::Serialize;

/// The most decimal places a display may show.
pub const MAX_DECIMALS: i64 = 10;

/// A fully resolved display (`reportfmt.Resolved`).
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct Display {
    /// `""` (raw), `number`, `currency`, `percent`, `duration`, `date`, `bool` or `text`.
    #[serde(deserialize_with = "nullable")]
    pub style: String,
    /// Places after the point, clamped to 0..=10 when used.
    pub decimals: i64,
    /// Whether the whole part is grouped by thousands.
    pub grouping: bool,
    /// The ISO currency code of a currency column.
    #[serde(deserialize_with = "nullable")]
    pub currency: String,
    /// `minus` or `parentheses`.
    #[serde(deserialize_with = "nullable")]
    pub negative: String,
    /// `standard` or `compact` (K, M, B, T).
    #[serde(deserialize_with = "nullable")]
    pub notation: String,
    /// Text written before the value.
    #[serde(deserialize_with = "nullable")]
    pub prefix: String,
    /// Text written after the value.
    #[serde(deserialize_with = "nullable")]
    pub suffix: String,
    /// How a date is written; dates are formatted by the host.
    #[serde(deserialize_with = "nullable")]
    pub date_style: String,
    /// `yesNo`, `trueFalse`, `onOff`, `check` or `oneZero`.
    #[serde(deserialize_with = "nullable")]
    pub bool_style: String,
    /// What one unit of a duration value is: `seconds`, `minutes`, `hours` or `days`.
    #[serde(deserialize_with = "nullable")]
    pub duration_unit: String,
    /// `clock`, `compact`, `verbose` or `decimalHours`.
    #[serde(deserialize_with = "nullable")]
    pub duration_style: String,
    /// What an empty value shows.
    #[serde(deserialize_with = "nullable")]
    pub null_text: String,
    /// Tone rules, first match wins.
    #[serde(skip_serializing_if = "Vec::is_empty")]
    #[serde(deserialize_with = "nullable")]
    pub rules: Vec<Rule>,
    /// The ranges a banded column groups into.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub band: Option<Band>,
}

impl Display {
    /// Whether the style renders a number (`IsNumeric`).
    #[must_use]
    pub fn is_numeric(&self) -> bool {
        matches!(
            self.style.as_str(),
            "number" | "currency" | "percent" | "duration"
        )
    }
}

/// An author's display overrides (`reportfmt.Spec`). Every field is optional.
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize)]
#[serde(default, rename_all = "camelCase")]
pub struct Spec {
    /// The style to show the column in.
    #[serde(deserialize_with = "nullable")]
    pub style: String,
    /// Places after the point.
    pub decimals: Option<i64>,
    /// Whether to group thousands.
    pub grouping: Option<bool>,
    /// The currency code, upper-cased when applied.
    #[serde(deserialize_with = "nullable")]
    pub currency: String,
    /// The negative style.
    #[serde(deserialize_with = "nullable")]
    pub negative: String,
    /// The notation.
    #[serde(deserialize_with = "nullable")]
    pub notation: String,
    /// A prefix.
    #[serde(deserialize_with = "nullable")]
    pub prefix: String,
    /// A suffix.
    #[serde(deserialize_with = "nullable")]
    pub suffix: String,
    /// The date style.
    #[serde(deserialize_with = "nullable")]
    pub date_style: String,
    /// The boolean style.
    #[serde(deserialize_with = "nullable")]
    pub bool_style: String,
    /// The duration unit.
    #[serde(deserialize_with = "nullable")]
    pub duration_unit: String,
    /// The duration style.
    #[serde(deserialize_with = "nullable")]
    pub duration_style: String,
    /// The empty-value text.
    #[serde(deserialize_with = "nullable")]
    pub null_text: String,
    /// Tone rules.
    #[serde(deserialize_with = "nullable")]
    pub rules: Vec<Rule>,
}

/// What a column's display is resolved from (`reportfmt.ResolveInput`).
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize)]
#[serde(default, rename_all = "camelCase")]
pub struct ResolveInput {
    /// The catalog field type: `string`, `int`, `decimal`, `bool`, `enum`, `epoch`, `ref`, `json`.
    #[serde(rename = "type")]
    #[serde(deserialize_with = "nullable")]
    pub field_type: String,
    /// The catalog format hint: `money`, `weight`, `percent`, `duration`, `distance`, `count`.
    #[serde(deserialize_with = "nullable")]
    pub hint: String,
    /// The date style a dimension's date bucket implies.
    #[serde(deserialize_with = "nullable")]
    pub date_style: String,
    /// The ranges a banded dimension groups into.
    pub band: Option<Band>,
    /// The author's overrides.
    pub spec: Option<Spec>,
}

/// Resolves a column's display.
#[must_use]
pub fn resolve(input: &ResolveInput) -> Display {
    let mut display = baseline(&input.field_type, &input.hint);
    if !input.date_style.is_empty() && display.style == "date" {
        display.date_style.clone_from(&input.date_style);
    }
    if let Some(band) = input.band.as_ref().filter(|band| !band.is_empty()) {
        display.band = Some(band.clone());
    }
    let Some(spec) = &input.spec else {
        return display;
    };
    if !spec.style.is_empty() && spec.style != display.style {
        display = restyle(display, &spec.style);
    }
    if let Some(decimals) = spec.decimals {
        display.decimals = decimals.clamp(0, MAX_DECIMALS);
    }
    if let Some(grouping) = spec.grouping {
        display.grouping = grouping;
    }
    if !spec.currency.is_empty() {
        display.currency = spec.currency.to_ascii_uppercase();
    }
    replace_if_set(&mut display.negative, &spec.negative);
    replace_if_set(&mut display.notation, &spec.notation);
    replace_if_set(&mut display.prefix, &spec.prefix);
    replace_if_set(&mut display.suffix, &spec.suffix);
    replace_if_set(&mut display.date_style, &spec.date_style);
    replace_if_set(&mut display.bool_style, &spec.bool_style);
    replace_if_set(&mut display.duration_unit, &spec.duration_unit);
    replace_if_set(&mut display.duration_style, &spec.duration_style);
    replace_if_set(&mut display.null_text, &spec.null_text);
    if !spec.rules.is_empty() {
        display.rules.clone_from(&spec.rules);
    }
    display
}

fn replace_if_set(field: &mut String, value: &str) {
    if !value.is_empty() {
        value.clone_into(field);
    }
}

fn baseline(field_type: &str, hint: &str) -> Display {
    let mut display = Display {
        style: String::from("text"),
        decimals: 0,
        grouping: true,
        currency: String::from("USD"),
        negative: String::from("minus"),
        notation: String::from("standard"),
        prefix: String::new(),
        suffix: String::new(),
        date_style: String::from("dateTime"),
        bool_style: String::from("yesNo"),
        duration_unit: String::from("seconds"),
        duration_style: String::from("clock"),
        null_text: String::new(),
        rules: Vec::new(),
        band: None,
    };
    match field_type {
        "epoch" => display.style = String::from("date"),
        "bool" => display.style = String::from("bool"),
        "int" | "decimal" => {
            display.style = String::from("number");
            if field_type == "decimal" {
                display.decimals = 2;
            }
            apply_hint(&mut display, hint);
        }
        _ => {}
    }
    display
}

fn apply_hint(display: &mut Display, hint: &str) {
    match hint {
        "money" => {
            display.style = String::from("currency");
            display.decimals = 2;
            display.negative = String::from("parentheses");
        }
        "percent" => {
            display.style = String::from("percent");
            display.decimals = 1;
        }
        "duration" => {
            display.style = String::from("duration");
            display.decimals = 0;
        }
        "distance" => {
            display.decimals = 1;
            display.suffix = String::from(" mi");
        }
        "weight" => {
            display.decimals = 0;
            display.suffix = String::from(" lbs");
        }
        "count" => display.decimals = 0,
        _ => {}
    }
}

fn restyle(mut display: Display, style: &str) -> Display {
    style.clone_into(&mut display.style);
    match style {
        "currency" => {
            display.decimals = 2;
            display.negative = String::from("parentheses");
        }
        "percent" => display.decimals = 1,
        "duration" => display.decimals = 0,
        "number" => display.suffix.clear(),
        _ => {}
    }
    display
}
