//! Value trees on the wire: the formula environment in, values out.
//!
//! A value tree is JSON with two tagged objects: `{"$date": "<RFC 3339>"}` and
//! `{"$duration": "<nanoseconds>"}`. Any other key starting with `$` is reserved and refused.
//!
//! Numbers are read from their JSON text by [`crate::wire::parse_decimal`], never through `f64`,
//! so `0.1` is exactly one tenth. That is why this is a small reader of its own rather than
//! `serde_json::Value`, which would round every number through a float. It refuses what JSON
//! refuses (trailing commas, comments, lone surrogates), duplicate keys, and nesting past
//! [`MAX_DEPTH`].
//!
//! Output trees follow the wire convention for decimals: a number is written as a JSON string,
//! because `JSON.parse` would turn it back into a float.

use super::time::parse_rfc3339;
use super::value::MAX_DEPTH;
use super::value::Value;
use crate::wire::parse_decimal;
use std::collections::BTreeMap;
use std::fmt::Write as _;
use std::rc::Rc;

/// Reads one value tree, or says (in developer English) why it is not one.
pub fn parse_value(text: &str) -> Result<Value, String> {
    let mut reader = Reader {
        bytes: text.as_bytes(),
        text,
        at: 0,
    };
    reader.skip_space();
    let value = reader.value(0)?;
    reader.skip_space();
    if reader.at != reader.bytes.len() {
        return Err(reader.fail("unexpected text after the value"));
    }
    Ok(value)
}

struct Reader<'a> {
    bytes: &'a [u8],
    text: &'a str,
    at: usize,
}

