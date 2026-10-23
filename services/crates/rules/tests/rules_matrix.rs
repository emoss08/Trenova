//! Rate matrix bands: the `testdata/matrix.json` corpus and the half-open band rule.

mod rules_support;

use proptest::prelude::*;
use rust_decimal::Decimal;
use trenova_rules::matrix::band::contains;

#[test]
fn corpus() {
    let ran = rules_support::run_corpus(include_str!("../testdata/matrix.json"));
    assert!(ran > 12, "only {ran} cases ran");
}

fn decimal() -> impl Strategy<Value = Decimal> {
    (-1_000_000i64..1_000_000, 0u32..4).prop_map(|(units, scale)| Decimal::new(units, scale))
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(1024))]

    #[test]
    fn contains_is_the_half_open_band(
        min in prop::option::of(decimal()),
        max in prop::option::of(decimal()),
        value in decimal(),
    ) {
        let expected = min.is_none_or(|min| min <= value) && max.is_none_or(|max| value < max);
        prop_assert_eq!(contains(min, max, value), expected);
    }

    #[test]
    fn adjacent_bands_never_both_claim_a_value(
        low in decimal(),
        width in 1i64..10_000,
        other in 1i64..10_000,
        value in decimal(),
    ) {
        let boundary = low + Decimal::from(width);
        let high = boundary + Decimal::from(other);
        let lower = contains(Some(low), Some(boundary), value);
        let upper = contains(Some(boundary), Some(high), value);
        prop_assert!(!(lower && upper));
        prop_assert_eq!(lower || upper, low <= value && value < high);
    }

    #[test]
    fn no_matrix_request_panics(input in "\\PC{0,300}") {
        let request = format!("{{\"op\":\"matrix.coverage\",\"input\":{input}}}");
        prop_assert!(!trenova_rules::ops::dispatch(request.as_bytes()).is_empty());
    }

    #[test]
    fn coverage_never_panics(
        bands in prop::collection::vec(
            (prop::option::of(-50i64..50), prop::option::of(-50i64..50), 0u8..3),
            0..12,
        ),
        position in -1i64..5,
    ) {
        let cells: Vec<serde_json::Value> = bands
            .iter()
            .map(|(min, max, zone)| serde_json::json!({"d0Key": format!("Z{zone}"), "d1Min": min, "d1Max": max}))
            .collect();
        let dimensions = serde_json::json!([
            {"position": 0, "kind": "Zone", "matchMode": "Exact"},
            {"position": position, "kind": "WeightBreak", "matchMode": "Range"}
        ]);
        let coverage = serde_json::json!({"op": "matrix.coverage", "input": {"dimensions": dimensions, "cells": cells}});
        prop_assert!(!trenova_rules::ops::dispatch(coverage.to_string().as_bytes()).is_empty());
    }
}
