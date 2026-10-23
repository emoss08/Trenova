//! Properties of the front end: tokens cover the source, limits hold, ids are numbered in
//! source order, every well-formed file formats to itself, messages are valid templates, and
//! nothing panics on any input.

use super::reference_blocks;
use crate::span::Utf16Index;
use crate::tariff::diagnostic::Severity;
use crate::tariff::syntax::ast::File;
use crate::tariff::syntax::format::format;
use crate::tariff::syntax::lexer::MAX_SOURCE_BYTES;
use crate::tariff::syntax::lexer::MAX_TOKENS;
use crate::tariff::syntax::outline::FoldKind;
use crate::tariff::syntax::outline::OutlineKind;
use crate::tariff::syntax::outline::outline;
use crate::tariff::syntax::parse;
use crate::tariff::syntax::shape;
use crate::tariff::syntax::token::TokenKind;
use crate::tariff::syntax::walk;
use proptest::prelude::*;

/// Every corpus source and every reference example.
fn samples() -> Vec<String> {
    let mut sources = reference_blocks();
    for area in [
        "lexing",
        "items",
        "statements",
        "expressions",
        "patterns",
        "types",
        "recovery",
        "formatting",
    ] {
        let path = std::path::PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("testdata/tariff/syntax")
            .join(format!("{area}.txt"));
        let text = std::fs::read_to_string(path).unwrap_or_default();
        for chunk in text.split("\n=== ") {
            if let Some((_, body)) = chunk.split_once('\n') {
                let source = body.split("\n---\n").next().unwrap_or_default();
                sources.push(format!("{source}\n"));
            }
        }
    }
    sources
}

/// Tokens are in order, do not overlap, end with the end of the file, and every byte outside
/// them is whitespace.
fn assert_covered(source: &str, file: &File) {
    let mut at = 0usize;
    for token in &file.tokens {
        let start = token.span.start as usize;
        let end = token.span.end as usize;
        assert!(
            start >= at && end >= start,
            "{source:?}: tokens out of order at {start}"
        );
        let gap = source.get(at..start).unwrap_or_default();
        assert!(
            gap.chars().all(char::is_whitespace) && !gap.contains('\n'),
            "{source:?}: bytes {at}..{start} ({gap:?}) are in no token"
        );
        at = end;
    }
    assert_eq!(
        at,
        source.len(),
        "{source:?}: the source does not end at the last token"
    );
    assert_eq!(file.tokens.last().map(|t| t.kind), Some(TokenKind::Eof));
}

fn assert_spans_inside(source: &str, file: &File) {
    let length = u32::try_from(source.len()).unwrap_or(u32::MAX);
    for diagnostic in &file.diagnostics {
        assert!(diagnostic.span.start <= diagnostic.span.end);
        assert!(diagnostic.span.end <= length, "{source:?}: {diagnostic:?}");
        for fix in &diagnostic.fixes {
            for change in &fix.changes {
                assert!(change.span.end <= length);
            }
        }
    }
}

fn assert_ids_in_source_order(file: &File) {
    let mut seen = Vec::new();
    walk::exprs(file, &mut |expr| seen.push((expr.id.0, expr.span.start)));
    for (index, (id, _)) in seen.iter().enumerate() {
        assert_eq!(*id as usize, index, "ids are not numbered in walk order");
    }
    assert_eq!(file.node_count as usize, seen.len());
    assert!(
        seen.windows(2).all(|pair| match pair {
            [(_, a), (_, b)] => a <= b,
            _ => true,
        }),
        "ids are not in source order"
    );
}

#[test]
fn every_sample_is_covered_by_its_tokens_with_ids_in_order() {
    for source in samples() {
        let file = parse(&source);
        assert_covered(&source, &file);
        assert_spans_inside(&source, &file);
        assert_ids_in_source_order(&file);
    }
}

#[test]
fn every_well_formed_sample_formats_to_itself_keeping_its_comments() {
    let mut formatted = 0;
    for source in samples() {
        let file = parse(&source);
        if file
            .diagnostics
            .iter()
            .any(|d| d.severity == Severity::Error)
        {
            assert!(
                format(&source).is_err(),
                "{source:?} has errors but was formatted"
            );
            continue;
        }
        let once = format(&source).unwrap_or_else(|errors| {
            panic!("{source:?} could not be formatted: {errors:?}");
        });
        assert_eq!(format(&once).as_deref(), Ok(once.as_str()), "{source:?}");
        assert_eq!(shape::file(&parse(&once)), shape::file(&file), "{source:?}");
        let comments = |file: &File| {
            file.tokens
                .iter()
                .filter(|t| matches!(t.kind, TokenKind::Comment | TokenKind::DocComment))
                .count()
        };
        assert_eq!(comments(&parse(&once)), comments(&file), "{source:?}");
        formatted += 1;
    }
    assert!(formatted > 150, "only {formatted} samples were formatted");
}

