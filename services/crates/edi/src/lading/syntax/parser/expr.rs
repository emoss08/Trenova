//! Expressions and written types.

use trenova_module_kit::message;
use trenova_module_kit::message::Message;
use trenova_module_kit::span::Span;

use super::Parser;
use crate::lading::syntax::ast::{
    Arg, BinOp, Expr, ExprKind, FieldInit, Ident, MatchArm, MemberKind, TextPart, TypeExpr,
    TypeExprKind, UnOp,
};
use crate::lading::syntax::lexer::{TextPiece, lex_at, text_pieces};
use crate::lading::syntax::token::{Tok, is_reserved};

/// The units a number may carry.
pub const UNITS: &[&str] = &[
    "lb", "kg", "oz", "t", "ton", "in", "ft", "yd", "mi", "cm", "m", "km", "ft3", "m3", "gal", "l",
    "°F", "°C", "s", "min", "h", "d",
];

impl Parser<'_> {
    /// An expression.
    pub(super) fn expr(&mut self) -> Expr {
        if !self.enter() {
            let span = self.token().span;
            self.skip_line();
            return Expr::error(span);
        }
        let expr = self.default_expr();
        self.leave();
        expr
    }

    /// An expression a block follows (`when cond {`), where `Type {` is not a record.
    pub(super) fn condition(&mut self) -> Expr {
        let outer = self.no_record;
        self.no_record = true;
        let expr = self.expr();
        self.no_record = outer;
        expr
    }

    /// A postfix chain only (`stop.address.postal`), so what follows on the line stays unread.
    pub(super) fn expr_postfix_only(&mut self) -> Expr {
        self.postfix()
    }

    /// An arithmetic expression, stopping before comparisons, `in` and `..`.
    pub(super) fn additive_only(&mut self) -> Expr {
        self.additive()
    }

    fn binary(lhs: Expr, op: BinOp, rhs: Expr) -> Expr {
        let span = lhs.span.cover(rhs.span);
        Expr::new(ExprKind::Binary(op, Box::new(lhs), Box::new(rhs)), span)
    }

    fn continue_line(&mut self) {
        if self.continuation_ahead(false) {
            self.skip_to_continuation();
        }
    }

    fn default_expr(&mut self) -> Expr {
        let lhs = self.or_expr();
        self.continue_line();
        if self.eat(Tok::QuestionQuestion).is_some() {
            self.skip_newlines_in_expr();
            let rhs = self.default_expr();
            return Self::binary(lhs, BinOp::Default, rhs);
        }
        lhs
    }

    /// After an operator, the expression may continue on the next line.
    fn skip_newlines_in_expr(&mut self) {
        while self.at(Tok::Newline) {
            self.advance();
        }
    }

    fn or_expr(&mut self) -> Expr {
        let mut lhs = self.and_expr();
        loop {
            self.continue_line();
            if self.eat_word("or").is_none() {
                return lhs;
            }
            self.skip_newlines_in_expr();
            let rhs = self.and_expr();
            lhs = Self::binary(lhs, BinOp::Or, rhs);
        }
    }

    fn and_expr(&mut self) -> Expr {
        let mut lhs = self.not_expr();
        loop {
            self.continue_line();
            if self.eat_word("and").is_none() {
                return lhs;
            }
            self.skip_newlines_in_expr();
            let rhs = self.not_expr();
            lhs = Self::binary(lhs, BinOp::And, rhs);
        }
    }

    fn not_expr(&mut self) -> Expr {
        if let Some(span) = self.eat_word("not") {
            let inner = self.not_expr();
            let span = span.cover(inner.span);
            return Expr::new(ExprKind::Unary(UnOp::Not, Box::new(inner)), span);
        }
        self.comparison()
    }

    fn comparison(&mut self) -> Expr {
        let lhs = self.via_expr();
        self.continue_line();
        let op = match self.peek() {
            Tok::EqEq => Some(BinOp::Eq),
            Tok::NotEq => Some(BinOp::Ne),
            Tok::Lt => Some(BinOp::Lt),
            Tok::Le => Some(BinOp::Le),
            Tok::Gt => Some(BinOp::Gt),
            Tok::Ge => Some(BinOp::Ge),
            Tok::Name if self.current_text() == "in" => Some(BinOp::In),
            _ => None,
        };
        let Some(op) = op else {
            return lhs;
        };
        if op == BinOp::In {
            self.eat_word("in");
        } else {
            self.advance();
        }
        self.skip_newlines_in_expr();
        let rhs = self.via_expr();
        Self::binary(lhs, op, rhs)
    }

    fn via_expr(&mut self) -> Expr {
        let mut value = self.additive();
        while self.at_word("via") {
            self.eat_word("via");
            let Some(crosswalk) = self.name("a crosswalk") else {
                return value;
            };
            let mut back = false;
            if self.at(Tok::Dot) && self.word_at(1, "back") {
                self.advance();
                self.eat_word("back");
                back = true;
            }
            let span = value.span.cover(self.since(crosswalk.span.start));
            value = Expr::new(
                ExprKind::Via {
                    value: Box::new(value),
                    crosswalk,
                    back,
                },
                span,
            );
        }
        value
    }

    fn additive(&mut self) -> Expr {
        let mut lhs = self.multiplicative();
        loop {
            self.continue_line();
            let op = match self.peek() {
                Tok::Plus => BinOp::Add,
                Tok::Minus => BinOp::Sub,
                _ => return lhs,
            };
            self.advance();
            self.skip_newlines_in_expr();
            let rhs = self.multiplicative();
            lhs = Self::binary(lhs, op, rhs);
        }
    }

    fn multiplicative(&mut self) -> Expr {
        let mut lhs = self.unary();
        loop {
            self.continue_line();
            let op = match self.peek() {
                Tok::Star => BinOp::Mul,
                Tok::Slash => BinOp::Div,
                _ => return lhs,
            };
            self.advance();
            self.skip_newlines_in_expr();
            let rhs = self.unary();
            lhs = Self::binary(lhs, op, rhs);
        }
    }

    fn unary(&mut self) -> Expr {
        if let Some(token) = self.eat(Tok::Minus) {
            let inner = self.unary();
            let span = token.span.cover(inner.span);
            return Expr::new(ExprKind::Unary(UnOp::Neg, Box::new(inner)), span);
        }
        if let Some(span) = self.eat_word("exists") {
            let inner = self.unary();
            let span = span.cover(inner.span);
            return Expr::new(ExprKind::Exists(Box::new(inner)), span);
        }
        self.postfix()
    }

    fn postfix(&mut self) -> Expr {
        let mut expr = self.primary();
        loop {
            if self.at(Tok::Newline) && self.continuation_ahead(false) {
                let mut index = self.pos;
                while self
                    .tokens
                    .get(index)
                    .is_some_and(|t| t.kind == Tok::Newline)
                {
                    index += 1;
                }
                if self.tokens.get(index).is_some_and(|t| t.kind == Tok::Dot) {
                    self.skip_to_continuation();
                } else {
                    return expr;
                }
            }
            match self.peek() {
                Tok::Dot => {
                    if self.word_at(1, "back") {
                        return expr;
                    }
                    self.advance();
                    expr = self.member(expr);
                }
                Tok::LParen if self.adjacent() => {
                    expr = self.call(expr);
                }
                Tok::LBracket if self.adjacent() => {
                    self.advance();
                    self.skip_newlines();
                    let index = self.expr();
                    self.skip_newlines();
                    self.expect(Tok::RBracket, "]");
                    let span = expr.span.cover(self.since(expr.span.start));
                    expr = Expr::new(
                        ExprKind::Index {
                            target: Box::new(expr),
                            index: Box::new(index),
                        },
                        span,
                    );
                }
                _ => return expr,
            }
        }
    }

    /// Whether the current token touches the previous one (`f(` and `x[`, not `x [`).
    fn adjacent(&self) -> bool {
        self.token().span.start == self.last_end()
    }

    fn member(&mut self, target: Expr) -> Expr {
        let token = self.token();
        let kind = match token.kind {
            Tok::Name => MemberKind::Name,
            Tok::X12 => MemberKind::X12,
            Tok::Digits | Tok::Number => MemberKind::Digits,
            Tok::TypeName => MemberKind::Type,
            _ => {
                self.error_here(Message::new(message!("Expected a name after the dot")));
                return target;
            }
        };
        self.advance();
        let member = Ident::new(self.text_of(token.span), token.span);
        let span = target.span.cover(token.span);
        Expr::new(
            ExprKind::Member {
                target: Box::new(target),
                member,
                kind,
            },
            span,
        )
    }

    fn call(&mut self, callee: Expr) -> Expr {
        self.advance();
        let mut args = Vec::new();
        loop {
            self.skip_newlines();
            if self.at(Tok::RParen) || self.at(Tok::Eof) {
                break;
            }
            let name = if matches!(self.peek(), Tok::Name) && self.peek_at(1) == Tok::Colon {
                let token = self.advance();
                self.advance();
                Some(Ident::new(self.text_of(token.span), token.span))
            } else {
                None
            };
            let value = self.expr();
            args.push(Arg { name, value });
            self.skip_newlines();
            if self.eat(Tok::Comma).is_none() {
                break;
            }
        }
        self.skip_newlines();
        self.expect(Tok::RParen, ")");
        let span = callee.span.cover(self.since(callee.span.start));
        Expr::new(
            ExprKind::Call {
                callee: Box::new(callee),
                args,
            },
            span,
        )
    }

    fn primary(&mut self) -> Expr {
        let token = self.token();
        let text = self.text_of(token.span).to_owned();
        match token.kind {
            Tok::Number => self.number(),
            Tok::Money => {
                self.advance();
                let digits = text.trim_start_matches('$').replace('_', "");
                Expr::new(ExprKind::Money(digits), token.span)
            }
            Tok::Text => {
                self.advance();
                self.text(&text, token.span)
            }
            Tok::RawText => {
                self.advance();
                let raw = text
                    .strip_prefix("r\"")
                    .and_then(|rest| rest.strip_suffix('"'))
                    .unwrap_or_default()
                    .to_owned();
                Expr::new(ExprKind::Raw(raw), token.span)
            }
            Tok::Date => {
                self.advance();
                Expr::new(ExprKind::Date(text[1..].to_owned()), token.span)
            }
            Tok::Time => {
                self.advance();
                Expr::new(ExprKind::Time(text[1..].to_owned()), token.span)
            }
            Tok::Instant => {
                self.advance();
                Expr::new(ExprKind::Instant(text[1..].to_owned()), token.span)
            }
            Tok::X12 => {
                self.advance();
                Expr::new(ExprKind::X12(text), token.span)
            }
            Tok::Digits => {
                self.advance();
                Expr::new(ExprKind::Digits(text), token.span)
            }
            Tok::TypeName => {
                self.advance();
                if self.at(Tok::LBrace) && !self.no_record {
                    let ty = Ident::new(text, token.span);
                    return self.record(Some(ty), token.span.start);
                }
                Expr::new(ExprKind::Type(text), token.span)
            }
            Tok::LParen => {
                self.advance();
                self.skip_newlines();
                let outer = self.no_record;
                self.no_record = false;
                let inner = self.expr();
                self.no_record = outer;
                self.skip_newlines();
                self.expect(Tok::RParen, ")");
                Expr::new(inner.kind, self.since(token.span.start))
            }
            Tok::LBracket => self.list(),
            Tok::LBrace => self.record(None, token.span.start),
            Tok::Name => self.word(&text, token.span),
            _ => {
                self.error_here(Message::new(message!("Expected a value")));
                if !self.at_line_end() {
                    self.advance();
                }
                Expr::error(token.span)
            }
        }
    }

    fn number(&mut self) -> Expr {
        let token = self.advance();
        let digits = self.text_of(token.span).replace('_', "");
        if self.at(Tok::Percent) && self.adjacent() {
            self.advance();
            return Expr::new(ExprKind::Percent(digits), self.since(token.span.start));
        }
        if self.peek() == Tok::Name {
            let unit = self.current_text().to_owned();
            let unit_is_in = unit == "in";
            let next = self.peek_at(1);
            let in_as_unit = !unit_is_in
                || matches!(
                    next,
                    Tok::Newline
                        | Tok::Eof
                        | Tok::RParen
                        | Tok::RBracket
                        | Tok::RBrace
                        | Tok::Comma
                        | Tok::DotDot
                        | Tok::Plus
                        | Tok::Minus
                        | Tok::Star
                        | Tok::Slash
                        | Tok::EqEq
                        | Tok::NotEq
                        | Tok::Lt
                        | Tok::Le
                        | Tok::Gt
                        | Tok::Ge
                );
            if UNITS.contains(&unit.as_str()) && in_as_unit {
                self.advance();
                return Expr::new(
                    ExprKind::Quantity(digits, unit),
                    self.since(token.span.start),
                );
            }
        }
        Expr::new(ExprKind::Number(digits), token.span)
    }

    fn text(&mut self, source: &str, span: Span) -> Expr {
        let mut parts = Vec::new();
        for piece in text_pieces(source, span.start) {
            match piece {
                TextPiece::Literal(text) => parts.push(TextPart::Literal(text)),
                TextPiece::Interpolation(inner, offset) => {
                    parts.push(TextPart::Expr(self.interpolation(&inner, offset)));
                }
            }
        }
        Expr::new(ExprKind::Text(parts), span)
    }

    fn interpolation(&mut self, inner: &str, offset: u32) -> Expr {
        let lexed = lex_at(inner, offset);
        let mut sub = Parser {
            source: self.source,
            tokens: lexed.tokens,
            comments: Vec::new(),
            pos: 0,
            diagnostics: lexed.diagnostics,
            keywords: Vec::new(),
            depth: self.depth,
            doc_floor: 0,
            no_record: false,
        };
        let expr = sub.expr();
        if !sub.at(Tok::Eof) {
            sub.error_here(Message::new(message!(
                "Expected the end of the interpolation"
            )));
        }
        self.diagnostics.append(&mut sub.diagnostics);
        self.keywords.append(&mut sub.keywords);
        expr
    }

    fn list(&mut self) -> Expr {
        let start = self.advance().span.start;
        let mut items = Vec::new();
        loop {
            self.skip_newlines();
            if self.at(Tok::RBracket) || self.at(Tok::Eof) {
                break;
            }
            items.push(self.expr());
            self.skip_newlines();
            if self.eat(Tok::Comma).is_none() {
                break;
            }
        }
        self.skip_newlines();
        self.expect(Tok::RBracket, "]");
        Expr::new(ExprKind::List(items), self.since(start))
    }

    /// `Type { field <- v }` or `{ name: v, ... }`.
    fn record(&mut self, ty: Option<Ident>, start: u32) -> Expr {
        self.expect(Tok::LBrace, "{");
        let mut fields = Vec::new();
        loop {
            self.skip_newlines();
            if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                break;
            }
            let Some(name) = self.ident("a field") else {
                self.skip_line();
                continue;
            };
            if self.eat(Tok::Colon).is_none() && self.eat(Tok::LArrow).is_none() {
                self.error_here(Message::new(message!("Expected : or <- after the field")));
                self.skip_line();
                continue;
            }
            self.skip_newlines();
            let value = self.expr();
            let otherwise = self.else_clause();
            fields.push(FieldInit {
                name,
                value,
                otherwise,
            });
            if self.eat(Tok::Comma).is_none() && !self.at_line_end() {
                self.error_here(Message::new(message!("Expected , or the end of the line")));
                self.skip_line();
            }
        }
        self.skip_newlines();
        self.expect(Tok::RBrace, "}");
        Expr::new(ExprKind::Record { ty, fields }, self.since(start))
    }

    fn word(&mut self, word: &str, span: Span) -> Expr {
        match word {
            "true" | "false" => {
                self.eat_word(word);
                Expr::new(ExprKind::Bool(word == "true"), span)
            }
            "none" => {
                self.eat_word(word);
                Expr::new(ExprKind::None, span)
            }
            "if" => self.if_expr(),
            "match" => self.match_expr(),
            "lookup" => self.lookup(),
            "each" => self.each_expr(),
            _ => {
                if self.peek_at(1) == Tok::FatArrow {
                    self.advance();
                    self.advance();
                    let body = self.expr();
                    let full = span.cover(body.span);
                    return Expr::new(
                        ExprKind::Lambda {
                            param: Ident::new(word, span),
                            body: Box::new(body),
                        },
                        full,
                    );
                }
                if is_reserved(word) {
                    self.error_here(Message::new(message!("Expected a value")));
                    self.advance();
                    return Expr::error(span);
                }
                self.advance();
                Expr::new(ExprKind::Name(word.to_owned()), span)
            }
        }
    }

    fn if_expr(&mut self) -> Expr {
        let start = self.token().span.start;
        self.eat_word("if");
        let cond = self.expr();
        self.skip_newlines();
        self.expect_word("then");
        self.skip_newlines();
        let then = self.expr();
        self.skip_newlines();
        self.expect_word("else");
        self.skip_newlines();
        let otherwise = self.expr();
        Expr::new(
            ExprKind::If {
                cond: Box::new(cond),
                then: Box::new(then),
                otherwise: Box::new(otherwise),
            },
            self.since(start),
        )
    }

    fn match_expr(&mut self) -> Expr {
        let start = self.token().span.start;
        self.eat_word("match");
        let value = self.or_expr();
        self.expect(Tok::LBrace, "{");
        let mut arms = Vec::new();
        let mut otherwise = None;
        loop {
            self.skip_newlines();
            if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                break;
            }
            if self.eat_word("else").is_some() {
                self.expect(Tok::FatArrow, "=>");
                otherwise = Some(Box::new(self.expr()));
                self.eat(Tok::Comma);
                continue;
            }
            let mut patterns = vec![self.or_expr()];
            while self.eat(Tok::Pipe).is_some() || self.eat(Tok::Comma).is_some() {
                patterns.push(self.or_expr());
            }
            if self.expect(Tok::FatArrow, "=>").is_none() {
                self.skip_line();
                continue;
            }
            let value = self.expr();
            arms.push(MatchArm { patterns, value });
            self.eat(Tok::Comma);
        }
        self.expect(Tok::RBrace, "}");
        Expr::new(
            ExprKind::Match {
                value: Box::new(value),
                arms,
                otherwise,
            },
            self.since(start),
        )
    }

    fn lookup(&mut self) -> Expr {
        let start = self.token().span.start;
        self.eat_word("lookup");
        let kind = self
            .type_name("a lookup kind, such as Customer or Location")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        self.expect(Tok::LParen, "(");
        let code = self.expr();
        self.expect(Tok::RParen, ")");
        let learn = if self.eat_word("learn").is_some() {
            Some(Box::new(self.expr()))
        } else {
            None
        };
        Expr::new(
            ExprKind::Lookup {
                kind,
                code: Box::new(code),
                learn,
            },
            self.since(start),
        )
    }

    fn each_expr(&mut self) -> Expr {
        let start = self.token().span.start;
        self.eat_word("each");
        let outer = self.no_record;
        self.no_record = true;
        let list = self.or_expr();
        self.no_record = outer;
        self.expect_word("as");
        let binder = self
            .name("a name for each item")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        let body = self.block();
        Expr::new(
            ExprKind::Each {
                list: Box::new(list),
                binder,
                body,
            },
            self.since(start),
        )
    }

    /// A written type: `Text`, `Code<S502>`, `[Stop]`, `[Text: Text]`, `Weight?`.
    pub(super) fn type_expr(&mut self) -> TypeExpr {
        let start = self.token().span.start;
        let mut ty = if self.eat(Tok::LBracket).is_some() {
            let inner = self.type_expr();
            let kind = if self.eat(Tok::Colon).is_some() {
                let value = self.type_expr();
                if !matches!(&inner.kind, TypeExprKind::Named(name, args) if name == "Text" && args.is_empty())
                {
                    self.error_at(inner.span, Message::new(message!("A map's keys are Text")));
                }
                TypeExprKind::Map(Box::new(value))
            } else {
                TypeExprKind::List(Box::new(inner))
            };
            self.expect(Tok::RBracket, "]");
            TypeExpr {
                kind,
                span: self.since(start),
            }
        } else if self.peek() == Tok::TypeName {
            let token = self.advance();
            let name = self.text_of(token.span).to_owned();
            let mut args = Vec::new();
            if self.eat(Tok::Lt).is_some() {
                loop {
                    if matches!(
                        self.peek(),
                        Tok::X12 | Tok::Digits | Tok::TypeName | Tok::Name
                    ) {
                        let arg = self.advance();
                        args.push(self.text_of(arg.span).to_owned());
                    } else {
                        self.error_here(Message::new(message!("Expected a type argument")));
                        break;
                    }
                    if self.eat(Tok::Comma).is_none() {
                        break;
                    }
                }
                self.expect(Tok::Gt, ">");
            }
            TypeExpr {
                kind: TypeExprKind::Named(name, args),
                span: self.since(start),
            }
        } else {
            self.error_here(Message::new(message!("Expected a type")));
            TypeExpr {
                kind: TypeExprKind::Error,
                span: self.token().span,
            }
        };
        while self.at(Tok::Question) && self.adjacent() {
            self.advance();
            ty = TypeExpr {
                kind: TypeExprKind::Optional(Box::new(ty)),
                span: self.since(start),
            };
        }
        ty
    }
}
