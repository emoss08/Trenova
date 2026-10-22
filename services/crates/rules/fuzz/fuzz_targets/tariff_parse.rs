//! Arbitrary text parsed as a Tariff file and loaded as a program: a tree every time, never a
//! panic, with every token, diagnostic and fix inside the text and the tokens covering it.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_rules::tariff::context::Context;
use trenova_rules::tariff::syntax::outline::outline;
use trenova_rules::tariff::syntax::parse;
use trenova_rules::tariff::syntax::program::load;

fuzz_target!(|data: &[u8]| {
    let source = String::from_utf8_lossy(data);
    let file = parse(&source);
    let length = u32::try_from(source.len()).unwrap_or(u32::MAX);
    let mut at = 0;
    for token in &file.tokens {
        assert!(token.span.start >= at && token.span.end >= token.span.start);
        at = token.span.end;
    }
    assert_eq!(at, length);
    for diagnostic in &file.diagnostics {
        assert!(diagnostic.span.start <= diagnostic.span.end && diagnostic.span.end <= length);
        for fix in &diagnostic.fixes {
            for change in &fix.changes {
                assert!(change.span.end <= length);
            }
        }
    }
    let _ = outline(&source, &file);
    let _ = load(&source, None, &Context::default());
});
