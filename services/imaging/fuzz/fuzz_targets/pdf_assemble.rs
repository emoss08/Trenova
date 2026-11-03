//! A PDF cut into an output by arbitrary page ranges and rotations, as a packet's turned
//! document is: an accepted output holds exactly the pages its ranges name, and a range past
//! the last page or a turn that is not a quarter turn is refused, never a panic.
//!
//! The first bytes choose the ranges; the rest is the document.

#![no_main]

#[path = "pdf_engine.rs"]
mod pdf_engine;

use libfuzzer_sys::fuzz_target;
use pdf_engine::{Kept, LIMITS, pdf, with_engine};
use trenova_imaging::engine::assemble::assemble;
use trenova_imaging::engine::pdf::page_count;
use trenova_proto::imaging::v1::{AssembleHeader, AssembleOutput, PageRange};

const TO_LAST: u32 = u32::MAX;
const CONTROL: usize = 12;

fn ranges(control: &[u8]) -> Vec<PageRange> {
    control
        .chunks(3)
        .map(|chunk| {
            let first = u32::from(chunk[0] % 8);
            let last = match chunk.get(1) {
                Some(byte) if *byte >= 0xF0 => TO_LAST,
                Some(byte) => first + u32::from(byte % 4),
                None => first,
            };
            let rotation = match chunk.get(2) {
                Some(byte) if *byte >= 0xFC => u32::from(*byte),
                Some(byte) => u32::from(byte % 4) * 90,
                None => 0,
            };
            PageRange {
                source: 0,
                first,
                last,
                rotation,
            }
        })
        .collect()
}

fuzz_target!(|data: &[u8]| {
    if data.len() <= CONTROL {
        return;
    }
    let (control, document) = data.split_at(CONTROL);
    let pages = ranges(control);
    with_engine(|engine| {
        let count = engine
            .pdf()
            .open(document)
            .map(|opened| page_count(&opened));
        let header = AssembleHeader {
            source_count: 1,
            outputs: vec![AssembleOutput {
                pages: pages.clone(),
            }],
            total_size: document.len() as u64,
        };
        let mut kept = Kept::default();
        let assembled = assemble(engine, &header, &[pdf(document)], LIMITS, &mut kept);
        let (Ok(summary), Ok(count)) = (assembled, count) else {
            return;
        };
        let expected: u32 = pages
            .iter()
            .map(|range| {
                let last = if range.last == TO_LAST {
                    count.saturating_sub(1)
                } else {
                    range.last
                };
                assert!(last < count, "a range past the last page was accepted");
                assert!(
                    range.rotation % 90 == 0,
                    "a turn that is not a quarter turn was accepted"
                );
                last - range.first + 1
            })
            .sum();
        assert_eq!(summary.output_page_counts, vec![expected]);
    });
});
