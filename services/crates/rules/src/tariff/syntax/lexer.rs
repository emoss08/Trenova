//! The lexer: source text to [`Token`]s, following the reference's "Lexical structure" and
//! Appendix A.
//!
//! - Identifiers are ASCII: a letter, then letters, digits and `_`. A word is a keyword, or is
//!   classified by its case: `camelCase` is a [`TokenKind::Name`], `PascalCase` a
//!   [`TokenKind::TypeName`], and a word with no lower-case letter and at least two capitals
//!   (`FSC`, `DIESEL_BASE`) a [`TokenKind::Code`]. A lone `_` is the wildcard.
//! - The words that mean something only in one place (`as`, `cents`, and the `round` clause's
//!   `to`, `up`, `down`, `half` and `even`) are names here; the parser marks them keywords where
//!   the grammar uses them.
//! - A unit is a [`TokenKind::Unit`] directly after a number (`500 lb`, `-5 °C`, `30 lb/ft3`) or
//!   after `$x/` (`$2.35/mi`). `in` after a number is the unit inches unless what follows on the
//!   line starts a value (`48 in` against `5 in [5, 6]`). `°F` and `°C` are units wherever they
//!   are written. `$` directly followed by `/` and a unit is the money unit (`.in($/cwt)`), a
//!   unit token followed by the denominators. The parser marks the units of a type or of
//!   `.in(…)`.
//! - `@` starts a date (`@2026-10-22`), a date-time (`@2026-10-22T08:00-05:00`) or a time of
//!   day (`@17:30`), all [`TokenKind::Date`]; [`parse_date_literal`] tells them apart.
//! - Text is split into literal runs ([`TokenKind::Text`], the first holding the opening quote and
//!   the last the closing one), escapes and interpolations, whose expressions are lexed like any
//!   other code between two [`TokenKind::Interpolation`] braces. Text ends at the end of its line.
//! - Every newline is a [`TokenKind::Newline`]; the parser decides which end a statement.
//!
//! The lexer never fails: what it cannot read is an [`TokenKind::Error`] token with a diagnostic,
//! and every byte of the source is inside a token or is whitespace.

use super::token::Token;
use super::token::TokenKind;
use trenova_module_kit::span::Span;
use trenova_module_kit::diagnostic::Change;
use trenova_module_kit::diagnostic::Diagnostic;
use trenova_module_kit::diagnostic::Fix;
use crate::tariff::syntax::ast::DateLit;
use crate::tariff::syntax::ast::DateTimeLit;
use crate::tariff::syntax::ast::TimeLit;
use crate::tariff::units;
use rust_decimal::Decimal;
use trenova_module_kit::message;
use trenova_module_kit::message::Arg;
use trenova_module_kit::message::Message;

/// The largest source the lexer reads, in bytes.
pub const MAX_SOURCE_BYTES: usize = 256 * 1024;

/// The most tokens one source may have.
pub const MAX_TOKENS: usize = 50_000;

/// How deeply text may be interpolated inside interpolated text.
pub const MAX_TEXT_NESTING: usize = 8;

/// The words the grammar reserves. None of them is ever a name, except after `.`/`?.` and before
/// `:` in an argument, a parameter's type, or a record field.
pub const KEYWORDS: &[&str] = &[
    "tariff",
    "library",
    "condition",
    "use",
    "input",
    "let",
    "var",
    "const",
    "fn",
    "type",
    "enum",
    "private",
    "charge",
    "note",
    "if",
    "else",
    "match",
    "for",
    "in",
    "where",
    "break",
    "continue",
    "return",
    "minimum",
    "maximum",
    "round",
    "require",
    "fail",
    "fails",
    "test",
    "given",
    "expect",
    "and",
    "or",
    "not",
    "true",
    "false",
    "none",
];

/// The words that are keywords only where the grammar uses them, and names everywhere else.
pub const CONTEXTUAL: &[&str] = &[
    "as", "cents", "to", "up", "down", "half", "even", "on", "freight",
];

/// What the lexer read.
#[derive(Debug, Clone, PartialEq)]
pub struct Lexed {
    /// The tokens, newlines and comments included, ending with one end-of-file token.
    pub tokens: Vec<Token>,
    /// What could not be read.
    pub diagnostics: Vec<Diagnostic>,
}

/// Whether `word` is a reserved keyword.
#[must_use]
pub fn is_keyword(word: &str) -> bool {
    KEYWORDS.contains(&word)
}

