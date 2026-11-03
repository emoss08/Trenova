//! Charges split between payers through `allocation.split`, built from arbitrary bytes so most
//! splits are valid: whenever a charge splits, its shares add up to exactly its total rounded
//! half to even, and when every charge splits, the payers' totals add up to the charges'.
//! Money is never made or lost in rounding.

#![no_main]

use libfuzzer_sys::fuzz_target;
use rust_decimal::{Decimal, RoundingStrategy};
use serde_json::{Value, json};
use std::str::FromStr;
use trenova_rules::ops::dispatch;

const PAYERS: [&str; 4] = ["a", "b", "c", ""];

struct Bytes<'a> {
    data: &'a [u8],
    at: usize,
}

impl Bytes<'_> {
    fn byte(&mut self) -> u8 {
        let byte = self.data.get(self.at).copied().unwrap_or(0);
        self.at += 1;
        byte
    }

    fn cents(&mut self) -> i64 {
        let raw = u32::from_le_bytes([self.byte(), self.byte(), self.byte(), self.byte()]);
        let cents = i64::from(raw % 100_000_000);
        if self.byte() >= 0xF8 { -cents } else { cents }
    }
}

fn money(cents: i64, places: u32) -> String {
    Decimal::new(cents, places).to_string()
}

fn rows(input: &mut Bytes<'_>, total_cents: i64) -> Vec<Value> {
    let count = usize::from(input.byte() % 5);
    let percent = input.byte() % 2 == 0;
    let mixed = input.byte() >= 0xF0;
    let mut left_cents = total_cents;
    let mut left_percent: i64 = 1_000_000;
    (0..count)
        .map(|index| {
            let last = index + 1 == count;
            let payer = PAYERS[usize::from(input.byte()) % PAYERS.len()];
            let by_percent = if mixed {
                input.byte() % 2 == 0
            } else {
                percent
            };
            let value = if by_percent {
                let take = if last {
                    left_percent
                } else {
                    left_percent * i64::from(input.byte()) / 255
                };
                left_percent -= take;
                json!({"billToCustomerId": payer, "method": "Percent", "percent": money(take, 4)})
            } else {
                let take = if last {
                    left_cents
                } else {
                    left_cents * i64::from(input.byte()) / 255
                };
                left_cents -= take;
                json!({"billToCustomerId": payer, "method": "Amount", "amount": money(take, 2)})
            };
            if input.byte() >= 0xFA {
                json!({"billToCustomerId": payer, "method": "Percent", "percent": "abc"})
            } else {
                value
            }
        })
        .collect()
}

fn request(data: &[u8]) -> Value {
    let mut input = Bytes { data, at: 0 };
    let base_cents = input.cents();
    let charges: Vec<Value> = (0..1 + input.byte() % 4)
        .map(|_| {
            if input.byte() % 2 == 0 {
                let total = input.cents();
                json!({"kind": "Freight", "total": money(total, 2), "allocations": rows(&mut input, total)})
            } else {
                let amount = input.cents();
                let unit = i64::from(input.byte() % 6);
                let method = ["Flat", "PerUnit", "Percentage"][usize::from(input.byte() % 3)];
                let total = match method {
                    "Flat" => amount * unit.max(1),
                    "PerUnit" => amount * unit,
                    _ => base_cents * amount / 10_000,
                };
                json!({
                    "kind": "Accessorial",
                    "method": method,
                    "amount": money(amount, 2),
                    "unit": unit,
                    "allocations": rows(&mut input, total),
                })
            }
        })
        .collect();
    json!({
        "op": "allocation.split",
        "input": {"defaultPayerId": "a", "base": money(base_cents, 2), "charges": charges},
    })
}

fn decimal(value: &Value) -> Decimal {
    Decimal::from_str(value.as_str().expect("a decimal string")).expect("a decimal")
}

fuzz_target!(|data: &[u8]| {
    let response = dispatch(request(data).to_string().as_bytes());
    let parsed: Value = serde_json::from_slice(&response).expect("the response is JSON");
    let Some(ok) = parsed.get("ok") else {
        return;
    };
    let mut split_total = Decimal::ZERO;
    let mut every_charge_split = true;
    for charge in ok["charges"].as_array().expect("charges") {
        let Some(shares) = charge.get("shares").and_then(Value::as_array) else {
            every_charge_split = false;
            continue;
        };
        let rounded = decimal(&charge["total"])
            .round_dp_with_strategy(2, RoundingStrategy::MidpointNearestEven);
        let sum: Decimal = shares.iter().map(|share| decimal(&share["amount"])).sum();
        assert_eq!(
            sum, rounded,
            "a charge's shares do not add up to it: {charge}"
        );
        split_total += sum;
    }
    match ok.get("payers").and_then(Value::as_array) {
        Some(payers) => {
            assert!(
                every_charge_split,
                "payers were summed though a charge did not split"
            );
            let mut payer_total = Decimal::ZERO;
            for payer in payers {
                let total = decimal(&payer["totalAmount"]);
                assert_eq!(
                    total,
                    decimal(&payer["freightAmount"]) + decimal(&payer["accessorialAmount"]),
                    "a payer's total is not its freight and accessorials: {payer}"
                );
                payer_total += total;
            }
            assert_eq!(
                payer_total, split_total,
                "the payers do not add up to the charges"
            );
        }
        None => assert!(
            !every_charge_split,
            "every charge split but no payer was summed"
        ),
    }
});
