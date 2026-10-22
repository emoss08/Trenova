//! Rendering one report value through its column's display (`reportfmt.Resolved.String`).
//!
//! The order of the decisions is Go's:
//!
//! 1. An empty value shows the null text.
//! 2. A raw display (style `""`) writes the value as it is.
//! 3. A number in a usable banded column shows the range it stands for.
//! 4. Otherwise the kind of value decides: a number in a date column is a date (the host writes
//!    it, since dates use time zones the module does not have), any other number is formatted
//!    by the numeric style, a boolean by the boolean style, and text with the prefix and suffix.
//!
//! A number is formatted exactly: a percent is shifted two places, compact notation picks K, M,
//! B or T on the unrounded magnitude, and the digits are rounded half away from zero
//! (`StringFixed`), so `1.005` is "1.01" with two places.

use crate::report::display::Display;
use crate::report::display::MAX_DECIMALS;
use crate::report::fixed::Digits;
use crate::report::fixed::div_magnitude;
use crate::report::fixed::group_thousands;
use crate::report::rules::tone_for;
use rust_decimal::Decimal;

/// A value as the formatter sees it.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Value<'a> {
    /// No value.
    Null,
    /// A number, exact.
    Number(Decimal),
    /// A boolean.
    Bool(bool),
    /// Text, including a number too large to read exactly, which is shown as written.
    Text(std::borrow::Cow<'a, str>),
    /// A JSON array or object, which a report never formats; it reads as empty.
    Other,
}

/// What a value renders to.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Rendered {
    /// The text to show.
    Text(String),
    /// A date for the host to write in its time zone: whole seconds since the Unix epoch,
    /// truncated toward zero.
    Date(Decimal),
}

/// Renders one value, with the tone its column's rules give it.
#[must_use]
pub fn render<'a>(display: &'a Display, value: &Value<'_>) -> (Rendered, Option<&'a str>) {
    let tone = match value {
        Value::Number(number) => tone_for(&display.rules, *number, display.style == "percent"),
        _ => None,
    };
    (render_value(display, value), tone)
}

fn render_value(display: &Display, value: &Value<'_>) -> Rendered {
    if matches!(value, Value::Null) {
        return Rendered::Text(display.null_text.clone());
    }
    if display.style.is_empty() {
        return Rendered::Text(raw(value));
    }
    if let (Some(band), Value::Number(number)) = (&display.band, value)
        && band.is_usable()
    {
        return Rendered::Text(band_label(display, *number));
    }
    match value {
        Value::Number(number) if display.style == "date" => Rendered::Date(number.trunc()),
        Value::Number(number) => Rendered::Text(numeric(display, *number)),
        Value::Bool(flag) => Rendered::Text(boolean(&display.bool_style, *flag).to_owned()),
        Value::Text(text) => Rendered::Text(text_value(display, text)),
        Value::Null | Value::Other => Rendered::Text(text_value(display, "")),
    }
}

fn raw(value: &Value<'_>) -> String {
    match value {
        Value::Number(number) => number.normalize().to_string(),
        Value::Bool(flag) => flag.to_string(),
        Value::Text(text) => text.to_string(),
        Value::Null | Value::Other => String::new(),
    }
}

fn text_value(display: &Display, text: &str) -> String {
    if text.is_empty() {
        return display.null_text.clone();
    }
    format!("{}{text}{}", display.prefix, display.suffix)
}

/// The text of a boolean in a boolean style.
#[must_use]
pub fn boolean(style: &str, flag: bool) -> &'static str {
    let (yes, no) = match style {
        "trueFalse" => ("True", "False"),
        "onOff" => ("On", "Off"),
        "check" => ("✓", "✗"),
        "oneZero" => ("1", "0"),
        _ => ("Yes", "No"),
    };
    if flag { yes } else { no }
}

fn places(decimals: i64) -> u32 {
    u32::try_from(decimals.clamp(0, MAX_DECIMALS)).unwrap_or(0)
}

const COMPACT: [(i64, &str); 4] = [(12, "T"), (9, "B"), (6, "M"), (3, "K")];