/// Whether `word` is a unit name the language knows (`lb`, `ft3`, `°F`).
#[must_use]
pub fn is_unit(word: &str) -> bool {
    word != "$" && word != "%" && units::simple(word).is_some()
}

/// The class of an identifier by its case.
#[must_use]
pub fn classify(word: &str) -> TokenKind {
    if is_keyword(word) {
        return TokenKind::Keyword;
    }
    let first = word.chars().next().unwrap_or('a');
    if first.is_ascii_lowercase() {
        return TokenKind::Name;
    }
    let lower = word.chars().any(|c| c.is_ascii_lowercase());
    let capitals = word.chars().filter(char::is_ascii_uppercase).count();
    if !lower && capitals >= 2 {
        TokenKind::Code
    } else {
        TokenKind::TypeName
    }
}

fn span(start: usize, end: usize) -> Span {
    Span::new(
        u32::try_from(start).unwrap_or(u32::MAX),
        u32::try_from(end).unwrap_or(u32::MAX),
    )
}

/// What the lexer is reading.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Mode {
    /// Code inside an interpolation, with how many braces are open inside it.
    Interpolation(usize),
    /// Text; `true` until its first token is written (which then holds the opening quote).
    Text { first: bool, start: usize },
}

struct Lexer<'a> {
    source: &'a str,
    position: usize,
    tokens: Vec<Token>,
    diagnostics: Vec<Diagnostic>,
    modes: Vec<Mode>,
    line_has_code: bool,
    stopped: bool,
}

/// Splits `source` into tokens.
#[must_use]
pub fn lex(source: &str) -> Lexed {
    let mut lexer = Lexer {
        source,
        position: 0,
        tokens: Vec::new(),
        diagnostics: Vec::new(),
        modes: Vec::new(),
        line_has_code: false,
        stopped: false,
    };
    if source.len() > MAX_SOURCE_BYTES {
        lexer.diagnostics.push(Diagnostic::error(
            "limit",
            span(0, 0),
            Message::with(
                message!("This file is larger than {0} KiB, the most a file can be"),
                [Arg::from(MAX_SOURCE_BYTES / 1024)],
            ),
        ));
        lexer.position = source.len();
        lexer.push(TokenKind::Error, 0);
    } else {
        lexer.run();
    }
    let end = span(source.len(), source.len());
    lexer.tokens.push(Token {
        kind: TokenKind::Eof,
        span: end,
    });
    Lexed {
        tokens: lexer.tokens,
        diagnostics: lexer.diagnostics,
    }
}

