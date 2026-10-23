//! The `tariff.*` operations this area answers: `tariff.format`, `tariff.tokens` and
//! `tariff.setName`.
//!
//! `tariff.format`: `{"source": "…"}` gives `{"formatted": "…"}`, or, for a file with errors,
//! `{"formatted": null, "diagnostics": [...]}` in the wire shape every operation uses
//! ([`WireDiagnostic`]).
//!
//! `tariff.tokens`: `{"source": "…"}` gives `{"tokens": [{"from": 0, "to": 6, "kind":
//! "keyword"}, …]}`: every token but newlines and the end of the file, in UTF-16 code units, each
//! classed for highlighting (`keyword`, `name`, `function`, `field`, `type`, `code`, `number`,
//! `unit`, `money`, `percent`, `date`, `text`, `escape`, `interpolation`, `punct`, `comment`,
//! `doc`, `error`). The lexer reads every `camelCase` word as a name; the tokens beside it say
//! more, so a name before `(` is classed `function` (called, or declared with `fn`) and a name
//! after `.` or `?.` is classed `field`. Both are read from the tokens alone, so a broken file
//! still highlights.
//!
//! `tariff.setName`: `{"source": "…", "name": "…"}` gives `{"source": "…", "change": {"range":
//! {"from": 7, "to": 18}, "insert": "\"…\""}}`: the file with its header naming `name`, and the
//! one edit that makes it, in UTF-16 code units, so an editor can apply it as a single undoable
//! change. The name is written as a text literal (`\"`, `\\`, `\n`, `\t`, `{{` and `}}`
//! escaped), so the header reads back exactly `name`. A header without its name gets one after
//! the keyword; a file without a header gets `tariff "…"` and a blank line at the top. A record's
//! name is its header's, so this is how anything that renames a record rewrites its source.

use super::ast::Header;
use super::format::format as format_source;
use super::parse;
use super::token::Token;
use super::token::TokenKind;
use crate::ops::OpResult;
use crate::ops::decode;
use crate::ops::encode;
use crate::span::Range;
use crate::span::Span;
use crate::span::Utf16Index;
use crate::tariff::diagnostic::WireDiagnostic;
use serde::Deserialize;
use serde::Serialize;
use serde_json::value::RawValue;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct SourceInput {
    source: String,
}

#[derive(Serialize)]
struct FormatOutput {
    formatted: Option<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    diagnostics: Vec<WireDiagnostic>,
}

/// `tariff.format`.
///
/// # Errors
///
/// When the input is not `{"source": "…"}`.
pub fn format(input: &RawValue) -> OpResult {
    let input: SourceInput = decode(input)?;
    let output = match format_source(&input.source) {
        Ok(formatted) => FormatOutput {
            formatted: Some(formatted),
            diagnostics: Vec::new(),
        },
        Err(errors) => {
            let index = Utf16Index::new(&input.source);
            FormatOutput {
                formatted: None,
                diagnostics: errors
                    .iter()
                    .map(|diagnostic| WireDiagnostic::new(diagnostic, &index))
                    .collect(),
            }
        }
    };
    encode(&output)
}

#[derive(Serialize)]
struct WireToken {
    from: u32,
    to: u32,
    kind: &'static str,
}

#[derive(Serialize)]
struct TokensOutput {
    tokens: Vec<WireToken>,
}

/// `tariff.tokens`.
///
/// # Errors
///
/// When the input is not `{"source": "…"}`.
pub fn tokens(input: &RawValue) -> OpResult {
    let input: SourceInput = decode(input)?;
    let file = parse(&input.source);
    let index = Utf16Index::new(&input.source);
    let tokens = file
        .tokens
        .iter()
        .enumerate()
        .filter(|(_, token)| !matches!(token.kind, TokenKind::Newline | TokenKind::Eof))
        .map(|(at, token)| {
            let range = index.range(token.span);
            WireToken {
                from: range.from,
                to: range.to,
                kind: highlight_class(&input.source, &file.tokens, at),
            }
        })
        .collect();
    encode(&TokensOutput { tokens })
}

