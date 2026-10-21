//! Arbitrary `Init` bodies decode to a policy or an error, never a panic.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_sandbox::wire::decode_init;

fuzz_target!(|data: &[u8]| {
    let _ = decode_init(data);
});
