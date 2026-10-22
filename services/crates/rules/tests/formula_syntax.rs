//! The lexer and the parser: the grammar the TypeScript parser and expr-lang read, the tree the
//! checker and the evaluator walk, and the syntax diagnostics the editor shows.

use rust_decimal::Decimal;
use std::str::FromStr;
use trenova_rules::formula::syntax;
use trenova_rules::formula::syntax::MAX_DEPTH;
use trenova_rules::formula::syntax::Program;
use trenova_rules::formula::syntax::ast::BinaryOp;
use trenova_rules::formula::syntax::ast::CompareOp;
use trenova_rules::formula::syntax::ast::Expr;
use trenova_rules::formula::syntax::ast::ExprKind;
use trenova_rules::formula::syntax::ast::Pointer;
use trenova_rules::formula::syntax::ast::StatementKind;
use trenova_rules::formula::syntax::ast::SyntaxCode;
use trenova_rules::formula::syntax::ast::UnaryOp;
use trenova_rules::formula::syntax::lexer;

fn parse(source: &str) -> Program {
    syntax::parse(source, &[])
}

fn messages(program: &Program) -> Vec<String> {
    program
        .diagnostics
        .iter()
        .map(|diagnostic| diagnostic.message.english())
        .collect()
}

fn only(source: &str) -> Expr {
    let program = parse(source);
    assert!(
        program.diagnostics.is_empty(),
        "{source}: {:?}",
        messages(&program)
    );
    program
        .statements
        .last()
        .expect("a statement")
        .value
        .clone()
}

fn number(source: &str) -> (Decimal, bool) {
    match only(source).kind {
        ExprKind::Number(number) => (number.value.expect("a value"), number.whole),
        other => panic!("{source} is not a number: {other:?}"),
    }
}

fn dec(text: &str) -> Decimal {
    Decimal::from_str(text).expect("a decimal")
}

fn string(source: &str) -> String {
    match only(source).kind {
        ExprKind::String(value) => value,
        other => panic!("{source} is not a string: {other:?}"),
    }
}

#[test]
fn numbers_read_separators_hex_octal_binary_and_a_trailing_dot() {
    assert_eq!(number("10_000"), (dec("10000"), true));
    assert_eq!(number("0x1F"), (dec("31"), true));
    assert_eq!(number("0o17"), (dec("15"), true));
    assert_eq!(number("0b101"), (dec("5"), true));
    assert_eq!(number("1."), (dec("1"), false));
    assert_eq!(number("2.5e3"), (dec("2500"), false));
    assert_eq!(number(".5"), (dec("0.5"), false));
}

#[test]
fn numbers_keep_their_exact_decimal_value() {
    assert_eq!(number("0.1").0.to_string(), "0.1");
    assert_eq!(number("1.50").0.to_string(), "1.50");
    assert_eq!(
        number("1234567890.123456789012345678").0.to_string(),
        "1234567890.123456789012345678"
    );
}

#[test]
fn a_number_too_large_for_a_decimal_is_reported() {
    let program = parse("100000000000000000000000000000000 + 1");
    assert_eq!(
        messages(&program),
        [
            "100000000000000000000000000000000 does not fit in a formula number, which holds up to 28 significant digits"
        ]
    );
    let ExprKind::Binary(binary) = &program.statements[0].value.kind else {
        panic!("a sum");
    };
    assert!(matches!(binary.left.kind, ExprKind::Number(number) if number.value.is_none()));
}

#[test]
fn a_range_after_a_whole_number_is_left_alone() {
    let source = "1..3";
    let tokens = lexer::tokenize(source, &[]).tokens;
    let texts: Vec<&str> = tokens
        .iter()
        .map(|token| &source[token.span.start as usize..token.span.end as usize])
        .collect();
    assert_eq!(texts, ["1", "..", "3", ""]);
}

