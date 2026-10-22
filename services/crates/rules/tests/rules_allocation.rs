//! Charge allocation: the `testdata/allocation.json` corpus and the split's invariants.

mod rules_support;

use proptest::prelude::*;
use rust_decimal::Decimal;
use trenova_rules::allocation::input::AllocationRow;
use trenova_rules::allocation::split::allocate;
use trenova_rules::report::fixed::round_bank;

#[test]
fn corpus() {
    let ran = rules_support::run_corpus(include_str!("../testdata/allocation.json"));
    assert!(ran > 50, "only {ran} cases ran");
}

/// Percents with six places that total exactly 100, from positive integer weights.
fn percents(weights: &[u32]) -> Vec<Decimal> {
    let total: u64 = weights.iter().map(|weight| u64::from(*weight)).sum();
    let mut percents = Vec::with_capacity(weights.len());
    let mut assigned = Decimal::ZERO;
    for (index, weight) in weights.iter().enumerate() {
        if index + 1 == weights.len() {
            percents.push(Decimal::ONE_HUNDRED - assigned);
        } else {
            let percent =
                (Decimal::from(*weight) * Decimal::ONE_HUNDRED / Decimal::from(total)).round_dp(6);
            assigned += percent;
            percents.push(percent);
        }
    }
    percents
}

fn rows(percents: &[Decimal]) -> Vec<AllocationRow> {
    percents
        .iter()
        .enumerate()
        .map(|(index, percent)| AllocationRow {
            id: None,
            bill_to_customer_id: Some(format!("cus_{index}")),
            method: Some(String::from("Percent")),
            percent: Some(*percent),
            amount: None,
            sequence: Some(i64::try_from(index).unwrap_or_default()),
        })
        .collect()
}

fn total_strategy() -> impl Strategy<Value = Decimal> {
    (-10_000_000_000i64..10_000_000_000i64, 0u32..=4)
        .prop_map(|(units, scale)| Decimal::new(units, scale))
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(512))]

    #[test]
    fn percent_shares_sum_exactly_to_the_rounded_total(
        total in total_strategy(),
        weights in prop::collection::vec(1u32..10_000, 1..8),
    ) {
        let percents = percents(&weights);
        prop_assume!(percents.iter().all(|percent| *percent > Decimal::ZERO));
        let rows = rows(&percents);
        let indexed: Vec<(usize, &AllocationRow)> = rows.iter().enumerate().collect();
        let shares = allocate(total, &indexed, "cus_0")
            .expect("no overflow")
            .expect("a valid split");
        let sum: Decimal = shares.iter().map(|share| share.amount).sum();
        prop_assert_eq!(sum, round_bank(total, 2));
    }

    #[test]
    fn percent_shares_are_within_a_minor_unit_of_the_exact_share(
        total in total_strategy(),
        weights in prop::collection::vec(1u32..10_000, 1..8),
    ) {
        let percents = percents(&weights);
        prop_assume!(percents.iter().all(|percent| *percent > Decimal::ZERO));
        let rows = rows(&percents);
        let indexed: Vec<(usize, &AllocationRow)> = rows.iter().enumerate().collect();
        let shares = allocate(total, &indexed, "cus_0")
            .expect("no overflow")
            .expect("a valid split");
        let half_cent = Decimal::new(5, 3);
        let cent = Decimal::new(1, 2);
        let last = shares.len() - 1;
        for (index, share) in shares.iter().enumerate() {
            let exact = total * percents[index] / Decimal::ONE_HUNDRED;
            let error = (share.amount - exact).abs();
            if index < last {
                // Every share but the last is the exact share rounded half to even.
                prop_assert!(error <= half_cent, "share {} is {} from {}", index, error, exact);
            } else if shares.len() <= 2 {
                // With two payers the last share is within one cent of its exact share.
                prop_assert!(error <= cent, "last share is {} from {}", error, exact);
            } else {
                // The last share absorbs every other share's rounding and the total's.
                let bound = half_cent * Decimal::from(shares.len());
                prop_assert!(error <= bound, "last share is {} from {}", error, exact);
            }
        }
    }

    #[test]
    fn amount_splits_that_are_accepted_add_up(
        total_cents in 0i64..10_000_000,
        cuts in prop::collection::vec(0u32..10_000, 0..6),
    ) {
        let total = Decimal::new(total_cents, 2);
        let mut amounts = Vec::new();
        let mut left = total_cents;
        for cut in cuts {
            let take = left * i64::from(cut) / 10_000;
            if take > 0 {
                amounts.push(Decimal::new(take, 2));
                left -= take;
            }
        }
        if left > 0 {
            amounts.push(Decimal::new(left, 2));
        }
        prop_assume!(!amounts.is_empty());
        let rows: Vec<AllocationRow> = amounts
            .iter()
            .enumerate()
            .map(|(index, amount)| AllocationRow {
                bill_to_customer_id: Some(format!("cus_{index}")),
                method: Some(String::from("Amount")),
                amount: Some(*amount),
                sequence: Some(i64::try_from(index).unwrap_or_default()),
                ..AllocationRow::default()
            })
            .collect();
        let indexed: Vec<(usize, &AllocationRow)> = rows.iter().enumerate().collect();
        let shares = allocate(total, &indexed, "cus_0")
            .expect("no overflow")
            .expect("a valid split");
        let sum: Decimal = shares.iter().map(|share| share.amount).sum();
        prop_assert_eq!(sum, total);
    }

    #[test]
    fn no_allocation_request_panics(input in "\\PC{0,400}") {
        for op in ["allocation.split", "allocation.remainder", "allocation.chargeTotal", "allocation.rebalance"] {
            let request = format!("{{\"op\":\"{op}\",\"input\":{input}}}");
            let response = trenova_rules::ops::dispatch(request.as_bytes());
            prop_assert!(!response.is_empty());
        }
    }

    #[test]
    fn huge_amounts_never_panic(
        amount in "-?[0-9]{1,28}(\\.[0-9]{1,10})?",
        unit in any::<i64>(),
        percent in "[0-9]{1,28}",
    ) {
        let request = format!(
            "{{\"op\":\"allocation.split\",\"input\":{{\"defaultPayerId\":\"a\",\"base\":\"{amount}\",\"charges\":[{{\"kind\":\"Accessorial\",\"method\":\"Percentage\",\"amount\":\"{amount}\",\"unit\":{unit},\"allocations\":[{{\"billToCustomerId\":\"a\",\"method\":\"Percent\",\"percent\":\"{percent}\"}},{{\"billToCustomerId\":\"b\",\"method\":\"Amount\",\"amount\":\"{amount}\"}}]}}]}}}}"
        );
        let response = trenova_rules::ops::dispatch(request.as_bytes());
        prop_assert!(!response.is_empty());
        let request = format!(
            "{{\"op\":\"allocation.rebalance\",\"input\":{{\"charges\":[{{\"total\":\"{amount}\",\"allocations\":[{{\"method\":\"Amount\",\"amount\":\"{amount}\"}},{{\"method\":\"Amount\",\"amount\":\"{percent}\"}}]}}]}}}}"
        );
        let response = trenova_rules::ops::dispatch(request.as_bytes());
        prop_assert!(!response.is_empty());
    }
}
