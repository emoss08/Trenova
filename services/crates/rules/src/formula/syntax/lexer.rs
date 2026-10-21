//! The tokens, as expr-lang's lexer reads them.
//!
//! Numbers take `_` separators, hexadecimal (`0x`), octal (`0o`) and binary (`0b`) forms, and a
//! trailing `.` (`1.`) that is not the start of a `..` range. Quoted strings take backslash
//! escapes and end at a newline when unclosed; backtick strings take none and span lines, a
//! doubled backtick standing for one. Identifiers are letters in any script, decimal digits and
//! `_`. Comments (`//` to the end of the line, `/* ... */`) and spaces produce no token; a
//! comment's span is kept so the formatter can refuse to drop it.
//!
//! The lexer never fails: a character the language has no use for becomes an error token with a
//! diagnostic, and an unclosed string or comment is reported and taken to the end it reached.

use super::ast::SyntaxCode;
use super::ast::SyntaxDiagnostic;
use super::ast::Token;
use super::ast::TokenKind;
use super::unicode;
use crate::message;
use crate::message::Arg;
use crate::message::Message;
use crate::span::Span;

/// The words the grammar reserves, besides the word operators.
pub const KEYWORDS: &[&str] = &[
    "true", "false", "nil", "in", "not", "and", "or", "let", "if", "else",
];

/// The word operators every formula has.
pub const WORD_OPERATORS: &[&str] = &["startsWith", "endsWith", "contains", "matches"];

/// What the lexer read.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Tokenized {
    /// The tokens, ending with one end-of-text token.
    pub tokens: Vec<Token>,
    /// Unclosed strings and comments, unknown characters and escapes.
    pub diagnostics: Vec<SyntaxDiagnostic>,
    /// Where the comments are.
    pub comments: Vec<Span>,
}

/// Whether a character may start an identifier.
#[must_use]
pub fn is_identifier_start(character: char) -> bool {
    character == '_' || is_letter(character)
}

/// Whether a character may continue an identifier.
#[must_use]
pub fn is_identifier_continue(character: char) -> bool {
    character == '_' || is_letter(character) || is_digit(character)
}

fn is_letter(character: char) -> bool {
    if character.is_ascii() {
        return character.is_ascii_alphabetic();
    }
    in_table(unicode::LETTERS, character)
}

fn is_digit(character: char) -> bool {
    if character.is_ascii() {
        return character.is_ascii_digit();
    }
    in_table(unicode::DIGITS, character)
}

fn in_table(table: &[(u32, u32)], character: char) -> bool {
    let code = u32::from(character);
    let index = table.partition_point(|(start, _)| *start <= code);
    index > 0 && table.get(index - 1).is_some_and(|(_, end)| code <= *end)
}

/// Whether a word is a keyword or one of the given word operators.
#[must_use]
pub fn is_keyword(word: &str, operators: &[&str]) -> bool {
    KEYWORDS.contains(&word) || WORD_OPERATORS.contains(&word) || operators.contains(&word)
}

struct Lexer<'a> {
    source: &'a str,
    position: usize,
    operators: &'a [&'a str],
    tokens: Vec<Token>,
    diagnostics: Vec<SyntaxDiagnostic>,
    comments: Vec<Span>,
}

fn span(start: usize, end: usize) -> Span {
    Span::new(
        u32::try_from(start).unwrap_or(u32::MAX),
        u32::try_from(end).unwrap_or(u32::MAX),
    )
}

/// Splits a formula into tokens. `operators` are the word operators the schema publishes beyond
/// the built-in four.
#[must_use]
pub fn tokenize(source: &str, operators: &[&str]) -> Tokenized {
    let mut lexer = Lexer {
        source,
        position: 0,
        operators,
        tokens: Vec::new(),
        diagnostics: Vec::new(),
        comments: Vec::new(),
    };
    lexer.run();
    let end = span(source.len(), source.len());
    lexer.tokens.push(Token {
        kind: TokenKind::Eof,
        span: end,
    });
    Tokenized {
        tokens: lexer.tokens,
        diagnostics: lexer.diagnostics,
        comments: lexer.comments,
    }
}

