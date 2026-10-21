//! Arbitrary EXIF blocks: an orientation of 1 to 8, or none, never a panic.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_imaging::engine::decode::exif_orientation;

fuzz_target!(|data: &[u8]| {
    if let Some(orientation) = exif_orientation(data) {
        assert!((1..=8).contains(&orientation));
    }
});
