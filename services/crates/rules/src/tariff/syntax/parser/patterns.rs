//! `match` patterns: literals, ranges with literal ends (or variant ends, `Mon..=Fri`), enum
//! variants, `none`, `_`, bindings, record shapes and `|` alternatives.

use super::Parser;
use crate::span::Span;
use crate::tariff::syntax::ast::Expr;
use crate::tariff::syntax::ast::ExprKind;
use crate::tariff::syntax::ast::Pattern;
use crate::tariff::syntax::ast::PatternKind;
use crate::tariff::syntax::ast::TextPart;
use crate::tariff::syntax::ast::UnaryOp;
use crate::tariff::syntax::token::TokenKind;
use trenova_module_kit::message;
use trenova_module_kit::message::Message;

impl Parser<'_> {
    /// A pattern and how deep it is.
    pub(super) fn pattern(&mut self) -> (Pattern, u8) {
        let start = self.peek().span.start;
        if !self.enter() {
            return (
                Pattern {
                    span: Span::new(start, start),
                    kind: PatternKind::Missing,
                },
                1,
            );
        }
        let (first, mut height) = self.alternative();
        let mut alternatives = vec![first];
        while self.at("|") && !self.stopped {
            self.bump();
            let (next, next_height) = self.alternative();
            height = height.max(next_height);
            alternatives.push(next);
        }
        self.leave();
        if alternatives.len() == 1
            && let Some(only) = alternatives.pop()
        {
            return (only, height);
        }
        let span = alternatives
            .iter()
            .fold(Span::new(start, start), |span, alt| span.cover(alt.span));
        (
            Pattern {
                span,
                kind: PatternKind::Or(alternatives),
            },
            height.saturating_add(1),
        )
    }

    fn alternative(&mut self) -> (Pattern, u8) {
        let token = self.peek();
        let simple = |kind| {
            (
                Pattern {
                    span: token.span,
                    kind,
                },
                1,
            )
        };
        if self.at("_") {
            self.bump();
            return simple(PatternKind::Wildcard);
        }
        if self.at_word("none") {
            self.bump();
            return simple(PatternKind::None);
        }
        if self.at("..") || self.at("..=") {
            return self.range_pattern(None);
        }
        if self.starts_literal() {
            let literal = self.pattern_literal();
            if (self.at("..") || self.at("..=")) && !self.newline_before() {
                return self.range_pattern(Some(literal));
            }
            return (
                Pattern {
                    span: literal.span,
                    kind: PatternKind::Literal(literal),
                },
                3,
            );
        }
        match token.kind {
            TokenKind::TypeName if self.variant_range_ahead() => {
                let start = self.path();
                self.range_pattern(Some(start))
            }
            TokenKind::TypeName | TokenKind::Code => self.variant_pattern(),
            TokenKind::Name => {
                self.bump();
                let name = self.name_of(&token);
                simple(PatternKind::Binding(name))
            }
            TokenKind::Punct if self.at("{") => self.record_pattern(),
            TokenKind::Error => {
                self.bump();
                simple(PatternKind::Missing)
            }
            _ => {
                self.expected(Message::new(message!(
                    "Expected a pattern: a value, a range, a variant, `none` or `_`"
                )));
                let at = self.previous_span().end;
                (
                    Pattern {
                        span: Span::new(at, at),
                        kind: PatternKind::Missing,
                    },
                    1,
                )
            }
        }
    }

    /// Whether the variant at the cursor (`Mon`, `Weekday.Mon`) starts a range: `..` or `..=`
    /// follows it on the same line.
    fn variant_range_ahead(&self) -> bool {
        let after = if self.peek_is(1, ".") && self.peek_nth(2).kind == TokenKind::TypeName {
            3
        } else {
            1
        };
        (self.peek_is(after, "..") || self.peek_is(after, "..=")) && !self.newline_before_nth(after)
    }

    /// Whether the cursor starts a literal a pattern can hold.
    fn starts_literal(&self) -> bool {
        match self.kind() {
            TokenKind::Number
            | TokenKind::Money
            | TokenKind::Percent
            | TokenKind::Date
            | TokenKind::Text => true,
            TokenKind::Keyword => self.at_word("true") || self.at_word("false"),
            TokenKind::Punct => {
                self.at("-")
                    && matches!(
                        self.peek_nth(1).kind,
                        TokenKind::Number | TokenKind::Money | TokenKind::Percent
                    )
            }
            _ => false,
        }
    }

    /// A literal, or a negated number, money or percent.
    fn pattern_literal(&mut self) -> Expr {
        if self.at("-") {
            let op = self.bump().span;
            let operand = self.pattern_literal();
            let span = op.cover(operand.span);
            return self.node_with(
                ExprKind::Unary {
                    op: UnaryOp::Neg,
                    op_span: op,
                    operand: Box::new(operand),
                },
                span,
                1,
            );
        }
        let expr = self.expr_primary_literal();
        if let ExprKind::Text(text) = &expr.kind
            && text
                .parts
                .iter()
                .any(|part| matches!(part, TextPart::Expr(_)))
        {
            self.error(
                expr.span,
                Message::new(message!("Text in a pattern can't interpolate values")),
            );
        }
        expr
    }

    fn expr_primary_literal(&mut self) -> Expr {
        if self.at_word("true") || self.at_word("false") {
            let token = self.bump();
            let value = self.slice(token.span) == "true";
            return self.leaf(ExprKind::Bool(value), token.span);
        }
        self.primary()
    }

    fn range_pattern(&mut self, start: Option<Expr>) -> (Pattern, u8) {
        let op = self.bump();
        let inclusive = self.slice(op.span) == "..=";
        let end = if self.newline_before() {
            None
        } else if self.starts_literal() {
            Some(self.pattern_literal())
        } else if self.kind() == TokenKind::TypeName {
            Some(self.path())
        } else {
            None
        };
        if start.is_none() && end.is_none() {
            self.error(
                op.span,
                Message::new(message!("A range needs a start, an end or both")),
            );
        }
        let mut span = op.span;
        for side in [&start, &end].into_iter().flatten() {
            span = span.cover(side.span);
        }
        (
            Pattern {
                span,
                kind: PatternKind::Range {
                    start,
                    end,
                    inclusive,
                },
            },
            3,
        )
    }

    fn variant_pattern(&mut self) -> (Pattern, u8) {
        let token = self.bump();
        let head = self.name_of(&token);
        if self.at(".") && self.word_kind_at(1) {
            self.bump();
            if let Some(variant) = self.any_name() {
                let span = head.span.cover(variant.span);
                return (
                    Pattern {
                        span,
                        kind: PatternKind::Variant {
                            enumeration: Some(head),
                            variant,
                        },
                    },
                    1,
                );
            }
        }
        (
            Pattern {
                span: head.span,
                kind: PatternKind::Variant {
                    enumeration: None,
                    variant: head,
                },
            },
            1,
        )
    }

    fn record_pattern(&mut self) -> (Pattern, u8) {
        let open = self.bump().span;
        let mut fields = Vec::new();
        let mut height = 1u8;
        while !self.at("}") && !self.at_eof() && !self.at_statement_line() && !self.stopped {
            let Some(name) = self.peek_is(1, ":").then(|| self.any_name()).flatten() else {
                self.expected(Message::new(message!(
                    "Expected a field, written `name: pattern`, or a closing brace"
                )));
                self.recover_in_list("}");
                continue;
            };
            self.bump();
            let (pattern, pattern_height) = self.pattern();
            height = height.max(pattern_height);
            fields.push((name, pattern));
            if self.eat(",").is_none() && !self.at_list_end("}") {
                self.expected(Message::new(message!(
                    "Expected `,` between the fields, or a closing brace"
                )));
                self.recover_in_list("}");
            }
        }
        let close = if let Some(close) = self.eat("}") {
            close
        } else {
            self.error(
                open,
                Message::new(message!("This pattern is missing its closing brace")),
            );
            self.previous_span()
        };
        if fields.is_empty()
            && self
                .slice(Span::new(open.end, close.start))
                .trim()
                .is_empty()
        {
            self.error(
                open.cover(close),
                Message::new(message!("A record pattern needs at least one field")),
            );
        }
        (
            Pattern {
                span: open.cover(close),
                kind: PatternKind::Record(fields),
            },
            height.saturating_add(2),
        )
    }
}