impl Lexer<'_> {
    fn rest(&self) -> &str {
        self.source.get(self.position..).unwrap_or_default()
    }

    fn peek(&self) -> Option<char> {
        self.rest().chars().next()
    }

    fn peek_at(&self, offset: usize) -> Option<char> {
        self.source.get(self.position + offset..)?.chars().next()
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
        if !matches!(
            kind,
            TokenKind::Newline | TokenKind::Comment | TokenKind::DocComment
        ) {
            self.line_has_code = true;
        }
        self.tokens.push(Token {
            kind,
            span: span(start, self.position),
        });
    }

    fn error(&mut self, start: usize, end: usize, message: Message) {
        self.diagnostics
            .push(Diagnostic::error("syntax", span(start, end), message));
    }

    fn run(&mut self) {
        while self.position < self.source.len() && !self.stopped {
            if self.tokens.len() >= MAX_TOKENS {
                self.stop_at_limit();
                break;
            }
            match self.modes.last().copied() {
                Some(Mode::Text { first, start }) => self.text(first, start),
                Some(Mode::Interpolation(depth)) => self.code(Some(depth)),
                None => self.code(None),
            }
        }
        if !self.stopped {
            self.close_open_text();
        }
    }

    fn stop_at_limit(&mut self) {
        self.stopped = true;
        let start = self.position;
        self.diagnostics.push(Diagnostic::error(
            "limit",
            span(start, start),
            Message::with(
                message!("This file has more than {0} tokens, the most a file can have"),
                [Arg::from(MAX_TOKENS)],
            ),
        ));
        self.position = self.source.len();
        self.push(TokenKind::Error, start);
    }

    /// Reports and leaves every text still open, at the end of a line or of the file.
    fn close_open_text(&mut self) {
        let mut reported = false;
        while let Some(mode) = self.modes.pop() {
            if let Mode::Text { start, .. } = mode
                && !reported
            {
                reported = true;
                let at = self.position;
                self.diagnostics.push(
                    Diagnostic::error(
                        "syntax",
                        span(start, at),
                        Message::new(message!("This text is missing its closing quote")),
                    )
                    .with_fix(Fix {
                        label: Message::new(message!("Close the text")),
                        changes: vec![Change {
                            span: span(at, at),
                            insert: String::from("\""),
                        }],
                    }),
                );
            } else if let Mode::Interpolation(_) = mode
                && !reported
            {
                reported = true;
                let at = self.position;
                self.diagnostics.push(Diagnostic::error(
                    "syntax",
                    span(at, at),
                    Message::new(message!("This interpolation is missing its closing brace")),
                ));
            }
        }
    }

    fn code(&mut self, interpolation: Option<usize>) {
        let Some(character) = self.peek() else {
            return;
        };
        let start = self.position;
        match character {
            '\n' => {
                if !self.modes.is_empty() {
                    self.close_open_text();
                }
                self.position += 1;
                self.push(TokenKind::Newline, start);
                self.line_has_code = false;
            }
            c if c.is_whitespace() => self.eat_while(|c| c != '\n' && c.is_whitespace()),
            '/' if self.rest().starts_with("//") => self.comment(start),
            '0'..='9' => self.number_literal(start),
            '$' => self.money(start),
            '@' => self.date(start),
            '"' => self.open_text(start),
            '°' => self.degrees(start),
            c if c.is_alphabetic() || c == '_' => self.word(start),
            '{' if interpolation.is_some() => {
                self.set_depth(interpolation.unwrap_or(0) + 1);
                self.position += 1;
                self.push(TokenKind::Punct, start);
            }
            '}' if interpolation == Some(0) => {
                self.modes.pop();
                self.position += 1;
                self.push(TokenKind::Interpolation, start);
            }
            '}' if interpolation.is_some() => {
                self.set_depth(interpolation.unwrap_or(1).saturating_sub(1));
                self.position += 1;
                self.push(TokenKind::Punct, start);
            }
            _ => self.punct(start),
        }
    }

    fn set_depth(&mut self, depth: usize) {
        if let Some(Mode::Interpolation(current)) = self.modes.last_mut() {
            *current = depth;
        }
    }

    fn comment(&mut self, start: usize) {
        let rest = self.rest();
        let doc = rest.starts_with("///") && !rest.starts_with("////") && !self.line_has_code;
        self.eat_while(|c| c != '\n');
        let kind = if doc {
            TokenKind::DocComment
        } else {
            TokenKind::Comment
        };
        self.push(kind, start);
        if !self.modes.is_empty() {
            self.close_open_text();
        }
    }

    fn word(&mut self, start: usize) {
        self.eat_while(|c| c.is_alphanumeric() || c == '_');
        let word = self.source.get(start..self.position).unwrap_or_default();
        if !word.is_ascii() {
            self.error(
                start,
                self.position,
                Message::with(
                    message!("`{0}` can't be a name: names use the letters a to z, digits and `_`"),
                    [Arg::from(word)],
                ),
            );
            self.push(TokenKind::Error, start);
            return;
        }
        if word == "_" {
            self.push(TokenKind::Punct, start);
            return;
        }
        if word.starts_with('_') {
            self.error(
                start,
                self.position,
                Message::new(message!("A name starts with a letter, not `_`")),
            );
            self.push(TokenKind::Error, start);
            return;
        }
        let kind = if CONTEXTUAL.contains(&word) {
            TokenKind::Name
        } else {
            classify(word)
        };
        self.push(kind, start);
    }

    fn degrees(&mut self, start: usize) {
        let after = self.peek_at('°'.len_utf8());
        let ends = self
            .peek_at('°'.len_utf8() + 1)
            .is_none_or(|c| !(c.is_ascii_alphanumeric() || c == '_'));
        if matches!(after, Some('F' | 'C')) && ends {
            self.position += '°'.len_utf8() + 1;
            self.push(TokenKind::Unit, start);
        } else {
            self.position += '°'.len_utf8();
            self.error(
                start,
                self.position,
                Message::new(message!("Write a temperature unit as `°F` or `°C`")),
            );
            self.push(TokenKind::Error, start);
        }
    }

    /// Reads digits with `_` and an optional fraction.
    fn digits(&mut self) {
        self.eat_while(|c| c.is_ascii_digit() || c == '_');
        if self.peek() == Some('.') && self.peek_at(1).is_some_and(|c| c.is_ascii_digit()) {
            self.position += 1;
            self.eat_while(|c| c.is_ascii_digit() || c == '_');
        }
    }

    fn number_literal(&mut self, start: usize) {
        self.digits();
        let text = self.source.get(start..self.position).unwrap_or_default();
        if parse_number(text).is_none() {
            self.error(
                start,
                self.position,
                Message::new(message!(
                    "This number has more digits than a number can hold (28)"
                )),
            );
            self.push(TokenKind::Error, start);
            return;
        }
        if self.peek() == Some('%') {
            self.position += 1;
            self.push(TokenKind::Percent, start);
            return;
        }
        self.push(TokenKind::Number, start);
        self.unit_after_number();
    }

    fn unit_after_number(&mut self) {
        let rest = self.rest();
        let gap = rest.len() - rest.trim_start_matches([' ', '\t']).len();
        let word_start = self.position + gap;
        let word = unit_word(self.source.get(word_start..).unwrap_or_default());
        let Some(word) = word else {
            return;
        };
        if word == "in" && starts_value(self.source.get(word_start + 2..).unwrap_or_default()) {
            return;
        }
        self.position = word_start + word.len();
        self.push(TokenKind::Unit, word_start);
        self.unit_denominators();
    }

    /// `/ft3` directly after a unit or a money amount: the `/` and the unit, repeated.
    fn unit_denominators(&mut self) {
        while self.peek() == Some('/') {
            let Some(word) = unit_word(self.source.get(self.position + 1..).unwrap_or_default())
            else {
                return;
            };
            let slash = self.position;
            self.position += 1;
            self.push(TokenKind::Punct, slash);
            let unit = self.position;
            self.position += word.len();
            self.push(TokenKind::Unit, unit);
        }
    }

    fn money(&mut self, start: usize) {
        if self.peek_at(1) == Some('/')
            && unit_word(self.source.get(start + 2..).unwrap_or_default()).is_some()
        {
            self.position += 1;
            self.push(TokenKind::Unit, start);
            self.unit_denominators();
            return;
        }
        if !self.peek_at(1).is_some_and(|c| c.is_ascii_digit()) {
            self.position += 1;
            self.error(
                start,
                self.position,
                Message::new(message!(
                    "Write money as `$` followed by the amount, like `$450`"
                )),
            );
            self.push(TokenKind::Error, start);
            return;
        }
        self.position += 1;
        self.digits();
        let text = self
            .source
            .get(start + 1..self.position)
            .unwrap_or_default();
        if parse_number(text).is_none() {
            self.error(
                start,
                self.position,
                Message::new(message!(
                    "This amount has more digits than a number can hold (28)"
                )),
            );
            self.push(TokenKind::Error, start);
            return;
        }
        self.push(TokenKind::Money, start);
        self.unit_denominators();
    }

    fn date(&mut self, start: usize) {
        self.position += 1;
        self.eat_while(|c| c.is_ascii_digit() || matches!(c, '-' | ':' | '+' | 'T' | 'Z'));
        let text = self.source.get(start..self.position).unwrap_or_default();
        let trailing = self
            .peek()
            .is_some_and(|c| c.is_ascii_alphanumeric() || c == '_');
        if trailing {
            self.eat_while(|c| c.is_ascii_alphanumeric() || c == '_');
        }
        let text_after = self.source.get(start..self.position).unwrap_or_default();
        let valid = !trailing && parse_date_literal(text).is_some();
        if valid {
            self.push(TokenKind::Date, start);
            return;
        }
        let message = if parse_date_shape(text_after) {
            Message::with(
                message!("{0} is not a real date and time"),
                [Arg::from(text_after)],
            )
        } else if parse_time_shape(text_after) {
            Message::with(
                message!(
                    "{0} is not a real time of day: hours run from 00 to 23, minutes and seconds from 00 to 59"
                ),
                [Arg::from(text_after)],
            )
        } else {
            Message::new(message!(
                "Write a date as `@2026-10-22`, a date and time as `@2026-10-22T08:00-05:00`, or a time of day as `@17:30`"
            ))
        };
        self.error(start, self.position, message);
        self.push(TokenKind::Error, start);
    }

    fn punct(&mut self, start: usize) {
        const LONG: &[&str] = &[
            "..=", "??", "?.", "=>", "->", "..", "==", "!=", "<=", "+=", "-=",
        ];
        const SHORT: &str = "+-*/%<>=.,:()[]{}|?";
        let rest = self.rest();
        if let Some(op) = LONG.iter().find(|op| rest.starts_with(**op)) {
            self.position += op.len();
            self.push(TokenKind::Punct, start);
            return;
        }
        let Some(character) = self.peek() else {
            return;
        };
        if SHORT.contains(character) {
            self.position += 1;
            self.push(TokenKind::Punct, start);
            return;
        }
        if character == '\'' {
            self.single_quoted(start);
            return;
        }
        let (length, message, replacement) = match character {
            '&' if rest.starts_with("&&") => (
                2,
                Message::new(message!("Write `and` instead of `&&`")),
                Some("and"),
            ),
            '!' => (
                1,
                Message::new(message!("Write `not` instead of `!`")),
                Some("not "),
            ),
            ';' => (
                1,
                Message::new(message!(
                    "A statement ends at the end of its line, so `;` is not needed"
                )),
                Some(""),
            ),
            _ => (
                character.len_utf8(),
                Message::with(
                    message!("`{0}` can't be used in a tariff"),
                    [Arg::from(character.to_string())],
                ),
                None,
            ),
        };
        self.position += length;
        let mut diagnostic = Diagnostic::error("syntax", span(start, self.position), message);
        if let Some(replacement) = replacement {
            diagnostic = diagnostic.with_fix(Fix {
                label: Message::with(
                    message!("Replace with `{0}`"),
                    [Arg::from(replacement.trim())],
                ),
                changes: vec![Change {
                    span: span(start, self.position),
                    insert: replacement.to_owned(),
                }],
            });
        }
        self.diagnostics.push(diagnostic);
        self.push(TokenKind::Error, start);
    }

    /// `'text'`: one error, with a fix writing it in double quotes.
    fn single_quoted(&mut self, start: usize) {
        let line = self.rest().split('\n').next().unwrap_or_default();
        let close = line.get(1..).and_then(|after| after.find('\''));
        let length = close.map_or(1, |close| close + 2);
        self.position += length;
        let mut diagnostic = Diagnostic::error(
            "syntax",
            span(start, self.position),
            Message::new(message!("Text is written in double quotes")),
        );
        if close.is_some() {
            let end = self.position;
            diagnostic = diagnostic.with_fix(Fix {
                label: Message::new(message!("Use double quotes")),
                changes: vec![
                    Change {
                        span: span(start, start + 1),
                        insert: String::from("\""),
                    },
                    Change {
                        span: span(end - 1, end),
                        insert: String::from("\""),
                    },
                ],
            });
        }
        self.diagnostics.push(diagnostic);
        self.push(TokenKind::Error, start);
    }

    fn open_text(&mut self, start: usize) {
        let nesting = self
            .modes
            .iter()
            .filter(|mode| matches!(mode, Mode::Text { .. }))
            .count();
        if nesting >= MAX_TEXT_NESTING {
            self.eat_while(|c| c != '\n');
            self.diagnostics.push(Diagnostic::error(
                "limit",
                span(start, self.position),
                Message::with(
                    message!("Text is interpolated more than {0} levels deep"),
                    [Arg::from(MAX_TEXT_NESTING)],
                ),
            ));
            self.push(TokenKind::Error, start);
            self.modes.clear();
            return;
        }
        self.position += 1;
        self.modes.push(Mode::Text { first: true, start });
        self.text_run(start);
    }

    fn set_text_started(&mut self) {
        if let Some(Mode::Text { first, .. }) = self.modes.last_mut() {
            *first = false;
        }
    }

    /// Reads literal characters from `run_start` up to the next escape, interpolation, closing
    /// quote or line end, and writes them (with the opening quote when this is the first run).
    fn text_run(&mut self, run_start: usize) {
        self.eat_while(|c| !matches!(c, '"' | '\\' | '{' | '}' | '\n'));
        let closing = self.peek() == Some('"');
        if closing {
            self.position += 1;
        }
        let first = matches!(self.modes.last(), Some(Mode::Text { first: true, .. }));
        if self.position > run_start || first {
            self.push(TokenKind::Text, run_start);
            self.set_text_started();
        }
        if closing {
            self.modes.pop();
        }
    }

    fn text(&mut self, first: bool, start: usize) {
        let at = self.position;
        let rest = self.rest();
        if first {
            self.text_run(start);
            return;
        }
        if rest.starts_with("{{") || rest.starts_with("}}") {
            self.position += 2;
            self.push(TokenKind::Escape, at);
        } else if rest.starts_with('{') {
            self.position += 1;
            self.push(TokenKind::Interpolation, at);
            self.modes.push(Mode::Interpolation(0));
        } else if rest.starts_with('}') {
            self.position += 1;
            self.push(TokenKind::Text, at);
            self.text_run(self.position);
        } else if rest.starts_with('\\') {
            self.escape(at);
        } else if rest.starts_with('\n') || rest.is_empty() {
            self.close_open_text();
        } else {
            self.text_run(at);
        }
    }

    fn escape(&mut self, at: usize) {
        let next = self.peek_at(1);
        match next {
            Some('"' | '\\' | 'n' | 't') => {
                self.position += 2;
                self.push(TokenKind::Escape, at);
            }
            Some('u') if self.peek_at(2) == Some('{') => {
                let body = self.source.get(at + 3..).unwrap_or_default();
                let digits = body.chars().take_while(char::is_ascii_hexdigit).count();
                let closed = body.get(digits..).is_some_and(|r| r.starts_with('}'));
                let valid = closed
                    && (1..=6).contains(&digits)
                    && u32::from_str_radix(body.get(..digits).unwrap_or_default(), 16)
                        .ok()
                        .and_then(char::from_u32)
                        .is_some();
                self.position = at + 3 + digits + usize::from(closed);
                if valid {
                    self.push(TokenKind::Escape, at);
                } else {
                    self.error(
                        at,
                        self.position,
                        Message::new(message!(
                            "Write a character code as `\\u` and up to six hexadecimal digits in braces"
                        )),
                    );
                    self.push(TokenKind::Error, at);
                }
            }
            Some(c) if c != '\n' => {
                self.position += 1 + c.len_utf8();
                let text = self.source.get(at..self.position).unwrap_or_default();
                self.error(
                    at,
                    self.position,
                    Message::with(
                        message!(
                            "`{0}` is not an escape; text knows `\\\"`, `\\\\`, `\\n`, `\\t` and `\\u`"
                        ),
                        [Arg::from(text)],
                    ),
                );
                self.push(TokenKind::Error, at);
            }
            _ => {
                self.position += 1;
                self.error(
                    at,
                    self.position,
                    Message::new(message!("A `\\` in text must be followed by an escape")),
                );
                self.push(TokenKind::Error, at);
            }
        }
    }
}

