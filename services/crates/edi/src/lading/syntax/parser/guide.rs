//! Lines of a guide file.

use trenova_module_kit::message;
use trenova_module_kit::message::Message;

use super::Parser;
use crate::lading::syntax::ast::{
    Doc, ElementRule, GuideDecl, GuideElement, Ident, Item, StructureRule,
};
use crate::lading::syntax::token::Tok;

impl Parser<'_> {
    pub(super) fn guide_item(&mut self, doc: Doc) -> Item {
        let start = self.token().span.start;
        if self.eat_word("loop").is_some() {
            let path = self.guide_path(false);
            let rules = self.structure_rules();
            self.end_line();
            return Item::Guide(GuideDecl::Loop {
                doc,
                path,
                rules,
                span: self.since(start),
            });
        }
        if self.eat_word("severity").is_some() {
            let kind_start = self.token().span.start;
            while !self.at_line_end() && !self.at_last_word_before_line_end() {
                self.advance();
            }
            let kind_span = self.since(kind_start);
            let kind = Ident::new(self.text_of(kind_span), kind_span);
            let level = self
                .name("error, warning or info")
                .unwrap_or_else(|| Ident::new("", self.token().span));
            self.end_line();
            return Item::Guide(GuideDecl::Severity {
                kind,
                level,
                span: self.since(start),
            });
        }
        if matches!(self.peek(), Tok::X12 | Tok::Digits) {
            let path = self.guide_path(true);
            let rules = self.structure_rules();
            let elements = if self.at(Tok::LBrace) {
                self.guide_elements()
            } else {
                Vec::new()
            };
            let note = if self.at(Tok::Text) {
                self.text_lit("a note")
            } else {
                None
            };
            self.end_line();
            return Item::Guide(GuideDecl::Segment {
                doc,
                path,
                rules,
                elements,
                note,
                span: self.since(start),
            });
        }
        self.error_here(Message::new(message!(
            "A guide line names a segment (0300.S5), a loop (loop 0300) or a severity"
        )));
        self.skip_line();
        Item::Error(self.since(start))
    }

    /// Whether the current token is the last word on its line.
    fn at_last_word_before_line_end(&self) -> bool {
        self.peek() == Tok::Name
            && matches!(
                self.peek_at(1),
                Tok::Newline | Tok::Eof | Tok::Semi | Tok::RBrace
            )
    }

    /// `0300.0310.N4` (a segment at the end) or `0300.0310` (loops only).
    fn guide_path(&mut self, segment: bool) -> Vec<Ident> {
        let mut path = Vec::new();
        loop {
            match self.peek() {
                Tok::Digits | Tok::Number => {
                    let token = self.advance();
                    path.push(Ident::new(self.text_of(token.span), token.span));
                }
                Tok::X12 if segment => {
                    let token = self.advance();
                    path.push(Ident::new(self.text_of(token.span), token.span));
                    break;
                }
                _ => {
                    self.error_here(if segment {
                        Message::new(message!("Expected a loop ID or a segment ID"))
                    } else {
                        Message::new(message!("Expected a loop ID"))
                    });
                    break;
                }
            }
            if self.eat(Tok::Dot).is_none() {
                break;
            }
        }
        path
    }

    fn structure_rules(&mut self) -> Vec<(StructureRule, trenova_module_kit::span::Span)> {
        let mut rules = Vec::new();
        loop {
            let start = self.token().span.start;
            let rule = if self.eat_word("required").is_some() {
                StructureRule::Required
            } else if self.eat_word("unused").is_some() {
                StructureRule::Unused
            } else if self.eat_word("max").is_some() {
                let Some((value, _)) = self.uint("a number") else {
                    break;
                };
                StructureRule::Max(value)
            } else if self.eat_word("min").is_some() {
                let Some((value, _)) = self.uint("a number") else {
                    break;
                };
                StructureRule::Min(value)
            } else if self.eat_word("syntax").is_some() {
                let mut notes = Vec::new();
                while self.peek() == Tok::X12 {
                    let token = self.advance();
                    notes.push(Ident::new(self.text_of(token.span), token.span));
                }
                if notes.is_empty() {
                    self.error_here(Message::new(message!(
                        "Expected a syntax note such as P0304"
                    )));
                }
                StructureRule::Syntax(notes)
            } else {
                break;
            };
            rules.push((rule, self.since(start)));
        }
        rules
    }

    fn guide_elements(&mut self) -> Vec<GuideElement> {
        self.advance();
        let mut elements = Vec::new();
        loop {
            self.skip_newlines();
            if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                break;
            }
            let start = self.token().span.start;
            let Some((position, _)) = self.uint("an element position such as 01") else {
                self.skip_line();
                continue;
            };
            let mut component = None;
            if self.at(Tok::Dot) {
                self.advance();
                component = self.uint("a component position").map(|(value, _)| value);
            }
            let mut rules = Vec::new();
            loop {
                let rule_start = self.token().span.start;
                let rule = if self.eat_word("required").is_some() {
                    ElementRule::Required
                } else if self.eat_word("unused").is_some() {
                    ElementRule::Unused
                } else if self.eat_word("codes").is_some() {
                    let mut codes = Vec::new();
                    loop {
                        if let Some(code) = self.text_lit("a code") {
                            codes.push(code);
                        }
                        if self.eat(Tok::Comma).is_none() {
                            break;
                        }
                    }
                    ElementRule::Codes(codes)
                } else if self.eat_word("length").is_some() {
                    let low = self.uint("a length").map_or(0, |(value, _)| value);
                    self.expect(Tok::DotDot, "..");
                    let high = self.uint("a length").map_or(0, |(value, _)| value);
                    ElementRule::Length(low, high)
                } else if self.eat_word("repeat").is_some() {
                    ElementRule::Repeat(self.uint("a number").map_or(1, |(value, _)| value))
                } else if self.eat_word("fixed").is_some() {
                    match self.text_lit("the fixed value") {
                        Some(value) => ElementRule::Fixed(value),
                        None => break,
                    }
                } else {
                    break;
                };
                rules.push((rule, self.since(rule_start)));
            }
            let note = if self.at(Tok::Text) {
                self.text_lit("a note")
            } else {
                None
            };
            elements.push(GuideElement {
                position,
                component,
                rules,
                note,
                span: self.since(start),
            });
            if !self.at_line_end() {
                self.error_here(Message::new(message!(
                    "Expected required, unused, codes, length, repeat, fixed or a note"
                )));
                self.skip_line();
            }
        }
        self.expect(Tok::RBrace, "}");
        elements
    }
}
