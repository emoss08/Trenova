//! The X12 character set, and bringing text into it.
//!
//! X12.6 defines a basic character set (capital letters, digits, the space and
//! `! " & ' ( ) * + , - . / : ; ? =`) and an extended one (lowercase letters and
//! ``% @ [ ] _ { } \ | < > ~ ^ ` # $``). Together they are exactly printable ASCII, U+0020 to
//! U+007E, and that is what a partner's translator accepts in an element: anything else (an
//! `ñ`, an en dash, a curly quote, a non-breaking space) is rejected with AK403 code 6.
//!
//! Trenova's records hold names and addresses as people typed them, so every value is folded
//! into the set before it is written: a character with an ASCII form is written as that form
//! (`ñ` as `n`, `ß` as `ss`, `–` as `-`, `½` as `1/2`), a tab or line break as a space, and
//! anything else (CJK, emoji, combining marks, control characters) is dropped. The table is
//! [`table::FOLDS`]; it is small enough to keep the WebAssembly module lean, where a general
//! Unicode normalization crate would carry decomposition tables for every script only to have
//! most of their output dropped.

mod table;

use std::borrow::Cow;

/// Whether `character` is in the X12 basic or extended character set.
#[must_use]
pub fn is_x12(character: char) -> bool {
    matches!(character, ' '..='~')
}

/// `text` in the X12 character set: borrowed when it already is, otherwise each character
/// outside the set written as its ASCII form or dropped.
#[must_use]
pub fn fold(text: &str) -> Cow<'_, str> {
    if text.chars().all(is_x12) {
        return Cow::Borrowed(text);
    }
    let mut folded = String::with_capacity(text.len());
    for character in text.chars() {
        push_folded(character, &mut folded);
    }
    Cow::Owned(folded)
}

fn push_folded(character: char, out: &mut String) {
    if is_x12(character) {
        out.push(character);
        return;
    }
    if matches!(
        character,
        '\t' | '\n' | '\u{0B}' | '\u{0C}' | '\r' | '\u{85}'
    ) {
        out.push(' ');
        return;
    }
    let code = u32::from(character);
    if (0xFF01..=0xFF5E).contains(&code) {
        if let Some(narrow) = char::from_u32(code - 0xFEE0) {
            out.push(narrow);
        }
        return;
    }
    if let Ok(index) = table::FOLDS.binary_search_by_key(&character, |(from, _)| *from) {
        out.push_str(table::FOLDS[index].1);
    }
}

#[cfg(test)]
mod tests {
    use std::borrow::Cow;

    use super::{fold, is_x12, table::FOLDS};

    #[test]
    fn the_set_is_printable_ascii() {
        assert!(is_x12(' '));
        assert!(is_x12('~'));
        assert!(is_x12('a'));
        assert!(!is_x12('\u{7F}'));
        assert!(!is_x12('\t'));
        assert!(!is_x12('é'));
    }

    #[test]
    fn text_already_in_the_set_is_borrowed() {
        assert!(matches!(fold("ACME DC #4 [dock] {a|b}"), Cow::Borrowed(_)));
    }

    #[test]
    fn letters_lose_their_accents() {
        assert_eq!(fold("Peñasco"), "Penasco");
        assert_eq!(fold("Société Générale"), "Societe Generale");
        assert_eq!(fold("Straße"), "Strasse");
        assert_eq!(fold("ÆRØ Łódź"), "AERO Lodz");
        assert_eq!(fold("Þórshöfn"), "THorshofn");
        assert_eq!(fold("Œuvre Ĳssel"), "OEuvre IJssel");
        assert_eq!(fold("Nguyễn Thị Minh Khai"), "Nguyen Thi Minh Khai");
        assert_eq!(fold("Târgu Mureș"), "Targu Mures");
    }

    #[test]
    fn combining_marks_are_dropped_from_decomposed_text() {
        assert_eq!(fold("Pe\u{303}n\u{303}asco"), "Penasco");
        assert_eq!(fold("Cafe\u{301}"), "Cafe");
    }

    #[test]
    fn typographic_punctuation_becomes_plain() {
        assert_eq!(fold("Dock 3–5"), "Dock 3-5");
        assert_eq!(fold("A — B"), "A - B");
        assert_eq!(
            fold("\u{201C}Fragile\u{201D} \u{2018}up\u{2019}"),
            "\"Fragile\" 'up'"
        );
        assert_eq!(fold("½ pallet, ¾ full"), "1/2 pallet, 3/4 full");
        assert_eq!(fold("Wait…"), "Wait...");
        assert_eq!(fold("℅ Acme № 5"), "c/o Acme No 5");
        assert_eq!(fold("Brand™ ©"), "BrandTM (C)");
        assert_eq!(fold("€40"), "EUR40");
    }

    #[test]
    fn spaces_and_line_breaks_become_spaces() {
        assert_eq!(fold("1\u{A0}Main"), "1 Main");
        assert_eq!(fold("1\u{2009}Main\u{3000}St"), "1 Main St");
        assert_eq!(fold("Suite 2\r\nDock\t4"), "Suite 2  Dock 4");
    }

    #[test]
    fn full_width_forms_are_narrowed() {
        assert_eq!(fold("ＡＢＣ１２３"), "ABC123");
    }

    #[test]
    fn what_has_no_ascii_form_is_dropped() {
        assert_eq!(fold("北京 Depot"), " Depot");
        assert_eq!(fold("Fresh 🍎 Co"), "Fresh  Co");
        assert_eq!(fold("👍🏽"), "");
        assert_eq!(fold("Москва"), "");
        assert_eq!(fold("a\u{200B}b\u{AD}c\u{0}d\u{7F}"), "abcd");
        assert_eq!(fold("\u{FE0F}"), "");
    }

    #[test]
    fn folding_can_lengthen_the_text() {
        let folded = fold("Groß ½");
        assert_eq!(folded, "Gross 1/2");
        assert_eq!(folded.chars().count(), 9);
    }

    #[test]
    fn the_table_is_sorted_and_writes_only_the_set() {
        assert!(FOLDS.windows(2).all(|pair| pair[0].0 < pair[1].0));
        for (from, to) in FOLDS {
            assert!(!is_x12(*from), "{from:?} is already in the set");
            assert!(to.chars().all(is_x12), "{from:?} folds to {to:?}");
            assert!(!to.is_empty(), "{from:?}");
        }
    }
}