/// The unit name at the start of `text`, when it is one and the word ends there.
fn unit_word(text: &str) -> Option<&str> {
    if let Some(rest) = text.strip_prefix('°') {
        let word = text.get(..'°'.len_utf8() + 1)?;
        let ends = rest
            .get(1..)
            .and_then(|r| r.chars().next())
            .is_none_or(|c| !(c.is_ascii_alphanumeric() || c == '_'));
        return (matches!(rest.chars().next(), Some('F' | 'C')) && ends).then_some(word);
    }
    let length = text
        .find(|c: char| !(c.is_ascii_alphanumeric() || c == '_'))
        .unwrap_or(text.len());
    let word = text.get(..length)?;
    (!word.is_empty() && is_unit(word)).then_some(word)
}

/// Whether the rest of a line after `in` starts a value, so `in` is the keyword, not inches.
fn starts_value(text: &str) -> bool {
    let text = text.trim_start_matches([' ', '\t']);
    let Some(first) = text.chars().next() else {
        return false;
    };
    if matches!(first, '[' | '(' | '"' | '@' | '$') || first.is_ascii_digit() {
        return true;
    }
    if text.starts_with("..") {
        return true;
    }
    if first.is_ascii_alphabetic() {
        let length = text
            .find(|c: char| !(c.is_ascii_alphanumeric() || c == '_'))
            .unwrap_or(text.len());
        let word = text.get(..length).unwrap_or_default();
        return !matches!(
            word,
            "and" | "or" | "else" | "where" | "if" | "to" | "as" | "in" | "not"
        );
    }
    false
}

