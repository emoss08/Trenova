//! Cuts a Lading source into tokens.
//!
//! The lexer never fails: what it can't read becomes an [`Tok::Error`] token and a finding, and
//! the parser carries on. Comments are kept aside (the formatter keeps them, and a `///` comment
//! is the doc of what follows). Newlines are tokens, since a line ends a statement.

use trenova_module_kit::diagnostic::Diagnostic;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::token::{Comment, Tok, Token};

/// What lexing a source produced.
#[derive(Debug, Clone, Default)]
pub struct Lexed {
    /// The tokens, ending with [`Tok::Eof`].
    pub tokens: Vec<Token>,
    /// The comments, in order.
    pub comments: Vec<Comment>,
    /// What could not be read.
    pub diagnostics: Vec<Diagnostic>,
}

/// Lexes `source`.
#[must_use]
pub fn lex(source: &str) -> Lexed {
    let mut lexer = Lexer {
        bytes: source.as_bytes(),
        source,
        pos: 0,
        out: Lexed::default(),
    };
    lexer.run();
    lexer.out
}

/// Lexes the inside of an interpolation, whose text starts at `offset` in the file.
#[must_use]
pub fn lex_at(source: &str, offset: u32) -> Lexed {
    let mut lexed = lex(source);
    for token in &mut lexed.tokens {
        token.span = Span::new(token.span.start + offset, token.span.end + offset);
    }
    for comment in &mut lexed.comments {
        comment.span = Span::new(comment.span.start + offset, comment.span.end + offset);
    }
    for diagnostic in &mut lexed.diagnostics {
        diagnostic.span = Span::new(diagnostic.span.start + offset, diagnostic.span.end + offset);
    }
    lexed
}

struct Lexer<'a> {
    bytes: &'a [u8],
    source: &'a str,
    pos: usize,
    out: Lexed,
}

fn span(start: usize, end: usize) -> Span {
    Span::new(
        u32::try_from(start).unwrap_or(u32::MAX),
        u32::try_from(end).unwrap_or(u32::MAX),
    )
}

