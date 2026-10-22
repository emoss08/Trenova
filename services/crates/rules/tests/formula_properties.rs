//! Properties every input holds: no text makes the language service panic or overflow its stack,
//! and formatting a formula that parses is idempotent and keeps its tree.

use proptest::prelude::*;
use serde_json::json;
use std::fmt::Write;
use trenova_rules::formula::check;
use trenova_rules::formula::check::Expect;
use trenova_rules::formula::check::Schema;
use trenova_rules::formula::syntax;
use trenova_rules::formula::syntax::FormatOutcome;
use trenova_rules::formula::syntax::Program;
use trenova_rules::formula::syntax::ast::Expr;
use trenova_rules::formula::syntax::ast::ExprKind;
use trenova_rules::formula::syntax::ast::StatementKind;
use trenova_rules::ops::dispatch;

const DESCRIBE: &str = include_str!(
    "../../../tms/internal/modules/rating/app/formula/testdata/shipment.describe.json"
);

fn schema() -> Schema {
    serde_json::from_str(DESCRIBE).expect("the schema parses")
}

fn exercise(source: &str) {
    let described: serde_json::Value = serde_json::from_str(DESCRIBE).expect("JSON");
    let units = source.encode_utf16().count();
    for request in [
        json!({"op": "formula.check", "input": {"source": source, "schema": described}}),
        json!({"op": "formula.check", "input": {"source": source, "expect": "predicate", "schema": null}}),
        json!({"op": "formula.format", "input": {"source": source, "width": 24}}),
    ] {
        let response = dispatch(request.to_string().as_bytes());
        assert!(response.starts_with(b"{\"ok\":"), "{source:?}");
    }
    for position in [0, units / 3, units / 2, units, units + 2] {
        for op in ["formula.signature", "formula.identifier"] {
            let request = json!({"op": op, "input": {"source": source, "position": position}});
            let response = dispatch(request.to_string().as_bytes());
            assert!(
                response.starts_with(b"{\"ok\":"),
                "{op} {source:?} at {position}"
            );
        }
    }
}

/// The tree's shape without spans: what must survive formatting.
fn shape(expr: &Expr) -> String {
    let children: Vec<String> = expr.children().into_iter().map(shape).collect();
    let head = match &expr.kind {
        ExprKind::Number(number) => format!("num {:?} {}", number.value, number.whole),
        ExprKind::String(value) => format!("str {value:?}"),
        ExprKind::Bool(value) => format!("bool {value}"),
        ExprKind::Nil => String::from("nil"),
        ExprKind::Pointer(pointer) => format!("ptr {pointer:?}"),
        ExprKind::Ident(name) => format!("id {name}"),
        ExprKind::Member(member) => format!("member {} {}", member.property.text, member.optional),
        ExprKind::Index(index) => format!("index {}", index.optional),
        ExprKind::Slice(slice) => format!(
            "slice {} {} {}",
            slice.start.is_some(),
            slice.end.is_some(),
            slice.optional
        ),
        ExprKind::Call(call) => format!("call {} {}", call.function.text, call.pipe),
        ExprKind::Method(method) => format!("method {} {}", method.method.text, method.optional),
        ExprKind::Unary(unary) => format!("unary {:?}", unary.op),
        ExprKind::Binary(binary) => format!("binary {:?} {}", binary.op, binary.not.is_some()),
        ExprKind::Chain(chain) => {
            let ops: Vec<_> = chain.operators.iter().map(|(op, _)| op).collect();
            format!("chain {ops:?}")
        }
        ExprKind::Ternary(ternary) => format!("ternary {}", ternary.then.is_some()),
        ExprKind::If(_) => String::from("if"),
        ExprKind::Array(_) => String::from("array"),
        ExprKind::Map(entries) => {
            let keys: Vec<&str> = entries
                .iter()
                .map(|entry| entry.key.text.as_str())
                .collect();
            format!("map {keys:?}")
        }
        ExprKind::Predicate(predicate) => format!("predicate {}", predicate.braces),
        ExprKind::Sequence(sequence) => {
            let names: Vec<&str> = sequence
                .lets
                .iter()
                .map(|entry| entry.name.text.as_str())
                .collect();
            format!("seq {names:?}")
        }
        ExprKind::Missing => String::from("missing"),
    };
    format!("({head} p{} {})", expr.parens, children.join(" "))
}

fn program_shape(program: &Program) -> Vec<String> {
    program
        .statements
        .iter()
        .map(|statement| {
            let kind = match &statement.kind {
                StatementKind::Let(name) => format!("let {}", name.text),
                StatementKind::Result => String::from("result"),
            };
            format!("{kind} {}", shape(&statement.value))
        })
        .collect()
}

