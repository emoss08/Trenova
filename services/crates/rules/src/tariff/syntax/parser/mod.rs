//! The error-tolerant parser: tokens to an [`ast::File`], following the reference's Appendix A.
//!
//! The parser reads the significant tokens only (newlines and comments are left out) and keeps,
//! for each, whether a newline came before it. That flag is what ends a statement, except where
//! the grammar continues a line: inside `( … )`, `[ … ]` and record braces (which never look at
//! it), after a binary operator, `,`, `=`, `=>` or `->` (whose operand is read wherever it is),
//! and before a line that starts with a binary operator, `.`, `?.` or `??` (which the expression
//! parser accepts across a newline).
//!
//! On an error the parser records one diagnostic, produces `Missing` nodes for what is absent
//! and resynchronises at the next statement boundary, so a half-written file still yields every
//! later item. It reports nothing at a token the lexer already reported, so one typo is one
//! diagnostic. Sources over the size or token limit, and trees deeper than [`MAX_DEPTH`], end
//! parsing with a `limit` diagnostic; the tree is never deeper than the limit.
//!
//! Split by responsibility: [`items`] (the header and file-level declarations), [`statements`],
//! [`expressions`] (with text literals), [`patterns`] and [`types`].

mod docs;
mod expressions;
mod items;
mod patterns;
mod statements;
mod types;

use super::ast;
use super::ast::Expr;
use super::ast::ExprKind;
use super::ast::Name;
use super::ast::NodeId;
use super::lexer;
use super::token::Token;
use super::token::TokenKind;
use super::walk;
use crate::message;
use crate::message::Arg;
use crate::message::Message;
use crate::span::Span;
use crate::tariff::diagnostic::Change;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::diagnostic::Fix;
use docs::DocTable;

/// The deepest a tree may be.
pub const MAX_DEPTH: usize = 128;

/// Parses a Tariff file.
#[must_use]
pub fn parse(source: &str) -> ast::File {
    let lexed = lexer::lex(source);
    if source.len() > lexer::MAX_SOURCE_BYTES {
        return ast::File {
            header: None,
            docs: None,
            items: Vec::new(),
            diagnostics: lexed.diagnostics,
            tokens: lexed.tokens,
            node_count: 0,
        };
    }
    let mut parser = Parser::new(source, lexed.tokens);
    parser.diagnostics = lexed.diagnostics;
    let mut file = parser.file();
    let mut next = 0u32;
    walk::exprs_mut(&mut file, &mut |expr| {
        expr.id = NodeId(next);
        next += 1;
    });
    file.node_count = next;
    file
}

/// Words that start a file-level declaration, where recovery stops.
const ITEM_KEYWORDS: &[&str] = &[
    "use",
    "input",
    "const",
    "type",
    "enum",
    "fn",
    "test",
    "private",
    "tariff",
    "library",
    "condition",
];

/// Words that start a statement and never a value: a list left open before one of them at the
/// start of a line was never closed.
const STATEMENT_KEYWORDS: &[&str] = &[
    "let",
    "var",
    "charge",
    "note",
    "for",
    "require",
    "fail",
    "return",
    "break",
    "continue",
    "minimum",
    "maximum",
    "round",
    "given",
    "expect",
    "use",
    "input",
    "const",
    "type",
    "enum",
    "fn",
    "test",
    "private",
    "tariff",
    "library",
    "condition",
];

