//! Report formatting: the `testdata/report.json` corpus and formatting that never panics.

mod rules_support;

use proptest::prelude::*;
use serde_json::Value;
use serde_json::json;

#[test]
fn corpus() {
    let ran = rules_support::run_corpus(include_str!("../testdata/report.json"));
    assert!(ran > 30, "only {ran} cases ran");
}

#[test]
fn one_value_shows_the_rounding_the_server_and_browser_now_share() {
    let request = json!({"op": "report.format", "input": {"columns": [
        {"display": {"style": "number", "decimals": 2}, "values": [1.005, 2.675]}
    ]}});
    let response = trenova_rules::ops::dispatch(request.to_string().as_bytes());
    let response: Value = serde_json::from_slice(&response).expect("JSON");
    assert_eq!(
        response,
        json!({"ok": {"columns": [{"values": [{"text": "1.01"}, {"text": "2.68"}]}]}})
    );
}

fn one_of(options: &'static [&'static str]) -> impl Strategy<Value = &'static str> {
    prop::sample::select(options)
}

fn number() -> impl Strategy<Value = String> {
    prop_oneof![
        "-?[0-9]{1,29}(\\.[0-9]{0,29})?",
        "-?[0-9]{1,3}e-?[0-9]{1,3}",
        any::<i64>().prop_map(|value| value.to_string()),
        any::<f64>()
            .prop_filter("finite", |value| value.is_finite())
            .prop_map(|value| value.to_string()),
    ]
}

fn value() -> impl Strategy<Value = Value> {
    prop_oneof![
        Just(Value::Null),
        any::<bool>().prop_map(Value::Bool),
        number().prop_map(|text| serde_json::from_str(&text).unwrap_or(Value::String(text))),
        number().prop_map(Value::String),
        "\\PC{0,12}".prop_map(Value::String),
        Just(json!([1, 2])),
        Just(json!({"a": 1})),
    ]
}

fn display() -> impl Strategy<Value = Value> {
    (
        one_of(&[
            "", "number", "currency", "percent", "duration", "date", "bool", "text", "x",
        ]),
        -20i64..30,
        any::<bool>(),
        one_of(&["", "USD", "EUR", "JPY", "XYZ"]),
        one_of(&["", "minus", "parentheses"]),
        one_of(&["", "standard", "compact"]),
        one_of(&["", "seconds", "minutes", "hours", "days"]),
        one_of(&["", "clock", "compact", "verbose", "decimalHours"]),
        prop_oneof![
            Just(Value::Null),
            number().prop_map(|width| json!({"width": width})),
            prop::collection::vec(number(), 0..5).prop_map(|edges| json!({"edges": edges})),
        ],
        prop::collection::vec(
            (
                one_of(&["gt", "gte", "lt", "lte", "eq", "between", "?"]),
                number(),
                number(),
            ),
            0..3,
        ),
    )
        .prop_map(
            |(
                style,
                decimals,
                grouping,
                currency,
                negative,
                notation,
                unit,
                duration,
                band,
                rules,
            )| {
                let rules: Vec<Value> = rules
                    .into_iter()
                    .map(|(op, value, upper)| {
                        json!({"op": op, "value": value, "upper": upper, "tone": "warning"})
                    })
                    .collect();
                json!({
                    "style": style, "decimals": decimals, "grouping": grouping,
                    "currency": currency, "negative": negative, "notation": notation,
                    "durationUnit": unit, "durationStyle": duration, "prefix": "<",
                    "suffix": ">", "band": band, "rules": rules
                })
            },
        )
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(512))]

    #[test]
    fn formatting_never_panics(
        display in display(),
        value_type in one_of(&["", "int", "decimal", "epoch", "bool", "string"]),
        values in prop::collection::vec(value(), 0..12),
    ) {
        let request = json!({"op": "report.format", "input": {"columns": [
            {"display": display, "valueType": value_type, "values": values}
        ]}});
        let response = trenova_rules::ops::dispatch(request.to_string().as_bytes());
        let response: Value = serde_json::from_slice(&response).expect("JSON");
        // A display whose decimals fail to read is a request error; anything else formats.
        if let Some(columns) = response.pointer("/ok/columns/0/values") {
            prop_assert_eq!(columns.as_array().map(Vec::len), Some(values.len()));
        } else {
            prop_assert!(response.get("error").is_some());
        }
    }

    #[test]
    fn exact_rounding_matches_decimal_rounding(
        mantissa in -1_000_000_000_000i64..1_000_000_000_000i64,
        scale in 0u32..12,
        places in 0i64..=10,
    ) {
        let value = rust_decimal::Decimal::new(mantissa, scale);
        let request = json!({"op": "report.format", "input": {"columns": [
            {"display": {"style": "number", "decimals": places}, "values": [value.to_string()], "valueType": "decimal"}
        ]}});
        let response = trenova_rules::ops::dispatch(request.to_string().as_bytes());
        let response: Value = serde_json::from_slice(&response).expect("JSON");
        let places = u32::try_from(places).unwrap_or_default();
        let rounded = value.round_dp_with_strategy(
            places,
            rust_decimal::RoundingStrategy::MidpointAwayFromZero,
        );
        let mut expected = format!("{:.*}", places as usize, rounded.abs());
        if value.is_sign_negative() && !value.is_zero() {
            expected.insert(0, '-');
        }
        prop_assert_eq!(response.pointer("/ok/columns/0/values/0/text"), Some(&Value::String(expected)));
    }

    #[test]
    fn no_report_request_panics(input in "\\PC{0,300}") {
        for op in ["report.format", "report.resolve"] {
            let request = format!("{{\"op\":\"{op}\",\"input\":{input}}}");
            prop_assert!(!trenova_rules::ops::dispatch(request.as_bytes()).is_empty());
        }
    }
}