impl Lexer<'_> {
    fn rest(&self) -> &str {
        self.source.get(self.position..).unwrap_or_default()
    }

    fn peek(&self) -> Option<char> {
        self.rest().chars().next()
    }

    fn peek_second(&self) -> Option<char> {
        self.rest().chars().nth(1)
    }

    fn bump(&mut self) -> Option<char> {
        let character = self.peek()?;
        self.position += character.len_utf8();
        Some(character)
    }

    fn eat_while(&mut self, accept: impl Fn(char) -> bool) {
        while let Some(character) = self.peek() {
            if !accept(character) {
                break;
            }
            self.position += character.len_utf8();
        }
    }

    fn push(&mut self, kind: TokenKind, start: usize) {
        self.tokens.push(Token {
            kind,
            span: span(start, self.position),
        });
    }

    fn report(&mut self, start: usize, end: usize, message: Message) {
        self.diagnostics.push(SyntaxDiagnostic {
            span: span(start, end),
            code: SyntaxCode::Syntax,
            message,
            fix: None,
        });
    }

    fn run(&mut self) {
        while let Some(character) = self.peek() {
            let start = self.position;
            if character.is_whitespace() {
                self.eat_while(char::is_whitespace);
            } else if self.rest().starts_with("//") {
                self.eat_while(|next| next != '\n');
                self.comments.push(span(start, self.position));
            } else if self.rest().starts_with("/*") {
                self.block_comment(start);
            } else if character.is_ascii_digit()
                || (character == '.' && self.peek_second().is_some_and(|c| c.is_ascii_digit()))
            {
                self.number(start);
            } else if character == '"' || character == '\'' {
                self.quoted(start, character);
            } else if character == '`' {
                self.raw(start);
            } else if is_identifier_start(character) {
                self.eat_while(is_identifier_continue);
                let word = self.source.get(start..self.position).unwrap_or_default();
                let kind = if is_keyword(word, self.operators) {
                    TokenKind::Keyword
                } else {
                    TokenKind::Ident
                };
                self.push(kind, start);
            } else if let Some(length) = operator_length(self.rest()) {
                self.position += length;
                self.push(TokenKind::Operator, start);
            } else if "(),;[]{}.".contains(character) {
                self.position += 1;
                self.push(TokenKind::Punct, start);
            } else if character == '#' {
                self.position += 1;
                if self.peek().is_some_and(is_identifier_start) {
                    self.eat_while(is_identifier_continue);
                }
                self.push(TokenKind::Pointer, start);
            } else {
                self.position += character.len_utf8();
                let text = self.source.get(start..self.position).unwrap_or_default();
                self.report(
                    start,
                    self.position,
                    Message::with(message!("Unexpected character '{0}'"), [Arg::from(text)]),
                );
                self.push(TokenKind::Error, start);
            }
        }
    }

    fn block_comment(&mut self, start: usize) {
        self.position += 2;
        if let Some(close) = self.rest().find("*/") {
            self.position += close + 2;
        } else {
            self.position = self.source.len();
            self.report(
                start,
                self.position,
                Message::new(message!("This comment is never closed")),
            );
        }
        self.comments.push(span(start, self.position));
    }

    fn number(&mut self, start: usize) {
        let digits = |c: char| c.is_ascii_digit() || c == '_';
        if self.peek() == Some('.') {
            self.position += 1;
            self.eat_while(digits);
            self.exponent();
            self.push(TokenKind::Number, start);
            return;
        }
        let rest = self.rest().as_bytes();
        if rest.len() > 2 && rest[0] == b'0' {
            let accept: Option<fn(char) -> bool> = match rest[1] {
                b'x' | b'X' => Some(|c| c.is_ascii_hexdigit() || c == '_'),
                b'o' | b'O' => Some(|c| matches!(c, '0'..='7' | '_')),
                b'b' | b'B' => Some(|c| matches!(c, '0' | '1' | '_')),
                _ => None,
            };
            if let Some(accept) = accept
                && char::from(rest[2]).is_ascii()
                && accept(char::from(rest[2]))
            {
                self.position += 2;
                self.eat_while(accept);
                self.push(TokenKind::Number, start);
                return;
            }
        }
        self.eat_while(digits);
        if self.peek() == Some('.') && self.peek_second() != Some('.') {
            self.position += 1;
            self.eat_while(digits);
        }
        self.exponent();
        self.push(TokenKind::Number, start);
    }

    fn exponent(&mut self) {
        let rest = self.rest().as_bytes();
        if rest.first().is_some_and(|b| *b == b'e' || *b == b'E') {
            let sign = usize::from(rest.get(1).is_some_and(|b| *b == b'+' || *b == b'-'));
            if rest
                .get(1 + sign)
                .is_some_and(|b| b.is_ascii_digit() || *b == b'_')
            {
                self.position += 1 + sign;
                self.eat_while(|c| c.is_ascii_digit() || c == '_');
            }
        }
    }

    fn quoted(&mut self, start: usize, quote: char) {
        self.position += 1;
        let mut closed = false;
        while let Some(character) = self.peek() {
            if character == quote {
                self.position += 1;
                closed = true;
                break;
            }
            if character == '\n' {
                break;
            }
            if character == '\\' {
                match self.peek_second() {
                    Some(next) if !is_line_terminator(next) => {
                        self.position += 1 + next.len_utf8();
                    }
                    _ => break,
                }
                continue;
            }
            self.bump();
        }
        if !closed {
            self.report(
                start,
                self.position,
                Message::new(message!("This text is never closed")),
            );
        }
        self.push(TokenKind::String, start);
        let text = self.source.get(start..self.position).unwrap_or_default();
        for (from, to, escape) in invalid_escapes(text) {
            self.report(
                start + from,
                start + to,
                Message::with(
                    message!("'{0}' is not an escape the formula language knows"),
                    [Arg::from(escape)],
                ),
            );
        }
    }

    fn raw(&mut self, start: usize) {
        self.position += 1;
        let mut closed = false;
        while let Some(character) = self.bump() {
            if character == '`' {
                if self.peek() == Some('`') {
                    self.position += 1;
                    continue;
                }
                closed = true;
                break;
            }
        }
        if !closed {
            self.report(
                start,
                self.position,
                Message::new(message!("This text is never closed")),
            );
        }
        self.push(TokenKind::String, start);
    }
}

