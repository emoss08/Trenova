//! The syntax corpus under `testdata/tariff/syntax/`: one file per area, each a list of cases.
//!
//! A case is a `=== name` line, the source, a `---` line and what the source must produce: for
//! `lexing.txt` the lexer's tokens and diagnostics, for `formatting.txt` its formatted text (or its refusal), and for
//! every other area the tree's [`shape`](crate::tariff::syntax::shape) followed by its
//! diagnostics, one `! code from..to message` line each. Run the tests with `TARIFF_BLESS=1` to
//! rewrite the expected sections from what the code produces, then review the diff.

use crate::tariff::syntax::format::format;
use crate::tariff::syntax::parse;
use crate::tariff::syntax::shape;
use crate::tariff::syntax::token::TokenKind;
use std::fmt::Write;
use std::path::PathBuf;

struct Case {
    name: String,
    source: String,
    expected: String,
}

fn path(area: &str) -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("testdata/tariff/syntax")
        .join(format!("{area}.txt"))
}

fn read(area: &str) -> Vec<Case> {
    let text = std::fs::read_to_string(path(area)).unwrap_or_default();
    let mut cases = Vec::new();
    for chunk in text
        .split("\n=== ")
        .map(|chunk| chunk.trim_start_matches("=== "))
    {
        if chunk.trim().is_empty() {
            continue;
        }
        let (name, body) = chunk.split_once('\n').unwrap_or((chunk, ""));
        let (source, expected) = if body == "---" {
            ("", "")
        } else if let Some(expected) = body.strip_prefix("---\n") {
            ("", expected)
        } else {
            body.split_once("\n---\n")
                .or_else(|| body.strip_suffix("\n---").map(|source| (source, "")))
                .unwrap_or((body, ""))
        };
        let source = if source.is_empty() {
            String::new()
        } else {
            format!("{source}\n")
        };
        cases.push(Case {
            name: name.trim().to_owned(),
            source,
            expected: expected.trim_end().to_owned(),
        });
    }
    cases
}

fn write(area: &str, cases: &[(String, String, String)]) {
    let mut out = String::new();
    for (name, source, actual) in cases {
        let source = source.trim_end_matches('\n');
        if source.is_empty() {
            let _ = write!(out, "=== {name}\n---\n{actual}\n\n");
        } else {
            let _ = write!(out, "=== {name}\n{source}\n---\n{actual}\n\n");
        }
    }
    let _ = std::fs::write(path(area), out.trim_end().to_owned() + "\n");
}

fn diagnostics(source: &str) -> String {
    let file = parse(source);
    let mut out = String::new();
    for diagnostic in &file.diagnostics {
        let _ = writeln!(
            out,
            "! {} {}..{} {}",
            diagnostic.code,
            diagnostic.span.start,
            diagnostic.span.end,
            diagnostic.message.english()
        );
    }
    out
}

fn tree(source: &str) -> String {
    let file = parse(source);
    format!("{}{}", shape::file(&file), diagnostics(source))
}

fn tokens(source: &str) -> String {
    let lexed = crate::tariff::syntax::lexer::lex(source);
    let mut out = String::new();
    for token in &lexed.tokens {
        if matches!(token.kind, TokenKind::Newline | TokenKind::Eof) {
            continue;
        }
        let text = source
            .get(token.span.start as usize..token.span.end as usize)
            .unwrap_or_default();
        let _ = writeln!(out, "{} {text:?}", token.kind.as_str());
    }
    for diagnostic in &lexed.diagnostics {
        let _ = writeln!(
            out,
            "! {} {}..{} {}",
            diagnostic.code,
            diagnostic.span.start,
            diagnostic.span.end,
            diagnostic.message.english()
        );
    }
    out
}

fn formatted(source: &str) -> String {
    match format(source) {
        Ok(text) => text,
        Err(errors) => errors
            .iter()
            .map(|d| format!("! refused: {} {}", d.code, d.message.english()))
            .collect::<Vec<_>>()
            .join("\n"),
    }
}

/// Runs every case of `area` through `run`, comparing with (or, blessing, rewriting) the
/// expected sections. Returns the number of cases.
fn run(area: &str, produce: fn(&str) -> String) -> usize {
    let cases = read(area);
    assert!(!cases.is_empty(), "no cases in {area}.txt");
    let bless = std::env::var_os("TARIFF_BLESS").is_some();
    let mut results = Vec::new();
    let mut failures = Vec::new();
    for case in &cases {
        let actual = produce(&case.source).trim_end().to_owned();
        if actual != case.expected {
            failures.push(format!(
                "--- {area}: {}\nexpected:\n{}\nactual:\n{actual}\n",
                case.name, case.expected
            ));
        }
        results.push((case.name.clone(), case.source.clone(), actual));
    }
    if bless {
        write(area, &results);
        return cases.len();
    }
    assert!(failures.is_empty(), "{}", failures.join("\n"));
    cases.len()
}

#[test]
fn lexing() {
    run("lexing", tokens);
}

#[test]
fn items() {
    run("items", tree);
}

#[test]
fn statements() {
    run("statements", tree);
}

#[test]
fn expressions() {
    run("expressions", tree);
}

#[test]
fn patterns() {
    run("patterns", tree);
}

#[test]
fn types() {
    run("types", tree);
}

#[test]
fn recovery() {
    run("recovery", tree);
}

#[test]
fn formatting() {
    run("formatting", formatted);
}

#[test]
fn formatting_is_idempotent_and_keeps_every_case_the_same_program() {
    for case in read("formatting") {
        let Ok(once) = format(&case.source) else {
            continue;
        };
        let twice = format(&once).unwrap_or_default();
        assert_eq!(once, twice, "{}: formatting is not idempotent", case.name);
        assert_eq!(
            shape::file(&parse(&once)),
            shape::file(&parse(&case.source)),
            "{}: formatting changed the tree",
            case.name
        );
    }
}

#[test]
fn the_corpus_has_at_least_two_hundred_cases() {
    let total: usize = [
        "lexing",
        "items",
        "statements",
        "expressions",
        "patterns",
        "types",
        "recovery",
        "formatting",
    ]
    .iter()
    .map(|area| read(area).len())
    .sum();
    assert!(total >= 200, "the corpus has {total} cases");
}

#[test]
fn the_corpus_reports_every_diagnostic_code() {
    let mut seen = std::collections::BTreeSet::new();
    for area in [
        "lexing",
        "items",
        "statements",
        "expressions",
        "patterns",
        "types",
        "recovery",
    ] {
        for case in read(area) {
            for diagnostic in parse(&case.source).diagnostics {
                seen.insert(diagnostic.code);
            }
        }
    }
    for code in ["syntax", "limit", "misplaced", "header", "doc"] {
        assert!(seen.contains(code), "no corpus case reports `{code}`");
    }
}
