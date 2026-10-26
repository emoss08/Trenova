//! Declarations: functions, crosswalks, constants, rules and the partner file's lines.

use trenova_module_kit::message;
use trenova_module_kit::message::Message;

use super::Parser;
use crate::lading::syntax::ast::{
    Body, ConstDecl, CrosswalkDecl, Direction, Doc, Expr, FnDecl, Ident, Interchange,
    InterchangeParty, Item, LetStmt, Pair, Param, RuleDecl, SettingDecl, TextLit, TypeExpr,
    TypeExprKind,
};
use crate::lading::syntax::token::Tok;

impl Parser<'_> {
    pub(super) fn fn_decl(&mut self, doc: Doc) -> FnDecl {
        let start = self.token().span.start;
        self.eat_word("fn");
        let name = self
            .name("the function's name")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        let mut params = Vec::new();
        if self.expect(Tok::LParen, "(").is_some() {
            loop {
                self.skip_newlines();
                if self.at(Tok::RParen) || self.at(Tok::Eof) {
                    break;
                }
                let Some(param) = self.name("a parameter") else {
                    self.skip_line();
                    break;
                };
                self.expect(Tok::Colon, ":");
                let ty = self.type_expr();
                let default = if self.eat(Tok::Assign).is_some() {
                    Some(self.expr())
                } else {
                    None
                };
                params.push(Param {
                    name: param,
                    ty,
                    default,
                });
                self.skip_newlines();
                if self.eat(Tok::Comma).is_none() {
                    break;
                }
            }
            self.expect(Tok::RParen, ")");
        }
        let ret = if self.expect(Tok::RArrow, "->").is_some() {
            self.type_expr()
        } else {
            TypeExpr {
                kind: TypeExprKind::Error,
                span: self.token().span,
            }
        };
        let body = if self.eat(Tok::Assign).is_some() {
            self.skip_newlines();
            Body {
                lets: Vec::new(),
                value: self.expr(),
            }
        } else if self.at(Tok::LBrace) {
            self.fn_block()
        } else {
            self.error_here(Message::new(message!("Expected = or {")));
            Body {
                lets: Vec::new(),
                value: Expr::error(self.token().span),
            }
        };
        FnDecl {
            doc,
            name,
            params,
            ret,
            body,
            span: self.since(start),
        }
    }

    /// `{ let ...; value }`
    pub(super) fn fn_block(&mut self) -> Body {
        self.advance();
        let mut lets: Vec<LetStmt> = Vec::new();
        let mut value = None;
        loop {
            self.skip_newlines();
            if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                break;
            }
            if self.at_word("let") {
                let start = self.token().span.start;
                self.eat_word("let");
                let name = self
                    .name("a name")
                    .unwrap_or_else(|| Ident::new("", self.token().span));
                self.expect(Tok::Assign, "=");
                let bound = self.expr();
                lets.push(LetStmt {
                    name,
                    value: bound,
                    span: self.since(start),
                });
                self.end_line_soft();
                continue;
            }
            if value.is_some() {
                self.error_here(Message::new(message!(
                    "A function body ends with one value"
                )));
                self.skip_line();
                continue;
            }
            value = Some(self.expr());
            self.end_line_soft();
        }
        let end = self.token().span;
        self.expect(Tok::RBrace, "}");
        Body {
            lets,
            value: value.unwrap_or_else(|| {
                self.error_at(
                    end,
                    Message::new(message!("A function body ends with its value")),
                );
                Expr::error(end)
            }),
        }
    }

    pub(super) fn crosswalk(&mut self, doc: Doc) -> CrosswalkDecl {
        let start = self.token().span.start;
        self.eat_word("crosswalk");
        let name = self
            .name("the crosswalk's name")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        self.expect(Tok::Colon, ":");
        let left = self.type_expr();
        self.expect(Tok::BiArrow, "<->");
        let right = self.type_expr();
        let mut pairs = Vec::new();
        let mut otherwise = Vec::new();
        if self.expect(Tok::LBrace, "{").is_some() {
            loop {
                self.skip_newlines();
                if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                    break;
                }
                if self.eat_word("else").is_some() {
                    let direction = self.direction();
                    let value = self.expr();
                    if let Some(direction) = direction {
                        otherwise.push((direction, value));
                    }
                    self.end_line_soft();
                    continue;
                }
                let pair_start = self.token().span.start;
                let left_values = self.pair_side();
                let Some(direction) = self.direction() else {
                    self.skip_line();
                    continue;
                };
                let right_values = self.pair_side();
                pairs.push(Pair {
                    left: left_values,
                    direction,
                    right: right_values,
                    span: self.since(pair_start),
                });
                self.end_line_soft();
            }
            self.expect(Tok::RBrace, "}");
        }
        CrosswalkDecl {
            doc,
            name,
            left,
            right,
            pairs,
            otherwise,
            span: self.since(start),
        }
    }

    fn direction(&mut self) -> Option<Direction> {
        if self.eat(Tok::BiArrow).is_some() {
            Some(Direction::Both)
        } else if self.eat(Tok::RArrow).is_some() {
            Some(Direction::Forward)
        } else if self.eat(Tok::LArrow).is_some() {
            Some(Direction::Backward)
        } else {
            self.error_here(Message::new(message!("Expected <->, -> or <-")));
            None
        }
    }

    fn pair_side(&mut self) -> Vec<Expr> {
        let mut values = vec![self.via_free_value()];
        while self.eat(Tok::Comma).is_some() {
            values.push(self.via_free_value());
        }
        values
    }

    /// A crosswalk value: a literal or enum variant, without reading an arrow as an operator.
    fn via_free_value(&mut self) -> Expr {
        self.expr_postfix_only()
    }

    pub(super) fn const_decl(&mut self, doc: Doc) -> ConstDecl {
        let start = self.token().span.start;
        self.eat_word("const");
        let name = self
            .name("the constant's name")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        self.expect(Tok::Assign, "=");
        let value = self.expr();
        self.end_line();
        ConstDecl {
            doc,
            name,
            value,
            span: self.since(start),
        }
    }

    pub(super) fn rule_decl(&mut self, doc: Doc) -> RuleDecl {
        let start = self.token().span.start;
        self.eat_word("rule");
        let title = self
            .text_lit("the rule's title")
            .unwrap_or_else(|| TextLit {
                value: String::new(),
                span: self.token().span,
            });
        let body = self.block();
        RuleDecl {
            doc,
            title,
            body,
            span: self.since(start),
        }
    }

    pub(super) fn partner_item(&mut self, doc: Doc) -> Item {
        let start = self.token().span.start;
        if self.eat_word("interchange").is_some() {
            return Item::Interchange(self.interchange(start));
        }
        if self.eat_word("zone").is_some() {
            let zone = self.zone_name();
            self.end_line();
            return zone.map_or_else(|| Item::Error(self.since(start)), Item::Zone);
        }
        if self.eat_word("currency").is_some() {
            let currency = self.text_lit("a currency code such as \"USD\"");
            self.end_line();
            return currency.map_or_else(|| Item::Error(self.since(start)), Item::Currency);
        }
        if self.eat_word("setting").is_some() {
            return Item::Setting(self.setting(doc, start));
        }
        self.error_here(Message::new(message!(
            "A partner file holds interchange, zone, currency and setting lines"
        )));
        self.skip_line();
        Item::Error(self.since(start))
    }

    /// A zone name: `America/Chicago`, `UTC`.
    pub(super) fn zone_name(&mut self) -> Option<Ident> {
        let start = self.token().span.start;
        if !matches!(self.peek(), Tok::TypeName | Tok::Name | Tok::X12) {
            self.error_here(Message::new(message!(
                "Expected a time zone such as America/Chicago"
            )));
            return None;
        }
        self.advance();
        while self.at(Tok::Slash) && self.adjacent_zone() {
            self.advance();
            if matches!(self.peek(), Tok::TypeName | Tok::Name | Tok::X12) {
                self.advance();
            }
        }
        let span = self.since(start);
        Some(Ident::new(self.text_of(span), span))
    }

    fn adjacent_zone(&self) -> bool {
        self.token().span.start == self.last_end()
    }

    fn setting(&mut self, doc: Doc, start: u32) -> SettingDecl {
        let name = self
            .name("the setting's name")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        self.expect(Tok::Colon, ":");
        let ty = self.type_expr();
        let default = if self.eat(Tok::Assign).is_some() {
            Some(self.comparison_free())
        } else {
            None
        };
        let range = if self.eat_word("in").is_some() {
            let low = self.comparison_free();
            self.expect(Tok::DotDot, "..");
            let high = self.comparison_free();
            Some((low, high))
        } else {
            None
        };
        self.end_line();
        SettingDecl {
            doc,
            name,
            ty,
            default,
            range,
            span: self.since(start),
        }
    }

    /// A value that stops before `in` and `..`.
    pub(super) fn comparison_free(&mut self) -> Expr {
        self.additive_only()
    }

    fn interchange(&mut self, start: u32) -> Interchange {
        let mut block = Interchange {
            them: None,
            us: None,
            group: None,
            version: None,
            separators: Vec::new(),
            usage: None,
            span: self.token().span,
        };
        if self.expect(Tok::LBrace, "{").is_none() {
            block.span = self.since(start);
            return block;
        }
        loop {
            self.skip_newlines();
            if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                break;
            }
            if self.eat_word("them").is_some() {
                block.them = self.party();
            } else if self.eat_word("us").is_some() {
                block.us = self.party();
            } else if self.eat_word("group").is_some() {
                let theirs = self.text_lit("their application code");
                self.expect(Tok::BiArrow, "<->");
                let ours = self.text_lit("our application code");
                if let (Some(theirs), Some(ours)) = (theirs, ours) {
                    block.group = Some((theirs, ours));
                }
            } else if self.eat_word("version").is_some() {
                if matches!(self.peek(), Tok::Digits | Tok::Number) {
                    let token = self.advance();
                    block.version = Some(Ident::new(self.text_of(token.span), token.span));
                } else {
                    self.error_here(Message::new(message!("Expected a version such as 004010")));
                }
            } else if self.eat_word("separators").is_some() {
                while self.at(Tok::Text) {
                    if let Some(text) = self.text_lit("a separator") {
                        block.separators.push(text);
                    }
                }
            } else if self.eat_word("usage").is_some() {
                block.usage = self.name("test or production");
            } else {
                self.error_here(Message::new(message!(
                    "Expected them, us, group, version, separators or usage"
                )));
                self.skip_line();
                continue;
            }
            self.end_line_soft();
        }
        self.expect(Tok::RBrace, "}");
        self.end_line();
        block.span = self.since(start);
        block
    }

    fn party(&mut self) -> Option<InterchangeParty> {
        let qualifier = if matches!(self.peek(), Tok::X12 | Tok::Digits | Tok::Number) {
            let token = self.advance();
            Ident::new(self.text_of(token.span), token.span)
        } else {
            self.error_here(Message::new(message!(
                "Expected an ISA qualifier such as ZZ or 01"
            )));
            return None;
        };
        let id = self.text_lit("the interchange ID")?;
        Some(InterchangeParty { qualifier, id })
    }
}
