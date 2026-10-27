//! A script and the host's replies, both arbitrary: the session must end every run with a step,
//! never panic, and never pause on anything but its three host functions.

#![no_main]

// Monty counts live memory through its allocator, so the script's memory limit raises
// MemoryError here as it does in the module, instead of the fuzzer running out of memory.
#[global_allocator]
static ALLOCATOR: monty_alloc::LimitedAllocator = monty_alloc::LimitedAllocator;

use libfuzzer_sys::fuzz_target;
use serde_json::json;
use trenova_scripts::{Progress, Start, Step, resume, start};

fuzz_target!(|data: &[u8]| {
    let mut parts = data.split(|byte| *byte == 0);
    let code = String::from_utf8_lossy(parts.next().unwrap_or_default()).into_owned();
    let mut replies = parts.map(|reply| String::from_utf8_lossy(reply).into_owned());

    let request = json!({
        "code": code,
        "index": json!([{"name": "shipment_list"}, {"name": "customer_get", "writes": true}]).to_string(),
        "inputs": json!({"region": "West"}).to_string(),
        "clock": {"unixSeconds": 1_793_000_000, "offsetSeconds": -18_000},
        "limits": {"outputChars": 512, "memoryBytes": 64 << 20, "cpuMs": 50, "recursion": 50, "calls": 16},
    });
    let Ok(start_request) = serde_json::from_value::<Start>(request) else {
        return;
    };

    let mut progress = start(&start_request);
    for _ in 0..=16 {
        match progress {
            Progress::Done(Step::Done(_)) => return,
            Progress::Paused(session, Step::Call { .. }) => {
                let reply = replies.next().unwrap_or_else(|| "{}".to_owned());
                progress = resume(*session, reply);
            }
            other => panic!("an impossible step: {:?}", other.step()),
        }
    }
    panic!("the session paused more often than its call limit allows");
});