/// A number in the display's numeric style (`numeric`).
#[must_use]
pub fn numeric(display: &Display, value: Decimal) -> String {
    if display.style == "duration" {
        return duration(display, value);
    }
    let percent = display.style == "percent";
    let negative = value.is_sign_negative() && !value.is_zero();
    let mut magnitude = Digits::of(value);
    if percent {
        magnitude = magnitude.shift(2);
    }
    let places = places(display.decimals);
    let mut suffix = "";
    if display.notation == "compact"
        && let Some((power, letter)) = COMPACT
            .iter()
            .find(|(power, _)| magnitude.at_least_pow10(*power))
    {
        magnitude = magnitude.shift(-power);
        suffix = letter;
    }
    let digits = magnitude.fixed(places);
    let body = if display.grouping {
        group_thousands(&digits)
    } else {
        digits
    };
    let mut text = String::with_capacity(body.len() + display.prefix.len() + 8);
    text.push_str(&display.prefix);
    if display.style == "currency" {
        text.push_str(&currency_symbol(&display.currency));
    }
    text.push_str(&body);
    text.push_str(suffix);
    if percent {
        text.push('%');
    }
    text.push_str(&display.suffix);
    signed(display, text, negative)
}

fn signed(display: &Display, body: String, negative: bool) -> String {
    if !negative || body.is_empty() {
        return body;
    }
    if display.negative == "parentheses" {
        format!("({body})")
    } else {
        format!("-{body}")
    }
}

/// The symbol a currency code is written with.
#[must_use]
pub fn currency_symbol(code: &str) -> String {
    match code {
        "USD" | "" => String::from("$"),
        "CAD" => String::from("C$"),
        "MXN" => String::from("MX$"),
        "EUR" => String::from("€"),
        "GBP" => String::from("£"),
        "AUD" => String::from("A$"),
        "JPY" => String::from("¥"),
        other => format!("{other} "),
    }
}

fn unit_seconds(unit: &str) -> u128 {
    match unit {
        "minutes" => 60,
        "hours" => 3600,
        "days" => 86_400,
        _ => 1,
    }
}

fn duration(display: &Display, value: Decimal) -> String {
    let negative = value.is_sign_negative() && !value.is_zero();
    let seconds = value
        .mantissa()
        .unsigned_abs()
        .saturating_mul(unit_seconds(&display.duration_unit));
    let body = match display.duration_style.as_str() {
        "decimalHours" => {
            let digits = div_magnitude(seconds, value.scale(), 3600, 0).map_or_else(
                || String::from("0"),
                |hours| hours.fixed(places(display.decimals.max(1))),
            );
            let digits = if display.grouping {
                group_thousands(&digits)
            } else {
                digits
            };
            format!("{digits} h")
        }
        style => {
            let whole = Digits::from_parts(seconds, i64::from(value.scale()))
                .whole()
                .unwrap_or(u128::MAX);
            match style {
                "compact" => compact_duration(whole, false),
                "verbose" => compact_duration(whole, true),
                _ => clock(whole),
            }
        }
    };
    signed(
        display,
        format!("{}{body}{}", display.prefix, display.suffix),
        negative,
    )
}

fn clock(seconds: u128) -> String {
    let hours = seconds / 3600;
    let minutes = (seconds % 3600) / 60;
    format!("{hours}:{minutes:02}")
}

fn compact_duration(seconds: u128, verbose: bool) -> String {
    let units = [
        (seconds / 86_400, "d", "day"),
        ((seconds % 86_400) / 3600, "h", "hour"),
        ((seconds % 3600) / 60, "m", "minute"),
    ];
    let parts: Vec<String> = units
        .iter()
        .filter(|(count, _, _)| *count != 0)
        .map(|(count, short, long)| {
            if verbose {
                let plural = if *count == 1 { "" } else { "s" };
                format!("{count} {long}{plural}")
            } else {
                format!("{count}{short}")
            }
        })
        .collect();
    if parts.is_empty() {
        return String::from(if verbose { "0 minutes" } else { "0m" });
    }
    parts.join(" ")
}

const RANGE_SEPARATOR: &str = " – ";

/// The range a banded value stands for, each end in the column's own display. A trailing unit
/// is written once, after the range; a leading symbol leads each end.
fn band_label(display: &Display, value: Decimal) -> String {
    let Some(band) = &display.band else {
        return numeric(display, value);
    };
    let bounds = band.bounds(value);
    let Some(upper) = bounds.upper else {
        return format!("{}+", numeric(display, bounds.lower));
    };
    let mut lead = display.clone();
    lead.suffix.clear();
    format!(
        "{}{RANGE_SEPARATOR}{}",
        numeric(&lead, bounds.lower),
        numeric(display, upper)
    )
}
