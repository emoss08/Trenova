//! Clauses of a flow file.

use trenova_module_kit::message;
use trenova_module_kit::message::Message;
use trenova_module_kit::span::Span;

use super::Parser;
use crate::lading::syntax::ast::{
    Action, ExpectAck, FlowDecl, Ident, Item, Otherwise, Receive, ReceiveArm, Respond, Send,
};
use crate::lading::syntax::token::Tok;

/// Words that start a top-level line of a flow file, so a clause's continuation stops there.
const FLOW_STARTS: &[&str] = &[
    "receive",
    "respond",
    "send",
    "expect",
    "test",
    "property",
    "fn",
    "crosswalk",
    "const",
];

impl Parser<'_> {
    pub(super) fn flow_item(&mut self) -> Item {
        let start = self.token().span.start;
        if self.at_word("receive") {
            return Item::Flow(FlowDecl::Receive(self.receive()));
        }
        if self.at_word("respond") {
            return Item::Flow(FlowDecl::Respond(self.respond()));
        }
        if self.at_word("send") {
            return Item::Flow(FlowDecl::Send(self.send()));
        }
        if self.at_word("expect") {
            return Item::Flow(FlowDecl::Expect(self.expect_ack()));
        }
        self.error_here(Message::new(message!(
            "A flow holds receive, respond, send and expect clauses"
        )));
        self.skip_line();
        Item::Error(self.since(start))
    }

    /// Steps to the next line when it continues this clause (starts with one of `words`).
    fn clause_continues(&mut self, words: &[&str]) -> bool {
        if !self.at_line_end() || self.at(Tok::RBrace) {
            return !self.at_line_end();
        }
        let mut index = self.pos;
        while self
            .tokens
            .get(index)
            .is_some_and(|token| matches!(token.kind, Tok::Newline | Tok::Semi))
        {
            index += 1;
        }
        let Some(token) = self.tokens.get(index).copied() else {
            return false;
        };
        let continues = match token.kind {
            Tok::Name => {
                let word = self.text_of(token.span);
                !FLOW_STARTS.contains(&word) && words.contains(&word)
            }
            Tok::RArrow => words.contains(&"->"),
            _ => false,
        };
        if continues {
            self.pos = index;
        }
        continues
    }

    fn set_name(&mut self) -> Ident {
        if matches!(self.peek(), Tok::Number | Tok::Digits) {
            let token = self.advance();
            return Ident::new(self.text_of(token.span), token.span);
        }
        self.error_here(Message::new(message!(
            "Expected a transaction set such as 204"
        )));
        Ident::new("", self.token().span)
    }

    fn receive(&mut self) -> Receive {
        let start = self.token().span.start;
        self.eat_word("receive");
        let set = self.set_name();
        self.expect_word("as");
        let binder = self
            .name("a name for the received document")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        let mut receive = Receive {
            set,
            binder,
            map: None,
            rules: None,
            key: None,
            acknowledge: None,
            arms: Vec::new(),
            span: Span::default(),
        };
        let words = ["via", "key", "acknowledge", "on", "->"];
        while self.clause_continues(&words) {
            if self.eat_word("via").is_some() {
                receive.map = self.name("a map");
                if self.eat(Tok::Comma).is_some() {
                    self.expect_word("rules");
                    receive.rules = self.name("a rules file");
                }
            } else if self.eat_word("key").is_some() {
                receive.key = Some(self.expr());
            } else if self.eat_word("acknowledge").is_some() {
                self.expect_word("with");
                if matches!(self.peek(), Tok::Number | Tok::Digits) {
                    let token = self.advance();
                    receive.acknowledge = Some(Ident::new(self.text_of(token.span), token.span));
                } else if self.at_word("none") {
                    let span = self.eat_word("none").unwrap_or_default();
                    receive.acknowledge = Some(Ident::new("none", span));
                } else {
                    self.error_here(Message::new(message!("Expected 997, 999 or none")));
                }
            } else if self.at_word("on") || self.at(Tok::RArrow) {
                let arm_start = self.token().span.start;
                let purpose = if self.eat_word("on").is_some() {
                    self.type_name("a purpose such as Original")
                } else {
                    None
                };
                self.expect(Tok::RArrow, "->");
                let action = self.action();
                receive.arms.push(ReceiveArm {
                    purpose,
                    action,
                    span: self.since(arm_start),
                });
            } else {
                self.error_here(Message::new(message!(
                    "Expected via, key, acknowledge, on or ->"
                )));
                self.skip_line();
            }
        }
        receive.span = self.since(start);
        receive
    }

    fn action(&mut self) -> Action {
        let start = self.token().span.start;
        if self.eat_word("create").is_some() {
            return Action::Create(self.since(start));
        }
        if self.eat_word("apply").is_some() {
            return Action::Apply(self.since(start));
        }
        if self.eat_word("ignore").is_some() {
            return Action::Ignore(self.since(start));
        }
        if self.eat_word("hold").is_some() {
            let message = self.expr();
            return Action::Hold(message, self.since(start));
        }
        if self.eat_word("update").is_some() {
            let fields = self.field_list();
            let mut keep = Vec::new();
            let mut hold_when = None;
            loop {
                if self.at_line_end() && !self.clause_continues(&["keep", "hold"]) {
                    break;
                }
                if self.eat_word("keep").is_some() {
                    keep = self.field_list();
                } else if self.eat_word("hold").is_some() {
                    self.expect_word("when");
                    let cond = self.expr();
                    let message = self.expr();
                    hold_when = Some((cond, message));
                } else {
                    break;
                }
            }
            return Action::Update {
                fields,
                keep,
                hold_when,
                span: self.since(start),
            };
        }
        if self.eat_word("cancel").is_some() {
            let unless = if self.eat_word("unless").is_some() {
                let state = self
                    .name("dispatched, delivered or invoiced")
                    .unwrap_or_else(|| Ident::new("", self.token().span));
                self.expect_word("else");
                Some((state, Box::new(self.action())))
            } else {
                None
            };
            return Action::Cancel {
                unless,
                span: self.since(start),
            };
        }
        self.error_here(Message::new(message!(
            "Expected create, update, cancel, apply, hold or ignore"
        )));
        Action::Error(self.since(start))
    }

    fn field_list(&mut self) -> Vec<Ident> {
        let mut fields = Vec::new();
        loop {
            if let Some(field) = self.name("a field") {
                fields.push(field);
            }
            if self.eat(Tok::Comma).is_none() {
                break;
            }
        }
        fields
    }

    fn respond(&mut self) -> Respond {
        let start = self.token().span.start;
        self.eat_word("respond");
        let set = self.set_name();
        let map = if self.eat_word("via").is_some() {
            self.name("a map")
        } else {
            None
        };
        let mut respond = Respond {
            set,
            map,
            accept: Vec::new(),
            decline: Vec::new(),
            otherwise: None,
            span: Span::default(),
        };
        while self.clause_continues(&["accept", "decline", "otherwise", "via"]) {
            if self.eat_word("via").is_some() {
                respond.map = self.name("a map");
            } else if self.eat_word("accept").is_some() {
                self.expect_word("automatically");
                self.expect_word("when");
                respond.accept.push(self.expr());
            } else if self.eat_word("decline").is_some() {
                self.expect_word("automatically");
                self.expect_word("when");
                respond.decline.push(self.expr());
            } else if self.eat_word("otherwise").is_some() {
                let other_start = self.token().span.start;
                let otherwise = if self.eat_word("planner").is_some() {
                    self.expect_word("decides");
                    let mut within = None;
                    let mut escalate = None;
                    if self.at_line_end() {
                        self.clause_continues(&["within"]);
                    }
                    if self.eat_word("within").is_some() {
                        within = Some(self.additive_only());
                        if self.at_line_end() {
                            self.clause_continues(&["else"]);
                        }
                        if self.eat_word("else").is_some() {
                            self.expect_word("escalate");
                            escalate = Some(self.expr());
                        }
                    }
                    Otherwise::Planner { within, escalate }
                } else if self.eat_word("accept").is_some() {
                    Otherwise::Accept
                } else if self.eat_word("decline").is_some() {
                    Otherwise::Decline(self.expr())
                } else {
                    self.error_here(Message::new(message!(
                        "Expected planner decides, accept or decline"
                    )));
                    self.skip_line();
                    continue;
                };
                respond.otherwise = Some((otherwise, self.since(other_start)));
            } else {
                self.error_here(Message::new(message!(
                    "Expected accept, decline or otherwise"
                )));
                self.skip_line();
            }
        }
        respond.span = self.since(start);
        respond
    }

    fn event_list(&mut self) -> Vec<Ident> {
        let mut events = Vec::new();
        loop {
            if let Some(event) = self.type_name("an event such as StopArrived") {
                events.push(event);
            }
            if self.eat(Tok::Comma).is_none() {
                break;
            }
        }
        events
    }

    fn send(&mut self) -> Send {
        let start = self.token().span.start;
        self.eat_word("send");
        let set = self.set_name();
        let mut send = Send {
            set,
            map: None,
            events: Vec::new(),
            filter: None,
            key: None,
            batch: None,
            until: None,
            span: Span::default(),
        };
        let words = ["via", "on", "where", "key", "batch", "until"];
        loop {
            if self.at_line_end() && !self.clause_continues(&words) {
                break;
            }
            if self.eat_word("via").is_some() {
                send.map = self.name("a map");
            } else if self.eat_word("on").is_some() {
                send.events.extend(self.event_list());
            } else if self.eat_word("where").is_some() {
                send.filter = Some(self.expr());
            } else if self.eat_word("key").is_some() {
                send.key = Some(self.expr());
            } else if self.eat_word("batch").is_some() {
                self.expect_word("within");
                send.batch = Some(self.additive_only());
            } else if self.eat_word("until").is_some() {
                send.until = self.type_name("an event");
            } else {
                self.error_here(Message::new(message!(
                    "Expected via, on, where, key, batch or until"
                )));
                self.skip_line();
                break;
            }
        }
        send.span = self.since(start);
        send
    }

    fn expect_ack(&mut self) -> ExpectAck {
        let start = self.token().span.start;
        self.eat_word("expect");
        let set = self.set_name();
        self.expect_word("for");
        self.expect_word("every");
        self.expect_word("send");
        let mut ack = ExpectAck {
            set,
            within: None,
            resend: None,
            escalate: None,
            span: Span::default(),
        };
        loop {
            if self.at_line_end() && !self.clause_continues(&["within", "else"]) {
                break;
            }
            if self.eat_word("within").is_some() {
                ack.within = Some(self.additive_only());
            } else if self.eat_word("else").is_some() {
                if self.eat_word("resend").is_some() {
                    ack.resend = self.uint("how many times");
                    self.expect_word("then");
                }
                self.expect_word("escalate");
                ack.escalate = Some(self.expr());
            } else {
                self.error_here(Message::new(message!("Expected within or else")));
                self.skip_line();
                break;
            }
        }
        ack.span = self.since(start);
        ack
    }
}