#[test]
fn a_source_over_the_size_limit_is_one_error_token() {
    let source = "x".repeat(MAX_SOURCE_BYTES + 1);
    let file = parse(&source);
    assert_eq!(file.diagnostics.len(), 1);
    assert_eq!(file.diagnostics[0].code, "limit");
    assert_eq!(file.tokens.len(), 2);
    assert_eq!(file.tokens[0].kind, TokenKind::Error);
    assert!(file.items.is_empty());
    assert_covered(&source, &file);
}

#[test]
fn a_source_over_the_token_limit_stops_with_a_limit_diagnostic() {
    let source = format!("tariff \"t\"\nlet a = [{}]\n", "1,".repeat(MAX_TOKENS));
    let file = parse(&source);
    assert!(file.diagnostics.iter().any(|d| d.code == "limit"));
    assert!(file.tokens.len() <= MAX_TOKENS + 2);
    assert_covered(&source, &file);
}

#[test]
fn utf16_ranges_count_non_ascii_text() {
    let source = "tariff \"é\"\nlet t = -5 °F\nlet s = \"🚚 {x}\"\n";
    let file = parse(source);
    let index = Utf16Index::new(source);
    let range = |kind: TokenKind, nth: usize| {
        file.tokens
            .iter()
            .filter(|t| t.kind == kind)
            .nth(nth)
            .map(|t| index.range(t.span))
            .map(|r| (r.from, r.to))
    };
    assert_eq!(range(TokenKind::Text, 0), Some((7, 10)));
    assert_eq!(range(TokenKind::Unit, 0), Some((22, 24)));
    assert_eq!(range(TokenKind::Text, 1), Some((33, 37)));
    assert_eq!(range(TokenKind::Name, 2), Some((38, 39)));
}

#[test]
fn the_outline_lists_declarations_charges_totals_and_tests() {
    let source = "tariff \"Dry van\"\n\n/// The rate.\n/// Per mile.\ninput rate: Money/mi = $2.35/mi\nconst LIMIT = 3\n\nfn deadhead(\n  empty: Distance,\n) -> Money = $1\n\nfor stop in shipment.stops {\n  if stop.appointment {\n    charge \"Detention, stop {stop.sequence}\" as DET = $1\n  }\n}\ncharge \"Linehaul\" = [\n  $1,\n].len * $1\nminimum $450\n\ntest \"x\" {\n  expect true\n}\n";
    let file = parse(source);
    assert!(file.diagnostics.is_empty(), "{:?}", file.diagnostics);
    let found = outline(source, &file);
    let entries: Vec<(OutlineKind, &str, Option<&str>, u32)> = found
        .items
        .iter()
        .map(|item| {
            (
                item.kind,
                item.name.as_str(),
                item.code.as_deref(),
                item.depth,
            )
        })
        .collect();
    assert_eq!(
        entries,
        vec![
            (OutlineKind::Header, "Dry van", None, 0),
            (OutlineKind::Input, "rate", None, 0),
            (OutlineKind::Const, "LIMIT", None, 0),
            (OutlineKind::Function, "deadhead", None, 0),
            (
                OutlineKind::Charge,
                "Detention, stop {stop.sequence}",
                Some("DET"),
                2
            ),
            (OutlineKind::Charge, "Linehaul", None, 0),
            (OutlineKind::Total, "minimum", None, 0),
            (OutlineKind::Test, "x", None, 0),
        ]
    );
    let rate = &found.items[1];
    assert_eq!(
        &source[rate.selection.start as usize..rate.selection.end as usize],
        "rate"
    );
    let kinds: Vec<FoldKind> = found.folding.iter().map(|(_, kind)| *kind).collect();
    assert!(kinds.contains(&FoldKind::Doc));
    assert!(kinds.contains(&FoldKind::List));
    assert_eq!(kinds.iter().filter(|k| **k == FoldKind::Block).count(), 3);
    for (span, _) in &found.folding {
        assert!(source[span.start as usize..span.end as usize].contains('\n'));
    }
}

/// Every `message!` template in the front end is a valid ICU template: braces are placeholders
/// (`{0}`, `{0, plural, …}`) or quoted (`'{'`).
#[test]
fn every_message_template_is_valid() {
    let sources = [
        include_str!("../lexer.rs"),
        include_str!("../parser/mod.rs"),
        include_str!("../parser/docs.rs"),
        include_str!("../parser/items.rs"),
        include_str!("../parser/statements.rs"),
        include_str!("../parser/expressions.rs"),
        include_str!("../parser/patterns.rs"),
        include_str!("../parser/types.rs"),
        include_str!("../program.rs"),
        include_str!("../format/mod.rs"),
    ];
    let mut checked = 0;
    for source in sources {
        let mut rest = source;
        while let Some(start) = rest.find("message!(") {
            rest = &rest[start + "message!(".len()..];
            let Some(open) = rest.find('"') else { break };
            let body = &rest[open + 1..];
            let mut end = 0;
            let bytes = body.as_bytes();
            while end < bytes.len()
                && !(bytes[end] == b'"' && bytes[end.saturating_sub(1)] != b'\\')
            {
                end += 1;
            }
            let template = &body[..end];
            let mut unquoted = String::new();
            let mut quoted = false;
            let characters: Vec<char> = template.chars().collect();
            for (index, character) in characters.iter().enumerate() {
                if *character == '\''
                    && characters
                        .get(index + 1)
                        .is_some_and(|c| *c == '{' || *c == '}')
                {
                    quoted = true;
                    continue;
                }
                if *character == '\'' && quoted {
                    quoted = false;
                    continue;
                }
                if !quoted {
                    unquoted.push(*character);
                }
            }
            let mut tail = unquoted.as_str();
            while let Some(brace) = tail.find('{') {
                let after = &tail[brace + 1..];
                let digits = after.chars().take_while(char::is_ascii_digit).count();
                assert!(digits > 0, "template {template:?} has a bare brace");
                tail = &after[digits..];
            }
            checked += 1;
            rest = &body[end..];
        }
    }
    assert!(checked > 100, "only {checked} templates were found");
}

