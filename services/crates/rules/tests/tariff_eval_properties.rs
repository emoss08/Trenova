//! Properties of the evaluator: a bill's lines always add up to its total, in every round mode
//! and in the billing currency, and evaluation never panics, whatever it is given.

use proptest::prelude::*;
use rust_decimal::Decimal;
use serde_json::{Value, json};
use std::fmt::Write as _;
use trenova_rules::tariff::eval::bill::{Bill, Clauses, Line, LineKind};
use trenova_rules::tariff::syntax::ast::RoundMode;

fn call(op: &str, input: &Value) -> Value {
    let request = json!({"op": op, "input": input}).to_string();
    let response = trenova_rules::ops::dispatch(request.as_bytes());
    serde_json::from_slice(&response).unwrap_or(Value::Null)
}

fn amount() -> impl Strategy<Value = Decimal> {
    (-5_000_000i64..50_000_000i64, 0u32..7).prop_map(|(units, scale)| Decimal::new(units, scale))
}

fn mode() -> impl Strategy<Value = RoundMode> {
    prop_oneof![
        Just(RoundMode::HalfUp),
        Just(RoundMode::HalfEven),
        Just(RoundMode::Up),
        Just(RoundMode::Down),
    ]
}

fn increment() -> impl Strategy<Value = Decimal> {
    prop_oneof![
        Just(Decimal::new(1, 2)),
        Just(Decimal::ONE),
        Just(Decimal::from(5)),
        Just(Decimal::new(25, 2)),
    ]
}

fn decimal(text: &Value) -> Decimal {
    text["$q"]
        .as_str()
        .and_then(|text| text.parse().ok())
        .unwrap_or(Decimal::MAX)
}

/// Well-formed expressions over values of every kind, most of which the checker would refuse:
/// the evaluator must answer each with a value or an error, never a panic.
fn expression() -> impl Strategy<Value = String> {
    let leaf = prop_oneof![
        Just("1"),
        Just("0"),
        Just("-3"),
        Just("2.5"),
        Just("$3"),
        Just("$2/mi"),
        Just("$18.5/cwt"),
        Just("4 lb"),
        Just("5 mi"),
        Just("0 mi"),
        Just("2 h"),
        Just("40 °F"),
        Just("18%"),
        Just("\"t\""),
        Just("@2026-01-05"),
        Just("@2026-01-05T08:00Z"),
        Just("none"),
        Just("true"),
        Just("[1, 2]"),
        Just("[]"),
        Just("{ a: 1 }"),
        Just("Delivery"),
        Just("shipment.weight"),
        Just("shipment.stops"),
        Just("shipment.distance"),
        Just("tables.T"),
        Just("rating.date"),
        Just("9_999_999_999_999_999_999_999_999_999"),
    ]
    .prop_map(str::to_owned);
    leaf.prop_recursive(4, 24, 3, |inner| {
        let binary = prop_oneof![
            Just("+"),
            Just("-"),
            Just("*"),
            Just("/"),
            Just("%"),
            Just("=="),
            Just("<"),
            Just("??"),
            Just("and"),
            Just("or"),
            Just("in"),
        ];
        let method = prop_oneof![
            Just(".round()"),
            Just(".len"),
            Just(".value"),
            Just(".in(cwt)"),
            Just(".sum()"),
            Just(".max(x => x)"),
            Just(".where(x => x)"),
            Just(".roundUp(to: 0 lb)"),
            Just(".pow(99)"),
            Just(".split(\"\")"),
            Just(".sortBy(x => x)"),
            Just(".weekday"),
            Just(".addDays(1)"),
            Just(".abs()"),
            Just(".clamp(1, 0)"),
            Just(".first()"),
            Just(".groupBy(x => x)"),
            Just(".sqrt()"),
            Just(".matches(\"(\")"),
            Just("[0]"),
            Just(".band(1 mi)"),
            Just(".interpolate(1 mi)"),
            Just("?.code"),
        ];
        prop_oneof![
            (inner.clone(), binary, inner.clone()).prop_map(|(a, op, b)| format!("({a} {op} {b})")),
            (inner.clone(), method).prop_map(|(a, m)| format!("({a}){m}")),
            (inner.clone(), inner.clone()).prop_map(|(a, b)| format!("[{a}, {b}]")),
            (inner.clone(), inner.clone(), inner.clone())
                .prop_map(|(c, a, b)| format!("(if {c} {{ {a} }} else {{ {b} }})")),
            (inner.clone(), inner.clone()).prop_map(|(a, b)| format!("{{ x: {a}, y: {b} }}")),
            (inner.clone(), inner)
                .prop_map(|(a, b)| format!("(match {a} {{\n  1 => {b}\n  _ => {b}\n}})")),
        ]
    })
}

