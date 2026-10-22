//! Properties the evaluator promises for every input: arithmetic never panics, budgets stop
//! every runaway formula, and no request to `formula.evaluate` makes the module panic.

use proptest::prelude::*;
use rust_decimal::Decimal;
use serde_json::json;
use trenova_rules::formula::eval::Code;
use trenova_rules::formula::eval::Limits;
use trenova_rules::formula::eval::Value;
use trenova_rules::formula::eval::budget::Meter;
use trenova_rules::formula::eval::interp;
use trenova_rules::formula::eval::interp::LookupSource;
use trenova_rules::formula::eval::interp::Setup;
use trenova_rules::formula::eval::number;
use trenova_rules::formula::eval::number::Rounding;
use trenova_rules::formula::eval::operators;
use trenova_rules::formula::eval::operators::Arithmetic;
use trenova_rules::formula::eval::parse_value;
use trenova_rules::formula::eval::time::Date;
use trenova_rules::formula::syntax;
use trenova_rules::ops::dispatch;

fn decimal() -> impl Strategy<Value = Decimal> {
    (any::<i128>(), 0u32..=28).prop_map(|(mantissa, scale)| {
        let bounded = mantissa % (1i128 << 96);
        Decimal::try_from_i128_with_scale(bounded, scale).unwrap_or(Decimal::ZERO)
    })
}

fn value() -> impl Strategy<Value = Value> {
    let leaf = prop_oneof![
        Just(Value::Nil),
        any::<bool>().prop_map(Value::Bool),
        decimal().prop_map(Value::Number),
        "[a-z0-9 ]{0,8}".prop_map(|text| Value::text(&text)),
        (any::<i64>(), any::<i32>()).prop_map(|(seconds, offset)| {
            Date::from_instant(seconds % 253_402_300_799, 0, offset % 86_399)
                .map_or(Value::Nil, Value::Date)
        }),
        any::<i64>().prop_map(Value::Duration),
    ];
    leaf.prop_recursive(3, 16, 4, |inner| {
        prop::collection::vec(inner, 0..4)
            .prop_map(|items| Value::list(items).unwrap_or(Value::Nil))
    })
}

proptest! {
    #[test]
    fn arithmetic_never_panics(left in value(), right in value()) {
        let mut meter = Meter::new(Limits::DEFAULT);
        for op in [
            Arithmetic::Add,
            Arithmetic::Sub,
            Arithmetic::Mul,
            Arithmetic::Div,
            Arithmetic::Rem,
            Arithmetic::Pow,
        ] {
            let _ = operators::arithmetic(op, &left, &right, &mut meter);
        }
        let _ = operators::equal(&left, &right);
        let _ = operators::membership(&left, &right);
        let _ = operators::index(&left, &right, &mut meter);
        let _ = operators::range(&left, &right, &mut meter);
    }

    #[test]
    fn decimal_functions_never_panic(value in decimal(), other in decimal(), places in -40i64..40) {
        for rounding in [Rounding::HalfAwayFromZero, Rounding::Up, Rounding::Down, Rounding::HalfEven] {
            let _ = number::round(value, places, rounding);
        }
        let _ = number::pow(value, other);
        let _ = number::pow(value, Decimal::from(places));
        let _ = number::sqrt(value);
        let _ = number::rem(value, other);
        let _ = number::to_i64(value);
    }

    #[test]
    fn money_sized_arithmetic_is_exact(
        a in (-1_000_000_000_000i64..1_000_000_000_000, 0u32..=6),
        b in (-1_000_000_000_000i64..1_000_000_000_000, 0u32..=6),
    ) {
        let (a, b) = (Decimal::new(a.0, a.1), Decimal::new(b.0, b.1));
        let sum = number::add(a, b).map_err(|error| TestCaseError::fail(error.message.english()))?;
        let back = number::sub(sum, b).map_err(|error| TestCaseError::fail(error.message.english()))?;
        prop_assert_eq!(back, a);
        let product = number::mul(a, b).map_err(|error| TestCaseError::fail(error.message.english()))?;
        if !b.is_zero() {
            let quotient = number::div(product, b).map_err(|error| TestCaseError::fail(error.message.english()))?;
            prop_assert_eq!(quotient, a);
        }
    }
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(48))]

    #[test]
    fn budgets_stop_runaway_formulas(
        sizes in prop::collection::vec(1u64..120_000, 1..4),
        kind in 0usize..5,
    ) {
        let mut source = String::from("#");
        for (level, size) in sizes.iter().enumerate() {
            source = match (kind + level) % 5 {
                0 => format!("map(1..{size}, {source})"),
                1 => format!("reduce(1..{size}, #acc + 1, 0) + len(map(1..2, {source}))"),
                2 => format!("filter(1..{size}, len(string({source})) > 0)"),
                3 => format!("sum(map(1..{size}, len(\"xxxxxxxxxx\" + string({source}))))"),
                _ => format!("len(join(map(1..{size}, \"ab\"), string({source})))"),
            };
        }
        let program = syntax::parse(&source, &[]);
        prop_assume!(program.diagnostics.is_empty());
        let env = parse_value("{}").map_err(TestCaseError::fail)?;
        let evaluation = interp::evaluate(
            &program,
            Setup {
                env: &env,
                lookups: LookupSource::Unavailable,
                limits: Limits::DEFAULT,
                trace_statements: false,
                record_lookups: false,
            },
        );
        if let Err(error) = evaluation.result {
            prop_assert!(
                matches!(error.code, Code::Budget | Code::Type),
                "{source}: {:?}",
                error.code
            );
        }
    }
}

