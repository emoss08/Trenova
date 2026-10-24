//! Arbitrary bytes as the host's first request: decoding must refuse or accept without panicking,
//! and an accepted start must run to a step under tight limits.

#![no_main]

// Monty counts live memory through its allocator, so the script's memory limit raises
// MemoryError here as it does in the module, instead of the fuzzer running out of memory.
#[global_allocator]
static ALLOCATOR: monty_alloc::LimitedAllocator = monty_alloc::LimitedAllocator;

use libfuzzer_sys::fuzz_target;
use trenova_scripts::{Progress, Request, start};

fuzz_target!(|data: &[u8]| {
    let Ok(Request::Start(mut request)) = serde_json::from_slice::<Request>(data) else {
        return;
    };
    request.limits.cpu_ms = request.limits.cpu_ms.min(50);
    request.limits.calls = request.limits.calls.min(16);
    request.limits.recursion = request.limits.recursion.min(100);
    request.limits.output_chars = request.limits.output_chars.min(4096);
    match start(&request) {
        Progress::Paused(..) | Progress::Done(_) => {}
    }
});
