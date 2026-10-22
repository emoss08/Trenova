//! The parser: a Pratt parser over the token stream, with expr-lang's grammar and precedence.
//!
//! It never fails. A part the author has not written yet becomes a `Missing` node, and every
//! problem is a diagnostic anchored to the span it is about, worded for the person typing, with a
//! one-line fix where there is an obvious one (`;` after a `let`, `==` for a stray `=`). After a
//! statement it cannot read, it skips to the next top-level `;`.
//!
//! Recursion is bounded: past [`MAX_DEPTH`] levels of nesting, or [`MAX_NODES`] nodes, it reports
//! the limit once and reads the rest of the text as its end, so no input can exhaust the stack or
//! make the tree arbitrarily large.

use super::MAX_DEPTH;
use super::MAX_NODES;
use super::ast::Binary;
use super::ast::BinaryOp;
use super::ast::Call;
use super::ast::Chain;
use super::ast::CompareOp;
use super::ast::Expr;
use super::ast::ExprKind;
use super::ast::If;
use super::ast::Index;
use super::ast::Let;
use super::ast::MapEntry;
use super::ast::Member;
use super::ast::Method;
use super::ast::Name;
use super::ast::Number;
use super::ast::Pointer;
use super::ast::Predicate;
use super::ast::Sequence;
use super::ast::Slice;
use super::ast::Statement;
use super::ast::StatementKind;
use super::ast::SyntaxCode;
use super::ast::SyntaxDiagnostic;
use super::ast::SyntaxFix;
use super::ast::Ternary;
use super::ast::Token;
use super::ast::TokenKind;
use super::ast::Unary;
use super::ast::UnaryOp;
use super::lexer;
use crate::formula::catalog;
use crate::message;
use crate::message::Arg;
use crate::message::Message;
use crate::span::Span;
use rust_decimal::Decimal;
use std::collections::BTreeMap;

/// An infix operator as the precedence climber sees it.
#[derive(Debug, Clone)]
enum Infix {
    Pipe,
    Op(BinaryOp),
}

fn infix(text: &str, operators: &[&str]) -> Option<(Infix, u16, bool)> {
    let op = match text {
        "|" => return Some((Infix::Pipe, 0, false)),
        "or" => BinaryOp::Or { word: true },
        "||" => BinaryOp::Or { word: false },
        "and" => BinaryOp::And { word: true },
        "&&" => BinaryOp::And { word: false },
        "==" => BinaryOp::Equal,
        "!=" => BinaryOp::NotEqual,
        "<" => BinaryOp::Compare(CompareOp::Less),
        ">" => BinaryOp::Compare(CompareOp::Greater),
        "<=" => BinaryOp::Compare(CompareOp::LessEqual),
        ">=" => BinaryOp::Compare(CompareOp::GreaterEqual),
        "in" => BinaryOp::In,
        "matches" => BinaryOp::Matches,
        "contains" => BinaryOp::Contains,
        "startsWith" => BinaryOp::StartsWith,
        "endsWith" => BinaryOp::EndsWith,
        ".." => BinaryOp::Range,
        "+" => BinaryOp::Add,
        "-" => BinaryOp::Subtract,
        "*" => BinaryOp::Multiply,
        "/" => BinaryOp::Divide,
        "%" => BinaryOp::Modulo,
        "**" => BinaryOp::Power { caret: false },
        "^" => BinaryOp::Power { caret: true },
        "??" => BinaryOp::Coalesce,
        word if operators.contains(&word) => BinaryOp::Word(Box::from(word)),
        _ => return None,
    };
    let precedence = match op {
        BinaryOp::Or { .. } => 10,
        BinaryOp::And { .. } => 15,
        BinaryOp::Range => 25,
        BinaryOp::Add | BinaryOp::Subtract => 30,
        BinaryOp::Multiply | BinaryOp::Divide | BinaryOp::Modulo => 60,
        BinaryOp::Power { .. } => 100,
        BinaryOp::Coalesce => 500,
        _ => 20,
    };
    let right = matches!(op, BinaryOp::Power { .. });
    Some((Infix::Op(op), precedence, right))
}

fn unary(text: &str) -> Option<(UnaryOp, u16)> {
    match text {
        "not" => Some((UnaryOp::Not { word: true }, 50)),
        "!" => Some((UnaryOp::Not { word: false }, 50)),
        "-" => Some((UnaryOp::Negate, 90)),
        "+" => Some((UnaryOp::Plus, 90)),
        _ => None,
    }
}

const fn opener_of(close: &str) -> &'static str {
    match close.as_bytes() {
        b")" => "(",
        b"]" => "[",
        _ => "{",
    }
}

/// The parser's state over one token stream.
pub(super) struct Parser<'a> {
    source: &'a str,
    tokens: &'a [Token],
    index: usize,
    operators: &'a [&'a str],
    pub(super) diagnostics: Vec<SyntaxDiagnostic>,
    reported: BTreeMap<u32, Vec<usize>>,
    nodes: usize,
    recursion: usize,
    stopped: bool,
    eof: Token,
}