/// A small well-formed program built from parts, for the formatting properties.
fn program() -> impl Strategy<Value = String> {
    let names = prop::sample::select(vec!["miles", "rate", "fee", "w", "count", "total"]);
    let values = prop::sample::select(vec![
        "1",
        "2.50",
        "500 lb",
        "$450",
        "$2.35/mi",
        "18%",
        "\"text {x}\"",
        "@2026-10-22",
        "true",
        "none",
        "[1, 2, 3]",
        "{ a: 1, b: $2 }",
        "shipment.weight ?? 0 lb",
        "f(1, to: $5)",
        "xs.count(s => s.kind == Delivery)",
        "if a { 1 } else { 2 }",
        "a and not b or c",
        "1..=3",
        "-x * (y + 2)",
        "tables.Zones[a, b]",
    ]);
    let statement =
        (names, values.clone(), values, 0..9usize).prop_map(|(name, a, b, shape)| match shape {
            0 => format!("let {name} = {a}"),
            1 => format!("let {name} = ({a}) + ({b})  // note"),
            2 => format!("if {a} {{\n  charge \"Line {{{name}}}\" as FSC = {b}\n}}"),
            3 => format!("for {name} in {a} where {b} {{\n\n  note \"x\"\n}}"),
            4 => format!("let {name} = match {a} {{\n  ..5 => {b}\n  _ => {a}\n}}"),
            5 => format!("require let {name} = {a} else \"no\"\n\n\n"),
            6 => format!("charge \"{name}\" (quantity: {a}, rate: {b})"),
            7 => format!("// {name}\nlet {name} = [{a}, {b}, {a}, {b}, {a}, {b}, {a}, {b}, {a}]"),
            _ => format!("test \"{name}\" {{\n  given shipment.weight = {a}\n  expect {b}\n}}"),
        });
    prop::collection::vec(statement, 0..8)
        .prop_map(|statements| format!("tariff \"p\"\n\n{}\n", statements.join("\n")))
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(256))]

    #[test]
    fn lexing_and_parsing_never_panic_on_any_text(source in "\\PC*") {
        let file = parse(&source);
        assert_covered(&source, &file);
        assert_spans_inside(&source, &file);
        let _ = format(&source);
        let _ = outline(&source, &file);
    }

    #[test]
    fn lexing_and_parsing_never_panic_on_tariff_like_text(
        source in "[a-zA-Z0-9_ $@%°é🚚\"{}()\\[\\].,:;=<>+*/?!|\\-\\\\\n]{0,200}"
    ) {
        let headed = format!("tariff \"t\"\n{source}");
        let file = parse(&headed);
        assert_covered(&headed, &file);
        assert_spans_inside(&headed, &file);
        assert_ids_in_source_order(&file);
        if let Ok(once) = format(&headed) {
            prop_assert_eq!(format(&once), Ok(once.clone()));
        }
    }

    #[test]
    fn formatting_is_idempotent_and_keeps_the_tree(source in program()) {
        let file = parse(&source);
        prop_assert!(file.diagnostics.is_empty(), "{:?}\n{}", file.diagnostics, source);
        let once = format(&source);
        prop_assert!(once.is_ok(), "{:?}\n{}", once, source);
        let once = once.unwrap_or_default();
        prop_assert_eq!(format(&once), Ok(once.clone()));
        prop_assert_eq!(shape::file(&parse(&once)), shape::file(&file));
    }
}

/// Typing a file produces every prefix of it. Each parses, and its diagnostics are all after
/// the last item it completed: the text already written is never blamed for what is missing.
#[test]
fn every_prefix_of_a_reference_example_is_blamed_only_where_it_is_incomplete() {
    for block in reference_blocks() {
        let full = parse(&block);
        for (cut, _) in block.char_indices().skip(1) {
            let prefix = &block[..cut];
            let file = parse(prefix);
            assert_covered(prefix, &file);
            let complete = full
                .items
                .iter()
                .filter(|item| (item.span.end as usize) < cut)
                .map(|item| item.span.end)
                .max()
                .unwrap_or(0);
            for diagnostic in &file.diagnostics {
                assert!(
                    diagnostic.span.start >= complete,
                    "{prefix:?}: {:?} blames text before {complete}",
                    diagnostic.message.english()
                );
            }
        }
    }
}
