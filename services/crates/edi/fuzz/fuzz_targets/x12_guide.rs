//! Checks arbitrary JSON as a guide, compiles it when the check passes, and reads a tender
//! with it: a guide from a person or an import must never panic the module or make a read
//! unbounded.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_edi::x12::guide::check::check_guide;
use trenova_edi::x12::guide::compiled::CompiledGuide;
use trenova_edi::x12::guide::model::Guide;
use trenova_edi::x12::read::{GuideChoice, ReadRequest, read};

const TENDER: &str =
    "ST*204*0001~B2**SIML**SHIP-1**PP~B2A*00~S5*1*CL~N1*SF*Shipper~S5*2*CU~SE*7*0001~";

fuzz_target!(|data: &[u8]| {
    let Ok(guide) = serde_json::from_slice::<Guide>(data) else {
        return;
    };
    let base = trenova_edi::x12::guide::builtin::by_id("x12.004010.204").map(CompiledGuide::guide);
    let _ = check_guide(&guide, base);
    if CompiledGuide::compile(guide.clone()).is_err() {
        return;
    }
    let request = ReadRequest {
        text: TENDER.to_owned(),
        separators: None,
        guides: vec![GuideChoice {
            transaction_set: guide.transaction_set.clone(),
            guide: Some(guide),
            builtin: None,
        }],
        include_segments: true,
        fragment: true,
    };
    let _ = read(&request);
});