/// The value of a number as written (`1_200`, `2.35`), or `None` when it has too many digits.
#[must_use]
pub fn parse_number(text: &str) -> Option<Decimal> {
    let digits: String = text.chars().filter(|c| *c != '_').collect();
    if digits.is_empty() || !digits.starts_with(|c: char| c.is_ascii_digit()) {
        return None;
    }
    Decimal::from_str_exact(&digits).ok()
}

fn two(text: &str) -> Option<u8> {
    if text.len() == 2 && text.bytes().all(|b| b.is_ascii_digit()) {
        text.parse().ok()
    } else {
        None
    }
}

/// Whether `text` has the shape of a date literal, valid or not.
fn parse_date_shape(text: &str) -> bool {
    let body = text.strip_prefix('@').unwrap_or(text);
    let bytes = body.as_bytes();
    bytes.len() >= 10
        && bytes.get(4) == Some(&b'-')
        && bytes.get(7) == Some(&b'-')
        && bytes
            .iter()
            .take(10)
            .enumerate()
            .all(|(index, byte)| index == 4 || index == 7 || byte.is_ascii_digit())
}

/// Whether `text` has the shape of a time-of-day literal, valid or not: `@HH:MM` or
/// `@HH:MM:SS` with digits in every place.
fn parse_time_shape(text: &str) -> bool {
    let body = text.strip_prefix('@').unwrap_or(text);
    let bytes = body.as_bytes();
    matches!(bytes.len(), 5 | 8)
        && bytes.iter().enumerate().all(|(index, byte)| {
            if index % 3 == 2 {
                *byte == b':'
            } else {
                byte.is_ascii_digit()
            }
        })
}

