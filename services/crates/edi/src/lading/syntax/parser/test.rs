//! Tests and properties.

use trenova_module_kit::message;
use trenova_module_kit::message::Message;
use trenova_module_kit::span::Span;

use super::Parser;
use crate::lading::syntax::ast::{
    Doc, Expr, ExprKind, Ident, PropertyDecl, TestDecl, TestStep, TextLit, Vary, VarySource,
};
use crate::lading::syntax::token::Tok;

/// The words `expect` takes before an outcome or action check.
const OUTCOMES: &[&str] = &[
    "held",
    "rejected",
    "declined",
    "warned",
    "escalated",
    "created",
    "updated",
    "cancelled",
    "applied",
    "ignored",
    "sent",
    "holds",
    "rejects",
    "declines",
    "warnings",
];

impl Parser<'_> {
    pub(super) fn test_decl(&mut self, doc: Doc) -> TestDecl {
        let start = self.token().span.start;
        self.eat_word("test");
        let name = self.text_lit("the test's name").unwrap_or_else(|| TextLit {
            value: String::new(),
            span: self.token().span,
        });
        let steps = self.steps(&mut Vec::new(), false);
        TestDecl {
            doc,
            name,
            steps,
            span: self.since(start),
        }
    }

    pub(super) fn property_decl(&mut self, doc: Doc) -> PropertyDecl {
        let start = self.token().span.start;
        self.eat_word("property");
        let name = self
            .text_lit("the property's name")
            .unwrap_or_else(|| TextLit {
                value: String::new(),
                span: self.token().span,
            });
        let mut varies = Vec::new();
        let steps = self.steps(&mut varies, true);
        PropertyDecl {
            doc,
            name,
            varies,
            steps,
            span: self.since(start),
        }
    }

    fn steps(&mut self, varies: &mut Vec<Vary>, allow_vary: bool) -> Vec<(TestStep, Span)> {
        let mut steps = Vec::new();
        if self.expect(Tok::LBrace, "{").is_none() {
            return steps;
        }
        loop {
            self.skip_newlines();
            if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                break;
            }
            let start = self.token().span.start;
            if allow_vary && self.at_word("vary") {
                if let Some(vary) = self.vary(start) {
                    varies.push(vary);
                }
            } else if let Some(step) = self.step() {
                steps.push((step, self.since(start)));
            }
            if !self.at_line_end() {
                self.error_here(Message::new(message!("Expected the end of the line")));
                self.skip_line();
            }
        }
        self.expect(Tok::RBrace, "}");
        steps
    }

    fn vary(&mut self, start: u32) -> Option<Vary> {
        self.eat_word("vary");
        let name = self.name("a name")?;
        self.expect_word("in");
        let source = if self.eat_word("samples").is_some() {
            VarySource::Samples(self.text_lit("a glob such as \"samples/**/*.x12\"")?)
        } else if self.at(Tok::LBracket) {
            VarySource::List(self.expr())
        } else if self.peek() == Tok::TypeName && self.peek_at(1) != Tok::LBrace {
            VarySource::Enum(self.type_name("an enum")?)
        } else {
            let low = self.additive_only();
            self.expect(Tok::DotDot, "..");
            let high = self.additive_only();
            VarySource::Range(low, high)
        };
        Some(Vary {
            name,
            source,
            span: self.since(start),
        })
    }

    fn step(&mut self) -> Option<TestStep> {
        if self.eat_word("given").is_some() {
            return self.given();
        }
        if self.at_word("expect") {
            return self.expect_step();
        }
        if self.eat_word("receive").is_some() {
            let file = if self.eat_word("file").is_some() {
                true
            } else {
                self.expect_word("text");
                false
            };
            let source = self.expr();
            return Some(TestStep::Receive { file, source });
        }
        if self.eat_word("respond").is_some() {
            if self.eat_word("accept").is_some() {
                return Some(TestStep::Respond {
                    accept: true,
                    reason: None,
                });
            }
            self.expect_word("decline");
            let reason = if self.at_line_end() {
                None
            } else {
                Some(self.expr())
            };
            return Some(TestStep::Respond {
                accept: false,
                reason,
            });
        }
        if self.eat_word("event").is_some() {
            let kind = self.type_name("an event such as StopArrived")?;
            let value = if self.at(Tok::LBrace) {
                Some(self.expr())
            } else {
                None
            };
            return Some(TestStep::Event { kind, value });
        }
        if self.eat_word("ack").is_some() {
            if self.eat_word("accepted").is_some() {
                return Some(TestStep::Ack(true));
            }
            self.expect_word("rejected");
            return Some(TestStep::Ack(false));
        }
        if self.eat_word("advance").is_some() {
            return Some(TestStep::Advance(self.expr()));
        }
        self.error_here(Message::new(message!(
            "Expected given, expect, receive, respond, event, ack or advance"
        )));
        self.skip_line();
        None
    }

    fn given(&mut self) -> Option<TestStep> {
        if self.eat_word("file").is_some() {
            return Some(TestStep::GivenFile(self.text_lit("a sample's path")?));
        }
        if self.eat_word("text").is_some() {
            return Some(TestStep::GivenText(self.expr()));
        }
        if self.eat_word("setting").is_some() {
            let name = self.name("a setting")?;
            self.expect(Tok::Assign, "=");
            let value = self.expr();
            return Some(TestStep::GivenSetting { name, value });
        }
        if self.eat_word("lookup").is_some() {
            let kind = self.type_name("a lookup kind")?;
            self.expect(Tok::LParen, "(");
            let code = self.expr();
            self.expect(Tok::RParen, ")");
            self.expect(Tok::Assign, "=");
            let label = if self.eat_word("none").is_some() {
                None
            } else {
                Some(self.expr())
            };
            return Some(TestStep::GivenLookup { kind, code, label });
        }
        if self.eat_word("now").is_some() {
            self.expect(Tok::Assign, "=");
            return Some(TestStep::GivenNow(self.expr()));
        }
        if self.peek() == Tok::Name {
            let document = self.name("a document")?;
            let value = self.expr();
            return Some(TestStep::GivenDocument { document, value });
        }
        self.error_here(Message::new(message!(
            "Expected file, text, setting, lookup, now or a document after given"
        )));
        None
    }

    fn expect_step(&mut self) -> Option<TestStep> {
        self.eat_word("expect");
        if self.at_word("x12") && matches!(self.peek_at(1), Tok::Name | Tok::EqEq) {
            self.eat_word("x12");
            let contains = if self.eat_word("contains").is_some() {
                true
            } else {
                self.expect(Tok::EqEq, "contains or ==")?;
                false
            };
            let text = self.expr();
            return Some(TestStep::ExpectX12 { contains, text });
        }
        let negated = self.at_word("no")
            && self.tokens.get(self.pos + 1).is_some_and(|token| {
                token.kind == Tok::Name && OUTCOMES.contains(&self.text_of(token.span))
            });
        if negated {
            self.eat_word("no");
        }
        let is_outcome = self.peek() == Tok::Name
            && OUTCOMES.contains(&self.current_text())
            && matches!(
                self.peek_at(1),
                Tok::Text | Tok::Newline | Tok::Eof | Tok::RBrace | Tok::Number | Tok::Digits
            );
        if negated || is_outcome {
            let token = self.advance();
            self.keywords.push(token.span);
            let what = Ident::new(self.text_of(token.span), token.span);
            let set = if what.name == "sent" && matches!(self.peek(), Tok::Number | Tok::Digits) {
                let set_token = self.advance();
                Some(Ident::new(self.text_of(set_token.span), set_token.span))
            } else {
                None
            };
            let detail = if self.at_line_end() {
                None
            } else if what.name == "sent" && matches!(self.current_text(), "accept" | "decline") {
                let token = self.advance();
                let word = self.text_of(token.span).to_owned();
                Some(Expr::new(ExprKind::Name(word), token.span))
            } else {
                Some(self.expr())
            };
            return Some(TestStep::ExpectOutcome {
                negated,
                what,
                set,
                detail,
            });
        }
        Some(TestStep::Expect(self.expr()))
    }
}
