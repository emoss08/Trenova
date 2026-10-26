//! An error-tolerant recursive-descent parser for Lading.
//!
//! The parser always returns a [`File`]: a line it can't read becomes an error node and a
//! finding, and parsing resumes on the next line. Lines end statements; inside brackets, after
//! an operator, comma or arrow, and before a line starting with `.`, an operator or `else`, a
//! statement continues.

mod decl;
pub(crate) mod expr;
mod flow;
mod guide;
mod stmt;
mod test;

use trenova_module_kit::diagnostic::Diagnostic;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::ast::{Endpoint, File, Header, Ident, Item, TextLit};
use super::lexer::{Lexed, TextPiece, lex, text_pieces};
use super::token::{Comment, Tok, Token};

/// The deepest an expression or block may nest before the parser stops descending.
const MAX_DEPTH: u32 = 128;

/// What parsing produced.
#[derive(Debug, Clone)]
pub struct Parsed {
    /// The tree.
    pub file: File,
    /// The comments, for the formatter.
    pub comments: Vec<Comment>,
    /// The tokens, for highlighting.
    pub tokens: Vec<Token>,
    /// What could not be read.
    pub diagnostics: Vec<Diagnostic>,
}

/// Parses one file.
#[must_use]
pub fn parse(source: &str) -> Parsed {
    let Lexed {
        tokens,
        comments,
        diagnostics,
    } = lex(source);
    let mut parser = Parser {
        source,
        tokens,
        comments,
        pos: 0,
        diagnostics,
        keywords: Vec::new(),
        depth: 0,
        doc_floor: 0,
        no_record: false,
    };
    let file = parser.file();
    Parsed {
        file,
        comments: parser.comments,
        tokens: parser.tokens,
        diagnostics: parser.diagnostics,
    }
}

pub(super) struct Parser<'a> {
    source: &'a str,
    tokens: Vec<Token>,
    comments: Vec<Comment>,
    pos: usize,
    diagnostics: Vec<Diagnostic>,
    keywords: Vec<Span>,
    depth: u32,
    doc_floor: u32,
    no_record: bool,
}