fn is_line_terminator(character: char) -> bool {
    matches!(character, '\n' | '\r' | '\u{2028}' | '\u{2029}')
}

/// The length of the operator at the start of `text`, longest first.
fn operator_length(text: &str) -> Option<usize> {
    const DOUBLE: &[&str] = &["??", "?.", "==", "!=", "<=", ">=", "&&", "||", "**", ".."];
    if DOUBLE.iter().any(|op| text.starts_with(op)) {
        return Some(2);
    }
    let first = text.chars().next()?;
    "-+*/%<>!?:^=|".contains(first).then_some(1)
}

/// The body of a string token: its quotes dropped, and whether it was closed.
fn body(text: &str) -> (char, &str, bool) {
    let mut characters = text.chars();
    let quote = characters.next().unwrap_or('"');
    let inner = characters.as_str();
    let closed = text.len() >= 2 && inner.ends_with(quote) && string_closed(text, quote);
    let inner = if closed {
        inner
            .get(..inner.len() - quote.len_utf8())
            .unwrap_or_default()
    } else {
        inner
    };
    (quote, inner, closed)
}

fn string_closed(text: &str, quote: char) -> bool {
    let bytes = text.as_bytes();
    if quote == '`' {
        let run = bytes
            .iter()
            .skip(1)
            .rev()
            .take_while(|byte| **byte == b'`')
            .count();
        return run % 2 == 1;
    }
    let backslashes = bytes
        .iter()
        .take(bytes.len().saturating_sub(1))
        .skip(1)
        .rev()
        .take_while(|byte| **byte == b'\\')
        .count();
    backslashes % 2 == 0
}

