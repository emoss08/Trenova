//! Statements of maps, rules and function bodies, and blocks.

use trenova_module_kit::message;
use trenova_module_kit::message::Message;

use super::Parser;
use crate::lading::syntax::ast::{
    Block, ElementAssign, Else, Expr, ExprKind, Ident, LValue, LetStmt, LvStep, Outcome,
    OutcomeKind, Stmt,
};
use crate::lading::syntax::token::Tok;

impl Parser<'_> {
    /// `{ statements }`
    pub(super) fn block(&mut self) -> Block {
        let start = self.token().span.start;
        if self.expect(Tok::LBrace, "{").is_none() {
            return Block {
                stmts: Vec::new(),
                span: self.since(start),
            };
        }
        let mut stmts = Vec::new();
        if self.enter() {
            loop {
                self.skip_newlines();
                if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                    break;
                }
                let before = self.pos;
                stmts.push(self.stmt());
                if self.pos == before {
                    let span = self.advance().span;
                    stmts.push(Stmt::Error(span));
                }
            }
            self.leave();
        }
        self.expect(Tok::RBrace, "}");
        Block {
            stmts,
            span: self.since(start),
        }
    }

    /// One statement, to the end of its line.
    pub(super) fn stmt(&mut self) -> Stmt {
        let stmt = self.stmt_inner();
        if !matches!(stmt, Stmt::Error(_)) {
            if self.eat(Tok::Comma).is_some() {
                return stmt;
            }
            if !self.at_line_end() {
                self.error_here(Message::new(message!("Expected the end of the line")));
                self.skip_line();
            }
        }
        stmt
    }

    fn stmt_inner(&mut self) -> Stmt {
        let start = self.token().span.start;
        match self.peek() {
            Tok::X12 if matches!(self.peek_at(1), Tok::LBrace) || self.word_at(1, "when") => {
                return self.segment_block();
            }
            Tok::Digits if matches!(self.peek_at(1), Tok::LBrace | Tok::LArrow) => {
                return self.loop_block();
            }
            Tok::Name => {}
            _ => {
                self.error_here(Message::new(message!(
                    "Expected an assignment, a segment, a loop or a rule statement"
                )));
                self.skip_line();
                return Stmt::Error(self.since(start));
            }
        }
        let word = self.current_text().to_owned();
        match word.as_str() {
            "let" => Stmt::Let(self.let_stmt()),
            "when" => {
                self.eat_word("when");
                let cond = self.condition();
                let body = self.block();
                Stmt::When {
                    cond,
                    body,
                    span: self.since(start),
                }
            }
            "each" => {
                self.eat_word("each");
                let list = self.condition();
                self.expect_word("as");
                let binder = self
                    .name("a name for each item")
                    .unwrap_or_else(|| Ident::new("", self.token().span));
                let body = self.block();
                Stmt::Each {
                    list,
                    binder,
                    body,
                    span: self.since(start),
                }
            }
            "for" => self.for_stmt(),
            "require" => {
                self.eat_word("require");
                let cond = self.expr();
                self.skip_newlines();
                self.expect_word("else");
                let outcome = self.outcome();
                Stmt::Require {
                    cond,
                    outcome,
                    span: self.since(start),
                }
            }
            "reject" | "decline" | "hold" | "warn" => {
                let outcome = self.outcome();
                self.continue_to("when");
                if self.expect_word("when").is_none() {
                    return Stmt::Error(self.since(start));
                }
                let cond = self.expr();
                Stmt::OutcomeWhen {
                    outcome,
                    cond,
                    span: self.since(start),
                }
            }
            "fix" => {
                self.eat_word("fix");
                let target = self.lvalue();
                self.expect(Tok::LArrow, "<-");
                let value = self.expr();
                Stmt::Fix {
                    target,
                    value,
                    span: self.since(start),
                }
            }
            _ => self.assign(),
        }
    }

    /// Steps over line ends when the next line starts with `word`.
    fn continue_to(&mut self, word: &str) {
        if self.continuation_word(word) {
            self.skip_newlines();
        }
    }

    fn let_stmt(&mut self) -> LetStmt {
        let start = self.token().span.start;
        self.eat_word("let");
        let name = self
            .name("a name")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        self.expect(Tok::Assign, "=");
        let value = self.expr();
        LetStmt {
            name,
            value,
            span: self.since(start),
        }
    }

    fn for_stmt(&mut self) -> Stmt {
        let start = self.token().span.start;
        self.eat_word("for");
        let binder = self
            .name("a name for each item")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        self.expect_word("in");
        let list = self.condition();
        let filter = if self.eat_word("where").is_some() {
            Some(self.condition())
        } else {
            None
        };
        let body = if self.at(Tok::LBrace) {
            self.block()
        } else {
            let stmt_start = self.token().span.start;
            let stmt = self.stmt_inner();
            crate::lading::syntax::ast::Block {
                stmts: vec![stmt],
                span: self.since(stmt_start),
            }
        };
        Stmt::For {
            binder,
            list,
            filter,
            body,
            span: self.since(start),
        }
    }

    pub(super) fn lvalue(&mut self) -> LValue {
        let start = self.token().span.start;
        let root = self
            .name("a field")
            .unwrap_or_else(|| Ident::new("", self.token().span));
        let mut steps = Vec::new();
        loop {
            if self.at(Tok::Dot) {
                self.advance();
                if let Some(field) = self.ident("a field") {
                    steps.push(LvStep::Field(field));
                }
            } else if self.at(Tok::LBracket) && self.adjacent_lv() {
                self.advance();
                let index = self.expr();
                self.expect(Tok::RBracket, "]");
                steps.push(LvStep::Index(index));
            } else {
                break;
            }
        }
        LValue {
            root,
            steps,
            span: self.since(start),
        }
    }

    fn adjacent_lv(&self) -> bool {
        self.token().span.start == self.last_end()
    }

    fn assign(&mut self) -> Stmt {
        let start = self.token().span.start;
        let target = self.lvalue();
        if self.expect(Tok::LArrow, "<-").is_none() {
            self.skip_line();
            return Stmt::Error(self.since(start));
        }
        let value = self.expr();
        let otherwise = self.else_clause();
        Stmt::Assign {
            target,
            value,
            otherwise,
            span: self.since(start),
        }
    }

    /// An optional `else value` or `else outcome`, on this line or the next.
    pub(super) fn else_clause(&mut self) -> Option<Else> {
        if self.at(Tok::Newline) && self.continuation_ahead(true) && self.continuation_word("else")
        {
            self.skip_newlines();
        }
        self.eat_word("else")?;
        if ["reject", "decline", "hold", "warn"]
            .iter()
            .any(|word| self.at_word(word))
        {
            return Some(Else::Outcome(self.outcome()));
        }
        Some(Else::Value(self.expr()))
    }

    /// `reject [at path] "m"`, `decline "m" [code "x"]`, `hold "m"`, `warn "m"`.
    pub(super) fn outcome(&mut self) -> Outcome {
        let start = self.token().span.start;
        let kind = if self.eat_word("reject").is_some() {
            let at = if self.eat_word("at").is_some() {
                Some(self.via_free_expr())
            } else {
                None
            };
            OutcomeKind::Reject(at)
        } else if self.eat_word("decline").is_some() {
            OutcomeKind::Decline(None)
        } else if self.eat_word("hold").is_some() {
            OutcomeKind::Hold
        } else if self.eat_word("warn").is_some() {
            OutcomeKind::Warn
        } else {
            self.error_here(Message::new(message!(
                "Expected reject, decline, hold or warn"
            )));
            return Outcome {
                kind: OutcomeKind::Warn,
                message: Expr::error(self.token().span),
                span: self.since(start),
            };
        };
        let message = self.expr();
        let kind = match kind {
            OutcomeKind::Decline(_) if self.at_word("code") => {
                self.eat_word("code");
                OutcomeKind::Decline(Some(self.expr()))
            }
            other => other,
        };
        Outcome {
            kind,
            message,
            span: self.since(start),
        }
    }

    /// A path for `reject at`: a postfix chain only, so the message after it is not swallowed.
    fn via_free_expr(&mut self) -> Expr {
        let expr = self.expr_postfix_only();
        if matches!(expr.kind, ExprKind::Error) {
            self.skip_line();
        }
        expr
    }

    fn segment_block(&mut self) -> Stmt {
        let start = self.token().span.start;
        let token = self.advance();
        let id = Ident::new(self.text_of(token.span), token.span);
        let cond = if self.eat_word("when").is_some() {
            Some(self.condition())
        } else {
            None
        };
        let mut elements = Vec::new();
        if self.expect(Tok::LBrace, "{").is_some() {
            loop {
                self.skip_newlines();
                if self.at(Tok::RBrace) || self.at(Tok::Eof) {
                    break;
                }
                match self.element_assign() {
                    Some(element) => elements.push(element),
                    None => self.skip_line(),
                }
                if self.eat(Tok::Comma).is_none() && !self.at_line_end() {
                    self.error_here(Message::new(message!("Expected , or the end of the line")));
                    self.skip_line();
                }
            }
            self.expect(Tok::RBrace, "}");
        }
        Stmt::Segment {
            id,
            cond,
            elements,
            span: self.since(start),
        }
    }

    fn element_assign(&mut self) -> Option<ElementAssign> {
        let start = self.token().span.start;
        if !matches!(self.peek(), Tok::Digits | Tok::Number) {
            self.error_here(Message::new(message!(
                "Expected an element position such as 01"
            )));
            return None;
        }
        let token = self.advance();
        let position = self.text_of(token.span).parse::<u32>().ok();
        let mut component = None;
        if self.at(Tok::Dot) && matches!(self.peek_at(1), Tok::Digits | Tok::Number) {
            self.advance();
            let part = self.advance();
            component = self.text_of(part.span).parse::<u32>().ok();
        }
        let position_span = self.since(start);
        let Some(position) = position else {
            self.error_at(
                position_span,
                Message::new(message!("Expected an element position such as 01")),
            );
            return None;
        };
        self.expect(Tok::LArrow, "<-")?;
        let value = self.expr();
        Some(ElementAssign {
            position: (position, component),
            position_span,
            value,
            span: self.since(start),
        })
    }

    fn loop_block(&mut self) -> Stmt {
        let start = self.token().span.start;
        let token = self.advance();
        let id = Ident::new(self.text_of(token.span), token.span);
        let each = if self.eat(Tok::LArrow).is_some() {
            self.expect_word("each");
            let list = self.condition();
            self.expect_word("as");
            let binder = self
                .name("a name for each item")
                .unwrap_or_else(|| Ident::new("", self.token().span));
            Some((list, binder))
        } else {
            None
        };
        let body = self.block();
        Stmt::Loop {
            id,
            each,
            body,
            span: self.since(start),
        }
    }
}