impl Reader<'_> {
    fn fail(&self, what: &str) -> String {
        format!("{what} at byte {}", self.at)
    }

    fn skip_space(&mut self) {
        while matches!(self.bytes.get(self.at), Some(b' ' | b'\t' | b'\n' | b'\r')) {
            self.at += 1;
        }
    }

    fn eat(&mut self, literal: &str) -> bool {
        if self.bytes[self.at..].starts_with(literal.as_bytes()) {
            self.at += literal.len();
            true
        } else {
            false
        }
    }

    fn value(&mut self, depth: u8) -> Result<Value, String> {
        match self.bytes.get(self.at) {
            Some(b'{') => self.object(depth),
            Some(b'[') => self.array(depth),
            Some(b'"') => Ok(Value::Text(Rc::from(self.string()?))),
            Some(b't') if self.eat("true") => Ok(Value::Bool(true)),
            Some(b'f') if self.eat("false") => Ok(Value::Bool(false)),
            Some(b'n') if self.eat("null") => Ok(Value::Nil),
            Some(b'-' | b'0'..=b'9') => self.number(),
            Some(_) => Err(self.fail("expected a value")),
            None => Err(self.fail("unexpected end of input")),
        }
    }

    fn nested(&self, depth: u8) -> Result<u8, String> {
        if depth >= MAX_DEPTH {
            return Err(self.fail(&format!("values nest deeper than {MAX_DEPTH} levels")));
        }
        Ok(depth + 1)
    }

    fn array(&mut self, depth: u8) -> Result<Value, String> {
        let depth = self.nested(depth)?;
        self.at += 1;
        let mut items = Vec::new();
        self.skip_space();
        if self.bytes.get(self.at) == Some(&b']') {
            self.at += 1;
            return Value::list(items).map_err(|_| self.fail("values nest too deeply"));
        }
        loop {
            self.skip_space();
            items.push(self.value(depth)?);
            self.skip_space();
            match self.bytes.get(self.at) {
                Some(b',') => self.at += 1,
                Some(b']') => {
                    self.at += 1;
                    return Value::list(items).map_err(|_| self.fail("values nest too deeply"));
                }
                _ => return Err(self.fail("expected ',' or ']'")),
            }
        }
    }

    fn object(&mut self, depth: u8) -> Result<Value, String> {
        let depth = self.nested(depth)?;
        let start = self.at;
        self.at += 1;
        let mut entries: BTreeMap<Rc<str>, Value> = BTreeMap::new();
        self.skip_space();
        if self.bytes.get(self.at) == Some(&b'}') {
            self.at += 1;
            return Value::map(entries).map_err(|_| self.fail("values nest too deeply"));
        }
        loop {
            self.skip_space();
            if self.bytes.get(self.at) != Some(&b'"') {
                return Err(self.fail("expected a key"));
            }
            let key = self.string()?;
            self.skip_space();
            if self.bytes.get(self.at) != Some(&b':') {
                return Err(self.fail("expected ':'"));
            }
            self.at += 1;
            self.skip_space();
            if let Some(tag) = key.strip_prefix('$') {
                if !entries.is_empty() {
                    return Err(self.fail("a $-tagged object has exactly one key"));
                }
                return self.tagged(tag, start);
            }
            let value = self.value(depth)?;
            if entries.insert(Rc::from(key), value).is_some() {
                return Err(self.fail("duplicate key"));
            }
            self.skip_space();
            match self.bytes.get(self.at) {
                Some(b',') => self.at += 1,
                Some(b'}') => {
                    self.at += 1;
                    return Value::map(entries).map_err(|_| self.fail("values nest too deeply"));
                }
                _ => return Err(self.fail("expected ',' or '}'")),
            }
        }
    }

    fn tagged(&mut self, tag: &str, start: usize) -> Result<Value, String> {
        if self.bytes.get(self.at) != Some(&b'"') {
            return Err(self.fail(&format!("${tag} takes a string")));
        }
        let payload = self.string()?;
        self.skip_space();
        if self.bytes.get(self.at) != Some(&b'}') {
            return Err(self.fail(&format!("${tag} must be the object's only key")));
        }
        self.at += 1;
        match tag {
            "date" => parse_rfc3339(&payload).map(Value::Date).ok_or_else(|| {
                format!("{payload:?} at byte {start} is not an RFC 3339 date with an offset")
            }),
            "duration" => {
                let valid = !payload.is_empty()
                    && payload
                        .strip_prefix('-')
                        .unwrap_or(&payload)
                        .bytes()
                        .all(|byte| byte.is_ascii_digit());
                valid
                    .then(|| payload.parse::<i64>().ok())
                    .flatten()
                    .map(Value::Duration)
                    .ok_or_else(|| {
                        format!("{payload:?} at byte {start} is not a whole number of nanoseconds")
                    })
            }
            _ => Err(format!(
                "${tag} at byte {start} is reserved: keys starting with $ are value tags"
            )),
        }
    }

    fn number(&mut self) -> Result<Value, String> {
        let start = self.at;
        if self.bytes.get(self.at) == Some(&b'-') {
            self.at += 1;
        }
        let digits = |reader: &mut Self| {
            let begin = reader.at;
            while reader.bytes.get(reader.at).is_some_and(u8::is_ascii_digit) {
                reader.at += 1;
            }
            reader.at - begin
        };
        let whole_start = self.at;
        let whole = digits(self);
        if whole == 0 || (whole > 1 && self.bytes.get(whole_start) == Some(&b'0')) {
            return Err(self.fail("malformed number"));
        }
        if self.bytes.get(self.at) == Some(&b'.') {
            self.at += 1;
            if digits(self) == 0 {
                return Err(self.fail("malformed number"));
            }
        }
        if matches!(self.bytes.get(self.at), Some(b'e' | b'E')) {
            self.at += 1;
            if matches!(self.bytes.get(self.at), Some(b'+' | b'-')) {
                self.at += 1;
            }
            if digits(self) == 0 {
                return Err(self.fail("malformed number"));
            }
        }
        parse_decimal(&self.text[start..self.at])
            .map(Value::Number)
            .map_err(|error| format!("{error} at byte {start}"))
    }

    fn string(&mut self) -> Result<String, String> {
        self.at += 1;
        let mut out = String::new();
        loop {
            let run_start = self.at;
            while let Some(byte) = self.bytes.get(self.at) {
                if *byte == b'"' || *byte == b'\\' || *byte < 0x20 {
                    break;
                }
                self.at += 1;
            }
            out.push_str(&self.text[run_start..self.at]);
            match self.bytes.get(self.at) {
                Some(b'"') => {
                    self.at += 1;
                    return Ok(out);
                }
                Some(b'\\') => {
                    self.at += 1;
                    self.escape(&mut out)?;
                }
                Some(_) => return Err(self.fail("control character in a string")),
                None => return Err(self.fail("unterminated string")),
            }
        }
    }

    fn escape(&mut self, out: &mut String) -> Result<(), String> {
        let Some(byte) = self.bytes.get(self.at).copied() else {
            return Err(self.fail("unterminated escape"));
        };
        self.at += 1;
        let character = match byte {
            b'"' => '"',
            b'\\' => '\\',
            b'/' => '/',
            b'b' => '\u{8}',
            b'f' => '\u{c}',
            b'n' => '\n',
            b'r' => '\r',
            b't' => '\t',
            b'u' => {
                let high = self.hex4()?;
                let code = if (0xD800..0xDC00).contains(&high) {
                    if !self.eat("\\u") {
                        return Err(self.fail("lone surrogate"));
                    }
                    let low = self.hex4()?;
                    if !(0xDC00..0xE000).contains(&low) {
                        return Err(self.fail("lone surrogate"));
                    }
                    0x10000 + ((high - 0xD800) << 10) + (low - 0xDC00)
                } else if (0xDC00..0xE000).contains(&high) {
                    return Err(self.fail("lone surrogate"));
                } else {
                    high
                };
                char::from_u32(code).ok_or_else(|| self.fail("invalid escape"))?
            }
            _ => return Err(self.fail("invalid escape")),
        };
        out.push(character);
        Ok(())
    }

    fn hex4(&mut self) -> Result<u32, String> {
        let slice = self
            .text
            .get(self.at..self.at + 4)
            .ok_or_else(|| self.fail("short \\u escape"))?;
        let code = u32::from_str_radix(slice, 16).map_err(|_| self.fail("invalid \\u escape"))?;
        self.at += 4;
        Ok(code)
    }
}