/// A string literal's value: escapes read in a quoted string, a doubled backtick in a raw one.
///
/// An escape the language does not know is kept as written; the lexer has reported it.
#[must_use]
pub fn string_value(text: &str) -> String {
    let (quote, inner, _) = body(text);
    if quote == '`' {
        return inner.replace("``", "`");
    }
    let normalised = inner.replace("\r\n", "\n").replace('\r', "\n");
    let mut out = String::with_capacity(normalised.len());
    let mut rest = normalised.as_str();
    while let Some(slash) = rest.find('\\') {
        out.push_str(rest.get(..slash).unwrap_or_default());
        let escape = rest.get(slash..).unwrap_or_default();
        if let Some((character, length)) = read_escape(escape) {
            out.push(character);
            rest = escape.get(length..).unwrap_or_default();
        } else {
            let length = escape
                .chars()
                .take(2)
                .map(char::len_utf8)
                .sum::<usize>()
                .max(1);
            out.push_str(escape.get(..length).unwrap_or(escape));
            rest = escape.get(length..).unwrap_or_default();
        }
    }
    out.push_str(rest);
    out
}

/// Reads one escape at the start of `text` (which starts with a backslash): the character it
/// stands for and how many bytes it takes, or `None` when the language does not know it.
fn read_escape(text: &str) -> Option<(char, usize)> {
    let bytes = text.as_bytes();
    let kind = *bytes.get(1)?;
    let simple = match kind {
        b'a' => Some('\u{7}'),
        b'b' => Some('\u{8}'),
        b'f' => Some('\u{c}'),
        b'n' => Some('\n'),
        b'r' => Some('\r'),
        b't' => Some('\t'),
        b'v' => Some('\u{b}'),
        b'\\' => Some('\\'),
        b'\'' => Some('\''),
        b'"' => Some('"'),
        b'`' => Some('`'),
        b'?' => Some('?'),
        _ => None,
    };
    if let Some(character) = simple {
        return Some((character, 2));
    }
    match kind {
        b'u' if bytes.get(2) == Some(&b'{') => {
            let close = text.get(3..)?.find('}')? + 3;
            let digits = text.get(3..close)?;
            if digits.is_empty() || digits.len() > 6 {
                return None;
            }
            let code = u32::from_str_radix(digits, 16).ok()?;
            Some((char::from_u32(code)?, close + 1))
        }
        b'x' | b'X' | b'u' | b'U' => {
            let count = match kind {
                b'u' => 4,
                b'U' => 8,
                _ => 2,
            };
            let digits = text.get(2..2 + count)?;
            if !digits.bytes().all(|byte| byte.is_ascii_hexdigit()) {
                return None;
            }
            let code = u32::from_str_radix(digits, 16).ok()?;
            Some((char::from_u32(code)?, 2 + count))
        }
        b'0'..=b'3' => {
            let digits = text.get(1..4)?;
            if !digits.bytes().all(|byte| matches!(byte, b'0'..=b'7')) {
                return None;
            }
            let code = u32::from_str_radix(digits, 8).ok()?;
            Some((char::from_u32(code)?, 4))
        }
        _ => None,
    }
}

/// The escapes in a quoted string token the language does not know, as byte ranges into the
/// token with their text.
fn invalid_escapes(text: &str) -> Vec<(usize, usize, String)> {
    let (quote, inner, _) = body(text);
    if quote == '`' {
        return Vec::new();
    }
    let offset = quote.len_utf8();
    let mut found = Vec::new();
    let mut index = 0;
    while let Some(slash) = inner.get(index..).and_then(|rest| rest.find('\\')) {
        let at = index + slash;
        let escape = inner.get(at..).unwrap_or_default();
        if let Some((_, length)) = read_escape(escape) {
            index = at + length;
            continue;
        }
        let length = escape
            .chars()
            .take(2)
            .map(char::len_utf8)
            .sum::<usize>()
            .max(1);
        let written = escape.get(..length).unwrap_or(escape);
        if length > 1 {
            found.push((offset + at, offset + at + length, written.to_owned()));
        }
        index = at + length;
    }
    found
}

/// A number literal's text without `_` separators or a trailing `.`, and whether it is an
/// integer literal.
#[must_use]
pub fn number_text(text: &str) -> (String, bool) {
    let digits: String = text.chars().filter(|c| *c != '_').collect();
    let lower = digits.to_ascii_lowercase();
    if lower.starts_with("0x") || lower.starts_with("0o") || lower.starts_with("0b") {
        return (lower, true);
    }
    let whole = !digits.contains(['.', 'e', 'E']);
    let digits = digits
        .strip_suffix('.')
        .map_or(digits.clone(), String::from);
    (digits, whole)
}
