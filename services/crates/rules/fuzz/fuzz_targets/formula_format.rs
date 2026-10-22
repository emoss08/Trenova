//! Arbitrary text formatted at a width the input picks: formatted text that parses and formats
//! to itself, or a refusal, never a panic.

#![no_main]

use libfuzzer_sys::fuzz_target;
use trenova_rules::formula::syntax;
use trenova_rules::formula::syntax::FormatOutcome;

fuzz_target!(|data: &[u8]| {
    let Some((&width, rest)) = data.split_first() else {
        return;
    };
    let source = String::from_utf8_lossy(rest);
    let width = usize::from(width).max(1);
    if let FormatOutcome::Formatted { text, .. } = syntax::format(&source, width, &[]) {
        assert!(syntax::parse(&text, &[]).diagnostics.is_empty());
        assert_eq!(
            syntax::format(&text, width, &[]),
            FormatOutcome::Formatted {
                text: text.clone(),
                changed: false
            }
        );
    }
});