fn leaf() -> impl Strategy<Value = String> {
    prop_oneof![
        Just(String::from("a")),
        Just(String::from("totalWeight")),
        Just(String::from("customer.name")),
        Just(String::from("tarifa_año")),
        Just(String::from("1")),
        Just(String::from("2.50")),
        Just(String::from("0x1F")),
        Just(String::from("10_000")),
        Just(String::from("1.")),
        Just(String::from("\"a b\"")),
        Just(String::from("'c'")),
        Just(String::from("`d``e`")),
        Just(String::from("true")),
        Just(String::from("nil")),
        Just(String::from("#")),
        Just(String::from("#acc")),
        Just(String::from(".weight")),
    ]
}

fn formula() -> impl Strategy<Value = String> {
    leaf().prop_recursive(5, 48, 4, |inner| {
        prop_oneof![
            (
                inner.clone(),
                prop::sample::select(vec![
                    "+",
                    "-",
                    "*",
                    "/",
                    "%",
                    "**",
                    "^",
                    "==",
                    "!=",
                    "<",
                    ">=",
                    "and",
                    "&&",
                    "or",
                    "||",
                    "in",
                    "contains",
                    "startsWith",
                    "..",
                    "??",
                ]),
                inner.clone()
            )
                .prop_map(|(l, op, r)| format!("({l} {op} {r})")),
            (inner.clone(), inner.clone(), inner.clone())
                .prop_map(|(a, b, c)| format!("({a} < {b} <= {c})")),
            (
                prop::sample::select(vec!["-", "not ", "!", "+"]),
                inner.clone()
            )
                .prop_map(|(op, operand)| format!("{op}({operand})")),
            (inner.clone(), inner.clone(), inner.clone())
                .prop_map(|(c, a, b)| format!("({c} ? {a} : {b})")),
            (inner.clone(), inner.clone()).prop_map(|(c, b)| format!("({c} ?: {b})")),
            (inner.clone(), inner.clone(), inner.clone())
                .prop_map(|(c, a, b)| format!("(if {c} {{ {a} }} else {{ {b} }})")),
            (
                prop::sample::select(vec!["round", "max", "len", "coalesce"]),
                prop::collection::vec(inner.clone(), 1..4)
            )
                .prop_map(|(name, args)| format!("{name}({})", args.join(", "))),
            (inner.clone(), inner.clone())
                .prop_map(|(list, body)| format!("filter({list}, {body})")),
            (inner.clone(), inner.clone())
                .prop_map(|(list, body)| format!("map({list}, {{{body}}})")),
            inner
                .clone()
                .prop_map(|value| format!("({value} | round(2))")),
            inner.clone().prop_map(|value| format!("({value}).Year()")),
            (inner.clone(), inner.clone())
                .prop_map(|(object, index)| format!("({object})[{index}]")),
            (inner.clone(), inner.clone())
                .prop_map(|(object, index)| format!("({object})?.[{index}:]")),
            inner
                .clone()
                .prop_map(|object| format!("({object})?.field")),
            prop::collection::vec(inner.clone(), 0..4)
                .prop_map(|items| format!("[{}]", items.join(", "))),
            (inner.clone(), inner.clone())
                .prop_map(|(a, b)| format!("{{k: {a}, \"other key\": {b}}}")),
            (inner.clone(), inner.clone())
                .prop_map(|(value, body)| format!("(let n = {value}; {body})")),
            (inner.clone(), inner.clone()).prop_map(|(a, b)| format!("({a} not in {b})")),
        ]
    })
}

fn program() -> impl Strategy<Value = String> {
    (
        prop::collection::vec(formula(), 0..3),
        formula(),
        prop::sample::select(vec![" ", "\n", "\n\n", ""]),
    )
        .prop_map(|(lets, result, gap)| {
            let mut out = String::new();
            for (index, value) in lets.iter().enumerate() {
                write!(out, "let v{index} = {value};{gap}").expect("writing to a string");
            }
            out.push_str(&result);
            out
        })
}

