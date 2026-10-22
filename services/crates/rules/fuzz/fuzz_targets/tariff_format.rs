//! Arbitrary text formatted as a Tariff file: formatted text that parses without errors to the
//! same tree and formats to itself, or a refusal, never a panic.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_rules::tariff::syntax::format::format;
use trenova_rules::tariff::syntax::parse;
use trenova_rules::tariff::syntax::shape;

fuzz_target!(|data: &[u8]| {
    let source = String::from_utf8_lossy(data);
    if let Ok(formatted) = format(&source) {
        let reparsed = parse(&formatted);
        assert_eq!(shape::file(&reparsed), shape::file(&parse(&source)));
        assert_eq!(format(&formatted).as_deref(), Ok(formatted.as_str()));
    }
});
