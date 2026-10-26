//! The tokens the editor colours, so no lexer lives in TypeScript.

use serde::Serialize;
use trenova_module_kit::span::{Range, Span, Utf16Index};

use super::parser::Parsed;
use super::token::{Tok, is_reserved};

/// One coloured range.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct Highlight {
    /// Where it is, in UTF-16 code units.
    pub range: Range,
    /// `keyword`, `name`, `type`, `x12`, `loop`, `number`, `text`, `date`, `operator`,
    /// `punctuation`, `comment` or `doc`.
    pub kind: &'static str,
}

/// The highlights of a parsed file, in order.
#[must_use]
pub fn highlights(source: &str, parsed: &Parsed) -> Vec<Highlight> {
    let index = Utf16Index::new(source);
    let mut spans: Vec<(Span, &'static str)> = Vec::with_capacity(parsed.tokens.len());
    let keywords: std::collections::BTreeSet<u32> =
        parsed.file.keywords.iter().map(|span| span.start).collect();
    for token in &parsed.tokens {
        let text = source
            .get(token.span.start as usize..token.span.end as usize)
            .unwrap_or_default();
        let kind = match token.kind {
            Tok::Name if keywords.contains(&token.span.start) || is_reserved(text) => "keyword",
            Tok::Name => "name",
            Tok::TypeName => "type",
            Tok::X12 => "x12",
            Tok::Digits => "loop",
            Tok::Number | Tok::Money => "number",
            Tok::Text | Tok::RawText => "text",
            Tok::Date | Tok::Time | Tok::Instant => "date",
            Tok::LArrow
            | Tok::RArrow
            | Tok::BiArrow
            | Tok::FatArrow
            | Tok::Assign
            | Tok::EqEq
            | Tok::NotEq
            | Tok::Lt
            | Tok::Le
            | Tok::Gt
            | Tok::Ge
            | Tok::Plus
            | Tok::Minus
            | Tok::Star
            | Tok::Slash
            | Tok::Percent
            | Tok::QuestionQuestion
            | Tok::Question
            | Tok::Pipe
            | Tok::DotDot => "operator",
            Tok::LBrace
            | Tok::RBrace
            | Tok::LParen
            | Tok::RParen
            | Tok::LBracket
            | Tok::RBracket
            | Tok::Comma
            | Tok::Semi
            | Tok::Dot
            | Tok::Colon => "punctuation",
            Tok::Newline | Tok::Error | Tok::Eof => continue,
        };
        spans.push((token.span, kind));
    }
    for comment in &parsed.comments {
        spans.push((comment.span, if comment.doc { "doc" } else { "comment" }));
    }
    spans.sort_by_key(|(span, _)| span.start);
    spans
        .into_iter()
        .map(|(span, kind)| Highlight {
            range: index.range(span),
            kind,
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::highlights;
    use crate::lading::syntax::parser::parse;

    #[test]
    fn keywords_names_and_x12_are_told_apart() {
        let source = "map m\n  from x12.004010.204\n  to loadTender\n\nkey <- B204 // the load\n";
        let parsed = parse(source);
        let kinds: Vec<&str> = highlights(source, &parsed).iter().map(|h| h.kind).collect();
        assert_eq!(kinds[0], "keyword");
        assert!(kinds.contains(&"x12"));
        assert!(kinds.contains(&"comment"));
        assert!(kinds.contains(&"operator"));
    }
}