fn quoted(text: &str) -> String {
    format!("'{text}'")
}

impl<'a> Parser<'a> {
    pub(super) fn new(source: &'a str, tokens: &'a [Token], operators: &'a [&'a str]) -> Self {
        let end = u32::try_from(source.len()).unwrap_or(u32::MAX);
        Self {
            source,
            tokens,
            index: 0,
            operators,
            diagnostics: Vec::new(),
            reported: BTreeMap::new(),
            nodes: 0,
            recursion: 0,
            stopped: false,
            eof: Token {
                kind: TokenKind::Eof,
                span: Span::new(end, end),
            },
        }
    }

    fn current(&self) -> Token {
        if self.stopped {
            return self.eof;
        }
        self.tokens.get(self.index).copied().unwrap_or(self.eof)
    }

    fn peek(&self) -> Token {
        if self.stopped {
            return self.eof;
        }
        self.tokens.get(self.index + 1).copied().unwrap_or(self.eof)
    }

    fn text(&self, token: Token) -> &'a str {
        self.source
            .get(token.span.start as usize..token.span.end as usize)
            .unwrap_or_default()
    }

    fn advance(&mut self) -> Token {
        let token = self.current();
        if !self.stopped && self.index + 1 < self.tokens.len() {
            self.index += 1;
        }
        token
    }

    fn prev_end(&self, fallback: u32) -> u32 {
        self.index
            .checked_sub(1)
            .and_then(|index| self.tokens.get(index))
            .map_or(fallback, |token| token.span.end)
    }

    fn is(&self, kind: TokenKind, text: &str) -> bool {
        let token = self.current();
        token.kind == kind && self.text(token) == text
    }

    fn is_kind(&self, kind: TokenKind) -> bool {
        self.current().kind == kind
    }

    fn is_op(&self, text: &str) -> bool {
        let token = self.current();
        matches!(token.kind, TokenKind::Operator | TokenKind::Keyword) && self.text(token) == text
    }

    fn is_punct(&self, text: &str) -> bool {
        self.is(TokenKind::Punct, text)
    }

    fn is_eof(&self) -> bool {
        self.is_kind(TokenKind::Eof)
    }

    fn report(&mut self, span: Span, message: Message) {
        self.report_fix(span, message, None);
    }

    fn report_fix(&mut self, span: Span, message: Message, fix: Option<SyntaxFix>) {
        if self.stopped {
            return;
        }
        let at = self.reported.entry(span.start).or_default();
        let repeated = at.iter().any(|index| {
            self.diagnostics
                .get(*index)
                .is_some_and(|known| known.message == message)
        });
        if !repeated {
            at.push(self.diagnostics.len());
            self.diagnostics.push(SyntaxDiagnostic {
                span,
                code: SyntaxCode::Syntax,
                message,
                fix,
            });
        }
    }

    fn stop(&mut self, span: Span, message: Message) {
        if self.stopped {
            return;
        }
        self.diagnostics.push(SyntaxDiagnostic {
            span,
            code: SyntaxCode::Limit,
            message,
            fix: None,
        });
        self.stopped = true;
    }

    fn missing(at: u32) -> Expr {
        Expr::leaf(ExprKind::Missing, Span::new(at, at))
    }

    /// Builds a node, counting it against the node budget and measuring its depth.
    fn node(&mut self, kind: ExprKind, span: Span) -> Expr {
        let mut expr = Expr::leaf(kind, span);
        let mut deepest = 0u16;
        expr.for_each_child(|child| deepest = deepest.max(child.depth));
        expr.depth = deepest.saturating_add(1);
        self.nodes += 1;
        if self.nodes > MAX_NODES {
            self.stop(
                span,
                Message::with(
                    message!("This formula is too long: a formula has at most {0} parts"),
                    [Arg::from(MAX_NODES)],
                ),
            );
            return Self::missing(span.start);
        }
        if usize::from(expr.depth) > MAX_DEPTH {
            self.stop_too_deep(span);
            return Self::missing(span.start);
        }
        expr
    }

    fn stop_too_deep(&mut self, span: Span) {
        self.stop(
            span,
            Message::with(
                message!("This formula nests more than {0} levels deep; split it into 'let' lines"),
                [Arg::from(MAX_DEPTH)],
            ),
        );
    }

    /// Runs `parse` one nesting level deeper, or stops the parse past the limit.
    fn nested(&mut self, parse: impl FnOnce(&mut Self) -> Expr) -> Expr {
        if self.recursion >= MAX_DEPTH {
            let at = self.current().span;
            self.stop_too_deep(at);
            return Self::missing(at.start);
        }
        self.recursion += 1;
        let expr = parse(self);
        self.recursion -= 1;
        expr
    }

    /// Skips to the `;` that ends the statement, or the end, after an error.
    fn recover(&mut self) {
        let mut depth = 0i64;
        while !self.is_eof() {
            let token = self.current();
            if token.kind == TokenKind::Punct {
                match self.text(token) {
                    "(" | "[" | "{" => depth += 1,
                    ")" | "]" | "}" => depth -= 1,
                    ";" if depth <= 0 => return,
                    _ => {}
                }
            }
            self.advance();
        }
    }

    pub(super) fn parse_program(&mut self) -> Vec<Statement> {
        let mut statements = Vec::new();
        while !self.is_eof() {
            let start = self.current().span.start;
            if self.is(TokenKind::Keyword, "let") {
                let (name, value, span, semicolon) = self.parse_let(true);
                statements.push(Statement {
                    span,
                    kind: StatementKind::Let(name),
                    value,
                    semicolon,
                });
                continue;
            }
            if self.is_punct(";") {
                let token = self.current();
                self.report(token.span, Message::new(message!("Unexpected ';'")));
                self.advance();
                continue;
            }
            let value = self.parse_expression(0);
            let end = self.prev_end(start).max(value.span.end).max(start);
            let mut statement = Statement {
                span: Span::new(start, end),
                kind: StatementKind::Result,
                value,
                semicolon: None,
            };
            if self.is_punct(";") {
                statement.semicolon = Some(self.advance().span);
                statements.push(statement);
                continue;
            }
            statements.push(statement);
            if !self.is_eof() {
                let token = self.current();
                self.report(
                    token.span,
                    Message::with(
                        message!("Unexpected {0}"),
                        [Arg::from(quoted(self.text(token)))],
                    ),
                );
                self.recover();
                if self.is_punct(";") {
                    self.advance();
                }
            }
        }
        statements
    }

    /// `let name = value;`, at the top level or inside a sequence: the name, the value, the span
    /// from `let` to the end of the value, and the `;`.
    fn parse_let(&mut self, top: bool) -> (Name, Expr, Span, Option<Span>) {
        let let_token = self.advance();
        let name = self.let_name(let_token);
        self.let_equals(&name);
        let value = if self.is_punct(";") || self.is_eof() {
            Self::missing(self.current().span.start)
        } else {
            self.parse_expression(0)
        };
        if value.is_missing() {
            if name.text.is_empty() {
                self.report(
                    name.span,
                    Message::new(message!("'let' needs a value after '='")),
                );
            } else {
                self.report(
                    name.span,
                    Message::with(
                        message!("'let {0}' needs a value after '='"),
                        [Arg::from(name.text.as_str())],
                    ),
                );
            }
        }
        let end = self.prev_end(let_token.span.end).max(let_token.span.end);
        let semicolon = self.let_end(top, end);
        (name, value, Span::new(let_token.span.start, end), semicolon)
    }

    fn let_name(&mut self, let_token: Token) -> Name {
        let mut name = Name {
            text: String::new(),
            span: Span::new(let_token.span.end, let_token.span.end),
        };
        if self.is_kind(TokenKind::Ident) {
            let token = self.advance();
            name = Name {
                text: self.text(token).to_owned(),
                span: token.span,
            };
        } else if self.is_kind(TokenKind::Keyword) && !self.is_op("=") {
            let token = self.advance();
            let text = self.text(token);
            self.report(
                token.span,
                Message::with(
                    message!("'{0}' is a keyword and cannot be a name"),
                    [Arg::from(text)],
                ),
            );
            name = Name {
                text: text.to_owned(),
                span: token.span,
            };
        } else {
            self.report(
                let_token.span,
                Message::new(message!("'let' needs a name: let fuel = …")),
            );
        }
        name
    }

    fn let_equals(&mut self, name: &Name) {
        if self.is_op("=") {
            self.advance();
        } else if self.is_op("==") {
            let token = self.advance();
            self.report_fix(
                token.span,
                Message::new(message!("'let' assigns with one '='")),
                Some(SyntaxFix {
                    label: Message::new(message!("Use '='")),
                    insert: String::from("="),
                }),
            );
        } else if name.text.is_empty() {
            self.report(
                name.span,
                Message::new(message!("'let' needs a name, '=' and a value")),
            );
        } else {
            self.report(
                name.span,
                Message::with(
                    message!("'let {0}' needs '=' and a value"),
                    [Arg::from(name.text.as_str())],
                ),
            );
        }
    }

    /// The `;` that ends a `let`, or a diagnostic with a fix that adds it at `end`.
    fn let_end(&mut self, top: bool, end: u32) -> Option<Span> {
        if self.is_punct(";") {
            return Some(self.advance().span);
        }
        if top && self.is_eof() {
            self.report_fix(
                Span::new(end, end),
                Message::new(message!(
                    "A 'let' line ends with ';', then the formula's amount follows on the next line"
                )),
                Some(SyntaxFix {
                    label: Message::new(message!("Add ';'")),
                    insert: String::from(";"),
                }),
            );
        } else {
            self.report_fix(
                Span::new(end, end),
                Message::new(message!("A 'let' line ends with ';'")),
                Some(SyntaxFix {
                    label: Message::new(message!("Add ';'")),
                    insert: String::from(";"),
                }),
            );
        }
        None
    }

    fn parse_expression(&mut self, precedence: u16) -> Expr {
        self.nested(|parser| parser.expression(precedence))
    }

    fn expression(&mut self, precedence: u16) -> Expr {
        if precedence == 0 && self.is(TokenKind::Keyword, "if") {
            return self.parse_if();
        }
        let mut left = self.parse_unary();
        let mut previous: Option<String> = None;

        loop {
            let mut token = self.current();
            let mut negate: Option<Token> = None;
            if token.kind == TokenKind::Keyword && self.text(token) == "not" {
                let next = self.peek();
                let next_text = self.text(next);
                let negatable = matches!(
                    next_text,
                    "contains" | "matches" | "startsWith" | "endsWith" | "in"
                ) || self.operators.contains(&next_text);
                if negatable {
                    negate = Some(token);
                    token = next;
                }
            }
            let mut text = self.text(token);
            if token.kind == TokenKind::Operator && text == "=" && negate.is_none() {
                self.report_fix(
                    token.span,
                    Message::new(message!("'=' assigns; use '==' to compare")),
                    Some(SyntaxFix {
                        label: Message::new(message!("Use '=='")),
                        insert: String::from("=="),
                    }),
                );
                text = "==";
            }
            if !matches!(token.kind, TokenKind::Operator | TokenKind::Keyword) {
                break;
            }
            let Some((op, op_precedence, right_assoc)) = infix(text, self.operators) else {
                break;
            };
            if op_precedence < precedence {
                break;
            }
            if negate.is_some() {
                self.advance();
            }
            self.advance();

            let op = match op {
                Infix::Pipe => {
                    match self.parse_pipe(left, token) {
                        Ok(call) => left = call,
                        Err(left) => return left,
                    }
                    continue;
                }
                Infix::Op(op) => op,
            };

            if previous.as_deref() == Some("??") && text != "??" {
                self.report(
                    token.span,
                    Message::with(
                        message!("Wrap the '??' part in parentheses before '{0}': (a ?? 0) {0} b"),
                        [Arg::from(text)],
                    ),
                );
            }

            let right = self.parse_expression(if right_assoc {
                op_precedence
            } else {
                op_precedence + 1
            });
            if right.is_missing() {
                self.report(
                    token.span,
                    Message::with(
                        message!("'{0}' needs a value on its right"),
                        [Arg::from(text)],
                    ),
                );
            }
            left = self.binary(left, op, token.span, negate.map(|not| not.span), right);
            previous = Some(text.to_owned());
        }

        if precedence == 0 && self.is_op("?") {
            return self.parse_ternary(left);
        }
        left
    }

    fn binary(
        &mut self,
        left: Expr,
        op: BinaryOp,
        op_span: Span,
        not: Option<Span>,
        right: Expr,
    ) -> Expr {
        let span = Span::new(left.span.start, right.span.end.max(left.span.end));
        if let BinaryOp::Compare(compare) = op
            && left.parens == 0
            && not.is_none()
        {
            match left.kind {
                ExprKind::Binary(Binary {
                    op: BinaryOp::Compare(first),
                    op_span: first_span,
                    not: None,
                    left: first_left,
                    right: middle,
                }) => {
                    let chain = Chain {
                        operands: vec![*first_left, *middle, right],
                        operators: vec![(first, first_span), (compare, op_span)],
                    };
                    return self.node(ExprKind::Chain(chain), span);
                }
                ExprKind::Chain(mut chain) => {
                    chain.operands.push(right);
                    chain.operators.push((compare, op_span));
                    return self.node(ExprKind::Chain(chain), span);
                }
                kind => {
                    let left = Expr { kind, ..left };
                    return self.node(
                        ExprKind::Binary(Binary {
                            op,
                            op_span,
                            not,
                            left: Box::new(left),
                            right: Box::new(right),
                        }),
                        span,
                    );
                }
            }
        }
        self.node(
            ExprKind::Binary(Binary {
                op,
                op_span,
                not,
                left: Box::new(left),
                right: Box::new(right),
            }),
            span,
        )
    }

    /// `value | name(args)`: the call, or the value alone when no function follows the pipe.
    fn parse_pipe(&mut self, left: Expr, pipe: Token) -> Result<Expr, Expr> {
        let name_token = self.current();
        if name_token.kind != TokenKind::Ident {
            self.report(
                pipe.span,
                Message::new(message!("'|' pipes into a function: value | round(2)")),
            );
            return Err(left);
        }
        self.advance();
        let name = self.text(name_token);
        let (args, parens) = if self.is_punct("(") {
            self.parse_arguments()
        } else {
            self.report_fix(
                name_token.span,
                Message::with(
                    message!("A function after '|' needs its parentheses: value | {0}()"),
                    [Arg::from(name)],
                ),
                Some(SyntaxFix {
                    label: Message::new(message!("Add ()")),
                    insert: format!("{name}()"),
                }),
            );
            (Vec::new(), name_token.span)
        };
        let span = Span::new(left.span.start, parens.end);
        let mut all = Vec::with_capacity(args.len() + 1);
        all.push(left);
        all.extend(args);
        Ok(self.call(
            Name {
                text: name.to_owned(),
                span: name_token.span,
            },
            all,
            parens,
            true,
            span,
        ))
    }

    /// A call node; a predicate function's second argument becomes a predicate.
    fn call(
        &mut self,
        function: Name,
        mut args: Vec<Expr>,
        parens: Span,
        pipe: bool,
        span: Span,
    ) -> Expr {
        let predicate = catalog::function(&function.text).is_some_and(|entry| entry.predicate);
        if predicate
            && args.len() > 1
            && !matches!(
                args[1].kind,
                ExprKind::Predicate(Predicate { braces: true, .. })
            )
        {
            let body = args.remove(1);
            let body_span = body.span;
            let wrapped = self.node(
                ExprKind::Predicate(Predicate {
                    body: Box::new(body),
                    braces: false,
                }),
                body_span,
            );
            args.insert(1, wrapped);
        }
        self.node(
            ExprKind::Call(Call {
                function,
                args,
                parens,
                pipe,
            }),
            span,
        )
    }

    fn parse_ternary(&mut self, test: Expr) -> Expr {
        self.advance();
        let then;
        let otherwise;
        if self.is_op(":") {
            self.advance();
            then = None;
            otherwise = self.parse_expression(0);
        } else {
            let consequent = self.parse_expression(0);
            if self.is_op(":") {
                self.advance();
                otherwise = self.parse_expression(0);
            } else {
                let token = self.current();
                self.report(
                    token.span,
                    Message::new(message!(
                        "'?' needs ':' and the value for the other case: test ? a : b"
                    )),
                );
                otherwise = Self::missing(token.span.start);
            }
            if consequent.is_missing() {
                let token = self.current();
                self.report(
                    token.span,
                    Message::new(message!("'?' needs a value for when the test holds")),
                );
            }
            then = Some(Box::new(consequent));
        }
        let span = Span::new(test.span.start, otherwise.span.end.max(test.span.end));
        self.node(
            ExprKind::Ternary(Ternary {
                test: Box::new(test),
                then,
                otherwise: Box::new(otherwise),
            }),
            span,
        )
    }

    fn parse_if(&mut self) -> Expr {
        self.nested(Self::if_expression)
    }

    fn if_expression(&mut self) -> Expr {
        let if_token = self.advance();
        let test = self.parse_expression(0);
        let then = self.parse_block("if");
        let otherwise = if self.is(TokenKind::Keyword, "else") {
            self.advance();
            if self.is(TokenKind::Keyword, "if") {
                self.parse_if()
            } else {
                self.parse_block("else")
            }
        } else {
            self.report(
                if_token.span,
                Message::new(message!("'if' needs an 'else' with the other value")),
            );
            Self::missing(self.current().span.start)
        };
        let end = self
            .prev_end(if_token.span.end)
            .max(otherwise.span.end)
            .max(if_token.span.end);
        self.node(
            ExprKind::If(If {
                test: Box::new(test),
                then: Box::new(then),
                otherwise: Box::new(otherwise),
            }),
            Span::new(if_token.span.start, end),
        )
    }

    fn parse_block(&mut self, after: &str) -> Expr {
        if !self.is_punct("{") {
            let token = self.current();
            self.report(
                token.span,
                Message::with(
                    message!("'{0}' needs its value in braces after it"),
                    [Arg::from(after)],
                ),
            );
            return Self::missing(token.span.start);
        }
        let open = self.advance();
        let body = if self.is_punct("}") {
            let token = self.current();
            self.report(
                Span::new(open.span.start, token.span.end),
                Message::with(
                    message!("The braces after '{0}' are empty; put its value inside them"),
                    [Arg::from(after)],
                ),
            );
            Self::missing(token.span.start)
        } else {
            self.parse_sequence()
        };
        if self.is_punct("}") {
            self.advance();
        } else {
            self.report(
                open.span,
                Message::with(
                    message!("'{0}' opened here is never closed"),
                    [Arg::from("{")],
                ),
            );
        }
        body
    }

    /// `let` lines followed by a value, where they are allowed besides the top level: inside
    /// parentheses, an `if` block and a predicate's braces.
    fn parse_sequence(&mut self) -> Expr {
        if !self.is(TokenKind::Keyword, "let") {
            return self.parse_expression(0);
        }
        let start = self.current().span.start;
        let mut lets = Vec::new();
        while self.is(TokenKind::Keyword, "let") {
            let (name, value, span, _) = self.parse_let(false);
            lets.push(Let { span, name, value });
        }
        let body = self.parse_expression(0);
        if body.is_missing()
            && let Some(last) = lets.last()
        {
            let span = last.span;
            self.report(
                span,
                Message::new(message!("A 'let' here needs the value it is for after it")),
            );
        }
        let end = body.span.end.max(start);
        self.node(
            ExprKind::Sequence(Sequence {
                lets,
                body: Box::new(body),
            }),
            Span::new(start, end),
        )
    }

    fn parse_unary(&mut self) -> Expr {
        let token = self.current();
        if matches!(token.kind, TokenKind::Operator | TokenKind::Keyword)
            && let Some((op, precedence)) = unary(self.text(token))
        {
            self.advance();
            let operand = self.parse_expression(precedence);
            if operand.is_missing() {
                self.report(
                    token.span,
                    Message::with(
                        message!("'{0}' needs a value after it"),
                        [Arg::from(self.text(token))],
                    ),
                );
            }
            let span = Span::new(token.span.start, operand.span.end.max(token.span.end));
            return self.node(
                ExprKind::Unary(Unary {
                    op,
                    op_span: token.span,
                    operand: Box::new(operand),
                }),
                span,
            );
        }
        let primary = self.parse_primary();
        self.parse_postfix(primary)
    }

    fn parse_postfix(&mut self, mut node: Expr) -> Expr {
        loop {
            if self.stopped {
                return node;
            }
            if self.is_op("?.")
                && self.peek().kind == TokenKind::Punct
                && self.text(self.peek()) == "["
            {
                self.advance();
                node = self.parse_index(node, true);
                continue;
            }
            if self.is_punct(".") || self.is_op("?.") {
                let dot = self.advance();
                if matches!(self.current().kind, TokenKind::Ident | TokenKind::Keyword) {
                    let property = self.advance();
                    let span = Span::new(node.span.start, property.span.end);
                    node = self.node(
                        ExprKind::Member(Member {
                            object: Box::new(node),
                            property: Name {
                                text: self.text(property).to_owned(),
                                span: property.span,
                            },
                            optional: self.text(dot) == "?.",
                        }),
                        span,
                    );
                    continue;
                }
                self.report(
                    dot.span,
                    Message::new(message!("'.' needs a field name after it")),
                );
                return node;
            }
            if self.is_punct("(") {
                node = self.parse_call(node);
                continue;
            }
            if self.is_punct("[") {
                node = self.parse_index(node, false);
                continue;
            }
            return node;
        }
    }

    fn parse_call(&mut self, callee: Expr) -> Expr {
        let open = self.current();
        let (args, parens) = self.parse_arguments();
        let span = Span::new(callee.span.start, parens.end);
        if callee.parens == 0 {
            match callee.kind {
                ExprKind::Ident(name) => {
                    return self.call(
                        Name {
                            text: name,
                            span: callee.span,
                        },
                        args,
                        parens,
                        false,
                        span,
                    );
                }
                ExprKind::Member(member) => {
                    return self.node(
                        ExprKind::Method(Method {
                            receiver: member.object,
                            method: member.property,
                            optional: member.optional,
                            args,
                            parens,
                        }),
                        span,
                    );
                }
                kind => {
                    let callee = Expr { kind, ..callee };
                    return self.uncallable(callee, open);
                }
            }
        }
        self.uncallable(callee, open)
    }

    fn uncallable(&mut self, callee: Expr, open: Token) -> Expr {
        if !callee.is_missing() {
            self.report(
                open.span,
                Message::with(message!("Unexpected {0}"), [Arg::from(quoted("("))]),
            );
        }
        callee
    }

    fn parse_index(&mut self, node: Expr, optional: bool) -> Expr {
        let open = self.advance();
        let mut start = None;
        let mut end = None;
        let mut slice = false;
        if !self.is_op(":") {
            start = Some(self.parse_expression(0));
        }
        if self.is_op(":") {
            slice = true;
            self.advance();
            if !self.is_punct("]") {
                end = Some(self.parse_expression(0));
            }
        }
        let to = if self.is_punct("]") {
            self.advance().span.end
        } else {
            self.report(
                open.span,
                Message::new(message!("'[' opened here is never closed")),
            );
            self.prev_end(open.span.end)
        };
        let span = Span::new(node.span.start, to.max(node.span.end));
        if slice {
            return self.node(
                ExprKind::Slice(Slice {
                    object: Box::new(node),
                    start: start.map(Box::new),
                    end: end.map(Box::new),
                    optional,
                }),
                span,
            );
        }
        let index = start.unwrap_or_else(|| Self::missing(open.span.end));
        self.node(
            ExprKind::Index(Index {
                object: Box::new(node),
                index: Box::new(index),
                optional,
            }),
            span,
        )
    }

    fn parse_arguments(&mut self) -> (Vec<Expr>, Span) {
        let open = self.advance();
        let mut args = Vec::new();
        if !self.is_punct(")") {
            loop {
                if self.is_punct(")") && !args.is_empty() {
                    break;
                }
                if self.is_punct(")") || self.is_eof() {
                    let previous = self
                        .index
                        .checked_sub(1)
                        .and_then(|index| self.tokens.get(index))
                        .map_or(open.span, |token| token.span);
                    self.report(
                        previous,
                        Message::new(message!("Remove the ',' or add a value after it")),
                    );
                    break;
                }
                args.push(self.parse_expression(0));
                if self.is_punct(",") {
                    self.advance();
                    continue;
                }
                break;
            }
        }
        if self.is_punct(")") {
            let close = self.advance();
            return (args, Span::new(open.span.start, close.span.end));
        }
        self.report(
            open.span,
            Message::new(message!("'(' opened here is never closed")),
        );
        if !self.is_eof() && !self.is_punct(";") {
            let token = self.current();
            self.report(
                token.span,
                Message::with(
                    message!("Unexpected {0} inside the call"),
                    [Arg::from(quoted(self.text(token)))],
                ),
            );
            self.recover();
        }
        let end = if self.is_eof() {
            self.current().span.start
        } else {
            self.prev_end(open.span.end)
        };
        (args, Span::new(open.span.start, end.max(open.span.end)))
    }

    fn parse_primary(&mut self) -> Expr {
        loop {
            let token = self.current();
            let text = self.text(token);
            match token.kind {
                TokenKind::Number => {
                    self.advance();
                    return self.number(token, text);
                }
                TokenKind::String => {
                    self.advance();
                    return self.node(ExprKind::String(lexer::string_value(text)), token.span);
                }
                TokenKind::Pointer => {
                    self.advance();
                    let pointer = match text {
                        "#" => Pointer::Element,
                        "#acc" => Pointer::Accumulator,
                        "#index" => Pointer::Index,
                        other => Pointer::Unknown(other.get(1..).unwrap_or_default().to_owned()),
                    };
                    return self.node(ExprKind::Pointer(pointer), token.span);
                }
                TokenKind::Ident => {
                    self.advance();
                    return self.node(ExprKind::Ident(text.to_owned()), token.span);
                }
                TokenKind::Keyword => return self.keyword(token, text),
                TokenKind::Punct => {
                    if matches!(text, ")" | "]" | "}") {
                        self.report(
                            token.span,
                            Message::with(
                                message!("'{0}' has no matching '{1}'"),
                                [Arg::from(text), Arg::from(opener_of(text))],
                            ),
                        );
                        self.advance();
                        continue;
                    }
                    return self.punct(token, text);
                }
                TokenKind::Operator => {
                    self.advance();
                    if text == "=" {
                        self.report_fix(
                            token.span,
                            Message::new(message!("'=' assigns; use '==' to compare")),
                            Some(SyntaxFix {
                                label: Message::new(message!("Use '=='")),
                                insert: String::from("=="),
                            }),
                        );
                    } else {
                        self.report(
                            token.span,
                            Message::with(
                                message!("'{0}' needs a value before it"),
                                [Arg::from(text)],
                            ),
                        );
                    }
                    return Self::missing(token.span.start);
                }
                TokenKind::Error => {
                    self.advance();
                    return Self::missing(token.span.end);
                }
                TokenKind::Eof => return Self::missing(token.span.start),
            }
        }
    }

    fn number(&mut self, token: Token, text: &str) -> Expr {
        let (digits, whole) = lexer::number_text(text);
        let value = decimal_value(&digits);
        if value.is_none() {
            self.report(
                token.span,
                Message::with(
                    message!(
                        "{0} does not fit in a formula number, which holds up to 28 significant digits"
                    ),
                    [Arg::from(text)],
                ),
            );
        }
        self.node(ExprKind::Number(Number { value, whole }), token.span)
    }

    fn keyword(&mut self, token: Token, text: &str) -> Expr {
        match text {
            "true" | "false" => {
                self.advance();
                self.node(ExprKind::Bool(text == "true"), token.span)
            }
            "nil" => {
                self.advance();
                self.node(ExprKind::Nil, token.span)
            }
            "if" => {
                let node = self.parse_if();
                self.report(
                    Span::new(node.span.start.min(token.span.start), token.span.end),
                    Message::new(message!(
                        "An 'if' that is part of a calculation needs parentheses around the whole 'if … else …'"
                    )),
                );
                node
            }
            "let" => {
                self.report(
                    token.span,
                    Message::new(message!(
                        "'let' lines come before the formula's amount, each ending with ';'"
                    )),
                );
                self.advance();
                Self::missing(token.span.end)
            }
            _ => {
                self.report(
                    token.span,
                    Message::with(message!("Unexpected '{0}' here"), [Arg::from(text)]),
                );
                self.advance();
                Self::missing(token.span.end)
            }
        }
    }

    fn punct(&mut self, token: Token, text: &str) -> Expr {
        match text {
            "(" => {
                self.advance();
                let mut inner = if self.is_punct(")") {
                    Self::missing(self.current().span.start)
                } else {
                    self.parse_sequence()
                };
                if inner.is_missing() {
                    self.report(token.span, Message::new(message!("'()' is empty")));
                }
                if self.is_punct(")") {
                    let close = self.advance();
                    inner.span = Span::new(token.span.start, close.span.end);
                    inner.parens = inner.parens.saturating_add(1);
                    return inner;
                }
                self.report(
                    token.span,
                    Message::new(message!("'(' opened here is never closed")),
                );
                inner
            }
            "[" => {
                self.advance();
                let mut items = Vec::new();
                while !self.is_punct("]") && !self.is_eof() {
                    items.push(self.parse_expression(0));
                    if self.is_punct(",") {
                        self.advance();
                    } else {
                        break;
                    }
                }
                let end = if self.is_punct("]") {
                    self.advance().span.end
                } else {
                    self.report(
                        token.span,
                        Message::new(message!("'[' opened here is never closed")),
                    );
                    self.prev_end(token.span.end).max(token.span.end)
                };
                self.node(ExprKind::Array(items), Span::new(token.span.start, end))
            }
            "{" => self.parse_braces(),
            "." if matches!(self.peek().kind, TokenKind::Ident | TokenKind::Keyword) => {
                let dot = self.advance();
                let property = self.advance();
                let pointer = self.node(
                    ExprKind::Pointer(Pointer::Implicit),
                    Span::new(dot.span.start, dot.span.start),
                );
                self.node(
                    ExprKind::Member(Member {
                        object: Box::new(pointer),
                        property: Name {
                            text: self.text(property).to_owned(),
                            span: property.span,
                        },
                        optional: false,
                    }),
                    Span::new(dot.span.start, property.span.end),
                )
            }
            _ => {
                self.report(
                    token.span,
                    Message::with(message!("Unexpected '{0}'"), [Arg::from(text)]),
                );
                self.advance();
                Self::missing(token.span.start)
            }
        }
    }

    fn parse_braces(&mut self) -> Expr {
        let open = self.advance();
        let first = self.current();
        let is_map = matches!(
            first.kind,
            TokenKind::Ident | TokenKind::String | TokenKind::Keyword | TokenKind::Number
        ) && self.text(self.peek()) == ":";
        if self.is_punct("}") {
            let close = self.advance();
            return self.node(
                ExprKind::Map(Vec::new()),
                Span::new(open.span.start, close.span.end),
            );
        }
        if !is_map {
            let body = self.parse_sequence();
            let end = if self.is_punct("}") {
                self.advance().span.end
            } else {
                self.report(
                    open.span,
                    Message::with(
                        message!("'{0}' opened here is never closed"),
                        [Arg::from("{")],
                    ),
                );
                body.span.end.max(open.span.end)
            };
            return self.node(
                ExprKind::Predicate(Predicate {
                    body: Box::new(body),
                    braces: true,
                }),
                Span::new(open.span.start, end),
            );
        }
        let mut entries = Vec::new();
        while !self.is_punct("}") && !self.is_eof() {
            let key = self.advance();
            if !self.is_op(":") {
                self.report(
                    key.span,
                    Message::new(message!("A map entry is written key: value")),
                );
                break;
            }
            self.advance();
            let key_text = self.text(key);
            let key = Name {
                text: if key.kind == TokenKind::String {
                    lexer::string_value(key_text)
                } else {
                    key_text.to_owned()
                },
                span: key.span,
            };
            let value = self.parse_expression(0);
            entries.push(MapEntry { key, value });
            if self.is_punct(",") {
                self.advance();
            } else {
                break;
            }
        }
        let end = if self.is_punct("}") {
            self.advance().span.end
        } else {
            self.report(
                open.span,
                Message::with(
                    message!("'{0}' opened here is never closed"),
                    [Arg::from("{")],
                ),
            );
            self.prev_end(open.span.end).max(open.span.end)
        };
        self.node(ExprKind::Map(entries), Span::new(open.span.start, end))
    }
}

/// The exact value of a number literal's digits, as [`lexer::number_text`] gives them, or `None`
/// when it does not fit in a decimal.
fn decimal_value(digits: &str) -> Option<Decimal> {
    let radix = match digits.get(..2) {
        Some("0x") => Some(16),
        Some("0o") => Some(8),
        Some("0b") => Some(2),
        _ => None,
    };
    if let Some(radix) = radix {
        let value = u128::from_str_radix(digits.get(2..)?, radix).ok()?;
        return Decimal::try_from_i128_with_scale(i128::try_from(value).ok()?, 0).ok();
    }
    let digits = if digits.starts_with('.') {
        format!("0{digits}")
    } else {
        digits.to_owned()
    };
    if digits.contains(['e', 'E']) {
        Decimal::from_scientific(&digits).ok()
    } else {
        Decimal::from_str_exact(&digits).ok()
    }
}