const TOKENS: &[&str] = &[
    "1",
    "0",
    "-1",
    "2.5",
    "1e3",
    "0x1F",
    "\"a\"",
    "`b`",
    "nil",
    "true",
    "false",
    "x",
    "y",
    "list",
    "map",
    "stops",
    "#",
    "#acc",
    "#index",
    ".w",
    "+",
    "-",
    "*",
    "/",
    "%",
    "**",
    "^",
    "??",
    "?.",
    "?",
    ":",
    "==",
    "!=",
    "<",
    "<=",
    ">",
    ">=",
    "and",
    "or",
    "not",
    "!",
    "in",
    "matches",
    "contains",
    "startsWith",
    "..",
    "(",
    ")",
    "[",
    "]",
    "{",
    "}",
    ",",
    ";",
    "let",
    "z",
    "=",
    "if",
    "else",
    "|",
    "len(",
    "sum(",
    "map(",
    "filter(",
    "reduce(",
    "lookup(\"t\", ",
    "lookup2(\"g\", ",
    "date(\"2026-01-01\")",
    "duration(\"1h\")",
    ".Year()",
    ".Hours()",
    "round(",
    "split(",
    "string(",
    "groupBy(",
    "sortBy(",
    "1..1000",
    "x[0]",
    "x[-1:]",
];

fn source() -> impl Strategy<Value = String> {
    prop::collection::vec(prop::sample::select(TOKENS), 0..24).prop_map(|tokens| tokens.join(" "))
}

fn tables() -> serde_json::Value {
    json!([
        {"name": "t", "axes": [{"match": "Range", "overflow": "Nearest"}], "cells": [
            {"at": [{"min": 0, "max": 10}], "value": 1},
            {"at": [{"min": 20}], "value": 2},
        ]},
        {"name": "g", "axes": [{"match": "Exact", "normalization": "Zip3"}, {"match": "Range", "overflow": "ClampToTopBand"}], "cells": [
            {"at": [{"key": "75201"}, {"min": 0, "max": 100}], "value": 3},
        ]},
    ])
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(512))]

    #[test]
    fn no_formula_makes_evaluate_panic(source in source(), predicate in any::<bool>(), traced in any::<bool>()) {
        let request = json!({
            "op": "formula.evaluate",
            "input": {
                "programs": [{"id": "p", "source": source, "expect": if predicate { "predicate" } else { "amount" }}],
                "env": {
                    "x": [1, "a", null, {"w": 2}],
                    "y": {"$date": "2026-03-01T10:00:00-06:00"},
                    "list": [3, 1, 2],
                    "stops": [{"w": 1}, {"w": null}],
                },
                "tables": tables(),
                "trace": {"statements": traced, "lookups": traced},
                "budget": {"steps": 20000},
            },
        });
        let response = dispatch(request.to_string().as_bytes());
        let parsed: serde_json::Value = serde_json::from_slice(&response)
            .map_err(|error| TestCaseError::fail(error.to_string()))?;
        prop_assert!(parsed.get("ok").is_some(), "{parsed}");
    }

    #[test]
    fn no_request_makes_evaluate_panic(bytes in prop::collection::vec(any::<u8>(), 0..200)) {
        let mut request = br#"{"op":"formula.evaluate","input":"#.to_vec();
        request.extend_from_slice(&bytes);
        request.push(b'}');
        let _ = dispatch(&request);
    }

    #[test]
    fn no_environment_makes_evaluate_panic(env in "[\\[\\]{}\":,0-9a-z$. -]{0,64}") {
        let request = format!(
            r#"{{"op":"formula.evaluate","input":{{"programs":[{{"id":"p","source":"x + 1","expect":"amount"}}],"env":{env}}}}}"#
        );
        let _ = dispatch(request.as_bytes());
    }
}
