//! Arbitrary text checked against the shipment schema, with and without it, and the cursor
//! operations at a position the input picks: an analysis every time, never a panic, and every
//! position inside the text.

#![no_main]

use libfuzzer_sys::fuzz_target;
use std::sync::OnceLock;
use trenova_rules::formula::check;
use trenova_rules::formula::check::Expect;
use trenova_rules::formula::check::Schema;

const DESCRIBE: &str = include_str!(
    "../../../../tms/internal/modules/rating/app/formula/testdata/shipment.describe.json"
);

fn schema() -> &'static Schema {
    static SCHEMA: OnceLock<Schema> = OnceLock::new();
    SCHEMA.get_or_init(|| serde_json::from_str(DESCRIBE).unwrap_or_default())
}

fuzz_target!(|data: &[u8]| {
    let Some((&selector, rest)) = data.split_first() else {
        return;
    };
    let source = String::from_utf8_lossy(rest);
    let expect = if selector & 1 == 0 {
        Expect::Amount
    } else {
        Expect::Predicate
    };
    let known = (selector & 2 == 0).then(schema);
    let analysis = check::check(&source, expect, known, None);
    let length = u32::try_from(source.len()).unwrap_or(u32::MAX);
    for diagnostic in &analysis.diagnostics {
        assert!(diagnostic.span.start <= diagnostic.span.end);
        assert!(diagnostic.span.end <= length);
    }
    let position = u32::from(selector) * length / 255;
    let _ = check::signature_at(&source, position, &[]);
    let _ = check::identifier_at(&source, position, known);
});
