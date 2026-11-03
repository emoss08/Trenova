//! Arbitrary bytes probed as a PDF: a readable document describes every page it counts, an
//! encrypted one says so, and anything else is a typed error, never a panic.

#![no_main]

#[path = "pdf_engine.rs"]
mod pdf_engine;

use libfuzzer_sys::fuzz_target;
use pdf_engine::{LIMITS, pdf, with_engine};
use trenova_imaging::engine::probe::probe;
use trenova_proto::imaging::v1::ProbeHeader;

fuzz_target!(|data: &[u8]| {
    with_engine(|engine| {
        for text_profile in [false, true] {
            let header = ProbeHeader {
                document: None,
                text_profile,
            };
            if let Ok(described) = probe(engine, &header, pdf(data), LIMITS)
                && !described.encrypted
            {
                assert_eq!(described.pages.len(), described.page_count as usize);
                for (index, page) in described.pages.iter().enumerate() {
                    assert_eq!(page.index as usize, index);
                    assert!(page.rotation % 90 == 0 && page.rotation < 360);
                }
            }
        }
    });
});