/// Whether `year` is a leap year.
const fn leap(year: i32) -> bool {
    (year % 4 == 0 && year % 100 != 0) || year % 400 == 0
}

/// The number of days in a month.
#[must_use]
pub const fn days_in_month(year: i32, month: u8) -> u8 {
    match month {
        1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
        4 | 6 | 9 | 11 => 30,
        2 if leap(year) => 29,
        2 => 28,
        _ => 0,
    }
}

/// `2026-10-22`, validated.
fn parse_date(text: &str) -> Option<DateLit> {
    let year_text = text.get(..4)?;
    if !year_text.bytes().all(|b| b.is_ascii_digit()) || text.get(4..5)? != "-" {
        return None;
    }
    let year: i32 = year_text.parse().ok()?;
    let month = two(text.get(5..7)?)?;
    if text.get(7..8)? != "-" {
        return None;
    }
    let day = two(text.get(8..10)?)?;
    if text.len() != 10 || !(1..=12).contains(&month) || day == 0 {
        return None;
    }
    (day <= days_in_month(year, month)).then_some(DateLit { year, month, day })
}

/// A date literal, `@` included: a date, a date-time or a time of day.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DateLiteral {
    /// `@2026-10-22`.
    Date(DateLit),
    /// `@2026-10-22T08:00-05:00`.
    DateTime(DateTimeLit),
    /// `@17:30`, `@07:30:15`.
    Time(TimeLit),
}