pub(crate) struct Parser<'a> {
    source: &'a str,
    tokens: Vec<Token>,
    /// Indexes into `tokens` of the significant tokens, ending with the end-of-file token.
    sig: Vec<usize>,
    /// For each significant token, whether a newline comes before it.
    newline: Vec<bool>,
    pos: usize,
    diagnostics: Vec<Diagnostic>,
    next_id: u32,
    /// The height of each expression built so far, by its provisional id.
    heights: Vec<u8>,
    nest: usize,
    stopped: bool,
    /// Whether the statement being read has reported an error, after which what it is still
    /// missing is not reported again: one typo, one diagnostic.
    statement_failed: bool,
    /// The nesting at which a `match` arm's value is read: there, a line starting with `-`
    /// begins the next arm (a negative pattern) instead of continuing the value.
    arm_nest: Option<usize>,
    /// Whether a `match` arm's guard is being read: a guard ends at its arm's `=>`, so there
    /// `x =>` is the guard's last value followed by the arrow, not a lambda.
    guard: bool,
    /// The token the statement being read starts at.
    statement_start: usize,
    docs: DocTable,
}

impl<'a> Parser<'a> {
    fn new(source: &'a str, tokens: Vec<Token>) -> Self {
        let mut sig = Vec::new();
        let mut newline = Vec::new();
        let mut pending = false;
        for (index, token) in tokens.iter().enumerate() {
            match token.kind {
                TokenKind::Newline => pending = true,
                TokenKind::Comment | TokenKind::DocComment => {}
                _ => {
                    sig.push(index);
                    newline.push(pending || index == 0);
                    pending = false;
                }
            }
        }
        let docs = DocTable::new(source, &tokens);
        Self {
            source,
            tokens,
            sig,
            newline,
            pos: 0,
            diagnostics: Vec::new(),
            next_id: 0,
            heights: Vec::new(),
            nest: 0,
            stopped: false,
            statement_failed: false,
            arm_nest: None,
            guard: false,
            statement_start: 0,
            docs,
        }
    }

    // ---- the token cursor ----

    fn token_at(&self, index: usize) -> Token {
        let at = self
            .sig
            .get(index)
            .or_else(|| self.sig.last())
            .copied()
            .unwrap_or(0);
        self.tokens.get(at).cloned().unwrap_or(Token {
            kind: TokenKind::Eof,
            span: Span::default(),
        })
    }

    fn peek(&self) -> Token {
        self.token_at(self.pos)
    }

    fn peek_nth(&self, ahead: usize) -> Token {
        self.token_at(self.pos + ahead)
    }

    fn kind(&self) -> TokenKind {
        self.peek().kind
    }

