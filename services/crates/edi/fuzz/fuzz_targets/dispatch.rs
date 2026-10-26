//! Sends arbitrary bytes through the module's dispatch, as a host would: every request gets
//! an envelope back, never a panic.

#![no_main]

use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    let response = trenova_edi::ops::dispatch(data);
    assert!(response.starts_with(b"{\"ok\":") || response.starts_with(b"{\"error\":"));
});