proptest! {
    #[test]
    fn lines_always_add_up_to_the_total(
        amounts in proptest::collection::vec((amount(), any::<bool>()), 0..12),
        minimum in proptest::option::of(amount()),
        maximum in proptest::option::of(amount()),
        mode in mode(),
        increment in increment(),
        rate in (1i64..5_000_000).prop_map(|units| Decimal::new(units, 6)),
    ) {
        let mut bill = Bill::default();
        for (index, (value, coded)) in amounts.iter().enumerate() {
            let _ = bill.charge(Line {
                kind: LineKind::Charge,
                label: Some(format!("line {index}")),
                code: coded.then(|| String::from("ACC")),
                amount: if *coded { value.abs() } else { *value },
                quantity: None,
                rate: None,
                span: None,
            });
        }
        let clauses = Clauses { minimum, maximum, round: Some((mode, increment)) };
        if let Ok(finished) = bill.finish(&clauses) {
            let sum: Decimal = finished.lines.iter().map(|line| line.amount).sum();
            prop_assert_eq!(sum, finished.total);
            let freight: Decimal = finished.lines.iter().filter(|line| line.freight()).map(|line| line.amount).sum();
            prop_assert_eq!(freight + finished.accessorial_total, finished.total);
            prop_assert!(finished.lines.iter().all(|line| !line.amount.is_zero()));
            if let Ok(converted) = finished.convert(rate, &clauses) {
                let sum: Decimal = converted.lines.iter().copied().sum();
                prop_assert_eq!(sum, converted.total);
            }
        }
    }

    #[test]
    fn evaluated_bills_add_up_in_every_round_mode(
        amounts in proptest::collection::vec(0i64..10_000_000, 1..8),
        scale in 0u32..5,
        clause in prop_oneof![
            Just(""), Just("round to cents"), Just("round half even to cents"),
            Just("round up to cents"), Just("round down to cents"), Just("round up to $5"),
            Just("round down to $1"), Just("round half even to $0.25"),
        ],
        minimum in proptest::option::of(0i64..100_000),
    ) {
        let mut source = String::from("tariff \"p\"\n");
        for (index, units) in amounts.iter().enumerate() {
            let _ = writeln!(source, "charge \"Line {index}\" = ${}", Decimal::new(*units, scale));
        }
        if let Some(minimum) = minimum {
            let _ = writeln!(source, "minimum ${minimum}");
        }
        source.push_str(clause);
        let response = call("tariff.evaluate", &json!({"programs": [{"id": "p", "source": source}]}));
        let result = &response["ok"]["results"][0];
        prop_assert_eq!(&result["status"], "rated", "{}", response);
        let lines = result["bill"]["lines"].as_array().cloned().unwrap_or_default();
        let sum: Decimal = lines.iter().map(|line| decimal(&line["amount"])).sum();
        prop_assert_eq!(sum, decimal(&result["bill"]["total"]));
    }

    #[test]
    fn evaluation_never_panics_on_arbitrary_expressions(expr in expression()) {
        let source = format!("tariff \"p\"\nlet value = {expr}\nnote \"{{value}}\"\ncharge \"x\" = $1");
        let context = json!({"tables": [{"code": "T", "valueType": "Money",
            "axes": [{"match": "Range", "kind": "Distance"}],
            "cells": [{"at": [{"min": "0"}], "value": "1"}]}]});
        let response = call("tariff.evaluate", &json!({"context": context, "trace": true,
            "programs": [{"id": "p", "source": source}]}));
        prop_assert!(response.get("ok").is_some(), "{}", response);
        let status = &response["ok"]["results"][0]["status"];
        prop_assert!(status == "rated" || status == "error" || status == "refused", "{}", response);
    }

    #[test]
    fn evaluation_never_panics_on_arbitrary_source(source in "\\PC{0,200}") {
        let response = call("tariff.evaluate", &json!({"programs": [{"id": "p", "source": source}]}));
        prop_assert!(response.get("ok").is_some());
        let response = call("tariff.test", &json!({"source": source}));
        prop_assert!(response.get("ok").is_some());
    }
}