/// The class the token at `at` is highlighted as: its kind's, except that a name is a `function`
/// when the next token is `(` and a `field` when the previous one is `.` or `?.`, so a method
/// call (`x.in(cwt)`) is a function.
fn highlight_class(source: &str, tokens: &[Token], at: usize) -> &'static str {
    let Some(token) = tokens.get(at) else {
        return "error";
    };
    if token.kind != TokenKind::Name {
        return token.kind.as_str();
    }
    let punct = |token: Option<&Token>, texts: &[&str]| {
        token.is_some_and(|token| {
            token.kind == TokenKind::Punct
                && source
                    .get(token.span.start as usize..token.span.end as usize)
                    .is_some_and(|text| texts.contains(&text))
        })
    };
    if punct(tokens.get(at + 1), &["("]) {
        return "function";
    }
    if punct(
        at.checked_sub(1).and_then(|before| tokens.get(before)),
        &[".", "?."],
    ) {
        return "field";
    }
    token.kind.as_str()
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct SetNameInput {
    source: String,
    name: String,
}

#[derive(Serialize)]
struct WireChange {
    range: Range,
    insert: String,
}

#[derive(Serialize)]
struct SetNameOutput {
    source: String,
    change: WireChange,
}

/// `tariff.setName`.
///
/// # Errors
///
/// When the input is not `{"source": "…", "name": "…"}`.
pub fn set_name(input: &RawValue) -> OpResult {
    let input: SetNameInput = decode(input)?;
    let file = parse(&input.source);
    let (span, insert) = header_edit(file.header.as_ref(), &input.name);
    let start = usize::try_from(span.start).unwrap_or(0);
    let end = usize::try_from(span.end).unwrap_or(0);
    let mut source = String::with_capacity(input.source.len() + insert.len());
    source.push_str(input.source.get(..start).unwrap_or_default());
    source.push_str(&insert);
    source.push_str(input.source.get(end..).unwrap_or_default());
    let range = Utf16Index::new(&input.source).range(span);
    encode(&SetNameOutput {
        source,
        change: WireChange { range, insert },
    })
}

/// The span to replace and the text to put there so the header names `name`: the quoted name
/// when there is one, a space and the name after a keyword without one, or a whole `tariff`
/// header at the top of a file without a header.
fn header_edit(header: Option<&Header>, name: &str) -> (Span, String) {
    let literal = text_literal(name);
    match header {
        Some(header) if header.name.span.start < header.name.span.end => {
            (header.name.span, literal)
        }
        Some(header) => {
            let end = header.span.end;
            (Span::new(end, end), format!(" {literal}"))
        }
        None => (Span::new(0, 0), format!("tariff {literal}\n\n")),
    }
}

/// `text` as a Tariff text literal that decodes back to exactly `text`.
fn text_literal(text: &str) -> String {
    let mut literal = String::with_capacity(text.len() + 2);
    literal.push('"');
    for character in text.chars() {
        match character {
            '"' => literal.push_str("\\\""),
            '\\' => literal.push_str("\\\\"),
            '\n' => literal.push_str("\\n"),
            '\t' => literal.push_str("\\t"),
            '{' => literal.push_str("{{"),
            '}' => literal.push_str("}}"),
            other => literal.push(other),
        }
    }
    literal.push('"');
    literal
}

#[cfg(test)]
mod tests {
    use crate::ops::dispatch;

    fn run(request: &str) -> serde_json::Value {
        serde_json::from_slice(&dispatch(request.as_bytes())).unwrap_or_default()
    }

