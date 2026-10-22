//! `tariff.scaffoldTest` writes a test that parses and passes under `tariff.test`.

use serde_json::{Value, json};

fn scaffold(input: &Value) -> Value {
    let raw = serde_json::value::to_raw_value(input).unwrap_or_else(|_| {
        serde_json::value::RawValue::from_string(String::from("{}")).unwrap_or_default()
    });
    trenova_rules::tariff::eval::ops::scaffold_test(&raw)
        .map(|output| serde_json::from_str(output.get()).unwrap_or(Value::Null))
        .unwrap_or(Value::Null)
}

fn call(op: &str, input: &Value) -> Value {
    let request = json!({"op": op, "input": input}).to_string();
    let response = trenova_rules::ops::dispatch(request.as_bytes());
    serde_json::from_slice(&response).unwrap_or(Value::Null)
}

fn sample() -> Value {
    let settings = trenova_rules::tariff::schema::Settings::default();
    trenova_rules::tariff::wire::encode(&trenova_rules::tariff::eval::record::sample(&settings))
}

fn merge(base: &mut Value, overlay: &Value) {
    match (base, overlay) {
        (Value::Object(base), Value::Object(overlay)) => {
            for (key, value) in overlay {
                match base.get_mut(key) {
                    Some(slot) => merge(slot, value),
                    None => {
                        base.insert(key.clone(), value.clone());
                    }
                }
            }
        }
        (slot, value) => *slot = value.clone(),
    }
}

fn stop(kind: &str, sequence: u32, extra: &Value) -> Value {
    let mut stop = json!({"sequence": {"$n": sequence.to_string()}, "kind": kind, "status": "New",
        "appointment": false, "window": {"start": null, "end": null}, "arrival": null,
        "departure": null, "dwell": null, "pieces": null, "weight": null,
        "location": {"city": "", "state": "", "zip": "", "zip3": "", "country": "", "timezone": "UTC"}});
    merge(&mut stop, extra);
    stop
}

const TARIFF: &str = r#"tariff "Scaffolded"

use std.accessorials.{ detention }

input rate: Money/mi = $2.35/mi
input fee: Money
input zones: List<Text> = []

require let weight = shipment.weight else "No weight"
charge "Linehaul" (quantity: shipment.distance, rate: rate)
charge "Handling {{fee}}" as HND = fee
for stop in shipment.stops where stop.dwell != none {
  let owed = detention(stop.dwell ?? 0 h, rate: $65/h)
  if owed > $0 {
    charge "Detention" as DET = owed
  }
}
if lane.origin.zip3 in zones {
  charge "Zone" as ZN = $5
}
note "{shipment.commodities.len} commodities, {weight}"
minimum $100
"#;

fn round_trip(record: &Value, inputs: &Value) -> (String, Value) {
    let output = scaffold(
        &json!({"source": TARIFF, "record": record, "inputs": inputs,
        "name": "a real \"load\" {saved}"}),
    );
    let test = output["test"]
        .as_str()
        .map_or_else(|| panic!("no test: {output}"), str::to_owned);
    let source = format!("{TARIFF}\n{test}");
    let response = call("tariff.test", &json!({"source": source}));
    (test, response)
}

#[test]
fn a_scaffolded_test_parses_and_passes() {
    let mut record = sample();
    merge(
        &mut record,
        &json!({
            "shipment": {
                "proNumber": "PRO-77", "status": "Completed", "weight": {"$q": "1240", "u": "lb"},
                "distance": {"$q": "412", "u": "mi"}, "hazmat": true,
                "temperature": {"min": {"$q": "34", "u": "°F"}, "max": null},
                "trailerType": {"code": "V53", "description": "Dry van 53'"},
                "stops": [
                    stop("Pickup", 1, &json!({"status": "Completed", "arrival": {"$date": "2026-01-05T08:00:00-06:00"},
                        "departure": {"$date": "2026-01-05T12:15:00-06:00"}, "dwell": {"$q": "4.25", "u": "h"},
                        "location": {"city": "Dallas", "state": "TX", "zip": "75201", "zip3": "752", "country": "US", "timezone": "America/Chicago"}})),
                    stop("Delivery", 2, &json!({"appointment": true, "dwell": {"$q": "3", "u": "h"}})),
                    stop("SplitDelivery", 7, &json!({})),
                ],
                "commodities": [{"name": "Widgets", "freightClass": "Class85", "weight": {"$q": "1240", "u": "lb"},
                    "pieces": {"$n": "4"}, "dimensions": {"length": null, "width": null, "height": null},
                    "volume": null, "density": null, "hazmat": false, "stackable": true}],
            },
            "lane": {"origin": {"zip3": "752", "zones": ["Z1"]}, "interstate": true},
            "fuel": {"price": {"$q": "3.2", "u": "$/gal"}},
            "rating": {"mode": "Buy", "sellTotal": {"$q": "900", "u": "$"}},
        }),
    );
    let stops = record["shipment"]["stops"].clone();
    record["shipment"]["pickup"] = stops[0].clone();
    record["shipment"]["delivery"] = stops[1].clone();
    let (test, response) = round_trip(
        &record,
        &json!({"fee": {"$q": "12.5", "u": "$"},
        "rate": {"$q": "2.35", "u": "$/mi"}, "zones": [""]}),
    );
    let tests = &response["ok"]["tests"];
    assert_eq!(tests[0]["status"], "passed", "{test}\n{response}");
    assert!(
        test.contains("given shipment.status = ShipmentStatus.Completed"),
        "{test}"
    );
    assert!(
        test.contains("given shipment.delivery = delivery(sequence: 2"),
        "{test}"
    );
    assert!(!test.contains("given input.rate"), "{test}");
    assert!(test.contains("given input.fee = $12.50"), "{test}");
    assert!(
        test.contains("expect charge(\"Linehaul\") == $968.20"),
        "{test}"
    );
    assert!(test.contains("expect charge(DET) == $211.25"), "{test}");
    assert!(test.contains("expect total =="), "{test}");
}

#[test]
fn a_sample_record_scaffolds_only_what_differs() {
    let (test, response) = round_trip(&Value::Null, &json!({"fee": {"$q": "1", "u": "$"}}));
    assert_eq!(
        response["ok"]["tests"][0]["status"], "passed",
        "{test}\n{response}"
    );
    assert_eq!(
        test,
        "test \"a real \\\"load\\\" {{saved}}\" {\n  given input.fee = $1.00\n  expect fails \"No weight\"\n}\n"
    );
}

#[test]
fn a_rating_that_fails_gives_its_error() {
    let output = scaffold(&json!({"source": "tariff \"t\"\ncharge \"x\" = $1 / 0", "name": "x"}));
    assert_eq!(output["test"], Value::Null);
    assert_eq!(output["error"]["code"], "divideByZero");
    let output = scaffold(&json!({"source": "tariff \"t\"\nlet = 1", "name": "x"}));
    assert_eq!(output["error"]["code"], "check");
    let output = scaffold(&json!({"source": "condition \"c\"\ntrue", "name": "x"}));
    assert_eq!(output["error"]["code"], "check");
}

#[test]
fn needs_are_reported() {
    let context = json!({"tables": [{"code": "Z", "valueType": "Money", "axes": [{"match": "Exact", "kind": "Zone"}]}]});
    let output = scaffold(
        &json!({"source": "tariff \"t\"\ncharge \"x\" = tables.Z[\"A\"] ?? $1", "name": "x", "context": context}),
    );
    assert_eq!(output["status"], "needs");
    assert_eq!(output["needs"], json!([{"table": "Z", "keys": ["A"]}]));
}
