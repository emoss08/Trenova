//! Values as statement traces and lookup receipts show them, exactly as the Go engine's
//! `traceValue` did, so the Studio and receipts read the same:
//!
//! - text longer than 500 characters is ellipsised (trimmed, cut to 499 characters, `…`);
//! - numbers are decimal strings (`"500"`, `"0.3"`);
//! - dates are RFC 3339 in UTC to the second; durations are Go's `1h30m0s`;
//! - lists and maps show 50 items and nest 3 levels, past which a list is `"[N items]"` and a
//!   map `"{N fields}"`; map entries whose value is nil are left out.

use super::time::format_duration;
use super::value::Value;
use super::value::format_number;
use serde_json::Map as JsonMap;
use serde_json::Value as Json;

const MAX_ITEMS: usize = 50;
const MAX_DEPTH: usize = 3;
const MAX_TEXT_CHARACTERS: usize = 500;

/// The value as a trace shows it.
#[must_use]
pub fn trace_value(value: &Value) -> Json {
    trace_at(value, 0)
}

fn trace_at(value: &Value, depth: usize) -> Json {
    match value {
        Value::Nil => Json::Null,
        Value::Bool(flag) => Json::Bool(*flag),
        Value::Text(text) => Json::String(ellipsize(text)),
        Value::Number(number) => Json::String(format_number(*number)),
        Value::Date(date) => Json::String(date.rfc3339_utc_seconds()),
        Value::Duration(nanos) => Json::String(format_duration(*nanos)),
        Value::List(list) => {
            if depth >= MAX_DEPTH {
                return Json::String(format!("[{} items]", list.items().len()));
            }
            Json::Array(
                list.items()
                    .iter()
                    .take(MAX_ITEMS)
                    .map(|item| trace_at(item, depth + 1))
                    .collect(),
            )
        }
        Value::Map(map) => {
            if depth >= MAX_DEPTH {
                return Json::String(format!("{{{} fields}}", map.entries().len()));
            }
            let mut out = JsonMap::new();
            for (key, item) in map.entries().iter().take(MAX_ITEMS) {
                let traced = trace_at(item, depth + 1);
                if !traced.is_null() {
                    out.insert(key.to_string(), traced);
                }
            }
            Json::Object(out)
        }
    }
}

/// Go's `stringutils.Ellipsize` applied past 500 characters.
fn ellipsize(text: &str) -> String {
    if text.chars().count() <= MAX_TEXT_CHARACTERS {
        return text.to_owned();
    }
    let trimmed = text.trim();
    if trimmed.chars().count() <= MAX_TEXT_CHARACTERS {
        return trimmed.to_owned();
    }
    let cut: String = trimmed.chars().take(MAX_TEXT_CHARACTERS - 1).collect();
    format!("{}\u{2026}", cut.trim_end_matches(' '))
}

#[cfg(test)]
mod tests {
    use super::trace_value;
    use crate::formula::eval::tree::parse_value;
    use crate::formula::eval::value::Value;

    fn traced(text: &str) -> String {
        trace_value(&parse_value(text).expect("tree")).to_string()
    }

    #[test]
    fn scalars_trace_as_go_did() {
        assert_eq!(traced("500.00"), "\"500\"");
        assert_eq!(traced("true"), "true");
        assert_eq!(
            traced(r#"{"$date": "2026-01-02T03:04:05-05:00"}"#),
            "\"2026-01-02T08:04:05Z\""
        );
        assert_eq!(traced(r#"{"$duration": "5400000000000"}"#), "\"1h30m0s\"");
    }

    #[test]
    fn long_text_is_ellipsised() {
        let text = Value::text(&"x".repeat(600));
        let out = trace_value(&text);
        let shown = out.as_str().expect("text");
        assert_eq!(shown.chars().count(), 500);
        assert!(shown.ends_with('\u{2026}'));
    }

    #[test]
    fn collections_are_bounded() {
        assert_eq!(
            traced(r#"{"b": null, "a": [1, [2, [3]]]}"#),
            r#"{"a":["1",["2","[1 items]"]]}"#
        );
        let wide = format!("[{}]", vec!["1"; 60].join(","));
        let out = trace_value(&parse_value(&wide).expect("tree"));
        assert_eq!(out.as_array().map(Vec::len), Some(50));
        assert_eq!(traced(r#"[[{"a": 1}]]"#), r#"[[{"a":"1"}]]"#);
        assert_eq!(traced(r#"[[[{"a": 1}]]]"#), r#"[[["{1 fields}"]]]"#);
    }
}