impl Lexer<'_> {
    fn peek(&self) -> u8 {
        self.bytes.get(self.pos).copied().unwrap_or(0)
    }

    fn peek_at(&self, ahead: usize) -> u8 {
        self.bytes.get(self.pos + ahead).copied().unwrap_or(0)
    }

    fn push(&mut self, kind: Tok, start: usize) {
        self.out.tokens.push(Token {
            kind,
            span: span(start, self.pos),
        });
    }

    fn last_kind(&self) -> Option<Tok> {
        self.out.tokens.last().map(|token| token.kind)
    }

    fn error(&mut self, start: usize, message: Message) {
        self.out
            .diagnostics
            .push(Diagnostic::error("syntax", span(start, self.pos), message));
        self.push(Tok::Error, start);
    }

    fn run(&mut self) {
        while self.pos < self.bytes.len() {
            let start = self.pos;
            let byte = self.peek();
            match byte {
                b' ' | b'\t' | b'\r' => self.pos += 1,
                b'\n' => {
                    self.pos += 1;
                    self.push(Tok::Newline, start);
                }
                b'/' if self.peek_at(1) == b'/' => self.comment(),
                b'"' => self.text(start),
                b'r' if self.peek_at(1) == b'"' => self.raw_text(start),
                b'@' => self.temporal(start),
                b'$' => self.money(start),
                b'0'..=b'9' => self.number(start),
                b'a'..=b'z' | b'_' => self.name(start),
                b'A'..=b'Z' => self.upper(start),
                _ if self.source[start..].starts_with('°') => self.degree(start),
                _ => self.punct(start),
            }
        }
        let end = self.bytes.len();
        self.out.tokens.push(Token {
            kind: Tok::Eof,
            span: span(end, end),
        });
    }

    fn comment(&mut self) {
        let start = self.pos;
        let doc = self.peek_at(2) == b'/' && self.peek_at(3) != b'/';
        while self.pos < self.bytes.len() && self.peek() != b'\n' {
            self.pos += 1;
        }
        self.out.comments.push(Comment {
            span: span(start, self.pos),
            doc,
        });
    }

    fn text(&mut self, start: usize) {
        self.pos += 1;
        let mut depth = 0usize;
        loop {
            match self.peek() {
                0 | b'\n' => {
                    self.error(start, Message::new(message!("This text is never closed")));
                    return;
                }
                b'\\' => self.pos += 2,
                b'{' if depth == 0 && self.peek_at(1) == b'{' => self.pos += 2,
                b'}' if depth == 0 && self.peek_at(1) == b'}' => self.pos += 2,
                b'{' => {
                    depth += 1;
                    self.pos += 1;
                }
                b'}' => {
                    depth = depth.saturating_sub(1);
                    self.pos += 1;
                }
                b'"' if depth == 0 => {
                    self.pos += 1;
                    self.push(Tok::Text, start);
                    return;
                }
                b'"' => self.skip_inner_text(),
                _ => self.pos += 1,
            }
        }
    }

    /// Skips a text inside an interpolation, so its quotes don't close the outer one.
    fn skip_inner_text(&mut self) {
        self.pos += 1;
        while !matches!(self.peek(), 0 | b'\n' | b'"') {
            if self.peek() == b'\\' {
                self.pos += 1;
            }
            self.pos += 1;
        }
        if self.peek() == b'"' {
            self.pos += 1;
        }
    }

    fn raw_text(&mut self, start: usize) {
        self.pos += 2;
        while !matches!(self.peek(), 0 | b'\n' | b'"') {
            self.pos += 1;
        }
        if self.peek() == b'"' {
            self.pos += 1;
            self.push(Tok::RawText, start);
        } else {
            self.error(start, Message::new(message!("This text is never closed")));
        }
    }

    fn digits(&mut self) -> usize {
        let start = self.pos;
        while self.peek().is_ascii_digit() {
            self.pos += 1;
        }
        self.pos - start
    }

    fn temporal(&mut self, start: usize) {
        self.pos += 1;
        let rest = &self.bytes[self.pos..];
        let is_date = rest.len() >= 10 && rest[4] == b'-' && rest[..4].iter().all(u8::is_ascii_digit);
        if is_date {
            self.pos += 10;
            if self.peek() == b'T' && self.peek_at(1).is_ascii_digit() {
                self.pos += 1;
                self.clock();
                self.zone_suffix();
                self.push(Tok::Instant, start);
            } else {
                self.push(Tok::Date, start);
            }
            return;
        }
        if self.peek().is_ascii_digit() {
            self.clock();
            self.push(Tok::Time, start);
            return;
        }
        self.error(
            start,
            Message::new(message!(
                "Expected a date (@2026-01-10), a time (@14:30) or an instant (@2026-01-10T14:30Z)"
            )),
        );
    }

    fn clock(&mut self) {
        self.digits();
        if self.peek() == b':' {
            self.pos += 1;
            self.digits();
        }
        if self.peek() == b':' {
            self.pos += 1;
            self.digits();
        }
    }

    fn zone_suffix(&mut self) {
        match self.peek() {
            b'Z' => self.pos += 1,
            b'+' | b'-' if self.peek_at(1).is_ascii_digit() => {
                self.pos += 1;
                self.digits();
                if self.peek() == b':' {
                    self.pos += 1;
                    self.digits();
                }
            }
            b'[' => {
                while !matches!(self.peek(), 0 | b'\n' | b']') {
                    self.pos += 1;
                }
                if self.peek() == b']' {
                    self.pos += 1;
                }
            }
            _ => {}
        }
    }

    fn money(&mut self, start: usize) {
        self.pos += 1;
        if !self.peek().is_ascii_digit() {
            self.error(start, Message::new(message!("Expected an amount after $")));
            return;
        }
        self.number_body();
        self.push(Tok::Money, start);
    }

    fn number_body(&mut self) {
        while self.peek().is_ascii_digit() || self.peek() == b'_' {
            self.pos += 1;
        }
        if self.peek() == b'.' && self.peek_at(1).is_ascii_digit() {
            self.pos += 1;
            while self.peek().is_ascii_digit() || self.peek() == b'_' {
                self.pos += 1;
            }
        }
    }

    fn number(&mut self, start: usize) {
        let after_dot = self.last_kind() == Some(Tok::Dot);
        let leading_zero = self.peek() == b'0' && self.peek_at(1).is_ascii_digit();
        if after_dot || leading_zero {
            self.digits();
            while self.peek().is_ascii_alphanumeric() {
                self.pos += 1;
            }
            self.push(Tok::Digits, start);
            return;
        }
        self.digits();
        if self.peek().is_ascii_uppercase() {
            while self.peek().is_ascii_alphanumeric() {
                self.pos += 1;
            }
            self.push(Tok::Digits, start);
            return;
        }
        self.pos = start;
        self.number_body();
        self.push(Tok::Number, start);
    }

    fn word_end(&mut self) {
        while self.peek().is_ascii_alphanumeric() || self.peek() == b'_' {
            self.pos += 1;
        }
    }

    fn name(&mut self, start: usize) {
        self.word_end();
        self.push(Tok::Name, start);
    }

    fn upper(&mut self, start: usize) {
        self.word_end();
        let word = &self.source[start..self.pos];
        let kind = if word.bytes().any(|byte| byte.is_ascii_lowercase()) {
            Tok::TypeName
        } else {
            Tok::X12
        };
        self.push(kind, start);
    }

    fn degree(&mut self, start: usize) {
        self.pos += '°'.len_utf8();
        if matches!(self.peek(), b'F' | b'C') {
            self.pos += 1;
            self.push(Tok::Name, start);
        } else {
            self.error(start, Message::new(message!("Expected °F or °C")));
        }
    }

    fn punct(&mut self, start: usize) {
        let pairs: &[(&str, Tok)] = &[
            ("<->", Tok::BiArrow),
            ("<-", Tok::LArrow),
            ("->", Tok::RArrow),
            ("=>", Tok::FatArrow),
            ("==", Tok::EqEq),
            ("!=", Tok::NotEq),
            ("<=", Tok::Le),
            (">=", Tok::Ge),
            ("??", Tok::QuestionQuestion),
            ("..", Tok::DotDot),
            ("{", Tok::LBrace),
            ("}", Tok::RBrace),
            ("(", Tok::LParen),
            (")", Tok::RParen),
            ("[", Tok::LBracket),
            ("]", Tok::RBracket),
            (",", Tok::Comma),
            (";", Tok::Semi),
            (".", Tok::Dot),
            (":", Tok::Colon),
            ("=", Tok::Assign),
            ("<", Tok::Lt),
            (">", Tok::Gt),
            ("+", Tok::Plus),
            ("-", Tok::Minus),
            ("*", Tok::Star),
            ("/", Tok::Slash),
            ("%", Tok::Percent),
            ("?", Tok::Question),
            ("|", Tok::Pipe),
        ];
        let rest = &self.source[start..];
        for (text, kind) in pairs {
            if rest.starts_with(text) {
                self.pos += text.len();
                self.push(*kind, start);
                return;
            }
        }
        let width = rest.chars().next().map_or(1, char::len_utf8);
        self.pos += width;
        let character = rest.chars().next().unwrap_or(' ').to_string();
        self.error(
            start,
            Message::with(message!("Unexpected character {0}"), [Arg::from(character)]),
        );
    }
}

