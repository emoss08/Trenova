//! Arbitrary text rated against the sample record through `tariff.evaluate`: one result for the
//! one program every time, metered, never a panic.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_rules::ops::dispatch;

fuzz_target!(|data: &[u8]| {
    let source = String::from_utf8_lossy(data);
    let request = serde_json::json!({
        "op": "tariff.evaluate",
        "input": {"programs": [{"id": "p", "source": source}], "trace": true},
    });
    let response = dispatch(request.to_string().as_bytes());
    let parsed: serde_json::Value =
        serde_json::from_slice(&response).expect("the response is JSON");
    let results = parsed["ok"]["results"].as_array().expect("an evaluation answers");
    assert_eq!(results.len(), 1);
});
