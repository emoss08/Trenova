//! Types as written: names with `<…>` arguments, quotients (`Money/cwt`, `Weight/Volume`,
//! `lb/ft3`), record types and `?` optionals.

use super::Parser;
use crate::message;
use crate::message::Message;
use crate::span::Span;
use crate::tariff::syntax::ast::FieldType;
use crate::tariff::syntax::ast::Name;
use crate::tariff::syntax::ast::TypeExpr;
use crate::tariff::syntax::ast::TypeExprKind;
use crate::tariff::syntax::lexer;
use crate::tariff::syntax::token::TokenKind;

impl Parser<'_> {
    /// A type.
    pub(super) fn type_expr(&mut self) -> TypeExpr {
        let start = self.peek().span.start;
        if !self.enter() {
            return TypeExpr {
                span: Span::new(start, start),
                kind: TypeExprKind::Missing,
            };
        }
        let mut ty = self.base_type();
        while self.at("?") && !self.newline_before() && !self.stopped {
            let mark = self.bump().span;
            let span = ty.span.cover(mark);
            if matches!(ty.kind, TypeExprKind::Optional(_)) {
                self.error(
                    mark,
                    Message::new(message!("A type is optional once; one `?` is enough")),
                );
                continue;
            }
            ty = TypeExpr {
                span,
                kind: TypeExprKind::Optional(Box::new(ty)),
            };
        }
        self.leave();
        ty
    }

    pub(super) fn missing_type(&self) -> TypeExpr {
        let at = self.previous_span().end;
        TypeExpr {
            span: Span::new(at, at),
            kind: TypeExprKind::Missing,
        }
    }

    fn base_type(&mut self) -> TypeExpr {
        if self.at("{") {
            return self.record_type();
        }
        match self.kind() {
            kind if is_type_word(kind) => {
                let token = self.bump();
                let name = self.name_of(&token);
                if self.at("/") && !self.newline_before() {
                    return self.quotient(name);
                }
                if token.kind == TokenKind::Unit || lexer::is_unit(&name.text) {
                    self.mark_span(name.span, TokenKind::Unit);
                } else if token.kind == TokenKind::Name || token.kind == TokenKind::Code {
                    self.mark_span(name.span, TokenKind::TypeName);
                }
                let mut args = Vec::new();
                let mut end = name.span;
                if self.at("<") && !self.newline_before() {
                    let (parsed, close) = self.type_arguments();
                    args = parsed;
                    end = close;
                }
                TypeExpr {
                    span: name.span.cover(end),
                    kind: TypeExprKind::Named { name, args },
                }
            }
            _ => {
                self.expected(Message::new(message!(
                    "Expected a type, like `Money`, `Weight` or `List<Stop>`"
                )));
                self.missing_type()
            }
        }
    }

    fn quotient(&mut self, numerator: Name) -> TypeExpr {
        self.bump();
        if lexer::is_unit(&numerator.text) {
            self.mark_span(numerator.span, TokenKind::Unit);
        }
        let denominator = if is_type_word(self.kind()) {
            let token = self.bump();
            let name = self.name_of(&token);
            if lexer::is_unit(&name.text) {
                self.mark_span(name.span, TokenKind::Unit);
            }
            name
        } else {
            self.expected(Message::new(message!(
                "Expected a unit or a dimension after `/`, like `Money/cwt` or `Money/Weight`"
            )));
            self.missing_name()
        };
        let span = numerator.span.cover(denominator.span);
        TypeExpr {
            span,
            kind: TypeExprKind::Quotient {
                numerator,
                denominator,
            },
        }
    }

    fn type_arguments(&mut self) -> (Vec<TypeExpr>, Span) {
        let open = self.bump().span;
        let mut args = Vec::new();
        while !self.at(">") && !self.at_eof() && !self.at_statement_line() && !self.stopped {
            args.push(self.type_expr());
            if self.eat(",").is_none() && !self.at(">") {
                self.expected(Message::new(message!("Expected `>` after the type")));
                self.recover_in_list(">");
                break;
            }
        }
        if args.is_empty() {
            self.error(open, Message::new(message!("Expected a type inside `<…>`")));
        }
        let close = if let Some(close) = self.eat(">") {
            close
        } else {
            self.error(
                open,
                Message::new(message!("This `<` is missing its closing `>`")),
            );
            self.previous_span()
        };
        (args, close)
    }

    fn record_type(&mut self) -> TypeExpr {
        let open = self.bump().span;
        let mut fields = Vec::new();
        while !self.at("}") && !self.at_eof() && !self.at_statement_line() && !self.stopped {
            let docs = self.take_doc();
            let Some(name) = self.peek_is(1, ":").then(|| self.any_name()).flatten() else {
                self.expected(Message::new(message!(
                    "Expected a field, written `name: Type`, or a closing brace"
                )));
                self.recover_in_list("}");
                continue;
            };
            self.bump();
            let ty = self.type_expr();
            fields.push(FieldType { docs, name, ty });
            if self.eat(",").is_none() && !self.at("}") {
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
                Message::new(message!("This record type is missing its closing brace")),
            );
            self.previous_span()
        };
        if fields.is_empty() {
            self.error(
                open.cover(close),
                Message::new(message!("A record type needs at least one field")),
            );
        }
        TypeExpr {
            span: open.cover(close),
            kind: TypeExprKind::Record(fields),
        }
    }
}

fn is_type_word(kind: TokenKind) -> bool {
    matches!(
        kind,
        TokenKind::TypeName | TokenKind::Name | TokenKind::Code | TokenKind::Unit
    )
}