/// Writes a value tree: numbers as JSON strings, dates as `{"$date": ...}` in their own offset,
/// durations as `{"$duration": "<nanoseconds>"}`, maps in key order.
#[must_use]
pub fn write_value(value: &Value) -> String {
    let mut out = String::new();
    write_into(value, &mut out);
    out
}

fn write_into(value: &Value, out: &mut String) {
    match value {
        Value::Nil => out.push_str("null"),
        Value::Bool(flag) => out.push_str(if *flag { "true" } else { "false" }),
        Value::Number(number) => write_string(&super::value::format_number(*number), out),
        Value::Text(text) => write_string(text, out),
        Value::Date(date) => {
            out.push_str("{\"$date\":");
            write_string(&date.rfc3339(), out);
            out.push('}');
        }
        Value::Duration(nanos) => {
            let _ = write!(out, "{{\"$duration\":\"{nanos}\"}}");
        }
        Value::List(list) => {
            out.push('[');
            for (index, item) in list.items().iter().enumerate() {
                if index > 0 {
                    out.push(',');
                }
                write_into(item, out);
            }
            out.push(']');
        }
        Value::Map(map) => {
            out.push('{');
            for (index, (key, item)) in map.entries().iter().enumerate() {
                if index > 0 {
                    out.push(',');
                }
                write_string(key, out);
                out.push(':');
                write_into(item, out);
            }
            out.push('}');
        }
    }
}

fn write_string(text: &str, out: &mut String) {
    out.push('"');
    for character in text.chars() {
        match character {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            control if u32::from(control) < 0x20 => {
                let _ = write!(out, "\\u{:04x}", u32::from(control));
            }
            other => out.push(other),
        }
    }
    out.push('"');
}

#[cfg(test)]
mod tests {
    use super::{parse_value, write_value};
    use crate::formula::eval::value::Value;

    #[test]
    fn numbers_are_read_exactly() {
        let value = parse_value(r#"{"a": 0.1, "b": [1.005, -2e-3, 12], "c": "x"}"#).expect("tree");
        assert_eq!(
            write_value(&value),
            r#"{"a":"0.1","b":["1.005","-0.002","12"],"c":"x"}"#
        );
    }

    #[test]
    fn dates_and_durations_are_tagged() {
        let value = parse_value(
            r#"{"at": {"$date": "2026-01-02T03:04:05-05:00"}, "for": {"$duration": "-90"}}"#,
        )
        .expect("tree");
        assert_eq!(
            write_value(&value),
            r#"{"at":{"$date":"2026-01-02T03:04:05-05:00"},"for":{"$duration":"-90"}}"#
        );
        assert!(matches!(
            parse_value(r#"{"$duration": "1h"}"#),
            Err(message) if message.contains("nanoseconds")
        ));
        assert!(parse_value(r#"{"$date": "2026-01-02"}"#).is_err());
        assert!(parse_value(r#"{"$other": "x"}"#).is_err());
        assert!(parse_value(r#"{"a": 1, "$date": "2026-01-02T00:00:00Z"}"#).is_err());
    }

    #[test]
    fn refuses_what_json_refuses() {
        for text in [
            "",
            "[1,]",
            "{\"a\":1,}",
            "01",
            "1.",
            "nul",
            "\"\\ud800\"",
            "{\"a\":1,\"a\":2}",
            "[1] [2]",
            "\"a\nb\"",
        ] {
            assert!(parse_value(text).is_err(), "{text:?}");
        }
    }

    #[test]
    fn escapes_round_trip() {
        let value = parse_value(r#""a\"b\\c\n\u00e9\ud83d\ude9a""#).expect("tree");
        assert!(matches!(&value, Value::Text(text) if &**text == "a\"b\\c\né🚚"));
        assert_eq!(write_value(&value), "\"a\\\"b\\\\c\\né🚚\"");
    }

    #[test]
    fn nesting_is_bounded() {
        let deep = format!("{}{}", "[".repeat(40), "]".repeat(40));
        assert!(parse_value(&deep).is_err());
        let fine = format!("{}{}", "[".repeat(10), "]".repeat(10));
        assert!(parse_value(&fine).is_ok());
    }
}
