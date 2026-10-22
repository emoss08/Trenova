//! The `tariff.*` operations this area answers: `tariff.format` and `tariff.tokens`.
//!
//! `tariff.format`: `{"source": "…"}` gives `{"formatted": "…"}`, or, for a file with errors,
//! `{"formatted": null, "diagnostics": [...]}` in the wire shape every operation uses
//! ([`WireDiagnostic`]).
//!
//! `tariff.tokens`: `{"source": "…"}` gives `{"tokens": [{"from": 0, "to": 6, "kind":
//! "keyword"}, …]}`: every token but newlines and the end of the file, in UTF-16 code units, each
//! classed for highlighting (`keyword`, `name`, `type`, `code`, `number`, `unit`, `money`,
//! `percent`, `date`, `text`, `escape`, `interpolation`, `punct`, `comment`, `doc`, `error`).

use super::format::format as format_source;
use super::parse;
use super::token::TokenKind;
use crate::ops::OpResult;
use crate::ops::decode;
use crate::ops::encode;
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
        .filter(|token| !matches!(token.kind, TokenKind::Newline | TokenKind::Eof))
        .map(|token| {
            let range = index.range(token.span);
            WireToken {
                from: range.from,
                to: range.to,
                kind: token.kind.as_str(),
            }
        })
        .collect();
    encode(&TokensOutput { tokens })
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
        assert!(found.contains(&(39, 41, String::from("name"))));
        assert!(found.contains(&(42, 45, String::from("unit"))));
        assert!(
            found
                .iter()
                .all(|(_, _, kind)| kind != "newline" && kind != "eof")
        );
    }
}