    fn slice(&self, span: Span) -> &'a str {
        self.source
            .get(span.start as usize..span.end as usize)
            .unwrap_or_default()
    }

    fn text_at(&self, ahead: usize) -> &'a str {
        self.slice(self.peek_nth(ahead).span)
    }

    fn text(&self) -> &'a str {
        self.text_at(0)
    }

    /// Whether a newline separates the current token from the one before it.
    fn newline_before(&self) -> bool {
        self.newline.get(self.pos).copied().unwrap_or(true)
    }

    /// Whether a newline comes before the token `ahead` places past the current one.
    fn newline_before_nth(&self, ahead: usize) -> bool {
        self.newline.get(self.pos + ahead).copied().unwrap_or(true)
    }

    /// Whether the cursor is on the contextual word `word` followed, on the same line, by text
    /// in quotes: the start of a `property`, `best`, `worst` or `option`.
    fn at_labelled(&self, word: &str) -> bool {
        self.at_word(word)
            && self.peek_nth(1).kind == TokenKind::Text
            && !self.newline_before_nth(1)
    }

    fn at_eof(&self) -> bool {
        self.kind() == TokenKind::Eof
    }

    /// Whether the current token is the punctuation `text`.
    fn at(&self, text: &str) -> bool {
        self.peek_is(0, text)
    }

    fn peek_is(&self, ahead: usize, text: &str) -> bool {
        let token = self.peek_nth(ahead);
        token.kind == TokenKind::Punct && self.slice(token.span) == text
    }

    /// Whether the current token is the keyword `word` (a reserved keyword, or a contextual one
    /// the lexer left as a name).
    fn at_word(&self, word: &str) -> bool {
        self.word_at(0, word)
    }

    fn word_at(&self, ahead: usize, word: &str) -> bool {
        let token = self.peek_nth(ahead);
        matches!(token.kind, TokenKind::Keyword | TokenKind::Name) && self.slice(token.span) == word
    }

    /// Whether the current token is any word: a name, type name, code or keyword.
    fn at_any_word(&self) -> bool {
        self.word_kind_at(0)
    }

    fn word_kind_at(&self, ahead: usize) -> bool {
        matches!(
            self.peek_nth(ahead).kind,
            TokenKind::Name | TokenKind::TypeName | TokenKind::Code | TokenKind::Keyword
        )
    }

    fn bump(&mut self) -> Token {
        let token = self.peek();
        if self.pos + 1 < self.sig.len() {
            self.pos += 1;
        }
        token
    }

    /// The span of the token before the current one.
    fn previous_span(&self) -> Span {
        if self.pos == 0 {
            return Span::new(0, 0);
        }
        self.token_at(self.pos - 1).span
    }

    /// Marks the current token's class for highlighting.
    fn mark(&mut self, kind: TokenKind) {
        self.mark_at(self.pos, kind);
    }

    fn mark_at(&mut self, index: usize, kind: TokenKind) {
        if let Some(at) = self.sig.get(index).copied()
            && let Some(token) = self.tokens.get_mut(at)
            && token.kind != TokenKind::Eof
        {
            token.kind = kind;
        }
    }

    /// Marks the token whose span is `span` (a name the parser has already read).
    fn mark_span(&mut self, span: Span, kind: TokenKind) {
        let index = self.sig.partition_point(|at| {
            self.tokens
                .get(*at)
                .is_some_and(|token| token.span.start < span.start)
        });
        if self
            .sig
            .get(index)
            .and_then(|at| self.tokens.get(*at))
            .is_some_and(|token| token.span == span)
        {
            self.mark_at(index, kind);
        }
    }

    fn eat(&mut self, text: &str) -> Option<Span> {
        if self.at(text) {
            return Some(self.bump().span);
        }
        None
    }

    fn eat_word(&mut self, word: &str) -> Option<Span> {
        if self.at_word(word) {
            self.mark(TokenKind::Keyword);
            return Some(self.bump().span);
        }
        None
    }

    // ---- diagnostics ----

    fn report(&mut self, diagnostic: Diagnostic) {
        if self.stopped {
            return;
        }
        let at = diagnostic.span.start;
        let duplicate =
            self.diagnostics.iter().rev().take(4).any(|existing| {
                existing.span.start == at && existing.severity == diagnostic.severity
            });
        if diagnostic.severity == crate::tariff::diagnostic::Severity::Error {
            self.statement_failed = true;
        }
        if !duplicate {
            self.diagnostics.push(diagnostic);
        }
    }

    fn error(&mut self, span: Span, message: Message) {
        self.report(Diagnostic::error("syntax", span, message));
    }

    /// Reports that something was expected at the current token, unless the lexer already
    /// reported that token.
    fn expected(&mut self, message: Message) {
        let token = self.peek();
        if token.kind == TokenKind::Error || self.statement_failed {
            return;
        }
        let span = self.here();
        self.error(span, message);
    }

    /// Where a diagnostic about the current position goes: the current token when it is on the
    /// same line, else the end of the previous token.
    fn here(&self) -> Span {
        let token = self.peek();
        if self.newline_before() || token.kind == TokenKind::Eof {
            let end = self.previous_span().end;
            return Span::new(end, end);
        }
        token.span
    }

    /// The end of a statement: a newline, a closing brace or the end of the file. Anything else
    /// is reported once and skipped.
    fn end_statement(&mut self) {
        if self.newline_before() || self.at("}") || self.at_eof() || self.stopped {
            return;
        }
        if self.kind() != TokenKind::Error {
            let token = self.peek();
            let found = self.slice(token.span).to_owned();
            self.report(
                Diagnostic::error(
                    "syntax",
                    token.span,
                    Message::with(
                        message!(
                            "Each statement goes on its own line; start a new line before `{0}`"
                        ),
                        [Arg::from(found)],
                    ),
                )
                .with_fix(Fix {
                    label: Message::new(message!("Start a new line")),
                    changes: vec![Change {
                        span: Span::new(token.span.start, token.span.start),
                        insert: String::from("\n"),
                    }],
                }),
            );
        }
        self.recover();
    }

    /// Skips to the next statement boundary: a token that starts a new line at the current
    /// nesting, a closing brace that closes the current block, or a file-level keyword.
    fn recover(&mut self) {
        let mut depth = 0usize;
        let mut first = true;
        while !self.at_eof() {
            let token = self.peek();
            let text = self.slice(token.span);
            if !first && depth == 0 && self.newline_before() {
                return;
            }
            if token.kind == TokenKind::Punct {
                match text {
                    "(" | "[" | "{" => depth += 1,
                    ")" | "]" | "}" if depth == 0 => {
                        if text == "}" {
                            return;
                        }
                    }
                    ")" | "]" | "}" => depth -= 1,
                    _ => {}
                }
            }
            if !first
                && depth == 0
                && token.kind == TokenKind::Keyword
                && ITEM_KEYWORDS.contains(&text)
            {
                return;
            }
            first = false;
            self.bump();
        }
    }

    /// Ends a statement that reported an error: what is left of its line is skipped without
    /// another diagnostic. Restores the enclosing statement's state.
    fn skip_rest_if_failed(&mut self, outer: bool) {
        let failed = std::mem::replace(&mut self.statement_failed, outer);
        if failed && !(self.newline_before() || self.at("}") || self.at_eof()) {
            self.recover();
        }
    }

    /// Whether a list closed by `close` ends here: at `close`, at the end of the file, or at a
    /// statement that shows the list was left open.
    fn at_list_end(&self, close: &str) -> bool {
        self.at(close) || self.at_eof() || self.at_statement_line()
    }

    /// Whether the current token starts a line with a statement keyword, so a list left open
    /// before it was never closed.
    fn at_statement_line(&self) -> bool {
        self.newline_before()
            && self.kind() == TokenKind::Keyword
            && STATEMENT_KEYWORDS.contains(&self.text())
            && !self.peek_is(1, ":")
    }

    /// Whether the current token is a file-level keyword written at the start of a line, in
    /// the first column, where a block left open before it was never closed.
    fn at_column_zero_item(&self) -> bool {
        let token = self.peek();
        let start = token.span.start as usize;
        let column_zero = start == 0
            || self
                .source
                .as_bytes()
                .get(start.wrapping_sub(1))
                .is_some_and(|byte| *byte == b'\n');
        column_zero
            && ((token.kind == TokenKind::Keyword
                && ITEM_KEYWORDS.contains(&self.slice(token.span)))
                || self.at_labelled("property"))
    }

    // ---- limits ----

    /// Enters a nested construct; false (with a `limit` diagnostic) when the tree would be too
    /// deep, after which parsing stops.
    fn enter(&mut self) -> bool {
        if self.stopped {
            return false;
        }
        if self.nest >= MAX_DEPTH {
            self.stop();
            return false;
        }
        self.nest += 1;
        true
    }

    fn leave(&mut self) {
        self.nest = self.nest.saturating_sub(1);
    }

    fn stop(&mut self) {
        if self.stopped {
            return;
        }
        let span = self.peek().span;
        self.diagnostics.push(Diagnostic::error(
            "limit",
            Span::new(span.start, span.start),
            Message::with(
                message!("This file nests deeper than {0} levels, the most a file can"),
                [Arg::from(MAX_DEPTH)],
            ),
        ));
        self.stopped = true;
        self.pos = self.sig.len().saturating_sub(1);
    }

    // ---- nodes ----

    fn height(&self, expr: &Expr) -> u8 {
        self.heights.get(expr.id.0 as usize).copied().unwrap_or(1)
    }

    /// Builds an expression whose children are `height` deep.
    fn node_with(&mut self, kind: ExprKind, span: Span, height: u8) -> Expr {
        let height = height.saturating_add(1);
        if self.nest + usize::from(height) > MAX_DEPTH {
            self.stop();
            return self.leaf(ExprKind::Missing, span);
        }
        self.leaf_with_height(kind, span, height)
    }

    fn leaf_with_height(&mut self, kind: ExprKind, span: Span, height: u8) -> Expr {
        let id = NodeId(self.next_id);
        self.next_id = self.next_id.saturating_add(1);
        self.heights.push(height);
        Expr {
            id,
            span,
            parens: 0,
            kind,
        }
    }

    fn leaf(&mut self, kind: ExprKind, span: Span) -> Expr {
        self.leaf_with_height(kind, span, 1)
    }

    /// An expression the author has not written, at the end of the previous token.
    fn missing(&mut self) -> Expr {
        let end = self.previous_span().end;
        self.leaf(ExprKind::Missing, Span::new(end, end))
    }

    fn missing_name(&self) -> Name {
        let end = self.previous_span().end;
        Name {
            text: String::new(),
            span: Span::new(end, end),
        }
    }

    fn name_of(&self, token: &Token) -> Name {
        Name {
            text: self.slice(token.span).to_owned(),
            span: token.span,
        }
    }

    /// Reads a word as a name, whatever its class (used after `.` and before `:`): keywords
    /// are marked as names.
    fn any_name(&mut self) -> Option<Name> {
        if !self.at_any_word() {
            return None;
        }
        if self.kind() == TokenKind::Keyword {
            self.mark(TokenKind::Name);
        }
        let token = self.bump();
        Some(self.name_of(&token))
    }

    /// Reads a word as a name when it is on the same line as what precedes it.
    fn same_line_name(&mut self) -> Option<Name> {
        if self.newline_before() {
            return None;
        }
        self.any_name()
    }

    /// Reads a camelCase name (or a contextual word) on the same line, reporting `message`
    /// when absent.
    fn value_name(&mut self, message: Message) -> Name {
        if self.newline_before() {
            self.expected(message);
            return self.missing_name();
        }
        match self.kind() {
            TokenKind::Name => {
                let token = self.bump();
                self.name_of(&token)
            }
            TokenKind::TypeName | TokenKind::Code => {
                let token = self.bump();
                let name = self.name_of(&token);
                self.report(
                    Diagnostic::error(
                        "syntax",
                        token.span,
                        Message::with(
                            message!("A value's name starts with a lower-case letter: `{0}`"),
                            [Arg::from(lower_first(&name.text))],
                        ),
                    )
                    .with_fix(rename_fix(token.span, lower_first(&name.text))),
                );
                name
            }
            TokenKind::Keyword => {
                let token = self.peek();
                let word = self.slice(token.span).to_owned();
                self.error(
                    token.span,
                    Message::with(
                        message!("`{0}` is a keyword, so it can't be used as a name"),
                        [Arg::from(word)],
                    ),
                );
                self.mark(TokenKind::Name);
                let token = self.bump();
                self.name_of(&token)
            }
            _ => {
                self.expected(message);
                self.missing_name()
            }
        }
    }
}

/// `word` with its first letter in lower case.
fn lower_first(word: &str) -> String {
    let mut characters = word.chars();
    match characters.next() {
        Some(first) if word.chars().all(|c| !c.is_ascii_lowercase()) => {
            first.to_ascii_lowercase().to_string() + &characters.as_str().to_ascii_lowercase()
        }
        Some(first) => first.to_ascii_lowercase().to_string() + characters.as_str(),
        None => String::new(),
    }
}

fn rename_fix(span: Span, to: String) -> Fix {
    Fix {
        label: Message::with(message!("Rename to `{0}`"), [Arg::from(to.clone())]),
        changes: vec![Change { span, insert: to }],
    }
}