    #[test]
    fn formats_a_file() {
        let response =
            run(r#"{"op":"tariff.format","input":{"source":"tariff \"t\"\nlet a=1+2\n"}}"#);
        assert_eq!(
            response["ok"]["formatted"],
            serde_json::json!("tariff \"t\"\nlet a = 1 + 2\n")
        );
        assert!(response["ok"].get("diagnostics").is_none());
    }

    #[test]
    fn refuses_a_file_with_errors_with_its_diagnostics() {
        let response =
            run(r#"{"op":"tariff.format","input":{"source":"tariff \"é\"\nlet a = 1 +\n"}}"#);
        let ok = &response["ok"];
        assert_eq!(ok["formatted"], serde_json::Value::Null);
        let diagnostic = &ok["diagnostics"][0];
        assert_eq!(diagnostic["code"], "syntax");
        assert_eq!(diagnostic["severity"], "error");
        assert_eq!(
            diagnostic["range"],
            serde_json::json!({"from": 21, "to": 22})
        );
        assert_eq!(
            diagnostic["message"]["message"],
            "Expected a value after `{0}`"
        );
        assert_eq!(diagnostic["message"]["args"], serde_json::json!(["+"]));
        assert!(diagnostic["fixes"].is_array());
        assert!(diagnostic["related"].is_array());
    }

    #[test]
    fn sends_fixes_in_utf16_ranges() {
        let response = run(r#"{"op":"tariff.format","input":{"source":"let é = 1\n"}}"#);
        let fix = &response["ok"]["diagnostics"][0]["fixes"][0];
        assert_eq!(
            fix["changes"][0]["range"],
            serde_json::json!({"from": 0, "to": 0})
        );
        assert_eq!(fix["changes"][0]["insert"], "tariff \"Untitled\"\n\n");
    }

    #[test]
    fn rejects_unknown_input_fields() {
        let response = run(r#"{"op":"tariff.tokens","input":{"source":"","extra":1}}"#);
        assert_eq!(response["error"]["code"], "invalid_input");
    }

    #[test]
    fn sends_tokens_in_utf16_ranges() {
        let source = "let t = -5 °F\nlet s = \"é 🚚\" // café\nx.in(cwt)\n";
        let request = serde_json::json!({"op": "tariff.tokens", "input": {"source": source}});
        let response = run(&request.to_string());
        let tokens = response["ok"]["tokens"]
            .as_array()
            .cloned()
            .unwrap_or_default();
        let found: Vec<(u64, u64, String)> = tokens
            .iter()
            .map(|t| {
                (
                    t["from"].as_u64().unwrap_or(0),
                    t["to"].as_u64().unwrap_or(0),
                    t["kind"].as_str().unwrap_or("").to_owned(),
                )
            })
            .collect();
        assert!(found.contains(&(11, 13, String::from("unit"))));
        assert!(found.contains(&(22, 28, String::from("text"))));
        assert!(found.contains(&(29, 36, String::from("comment"))));
        assert!(found.contains(&(37, 38, String::from("name"))));
        assert!(found.contains(&(39, 41, String::from("function"))));
        assert!(found.contains(&(42, 45, String::from("unit"))));
        assert!(
            found
                .iter()
                .all(|(_, _, kind)| kind != "newline" && kind != "eof")
        );
    }
    fn set_name(source: &str, name: &str) -> serde_json::Value {
        let request =
            serde_json::json!({"op": "tariff.setName", "input": {"source": source, "name": name}});
        run(&request.to_string())["ok"].clone()
    }

    #[test]
    fn renames_the_header_with_one_change_in_utf16_ranges() {
        let source = "tariff \"🚚 Old name\"\nlet a = 1\n";
        let out = set_name(source, "Expedited by service level");
        assert_eq!(
            out["source"],
            "tariff \"Expedited by service level\"\nlet a = 1\n"
        );
        assert_eq!(
            out["change"],
            serde_json::json!({
                "range": {"from": 7, "to": 20},
                "insert": "\"Expedited by service level\"",
            })
        );
    }

    #[test]
    fn writes_a_name_the_header_reads_back_exactly() {
        for name in [
            "Plain",
            "Say \"hi\"",
            "back\\slash",
            "{braces} and }}",
            "tab\there",
            "line\nbreak",
            "Fuel 🚚 ünïcode",
            "",
        ] {
            let out = set_name("tariff \"x\"\n", name);
            let renamed = out["source"].as_str().unwrap_or_default();
            let file = crate::tariff::syntax::parse(renamed);
            let header = file.header.as_ref().map(|header| header.name.text.as_str());
            assert_eq!(header, Some(name), "{renamed}");
        }
    }

    #[test]
    fn names_a_header_written_without_its_name() {
        let out = set_name("tariff\nlet a = 1\n", "Linehaul");
        assert_eq!(out["source"], "tariff \"Linehaul\"\nlet a = 1\n");
        assert_eq!(
            out["change"],
            serde_json::json!({"range": {"from": 6, "to": 6}, "insert": " \"Linehaul\""})
        );
    }

    #[test]
    fn adds_a_header_to_a_file_without_one() {
        let out = set_name("let a = 1\n", "Linehaul");
        assert_eq!(out["source"], "tariff \"Linehaul\"\n\nlet a = 1\n");
        assert_eq!(
            out["change"],
            serde_json::json!({"range": {"from": 0, "to": 0}, "insert": "tariff \"Linehaul\"\n\n"})
        );
    }

    #[test]
    fn keeps_a_library_header_a_library_header() {
        let out = set_name("library \"org.fees\"\nfn a() -> Int = 1\n", "org.charges");
        assert_eq!(
            out["source"],
            "library \"org.charges\"\nfn a() -> Int = 1\n"
        );
    }

    #[test]
    fn refuses_a_set_name_request_without_a_name() {
        let response = run(r#"{"op":"tariff.setName","input":{"source":"tariff \"x\"\n"}}"#);
        assert_eq!(response["error"]["code"], "invalid_input");
    }

    #[test]
    fn classes_a_called_name_as_a_function_and_a_member_as_a_field() {
        let source = "fn trip(miles: Distance) -> Money = rate * miles\nlet w = shipment.weight\nlet s = stop?.city\nlet c = shipment.distance.in(mi)\nlet b = trip(\n";
        let request = serde_json::json!({"op": "tariff.tokens", "input": {"source": source}});
        let response = run(&request.to_string());
        let names: Vec<(String, String)> = response["ok"]["tokens"]
            .as_array()
            .cloned()
            .unwrap_or_default()
            .iter()
            .map(|token| {
                let from = usize::try_from(token["from"].as_u64().unwrap_or(0)).unwrap_or(0);
                let to = usize::try_from(token["to"].as_u64().unwrap_or(0)).unwrap_or(0);
                (
                    source.get(from..to).unwrap_or("").to_owned(),
                    token["kind"].as_str().unwrap_or("").to_owned(),
                )
            })
            .filter(|(_, kind)| matches!(kind.as_str(), "name" | "function" | "field"))
            .collect();
        let expected: Vec<(String, String)> = [
            ("trip", "function"),
            ("miles", "name"),
            ("rate", "name"),
            ("miles", "name"),
            ("w", "name"),
            ("shipment", "name"),
            ("weight", "field"),
            ("s", "name"),
            ("stop", "name"),
            ("city", "field"),
            ("c", "name"),
            ("shipment", "name"),
            ("distance", "field"),
            ("in", "function"),
            ("b", "name"),
            ("trip", "function"),
        ]
        .iter()
        .map(|(text, kind)| ((*text).to_owned(), (*kind).to_owned()))
        .collect();
        assert_eq!(names, expected);
    }

    #[test]
    fn highlights_a_contextual_word_as_a_keyword_only_where_the_grammar_uses_it() {
        let source = "tariff \"t\"\nfn reach(to: Int) -> Int = to\nround up to cents\n";
        let request = serde_json::json!({"op": "tariff.tokens", "input": {"source": source}});
        let response = run(&request.to_string());
        let kinds: Vec<(String, String)> = response["ok"]["tokens"]
            .as_array()
            .cloned()
            .unwrap_or_default()
            .iter()
            .map(|token| {
                let from = usize::try_from(token["from"].as_u64().unwrap_or(0)).unwrap_or(0);
                let to = usize::try_from(token["to"].as_u64().unwrap_or(0)).unwrap_or(0);
                (
                    source.get(from..to).unwrap_or("").to_owned(),
                    token["kind"].as_str().unwrap_or("").to_owned(),
                )
            })
            .filter(|(text, _)| matches!(text.as_str(), "to" | "cents" | "up"))
            .collect();
        let expected: Vec<(String, String)> = [
            ("to", "name"),
            ("to", "name"),
            ("up", "keyword"),
            ("to", "keyword"),
            ("cents", "keyword"),
        ]
        .iter()
        .map(|(text, kind)| ((*text).to_owned(), (*kind).to_owned()))
        .collect();
        assert_eq!(kinds, expected);
    }
}