/// `17:30` or `07:30:15`, validated: hours 0 to 23, minutes and seconds 0 to 59.
fn parse_clock(text: &str) -> Option<TimeLit> {
    let mut parts = text.split(':');
    let hour = two(parts.next()?)?;
    let minute = two(parts.next()?)?;
    let second = match parts.next() {
        Some(second) => two(second)?,
        None => 0,
    };
    if parts.next().is_some() || hour > 23 || minute > 59 || second > 59 {
        return None;
    }
    Some(TimeLit {
        hour,
        minute,
        second,
    })
}

/// Reads a date literal (`@` included), validating the calendar date, the time (hours 0 to 23)
/// and the offset (within ±18:00). `@` followed directly by `HH:` is a time of day.
#[must_use]
pub fn parse_date_literal(text: &str) -> Option<DateLiteral> {
    let body = text.strip_prefix('@')?;
    if body.as_bytes().get(2) == Some(&b':') {
        return parse_clock(body).map(DateLiteral::Time);
    }
    let Some((date_text, time_text)) = body.split_once('T') else {
        return parse_date(body).map(DateLiteral::Date);
    };
    let date = parse_date(date_text)?;
    let (clock, offset_minutes) = if let Some(clock) = time_text.strip_suffix('Z') {
        (clock, 0i16)
    } else {
        let split = time_text.rfind(['+', '-'])?;
        let (clock, offset) = time_text.split_at(split);
        let negative = offset.starts_with('-');
        let offset = offset.get(1..)?;
        let (hours, minutes) = offset.split_once(':')?;
        let hours = two(hours)?;
        let minutes = two(minutes)?;
        if minutes > 59 || u16::from(hours) * 60 + u16::from(minutes) > 18 * 60 {
            return None;
        }
        let total = i16::from(hours) * 60 + i16::from(minutes);
        (clock, if negative { -total } else { total })
    };
    let time = parse_clock(clock)?;
    Some(DateLiteral::DateTime(DateTimeLit {
        date,
        hour: time.hour,
        minute: time.minute,
        second: time.second,
        offset_minutes,
    }))
}