/// The parts of a text token: literal pieces and interpolations.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TextPiece {
    /// Literal text, escapes decoded.
    Literal(String),
    /// An interpolation: the source inside the braces, and where it starts in the file.
    Interpolation(String, u32),
}

/// Splits a text token's source (quotes included) into pieces.
#[must_use]
pub fn text_pieces(token_source: &str, token_start: u32) -> Vec<TextPiece> {
    let inner = token_source
        .strip_prefix('"')
        .and_then(|rest| rest.strip_suffix('"'))
        .unwrap_or(token_source);
    let base = token_start + 1;
    let mut pieces = Vec::new();
    let mut literal = String::new();
    let mut chars = inner.char_indices().peekable();
    while let Some((offset, character)) = chars.next() {
        match character {
            '\\' => {
                if let Some((_, escaped)) = chars.next() {
                    match escaped {
                        'n' => literal.push('\n'),
                        't' => literal.push('\t'),
                        'u' => {
                            let mut hex = String::new();
                            if chars.peek().map(|(_, c)| *c) == Some('{') {
                                chars.next();
                                for (_, c) in chars.by_ref() {
                                    if c == '}' {
                                        break;
                                    }
                                    hex.push(c);
                                }
                            }
                            if let Some(decoded) =
                                u32::from_str_radix(&hex, 16).ok().and_then(char::from_u32)
                            {
                                literal.push(decoded);
                            }
                        }
                        other => literal.push(other),
                    }
                }
            }
            '{' if chars.peek().map(|(_, c)| *c) == Some('{') => {
                chars.next();
                literal.push('{');
            }
            '}' if chars.peek().map(|(_, c)| *c) == Some('}') => {
                chars.next();
                literal.push('}');
            }
            '{' => {
                if !literal.is_empty() {
                    pieces.push(TextPiece::Literal(std::mem::take(&mut literal)));
                }
                let start = offset + 1;
                let mut depth = 1usize;
                let mut end = start;
                let mut in_text = false;
                for (inner_offset, inner_char) in chars.by_ref() {
                    end = inner_offset;
                    match inner_char {
                        '"' => in_text = !in_text,
                        '{' if !in_text => depth += 1,
                        '}' if !in_text => {
                            depth -= 1;
                            if depth == 0 {
                                break;
                            }
                        }
                        _ => {}
                    }
                }
                let source = inner.get(start..end).unwrap_or_default().to_owned();
                pieces.push(TextPiece::Interpolation(
                    source,
                    base + u32::try_from(start).unwrap_or(0),
                ));
            }
            other => literal.push(other),
        }
    }
    if !literal.is_empty() || pieces.is_empty() {
        pieces.push(TextPiece::Literal(literal));
    }
    pieces
}