proptest! {
    #![proptest_config(ProptestConfig {
        failure_persistence: None,
        ..ProptestConfig::with_cases(256)
    })]

    #[test]
    fn no_text_makes_the_service_panic(source in ".{0,200}") {
        exercise(&source);
    }

    #[test]
    fn no_bracket_soup_makes_the_service_panic(source in "[()\\[\\]{}a1,;.#?:!=<'\"`\\\\ letinf|-]{0,300}") {
        exercise(&source);
    }

    #[test]
    fn no_unicode_makes_the_service_panic(source in "\\PC{0,120}") {
        exercise(&source);
    }

    #[test]
    fn formatting_is_idempotent_and_keeps_the_tree(source in program(), width in 16usize..120) {
        let program = syntax::parse(&source, &[]);
        prop_assume!(program.diagnostics.is_empty());
        let FormatOutcome::Formatted { text, .. } = syntax::format(&source, width, &[]) else {
            return Err(TestCaseError::fail(format!("{source:?} was refused")));
        };
        let reparsed = syntax::parse(&text, &[]);
        prop_assert!(reparsed.diagnostics.is_empty(), "{text:?}: {:?}", reparsed.diagnostics);
        prop_assert_eq!(program_shape(&program), program_shape(&reparsed), "{}", text);
        prop_assert_eq!(
            syntax::format(&text, width, &[]),
            FormatOutcome::Formatted { text: text.clone(), changed: false }
        );
    }

    #[test]
    fn any_text_that_formats_formats_to_itself(source in "[a-c1-3 ()\\[\\],.#?:!=<+*%|\"'\\n-]{0,60}|(let x = )?[a1(\\[ .,+?:-]{0,30}", width in 8usize..40) {
        if let FormatOutcome::Formatted { text, .. } = syntax::format(&source, width, &[]) {
            prop_assert!(syntax::parse(&text, &[]).diagnostics.is_empty(), "{:?} -> {:?}", source, text);
            prop_assert_eq!(
                program_shape(&syntax::parse(&source, &[])),
                program_shape(&syntax::parse(&text, &[]))
            );
            prop_assert_eq!(
                syntax::format(&text, width, &[]),
                FormatOutcome::Formatted { text: text.clone(), changed: false }
            );
        }
    }

    #[test]
    fn checking_a_generated_formula_never_panics(source in program()) {
        let analysis = check::check(&source, Expect::Amount, Some(&schema()), None);
        prop_assert!(analysis.diagnostics.iter().all(|d| d.span.start <= d.span.end));
    }
}

fn deep_cases() -> Vec<String> {
    let deep = 50_000;
    vec![
        format!("{}1{}", "(".repeat(deep), ")".repeat(deep)),
        format!("{}1", "-".repeat(deep)),
        format!("{}1", "not ".repeat(deep / 4)),
        format!("{}1", "[".repeat(deep)),
        format!("{}1", "{".repeat(deep)),
        format!("{}1", "round(".repeat(deep / 6)),
        format!("1{}", " ** 2".repeat(deep / 5)),
        format!("1{}", " + 2".repeat(deep / 4)),
        format!("a{}", ".b".repeat(deep / 2)),
        format!("a{}", "[0]".repeat(deep / 3)),
        format!("{}0", "1 ? 2 : ".repeat(deep / 8)),
        format!("if a {{ 1 }}{}", " else if a { 1 }".repeat(deep / 16)),
        format!("{}1", "(let a = 1; ".repeat(deep / 12)),
        format!("{}", ")".repeat(deep)),
        "let a = 1; ".repeat(3_000),
        format!("[{}]", vec!["(1 + 2)"; 3_000].join(", ")),
    ]
}

#[test]
fn the_deepest_formulas_fit_in_a_webassembly_sized_stack() {
    let worker = std::thread::Builder::new()
        .stack_size(1 << 20)
        .spawn(|| {
            for source in deep_cases() {
                let analysis = check::check(&source, Expect::Amount, Some(&schema()), None);
                assert!(!analysis.accepted() || source.starts_with("let"));
                let _ = syntax::format(&source, 80, &[]);
                let _ = check::signature_at(&source, 3, &[]);
            }
        })
        .expect("a thread");
    assert!(worker.join().is_ok(), "the stack overflowed");
}

#[test]
fn the_deepest_formula_that_parses_formats_and_checks() {
    let depth = syntax::MAX_DEPTH - 2;
    let source = format!("{}1{}", "round(".repeat(depth / 2), ")".repeat(depth / 2));
    let worker = std::thread::Builder::new()
        .stack_size(1 << 20)
        .spawn(move || {
            let analysis = check::check(&source, Expect::Amount, Some(&schema()), None);
            assert!(analysis.accepted(), "{:?}", analysis.diagnostics);
            assert!(matches!(
                syntax::format(&source, 20, &[]),
                FormatOutcome::Formatted { .. }
            ));
        })
        .expect("a thread");
    assert!(worker.join().is_ok(), "the stack overflowed");
}