#[cfg(test)]
mod tests {
    use super::DateLiteral;
    use super::TokenKind;
    use super::lex;
    use super::parse_date_literal;

    fn kinds(source: &str) -> Vec<(TokenKind, String)> {
        lex(source)
            .tokens
            .iter()
            .map(|t| {
                (
                    t.kind,
                    source
                        .get(t.span.start as usize..t.span.end as usize)
                        .unwrap_or_default()
                        .to_owned(),
                )
            })
            .collect()
    }

    #[test]
    fn reads_units_after_numbers() {
        let tokens = kinds("500 lb + 30 lb/ft3 - 5 °C");
        assert_eq!(tokens[1], (TokenKind::Unit, String::from("lb")));
        assert_eq!(tokens[4], (TokenKind::Unit, String::from("lb")));
        assert_eq!(tokens[6], (TokenKind::Unit, String::from("ft3")));
        assert_eq!(tokens[9], (TokenKind::Unit, String::from("°C")));
    }

    #[test]
    fn in_after_a_number_is_the_keyword_before_a_value() {
        let tokens = kinds("5 in [5]");
        assert_eq!(tokens[1].0, TokenKind::Keyword);
        let tokens = kinds("48 in");
        assert_eq!(tokens[1].0, TokenKind::Unit);
    }

    #[test]
    fn validates_dates() {
        assert!(matches!(
            parse_date_literal("@2024-02-29"),
            Some(DateLiteral::Date(_))
        ));
        assert!(parse_date_literal("@2026-02-29").is_none());
        assert!(parse_date_literal("@2026-10-22T24:00Z").is_none());
        assert!(parse_date_literal("@2026-10-22T08:00+18:01").is_none());
        assert!(matches!(
            parse_date_literal("@2026-10-22T08:00:30-05:00"),
            Some(DateLiteral::DateTime(dt)) if dt.offset_minutes == -300 && dt.second == 30
        ));
    }

    #[test]
    fn validates_times_of_day() {
        assert!(matches!(
            parse_date_literal("@17:30"),
            Some(DateLiteral::Time(time)) if time.hour == 17 && time.minute == 30 && time.second == 0
        ));
        assert!(matches!(
            parse_date_literal("@07:30:15"),
            Some(DateLiteral::Time(time)) if time.second == 15
        ));
        assert!(parse_date_literal("@24:00").is_none());
        assert!(parse_date_literal("@12:60").is_none());
        assert!(parse_date_literal("@7:30").is_none());
        assert!(parse_date_literal("@12:00Z").is_none());
        let tokens = kinds("@08:00..@17:00");
        assert_eq!(tokens[0], (TokenKind::Date, String::from("@08:00")));
        assert_eq!(tokens[1], (TokenKind::Punct, String::from("..")));
        assert_eq!(tokens[2], (TokenKind::Date, String::from("@17:00")));
        let errors = lex("@25:00").diagnostics;
        assert_eq!(
            errors.first().map(|d| d.message.english()),
            Some(String::from(
                "@25:00 is not a real time of day: hours run from 00 to 23, minutes and seconds from 00 to 59"
            ))
        );
    }

    #[test]
    fn splits_text_into_runs_escapes_and_interpolations() {
        let tokens = kinds("\"a\\n{x}b{{\"");
        let found: Vec<TokenKind> = tokens.iter().map(|t| t.0).collect();
        assert_eq!(
            found,
            vec![
                TokenKind::Text,
                TokenKind::Escape,
                TokenKind::Interpolation,
                TokenKind::Name,
                TokenKind::Interpolation,
                TokenKind::Text,
                TokenKind::Escape,
                TokenKind::Text,
                TokenKind::Eof
            ]
        );
    }
}