#[cfg(test)]
mod tests {
    use super::{TextPiece, lex, text_pieces};
    use crate::lading::syntax::token::Tok;

    fn kinds(source: &str) -> Vec<Tok> {
        lex(source).tokens.iter().map(|token| token.kind).collect()
    }

    #[test]
    fn x12_paths_lex_as_references_and_loop_ids() {
        assert_eq!(
            kinds("stop.0310.N104"),
            vec![Tok::Name, Tok::Dot, Tok::Digits, Tok::Dot, Tok::X12, Tok::Eof]
        );
        assert_eq!(kinds("each 0300"), vec![Tok::Name, Tok::Digits, Tok::Eof]);
        assert_eq!(kinds("seg.02"), vec![Tok::Name, Tok::Dot, Tok::Digits, Tok::Eof]);
        assert_eq!(kinds("2000A"), vec![Tok::Digits, Tok::Eof]);
    }

    #[test]
    fn numbers_ranges_and_quantities() {
        assert_eq!(
            kinds("5..9"),
            vec![Tok::Number, Tok::DotDot, Tok::Number, Tok::Eof]
        );
        assert_eq!(kinds("45_000 lb"), vec![Tok::Number, Tok::Name, Tok::Eof]);
        assert_eq!(kinds("12.50"), vec![Tok::Number, Tok::Eof]);
        assert_eq!(kinds("-10 °F"), vec![Tok::Minus, Tok::Number, Tok::Name, Tok::Eof]);
        assert_eq!(kinds("$1_850.00"), vec![Tok::Money, Tok::Eof]);
        assert_eq!(kinds("5%"), vec![Tok::Number, Tok::Percent, Tok::Eof]);
    }

    #[test]
    fn temporal_literals() {
        assert_eq!(kinds("@2026-01-10"), vec![Tok::Date, Tok::Eof]);
        assert_eq!(kinds("@14:30"), vec![Tok::Time, Tok::Eof]);
        assert_eq!(kinds("@2026-01-10T14:30Z"), vec![Tok::Instant, Tok::Eof]);
        assert_eq!(
            kinds("@2026-01-10T08:30[America/Chicago]"),
            vec![Tok::Instant, Tok::Eof]
        );
        assert_eq!(kinds("@2026-01-10T08:30-06:00"), vec![Tok::Instant, Tok::Eof]);
    }

    #[test]
    fn names_types_and_x12() {
        assert_eq!(
            kinds("stopTypes StopType B2A01 ZZ"),
            vec![Tok::Name, Tok::TypeName, Tok::X12, Tok::X12, Tok::Eof]
        );
    }

    #[test]
    fn arrows_and_operators() {
        assert_eq!(
            kinds("a <- b -> c <-> d => e ?? f"),
            vec![
                Tok::Name,
                Tok::LArrow,
                Tok::Name,
                Tok::RArrow,
                Tok::Name,
                Tok::BiArrow,
                Tok::Name,
                Tok::FatArrow,
                Tok::Name,
                Tok::QuestionQuestion,
                Tok::Name,
                Tok::Eof
            ]
        );
    }

    #[test]
    fn texts_keep_interpolations_whole() {
        let lexed = lex(r#""Over {partner.maxWeight} and {f("x")}" rest"#);
        assert_eq!(lexed.tokens[0].kind, Tok::Text);
        assert_eq!(lexed.tokens[1].kind, Tok::Name);
        let source = r#""Stop {stop.S501} {{x}}""#;
        let pieces = text_pieces(source, 0);
        assert_eq!(
            pieces,
            vec![
                TextPiece::Literal("Stop ".into()),
                TextPiece::Interpolation("stop.S501".into(), 7),
                TextPiece::Literal(" {x}".into()),
            ]
        );
    }

    #[test]
    fn comments_are_kept_aside() {
        let lexed = lex("/// doc\nx // note\n");
        assert_eq!(lexed.comments.len(), 2);
        assert!(lexed.comments[0].doc);
        assert!(!lexed.comments[1].doc);
        assert_eq!(
            lexed.tokens.iter().map(|t| t.kind).collect::<Vec<_>>(),
            vec![Tok::Newline, Tok::Name, Tok::Newline, Tok::Eof]
        );
    }

    #[test]
    fn unknown_characters_are_errors_not_panics() {
        let lexed = lex("a # b ~ \"open");
        assert!(!lexed.diagnostics.is_empty());
        assert_eq!(lexed.tokens.last().map(|t| t.kind), Some(Tok::Eof));
    }
}
