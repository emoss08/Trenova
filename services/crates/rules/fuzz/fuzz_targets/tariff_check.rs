//! Arbitrary text checked as a Tariff file through the operation the hosts call: an answer every
//! time, never a panic, and a response that is always one JSON envelope.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_rules::ops::dispatch;

fuzz_target!(|data: &[u8]| {
    let source = String::from_utf8_lossy(data);
    let request = serde_json::json!({"op": "tariff.check", "input": {"source": source}});
    let response = dispatch(request.to_string().as_bytes());
    let parsed: serde_json::Value =
        serde_json::from_slice(&response).expect("the response is JSON");
    assert!(parsed.get("ok").is_some() || parsed.get("error").is_some());
});
