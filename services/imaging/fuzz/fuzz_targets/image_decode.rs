//! Arbitrary bytes through every Rust-side decoder path: an image decodes, or
//! fails with a typed error, within the pixel budget, and never panics.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_imaging::engine::decode::{DecodeOptions, decode};

fuzz_target!(|data: &[u8]| {
    let options = DecodeOptions {
        max_pixels: 4_000_000,
        want_width: 0,
        want_height: 0,
    };
    if let Ok(decoded) = decode(data, options, None) {
        assert!(u64::from(decoded.source_width) * u64::from(decoded.source_height) <= 4_000_000);
    }
    let reduced = DecodeOptions {
        want_width: 64,
        want_height: 64,
        ..options
    };
    let _ = decode(data, reduced, None);
});