#[test]
fn strings_read_escapes_in_quotes_and_none_in_backticks() {
    assert_eq!(string(r#""a\nb""#), "a\nb");
    assert_eq!(string(r"'it\'s'"), "it's");
    assert_eq!(string(r#""ét\x41""#), "étA");
    assert_eq!(string(r#""\u{1F69A}""#), "🚚");
    assert_eq!(string(r#""\101""#), "A");
    assert_eq!(string(r"`a\n`"), r"a\n");
    assert_eq!(string("`a``b`"), "a`b");
    assert_eq!(string("`two\nlines`"), "two\nlines");
}

#[test]
fn an_escape_the_language_does_not_know_is_reported_where_it_is() {
    let program = parse(r#""a\qb""#);
    assert_eq!(
        messages(&program),
        [r"'\q' is not an escape the formula language knows"]
    );
    assert_eq!(
        (
            program.diagnostics[0].span.start,
            program.diagnostics[0].span.end
        ),
        (2, 4)
    );
}

#[test]
fn a_doubled_backtick_does_not_close_a_raw_string() {
    assert!(parse("len(`a``b`)").diagnostics.is_empty());
    assert!(messages(&parse("len(`a``)")).contains(&String::from("This text is never closed")));
}

#[test]
fn a_quoted_string_ends_unclosed_at_the_line_end() {
    let program = parse("\"abc\n+ 1");
    assert!(messages(&program).contains(&String::from("This text is never closed")));
    assert_eq!(program.diagnostics[0].span.end, 4);
}

#[test]
fn reads_non_ascii_names_named_pointers_and_optional_indexes() {
    assert_eq!(
        only("let tarifa_año = 2; tarifa_año").kind,
        ExprKind::Ident(String::from("tarifa_año"))
    );
    assert!(matches!(
        only("reduce([1, 2], #acc + #index, 0)").kind,
        ExprKind::Call(_)
    ));
    assert!(matches!(
        only("stops?.[0]").kind,
        ExprKind::Index(index) if index.optional
    ));
    assert_eq!(
        only("料金 * 2").children()[0].kind,
        ExprKind::Ident(String::from("料金"))
    );
}

#[test]
fn identifiers_take_letters_digits_and_underscores_of_any_script() {
    assert!(lexer::is_identifier_start('é'));
    assert!(lexer::is_identifier_start('料'));
    assert!(lexer::is_identifier_start('_'));
    assert!(!lexer::is_identifier_start('٣'));
    assert!(lexer::is_identifier_continue('٣'));
    assert!(!lexer::is_identifier_start('Ⅻ'));
    assert!(!lexer::is_identifier_continue('\u{0301}'));
}

#[test]
fn a_let_is_scoped_inside_parentheses_an_if_block_and_a_predicate() {
    let grouped = only("(let a = 2; a * 3)");
    assert_eq!(grouped.parens, 1);
    let ExprKind::Sequence(sequence) = &grouped.kind else {
        panic!("a sequence");
    };
    assert_eq!(sequence.lets[0].name.text, "a");

    let ExprKind::If(branch) = only("if isRush { let a = 2; a } else { 0 }").kind else {
        panic!("an if");
    };
    assert!(matches!(branch.then.kind, ExprKind::Sequence(_)));

    let ExprKind::Call(call) = only("filter([1, 2, 3], { let x = #; x > 1 })").kind else {
        panic!("a call");
    };
    let ExprKind::Predicate(predicate) = &call.args[1].kind else {
        panic!("a predicate");
    };
    assert!(predicate.braces);
    assert!(matches!(predicate.body.kind, ExprKind::Sequence(_)));
}

#[test]
fn a_trailing_comma_is_taken_in_calls_lists_and_maps() {
    let ExprKind::Call(call) = only("max(1, 2,)").kind else {
        panic!("a call");
    };
    assert_eq!(call.args.len(), 2);
    assert!(matches!(only("[1, 2,]").kind, ExprKind::Array(items) if items.len() == 2));
    assert!(
        matches!(only("{a: 1, \"b c\": 2,}").kind, ExprKind::Map(entries) if entries.len() == 2)
    );
}

#[test]
fn map_keys_are_names_strings_or_numbers() {
    let ExprKind::Map(entries) = only("{a: 1, \"b c\": 2, 3: 4}").kind else {
        panic!("a map");
    };
    let keys: Vec<&str> = entries
        .iter()
        .map(|entry| entry.key.text.as_str())
        .collect();
    assert_eq!(keys, ["a", "b c", "3"]);
}

#[test]
fn a_bare_predicate_argument_becomes_a_predicate() {
    let ExprKind::Call(call) = only("count(stops, .weight > 0)").kind else {
        panic!("a call");
    };
    let ExprKind::Predicate(predicate) = &call.args[1].kind else {
        panic!("a predicate");
    };
    assert!(!predicate.braces);
    assert_eq!(call.args[1].span, predicate.body.span);
    let ExprKind::Call(sum) = only("sum(stops, .weight)").kind else {
        panic!("a call");
    };
    assert!(
        matches!(sum.args[1].kind, ExprKind::Member(_)),
        "sum takes no predicate, so its argument stays as written"
    );
}

#[test]
fn the_item_a_leading_dot_reads_is_an_implicit_pointer() {
    let ExprKind::Member(member) = only(".weight").kind else {
        panic!("a member");
    };
    assert_eq!(member.property.text, "weight");
    assert_eq!(member.object.kind, ExprKind::Pointer(Pointer::Implicit));
    assert_eq!(member.object.span.start, member.object.span.end);
}

#[test]
fn chained_comparisons_are_one_chain_and_parentheses_stop_one() {
    let ExprKind::Chain(chain) = only("0 < x <= 5000").kind else {
        panic!("a chain");
    };
    assert_eq!(chain.operands.len(), 3);
    let ops: Vec<CompareOp> = chain.operators.iter().map(|(op, _)| *op).collect();
    assert_eq!(ops, [CompareOp::Less, CompareOp::LessEqual]);
    let ExprKind::Chain(longer) = only("0 < a < 2000 <= 3000").kind else {
        panic!("a chain");
    };
    assert_eq!(longer.operands.len(), 4);
    let ExprKind::Binary(grouped) = only("(1 < 2) < 3").kind else {
        panic!("a comparison");
    };
    assert_eq!(grouped.op, BinaryOp::Compare(CompareOp::Less));
    assert_eq!(grouped.left.parens, 1);
    assert!(
        matches!(only("a == b < c").kind, ExprKind::Binary(binary) if binary.op == BinaryOp::Compare(CompareOp::Less))
    );
}

#[test]
fn operators_keep_the_authors_spelling() {
    let ExprKind::Binary(and) = only("a and b").kind else {
        panic!("and");
    };
    assert_eq!(and.op, BinaryOp::And { word: true });
    let ExprKind::Binary(symbol) = only("a && b").kind else {
        panic!("&&");
    };
    assert_eq!(symbol.op.text(), "&&");
    let ExprKind::Binary(caret) = only("2 ^ 3").kind else {
        panic!("^");
    };
    assert_eq!(caret.op, BinaryOp::Power { caret: true });
    let ExprKind::Unary(bang) = only("!a").kind else {
        panic!("!");
    };
    assert_eq!(bang.op, UnaryOp::Not { word: false });
}

#[test]
fn precedence_follows_the_operator_table() {
    let ExprKind::Binary(sum) = only("1 + 2 * 3").kind else {
        panic!("a sum");
    };
    assert_eq!(sum.op, BinaryOp::Add);
    let ExprKind::Binary(power) = only("2 ** 3 ** 2").kind else {
        panic!("a power");
    };
    assert!(
        matches!(power.right.kind, ExprKind::Binary(_)),
        "** is right-associative"
    );
    let ExprKind::Unary(negate) = only("-2 ** 2").kind else {
        panic!("a negation");
    };
    assert!(matches!(negate.operand.kind, ExprKind::Binary(_)));
    let ExprKind::Binary(or) = only("a or b and c").kind else {
        panic!("an or");
    };
    assert_eq!(or.op, BinaryOp::Or { word: true });
    let ExprKind::Binary(range) = only("1..2 + 3").kind else {
        panic!("a range");
    };
    assert_eq!(range.op, BinaryOp::Range);
}

#[test]
fn not_before_a_word_operator_negates_it() {
    let ExprKind::Binary(binary) = only("code not in [\"A\", \"B\"]").kind else {
        panic!("an in");
    };
    assert_eq!(binary.op, BinaryOp::In);
    assert!(binary.not.is_some());
    let ExprKind::Binary(contains) = only("name not contains \"x\"").kind else {
        panic!("a contains");
    };
    assert!(contains.not.is_some());
}

#[test]
fn schema_word_operators_parse_at_comparison_precedence() {
    let program = syntax::parse("code soundsLike \"abc\" ? 1 : 0", &["soundsLike"]);
    assert!(program.diagnostics.is_empty());
    let ExprKind::Ternary(ternary) = &program.statements[0].value.kind else {
        panic!("a ternary");
    };
    assert!(
        matches!(&ternary.test.kind, ExprKind::Binary(binary) if binary.op == BinaryOp::Word(Box::from("soundsLike")))
    );
}

#[test]
fn ternaries_elvis_and_if_are_told_apart() {
    assert!(matches!(only("a ? 1 : 2").kind, ExprKind::Ternary(ternary) if ternary.then.is_some()));
    assert!(matches!(only("a ?: 2").kind, ExprKind::Ternary(ternary) if ternary.then.is_none()));
    let ExprKind::If(branch) = only("if a { 1 } else if b { 2 } else { 3 }").kind else {
        panic!("an if");
    };
    assert!(matches!(branch.otherwise.kind, ExprKind::If(_)));
    let ExprKind::Ternary(nested) = only("a ? 1 : b ? 2 : 3").kind else {
        panic!("a ternary");
    };
    assert!(matches!(nested.otherwise.kind, ExprKind::Ternary(_)));
}

#[test]
fn an_if_spans_through_its_last_brace() {
    let source = "if a { 1 } else { 2 }";
    assert_eq!(only(source).span.end as usize, source.len());
}

#[test]
fn methods_calls_pipes_slices_and_optional_members() {
    let ExprKind::Method(method) = only("pickupDate.Year()").kind else {
        panic!("a method");
    };
    assert_eq!(method.method.text, "Year");
    assert!(matches!(method.receiver.kind, ExprKind::Ident(_)));
    let ExprKind::Call(pipe) = only("totalWeight | round(2)").kind else {
        panic!("a pipe");
    };
    assert!(pipe.pipe);
    assert_eq!(pipe.function.text, "round");
    assert_eq!(pipe.args.len(), 2);
    let ExprKind::Slice(slice) = only("origin.zip[0:3]").kind else {
        panic!("a slice");
    };
    assert!(slice.start.is_some() && slice.end.is_some());
    assert!(matches!(only("a[:2]").kind, ExprKind::Slice(slice) if slice.start.is_none()));
    assert!(matches!(only("a[1:]").kind, ExprKind::Slice(slice) if slice.end.is_none()));
    assert!(matches!(only("shipmentType?.code").kind, ExprKind::Member(member) if member.optional));
    assert!(matches!(only("now().Year()").kind, ExprKind::Method(_)));
}

#[test]
fn the_pipe_without_parentheses_offers_them() {
    let program = parse("totalWeight | round");
    let fix = program.diagnostics[0].fix.as_ref().expect("a fix");
    assert_eq!(fix.insert, "round()");
}

#[test]
fn let_lines_come_before_the_result() {
    let program = parse("let a = 1;\nlet b = a * 2;\na + b");
    assert!(program.diagnostics.is_empty());
    let kinds: Vec<bool> = program
        .statements
        .iter()
        .map(|statement| matches!(statement.kind, StatementKind::Let(_)))
        .collect();
    assert_eq!(kinds, [true, true, false]);
    let StatementKind::Let(name) = &program.statements[1].kind else {
        panic!("a let");
    };
    assert_eq!(name.text, "b");
    assert_eq!(program.statements[1].span.start, 11);
    assert_eq!(
        program.statements[0].semicolon.map(|span| span.start),
        Some(9)
    );
}

#[test]
fn an_unclosed_paren_is_anchored_on_the_paren() {
    let program = parse("round(baseRate * 2");
    assert_eq!(
        (
            program.diagnostics[0].span.start,
            program.diagnostics[0].span.end
        ),
        (5, 6)
    );
    assert!(
        program.diagnostics[0]
            .message
            .english()
            .contains("never closed")
    );
}

#[test]
fn a_let_without_its_semicolon_offers_one() {
    let program = parse("let a = 1\na + 1");
    assert!(
        program
            .diagnostics
            .iter()
            .any(|diagnostic| diagnostic.fix.as_ref().is_some_and(|fix| fix.insert == ";"))
    );
}

#[test]
fn a_stray_equals_offers_double_equals() {
    let program = parse("totalStops = 2 ? 1 : 0");
    assert!(program.diagnostics.iter().any(|diagnostic| {
        diagnostic
            .fix
            .as_ref()
            .is_some_and(|fix| fix.insert == "==")
    }));
}

#[test]
fn word_operators_not_and_the_ternary_parse_together() {
    let program = parse("customer.name startsWith \"A\" and not (hasHazmat) ? 1 : 0");
    assert!(program.diagnostics.is_empty());
    assert!(matches!(
        program.statements[0].value.kind,
        ExprKind::Ternary(_)
    ));
}

#[test]
fn coalesce_mixed_with_an_operator_asks_for_parentheses() {
    let program = parse("weight ?? 0 * 2");
    assert!(
        program.diagnostics[0]
            .message
            .english()
            .contains("parentheses")
    );
}

#[test]
fn syntax_messages_say_what_is_wrong() {
    let cases: &[(&str, &str)] = &[
        ("1 +", "'+' needs a value on its right"),
        ("-", "'-' needs a value after it"),
        ("* 2", "'*' needs a value before it"),
        ("a.", "'.' needs a field name after it"),
        ("(1 + 2", "'(' opened here is never closed"),
        ("[1, 2", "'[' opened here is never closed"),
        ("a[1", "'[' opened here is never closed"),
        ("{a: 1", "'{' opened here is never closed"),
        (") 1", "')' has no matching '('"),
        ("1)", "Unexpected ')'"),
        ("()", "'()' is empty"),
        (
            "a ? 1",
            "'?' needs ':' and the value for the other case: test ? a : b",
        ),
        ("a ? : 1 : 2", "Unexpected ':'"),
        ("if a { 1 }", "'if' needs an 'else' with the other value"),
        ("if a 1 else 2", "'if' needs its value in braces after it"),
        (
            "if a {} else { 1 }",
            "The braces after 'if' are empty; put its value inside them",
        ),
        (
            "2 + if a { 1 } else { 2 }",
            "An 'if' that is part of a calculation needs parentheses around the whole 'if … else …'",
        ),
        (
            "1 + let",
            "'let' lines come before the formula's amount, each ending with ';'",
        ),
        ("1 + else", "Unexpected 'else' here"),
        ("let = 1; 2", "'let' needs a name: let fuel = …"),
        ("let a 1; a", "'let a' needs '=' and a value"),
        ("let a = ; a", "'let a' needs a value after '='"),
        ("let a == 1; a", "'let' assigns with one '='"),
        ("let if = 1; 1", "'if' is a keyword and cannot be a name"),
        (
            "let a = 1",
            "A 'let' line ends with ';', then the formula's amount follows on the next line",
        ),
        ("(let a = 1 a)", "A 'let' line ends with ';'"),
        (
            "(let a = 1;)",
            "A 'let' here needs the value it is for after it",
        ),
        ("1 2", "Unexpected '2'"),
        (";", "Unexpected ';'"),
        ("x | 1", "'|' pipes into a function: value | round(2)"),
        ("round(1,", "Remove the ',' or add a value after it"),
        ("round(1 2)", "Unexpected '2' inside the call"),
        ("{a: 1, b}", "A map entry is written key: value"),
        ("a @ b", "Unexpected character '@'"),
        ("/* open", "This comment is never closed"),
        ("(1)(2)", "Unexpected '('"),
    ];
    for (source, message) in cases {
        let found = messages(&parse(source));
        assert!(
            found.iter().any(|text| text == message),
            "{source}: wanted {message:?}, got {found:?}"
        );
    }
}

#[test]
fn every_program_with_a_missing_part_has_a_diagnostic() {
    for source in [
        "1 +",
        "if a {} else {1}",
        "(let a = ; a)",
        "a[",
        "round(",
        "{a: }",
        "[,]",
        "a ?",
        "f(,)",
        "let a = 1;",
        "!",
        "a ?? ",
        "x |",
        "#[",
    ] {
        let program = parse(source);
        let mut missing = false;
        program.walk(&mut |expr| missing |= expr.is_missing());
        assert!(
            !missing || !program.diagnostics.is_empty(),
            "{source} has a missing node and no diagnostic"
        );
    }
}

#[test]
fn an_empty_formula_is_empty() {
    assert!(parse("   ").empty);
    assert!(parse("// only a comment").empty);
    assert!(parse("").statements.is_empty());
    let open = parse("/* open");
    assert!(open.empty);
    assert_eq!(messages(&open), ["This comment is never closed"]);
}

#[test]
fn comments_are_kept_out_of_the_tokens() {
    let program = parse("1 // one\n+ /* two */ 2");
    assert!(program.diagnostics.is_empty());
    assert_eq!(program.comments.len(), 2);
    assert_eq!(program.tokens.len(), 4);
}

#[test]
fn nesting_past_the_limit_stops_with_one_diagnostic() {
    let deep = format!("{}1{}", "(".repeat(5_000), ")".repeat(5_000));
    let program = parse(&deep);
    assert_eq!(program.diagnostics.len(), 1);
    assert_eq!(program.diagnostics[0].code, SyntaxCode::Limit);
    assert_eq!(
        program.diagnostics[0].message.english(),
        format!("This formula nests more than {MAX_DEPTH} levels deep; split it into 'let' lines")
    );
    let sum = (0..300)
        .map(|i| format!("a{i}"))
        .collect::<Vec<_>>()
        .join(" + ");
    let program = parse(&sum);
    assert_eq!(program.diagnostics[0].code, SyntaxCode::Limit);
    let mut deepest = 0;
    program.walk(&mut |expr| deepest = deepest.max(expr.depth()));
    assert!(usize::from(deepest) <= MAX_DEPTH);
}

#[test]
fn stray_closers_do_not_recurse() {
    let closers = ")".repeat(20_000);
    let program = parse(&format!("1 + {closers}"));
    assert!(!program.diagnostics.is_empty());
}

#[test]
fn too_many_parts_stop_with_one_diagnostic() {
    let list = format!("[{}]", vec!["1"; 1_200].join(", "));
    let program = parse(&list);
    let limits: Vec<String> = program
        .diagnostics
        .iter()
        .filter(|diagnostic| diagnostic.code == SyntaxCode::Limit)
        .map(|diagnostic| diagnostic.message.english())
        .collect();
    assert_eq!(
        limits,
        ["This formula is too long: a formula has at most 1000 parts"]
    );
}

#[test]
fn a_formula_past_the_length_limit_is_not_read() {
    let long = "1 + ".repeat(10_000);
    let program = parse(&long);
    assert!(program.statements.is_empty());
    assert_eq!(program.diagnostics.len(), 1);
    assert_eq!(program.diagnostics[0].span.start, 32_768);
    assert_eq!(
        program.diagnostics[0].message.english(),
        "This formula is too long: a formula holds at most 32768 characters"
    );
}

#[test]
fn every_conformance_formula_parses() {
    let corpus: serde_json::Value =
        serde_json::from_str(include_str!("../testdata/conformance.json"))
            .expect("the corpus is JSON");
    for case in corpus["cases"].as_array().expect("cases") {
        let source = case["expression"].as_str().expect("an expression");
        let program = parse(source);
        let syntax_refused = matches!(
            case["id"].as_str(),
            Some("if-as-an-operand" | "pipe-without-parentheses")
        );
        if case["accepted"].as_bool() == Some(true) {
            assert!(
                program.diagnostics.is_empty(),
                "{source}: {:?}",
                messages(&program)
            );
        } else if syntax_refused {
            assert!(!program.diagnostics.is_empty(), "{source}");
        }
    }
}