impl Parser<'_> {
    // ----- tokens -----

    fn token(&self) -> Token {
        self.tokens.get(self.pos).copied().unwrap_or(Token {
            kind: Tok::Eof,
            span: self.end_span(),
        })
    }

    fn end_span(&self) -> Span {
        let end = u32::try_from(self.source.len()).unwrap_or(u32::MAX);
        Span::new(end, end)
    }

    fn peek(&self) -> Tok {
        self.token().kind
    }

    fn peek_at(&self, ahead: usize) -> Tok {
        self.tokens
            .get(self.pos + ahead)
            .map_or(Tok::Eof, |token| token.kind)
    }

    fn text_of(&self, span: Span) -> &str {
        self.source
            .get(span.start as usize..span.end as usize)
            .unwrap_or_default()
    }

    fn current_text(&self) -> &str {
        self.text_of(self.token().span)
    }

    fn advance(&mut self) -> Token {
        let token = self.token();
        if self.pos < self.tokens.len() {
            self.pos += 1;
        }
        token
    }

    fn last_end(&self) -> u32 {
        if self.pos == 0 {
            return 0;
        }
        self.tokens
            .get(self.pos - 1)
            .map_or(0, |token| token.span.end)
    }

    fn since(&self, start: u32) -> Span {
        Span::new(start, self.last_end().max(start))
    }

    fn at(&self, kind: Tok) -> bool {
        self.peek() == kind
    }

    fn eat(&mut self, kind: Tok) -> Option<Token> {
        if self.at(kind) {
            Some(self.advance())
        } else {
            None
        }
    }

    /// Whether the current token is the word `word`.
    fn at_word(&self, word: &str) -> bool {
        self.peek() == Tok::Name && self.current_text() == word
    }

    fn word_at(&self, ahead: usize, word: &str) -> bool {
        self.tokens
            .get(self.pos + ahead)
            .is_some_and(|token| token.kind == Tok::Name && self.text_of(token.span) == word)
    }

    /// Eats the keyword `word`, recording it for highlighting.
    fn eat_word(&mut self, word: &str) -> Option<Span> {
        if self.at_word(word) {
            let span = self.advance().span;
            self.keywords.push(span);
            Some(span)
        } else {
            None
        }
    }

    fn expect_word(&mut self, word: &str) -> Option<Span> {
        let found = self.eat_word(word);
        if found.is_none() {
            self.error_here(Message::with(message!("Expected {0}"), [Arg::from(word)]));
        }
        found
    }

    fn expect(&mut self, kind: Tok, what: &'static str) -> Option<Token> {
        let found = self.eat(kind);
        if found.is_none() {
            self.error_here(Message::with(message!("Expected {0}"), [Arg::from(what)]));
        }
        found
    }

    fn error_at(&mut self, span: Span, message: Message) {
        if self
            .diagnostics
            .last()
            .is_some_and(|last| last.span == span)
        {
            return;
        }
        self.diagnostics
            .push(Diagnostic::error("syntax", span, message));
    }

    fn error_here(&mut self, message: Message) {
        let span = self.token().span;
        self.error_at(span, message);
    }

    fn skip_newlines(&mut self) {
        while matches!(self.peek(), Tok::Newline | Tok::Semi) {
            self.advance();
        }
    }

    fn at_line_end(&self) -> bool {
        matches!(
            self.peek(),
            Tok::Newline | Tok::Eof | Tok::RBrace | Tok::Semi
        )
    }

    /// Skips to the end of the line, stepping over anything bracketed.
    fn skip_line(&mut self) {
        let mut depth = 0usize;
        loop {
            match self.peek() {
                Tok::Eof => return,
                Tok::Newline | Tok::Semi | Tok::RBrace if depth == 0 => return,
                Tok::LBrace | Tok::LParen | Tok::LBracket => depth += 1,
                Tok::RBrace | Tok::RParen | Tok::RBracket => depth = depth.saturating_sub(1),
                _ => {}
            }
            self.advance();
        }
    }

    /// Ends a statement: the line must end here.
    fn end_line(&mut self) {
        if !self.at_line_end() {
            self.error_here(Message::new(message!("Expected the end of the line")));
            self.skip_line();
        }
    }

    /// Whether the next line continues this one: it starts with `.`, an operator or `else`.
    fn continuation_ahead(&self, allow_else: bool) -> bool {
        let mut index = self.pos;
        while self
            .tokens
            .get(index)
            .is_some_and(|token| token.kind == Tok::Newline)
        {
            index += 1;
        }
        if index == self.pos {
            return false;
        }
        let Some(token) = self.tokens.get(index) else {
            return false;
        };
        match token.kind {
            Tok::Dot
            | Tok::QuestionQuestion
            | Tok::EqEq
            | Tok::NotEq
            | Tok::Lt
            | Tok::Le
            | Tok::Gt
            | Tok::Ge
            | Tok::Plus
            | Tok::Star
            | Tok::Slash => true,
            Tok::Name => {
                let word = self.text_of(token.span);
                matches!(word, "and" | "or") || (allow_else && word == "else")
            }
            _ => false,
        }
    }

    fn skip_to_continuation(&mut self) {
        while self.at(Tok::Newline) {
            self.advance();
        }
    }

    fn enter(&mut self) -> bool {
        if self.depth >= MAX_DEPTH {
            self.error_here(Message::new(message!("This is nested too deeply")));
            return false;
        }
        self.depth += 1;
        true
    }

    fn leave(&mut self) {
        self.depth = self.depth.saturating_sub(1);
    }

    // ----- names and literals -----

    fn ident(&mut self, what: &'static str) -> Option<Ident> {
        if matches!(self.peek(), Tok::Name | Tok::TypeName | Tok::X12) {
            let token = self.advance();
            return Some(Ident::new(self.text_of(token.span), token.span));
        }
        self.error_here(Message::with(message!("Expected {0}"), [Arg::from(what)]));
        None
    }

    fn name(&mut self, what: &'static str) -> Option<Ident> {
        if self.peek() == Tok::Name {
            let token = self.advance();
            return Some(Ident::new(self.text_of(token.span), token.span));
        }
        self.error_here(Message::with(message!("Expected {0}"), [Arg::from(what)]));
        None
    }

    fn type_name(&mut self, what: &'static str) -> Option<Ident> {
        if self.peek() == Tok::TypeName {
            let token = self.advance();
            return Some(Ident::new(self.text_of(token.span), token.span));
        }
        self.error_here(Message::with(message!("Expected {0}"), [Arg::from(what)]));
        None
    }

    /// A dotted name: `x12.004010.204`, `acme.204`.
    fn dotted(&mut self, what: &'static str) -> Option<Ident> {
        if !matches!(
            self.peek(),
            Tok::Name | Tok::X12 | Tok::Digits | Tok::Number | Tok::TypeName
        ) {
            self.error_here(Message::with(message!("Expected {0}"), [Arg::from(what)]));
            return None;
        }
        let start = self.advance().span.start;
        while self.at(Tok::Dot)
            && matches!(
                self.peek_at(1),
                Tok::Name | Tok::X12 | Tok::Digits | Tok::Number | Tok::TypeName
            )
        {
            self.advance();
            self.advance();
        }
        let span = self.since(start);
        Some(Ident::new(self.text_of(span), span))
    }

    /// A text literal without interpolation.
    fn text_lit(&mut self, what: &'static str) -> Option<TextLit> {
        if self.peek() == Tok::Text {
            let token = self.advance();
            let source = self.text_of(token.span).to_owned();
            let value = text_pieces(&source, token.span.start)
                .into_iter()
                .map(|piece| match piece {
                    TextPiece::Literal(text) => text,
                    TextPiece::Interpolation(text, _) => format!("{{{text}}}"),
                })
                .collect();
            return Some(TextLit {
                value,
                span: token.span,
            });
        }
        self.error_here(Message::with(message!("Expected {0}"), [Arg::from(what)]));
        None
    }

    fn uint(&mut self, what: &'static str) -> Option<(u32, Span)> {
        if matches!(self.peek(), Tok::Number | Tok::Digits) {
            let token = self.advance();
            if let Ok(value) = self.text_of(token.span).replace('_', "").parse::<u32>() {
                return Some((value, token.span));
            }
            self.error_at(
                token.span,
                Message::with(message!("Expected {0}"), [Arg::from(what)]),
            );
            return None;
        }
        self.error_here(Message::with(message!("Expected {0}"), [Arg::from(what)]));
        None
    }

    /// The doc comment directly above the token at the current position.
    fn doc(&self) -> Option<String> {
        let start = self.token().span.start;
        let mut lines = Vec::new();
        for comment in &self.comments {
            if comment.span.end > start {
                break;
            }
            if comment.span.start < self.doc_floor {
                continue;
            }
            if comment.doc {
                let text = self.text_of(comment.span);
                lines.push(text.trim_start_matches('/').trim().to_owned());
            } else {
                lines.clear();
            }
        }
        if lines.is_empty() {
            None
        } else {
            Some(lines.join("\n"))
        }
    }

    // ----- the file -----

    fn file(&mut self) -> File {
        self.skip_newlines();
        let header = self.header();
        let mut items = Vec::new();
        loop {
            self.skip_newlines();
            if self.at(Tok::Eof) {
                break;
            }
            if self.at(Tok::RBrace) {
                let span = self.advance().span;
                self.error_at(span, Message::new(message!("This brace closes nothing")));
                continue;
            }
            let before = self.pos;
            let item = self.item(&header);
            items.push(item);
            if self.pos == before {
                let span = self.advance().span;
                items.push(Item::Error(span));
            }
            self.doc_floor = self.last_end();
        }
        let end = u32::try_from(self.source.len()).unwrap_or(u32::MAX);
        File {
            header,
            items,
            span: Span::new(0, end),
            keywords: std::mem::take(&mut self.keywords),
        }
    }

    fn header(&mut self) -> Header {
        let start = self.token().span.start;
        let header = if self.eat_word("partner").is_some() {
            let name = self.name("the partner's name");
            let label = if self.at(Tok::Text) {
                self.text_lit("its label")
            } else {
                None
            };
            name.map(|name| Header::Partner { name, label })
        } else if self.eat_word("guide").is_some() {
            let name = self.dotted("the guide's name");
            self.expect_word("from");
            let base = self.dotted("the guide it narrows");
            let title = if self.at(Tok::Text) {
                self.text_lit("its title")
            } else {
                None
            };
            match (name, base) {
                (Some(name), Some(base)) => Some(Header::Guide { name, base, title }),
                _ => None,
            }
        } else if self.eat_word("map").is_some() {
            self.map_header()
        } else if self.eat_word("rules").is_some() {
            let name = self.name("the rules' name");
            self.skip_newlines();
            self.expect_word("for");
            let map = self.name("the map they check");
            let uses = self.uses();
            match (name, map) {
                (Some(name), Some(map)) => Some(Header::Rules { name, map, uses }),
                _ => None,
            }
        } else if self.eat_word("flow").is_some() {
            let name = self.name("the flow's name");
            self.skip_newlines();
            self.expect_word("with");
            let partner = self.name("the partner");
            let uses = self.uses();
            match (name, partner) {
                (Some(name), Some(partner)) => Some(Header::Flow {
                    name,
                    partner,
                    uses,
                }),
                _ => None,
            }
        } else if self.eat_word("library").is_some() {
            let name = self.name("the library's name");
            let uses = self.uses();
            name.map(|name| Header::Library { name, uses })
        } else {
            self.error_here(Message::new(message!(
                "A file starts with partner, guide, map, rules, flow or library"
            )));
            None
        };
        if header.is_none() {
            self.skip_line();
        } else {
            self.end_line();
        }
        header.unwrap_or_else(|| Header::Missing(self.since(start)))
    }

    fn uses(&mut self) -> Vec<Ident> {
        let mut uses = Vec::new();
        if self.continuation_word("use") {
            self.skip_newlines();
        }
        if self.eat_word("use").is_some() {
            loop {
                if let Some(name) = self.name("a library") {
                    uses.push(name);
                }
                if self.eat(Tok::Comma).is_none() {
                    break;
                }
            }
        }
        uses
    }

    /// Whether the next line starts with `word` (a header continued over lines).
    fn continuation_word(&self, word: &str) -> bool {
        let mut index = self.pos;
        while self
            .tokens
            .get(index)
            .is_some_and(|token| matches!(token.kind, Tok::Newline))
        {
            index += 1;
        }
        self.tokens
            .get(index)
            .is_some_and(|token| token.kind == Tok::Name && self.text_of(token.span) == word)
    }

    fn map_header(&mut self) -> Option<Header> {
        let name = self.name("the map's name");
        self.skip_newlines();
        self.expect_word("from");
        let from = self.endpoint();
        self.skip_newlines();
        self.expect_word("to");
        let to = self.endpoint();
        let uses = self.uses();
        name.map(|name| Header::Map {
            name,
            from,
            to,
            uses,
        })
    }

    fn endpoint(&mut self) -> Endpoint {
        let start = self.token().span.start;
        if self.eat_word("json").is_some() {
            return Endpoint::Json(self.since(start));
        }
        if self.at_word("csv") && self.peek_at(1) == Tok::LBrace {
            self.eat_word("csv");
            return self.csv_columns(start);
        }
        match self.dotted("a guide, a document or csv") {
            Some(ident) if ident.name.contains('.') => Endpoint::Guide(ident),
            Some(ident) => Endpoint::Document(ident),
            None => Endpoint::Missing(self.since(start)),
        }
    }

    fn csv_columns(&mut self, start: u32) -> Endpoint {
        let mut columns = Vec::new();
        self.expect(Tok::LBrace, "{");
        loop {
            self.skip_newlines();
            if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                break;
            }
            let Some(name) = self.ident("a column") else {
                self.skip_line();
                continue;
            };
            self.expect(Tok::Colon, ":");
            let ty = self.type_expr();
            let format = if self.at_line_end() || self.at(Tok::Comma) {
                None
            } else {
                Some(self.expr())
            };
            columns.push(super::ast::CsvColumn { name, ty, format });
            self.eat(Tok::Comma);
            self.end_line_soft();
        }
        self.expect(Tok::RBrace, "}");
        Endpoint::Csv {
            columns,
            span: self.since(start),
        }
    }

    /// Ends a line inside a block, where `}` also ends it.
    fn end_line_soft(&mut self) {
        if !(self.at_line_end() || self.at(Tok::Comma)) {
            self.error_here(Message::new(message!("Expected the end of the line")));
            self.skip_line();
        }
    }

    fn item(&mut self, header: &Header) -> Item {
        let doc = self.doc();
        if self.at_word("fn") {
            return Item::Fn(self.fn_decl(doc));
        }
        if self.at_word("crosswalk") {
            return Item::Crosswalk(self.crosswalk(doc));
        }
        if self.at_word("const") {
            return Item::Const(self.const_decl(doc));
        }
        if self.at_word("test") && self.peek_at(1) == Tok::Text {
            return Item::Test(self.test_decl(doc));
        }
        if self.at_word("property") && self.peek_at(1) == Tok::Text {
            return Item::Property(self.property_decl(doc));
        }
        match header {
            Header::Partner { .. } => self.partner_item(doc),
            Header::Guide { .. } => self.guide_item(doc),
            Header::Flow { .. } => self.flow_item(),
            Header::Rules { .. } if self.at_word("rule") => Item::Rule(self.rule_decl(doc)),
            Header::Library { .. } | Header::Missing(_) => {
                let start = self.token().span.start;
                self.error_here(Message::new(message!(
                    "A library holds fn, crosswalk, const, test and property declarations"
                )));
                self.skip_line();
                Item::Error(self.since(start))
            }
            Header::Map { .. } | Header::Rules { .. } => Item::Stmt(self.stmt()),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::parse;
    use crate::lading::syntax::ast::{FlowDecl, Header, Item, Stmt};

    const CORPUS: &[(&str, &str)] = &[
        (
            "partner",
            include_str!("../../../../testdata/lading/acme/partner.lading"),
        ),
        (
            "guide",
            include_str!("../../../../testdata/lading/acme/guides/acme-204.lading"),
        ),
        (
            "library",
            include_str!("../../../../testdata/lading/acme/lib/common.lading"),
        ),
        (
            "map",
            include_str!("../../../../testdata/lading/acme/maps/tender-in.lading"),
        ),
        (
            "map",
            include_str!("../../../../testdata/lading/acme/maps/status-out.lading"),
        ),
        (
            "rules",
            include_str!("../../../../testdata/lading/acme/rules/tender-checks.lading"),
        ),
        (
            "flow",
            include_str!("../../../../testdata/lading/acme/flows/tender.lading"),
        ),
    ];

    #[test]
    fn the_corpus_parses_without_findings() {
        for (kind, source) in CORPUS {
            let parsed = parse(source);
            let messages: Vec<String> = parsed
                .diagnostics
                .iter()
                .map(|d| {
                    format!(
                        "{}..{} {} at {:?}",
                        d.span.start,
                        d.span.end,
                        d.message.english(),
                        source.get(d.span.start as usize..(d.span.end as usize).min(source.len()))
                    )
                })
                .collect();
            assert!(messages.is_empty(), "{kind}: {messages:#?}");
            assert_eq!(parsed.file.header.kind(), *kind);
            assert!(
                parsed
                    .file
                    .items
                    .iter()
                    .all(|item| !matches!(item, Item::Error(_))),
                "{kind}: {:#?}",
                parsed.file.items
            );
        }
    }

    #[test]
    fn a_map_reads_its_header_and_each_block() {
        let parsed = parse(CORPUS[3].1);
        let Header::Map { name, uses, .. } = &parsed.file.header else {
            panic!("not a map");
        };
        assert_eq!(name.name, "tenderIn");
        assert_eq!(uses.len(), 1);
        let assigns = parsed
            .file
            .items
            .iter()
            .filter(|item| matches!(item, Item::Stmt(Stmt::Assign { .. })))
            .count();
        assert_eq!(assigns, 9);
    }

    #[test]
    fn a_flow_reads_every_clause() {
        let parsed = parse(CORPUS[6].1);
        let flows: Vec<&FlowDecl> = parsed
            .file
            .items
            .iter()
            .filter_map(|item| match item {
                Item::Flow(flow) => Some(flow),
                _ => None,
            })
            .collect();
        assert_eq!(flows.len(), 5);
        let FlowDecl::Receive(receive) = flows[0] else {
            panic!("not a receive");
        };
        assert_eq!(receive.arms.len(), 3);
        assert!(receive.key.is_some());
        assert_eq!(
            receive.rules.as_ref().map(|r| r.name.as_str()),
            Some("tenderChecks")
        );
    }

    #[test]
    fn broken_lines_recover_on_the_next() {
        let parsed = parse(
            "map m\n  from loadTender\n  to x12.004010.214\n\nB10 { 01 <- }\nLX { 01 <- 1 }\n",
        );
        assert!(!parsed.diagnostics.is_empty());
        assert!(
            parsed.file.items.iter().any(
                |item| matches!(item, Item::Stmt(Stmt::Segment { id, .. }) if id.name == "LX")
            )
        );
    }

    #[test]
    fn deep_nesting_is_refused_not_overflowed() {
        let source = format!(
            "map m\n from loadTender\n to x12.004010.214\nkey <- {}1{}\n",
            "(".repeat(5000),
            ")".repeat(5000)
        );
        let parsed = parse(&source);
        assert!(!parsed.diagnostics.is_empty());
    }
}
