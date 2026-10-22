//! Statements and blocks.

use super::Parser;
use crate::message;
use crate::message::Arg;
use crate::message::Message;
use crate::span::Span;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::syntax::ast::Arm;
use crate::tariff::syntax::ast::ArmBody;
use crate::tariff::syntax::ast::AssignOp;
use crate::tariff::syntax::ast::Binding;
use crate::tariff::syntax::ast::Block;
use crate::tariff::syntax::ast::Charge;
use crate::tariff::syntax::ast::Condition;
use crate::tariff::syntax::ast::Expect;
use crate::tariff::syntax::ast::ForStmt;
use crate::tariff::syntax::ast::IfStmt;
use crate::tariff::syntax::ast::MatchStmt;
use crate::tariff::syntax::ast::Name;
use crate::tariff::syntax::ast::RoundMode;
use crate::tariff::syntax::ast::RoundTo;
use crate::tariff::syntax::ast::Stmt;
use crate::tariff::syntax::ast::StmtKind;
use crate::tariff::syntax::ast::Text;
use crate::tariff::syntax::ast::TotalClause;
use crate::tariff::syntax::token::TokenKind;

/// Where a block is, which decides the position rules its statements follow.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum BlockKind {
    /// The file itself.
    File,
    /// A function body.
    Function,
    /// A test body, where `given` and `expect` are allowed.
    Test,
    /// Any other block: an `if`, `for` or `match` body.
    Nested,
}

const DECLARATION_KEYWORDS: &[&str] = &[
    "use", "input", "const", "type", "enum", "fn", "test", "private",
];

impl Parser<'_> {
    /// `{ statements }`.
    pub(super) fn block(&mut self, kind: BlockKind) -> Block {
        let open = self.bump().span;
        let mut stmts = Vec::new();
        if !self.enter() {
            return Block { stmts, span: open };
        }
        loop {
            if self.at("}") || self.at_eof() || self.stopped {
                break;
            }
            if self.at_column_zero_item() {
                break;
            }
            let before = self.pos;
            stmts.push(self.statement(kind));
            self.end_statement();
            if self.pos == before {
                self.bump();
            }
        }
        self.leave();
        let end = if let Some(close) = self.eat("}") {
            close.end
        } else {
            self.error(
                open,
                Message::new(message!("This block is missing its closing brace")),
            );
            self.previous_span().end
        };
        Block {
            stmts,
            span: Span::new(open.start, end),
        }
    }

    /// One statement.
    pub(super) fn statement(&mut self, block: BlockKind) -> Stmt {
        let start = self.peek().span.start;
        if !self.enter() {
            return Stmt {
                span: Span::new(start, start),
                kind: StmtKind::Break,
            };
        }
        let kind = self.statement_kind(block);
        self.leave();
        let end = self.previous_span().end.max(start);
        Stmt {
            span: Span::new(start, end),
            kind,
        }
    }

    fn statement_kind(&mut self, block: BlockKind) -> StmtKind {
        if self.kind() == TokenKind::Keyword {
            let word = self.text();
            if DECLARATION_KEYWORDS.contains(&word) {
                return self.misplaced_item(block, word);
            }
            if let Some(kind) = self.keyword_statement(block, word) {
                return kind;
            }
        }
        if matches!(self.kind(), TokenKind::Name)
            && (self.peek_is(1, "=") || self.peek_is(1, "+=") || self.peek_is(1, "-="))
        {
            return self.assignment();
        }
        if self.at("}") && block == BlockKind::File {
            let span = self.bump().span;
            self.error(
                span,
                Message::new(message!("This closing brace has no opening brace")),
            );
            return StmtKind::Expr(self.missing());
        }
        let expr = self.expr();
        if (self.at("=") || self.at("+=") || self.at("-=")) && !self.newline_before() {
            let span = self.peek().span;
            self.error(
                span,
                Message::new(message!("Only a `var` can be assigned, by its name")),
            );
            self.bump();
            self.expr();
        }
        StmtKind::Expr(expr)
    }

    fn misplaced_item(&mut self, block: BlockKind, word: &str) -> StmtKind {
        let span = self.peek().span;
        let word = word.to_owned();
        let item = self.item(block);
        let message = if word == "test" {
            message!("A `test` can only be written at the top level of a file, not inside a block")
        } else if word == "private" {
            message!("`private` can only be used at the top level of a file, not inside a block")
        } else {
            message!("`{0}` can only be used at the top level of a file, not inside a block")
        };
        self.report(Diagnostic::error(
            "misplaced",
            span,
            Message::with(message, [Arg::from(word)]),
        ));
        StmtKind::Item(Box::new(item))
    }

    fn keyword_statement(&mut self, block: BlockKind, word: &str) -> Option<StmtKind> {
        Some(match word {
            "let" => StmtKind::Let(self.binding()),
            "var" => StmtKind::Var(self.binding()),
            "charge" if !self.peek_is(1, "(") => StmtKind::Charge(self.charge()),
            "note" => {
                self.bump();
                StmtKind::Note(self.required_text(Message::new(message!(
                    "Expected the note's text in quotes after `note`"
                ))))
            }
            "if" => StmtKind::If(self.if_statement()),
            "match" => StmtKind::Match(self.match_statement()),
            "for" => StmtKind::For(self.for_statement()),
            "require" => self.require(),
            "fail" => {
                self.bump();
                StmtKind::Fail(self.required_text(Message::new(message!(
                    "Expected the reason in quotes after `fail`, like `fail \"No rate for this lane\"`"
                ))))
            }
            "return" => {
                self.bump();
                StmtKind::Return(self.expr())
            }
            "break" => {
                self.bump();
                StmtKind::Break
            }
            "continue" => {
                self.bump();
                StmtKind::Continue
            }
            "minimum" => {
                self.bump();
                StmtKind::Total(TotalClause::Minimum(self.expr()))
            }
            "maximum" => {
                self.bump();
                StmtKind::Total(TotalClause::Maximum(self.expr()))
            }
            "round" => StmtKind::Total(self.round()),
            "given" => self.given(block),
            "expect" => self.expect_statement(block),
            "else" => {
                let span = self.bump().span;
                self.error(
                    span,
                    Message::new(message!("This `else` has no `if` before it")),
                );
                self.recover();
                StmtKind::Expr(self.missing())
            }
            _ => return None,
        })
    }

    fn binding(&mut self) -> Binding {
        let keyword = self.bump();
        let word = self.slice(keyword.span).to_owned();
        let name = self.value_name(Message::with(
            message!("Expected a name after `{0}`"),
            [Arg::from(word)],
        ));
        let ty = self.eat(":").map(|_| self.type_expr());
        let value = if self.eat("=").is_some() {
            self.expr()
        } else {
            self.expected(Message::with(
                message!("Expected `=` and the value of `{0}`"),
                [Arg::from(name.text.clone())],
            ));
            self.missing()
        };
        Binding { name, ty, value }
    }

    fn assignment(&mut self) -> StmtKind {
        let token = self.bump();
        let target = self.name_of(&token);
        let op = match self.bump().span {
            span if self.slice(span) == "+=" => AssignOp::Add,
            span if self.slice(span) == "-=" => AssignOp::Sub,
            _ => AssignOp::Set,
        };
        let value = self.expr();
        StmtKind::Assign { target, op, value }
    }

    /// Text in quotes, reporting `message` when there is none.
    pub(super) fn required_text(&mut self, message: Message) -> Text {
        if self.kind() == TokenKind::Text {
            return self.text_literal();
        }
        self.expected(message);
        let at = self.previous_span().end;
        Text {
            parts: Vec::new(),
            span: Span::new(at, at),
        }
    }

    fn charge(&mut self) -> Charge {
        self.bump();
        let label = self.required_text(Message::new(message!(
            "Expected the charge's label in quotes after `charge`, like `charge \"Linehaul\"`"
        )));
        let code = if self.at_word("as") && !self.newline_before() {
            self.eat_word("as");
            Some(self.charge_code())
        } else {
            None
        };
        let options = if self.at("(") && !self.newline_before() {
            let args = self.arguments();
            for arg in &args {
                if arg.name.is_none() {
                    self.error(
                        arg.span,
                        Message::new(message!(
                            "A charge's options are named: `quantity: …` and `rate: …`"
                        )),
                    );
                }
            }
            args
        } else {
            Vec::new()
        };
        let amount = if self.at("=") && !self.newline_before() {
            self.bump();
            Some(self.expr())
        } else {
            None
        };
        Charge {
            label,
            code,
            options,
            amount,
        }
    }

    fn charge_code(&mut self) -> Name {
        match self.kind() {
            TokenKind::Code => {
                let token = self.bump();
                self.name_of(&token)
            }
            TokenKind::Name | TokenKind::TypeName | TokenKind::Keyword => {
                let token = self.bump();
                let name = self.name_of(&token);
                if name.text.chars().any(|c| c.is_ascii_lowercase()) || name.text.len() < 2 {
                    let fixed = name.text.to_ascii_uppercase();
                    self.report(
                        Diagnostic::error(
                            "syntax",
                            token.span,
                            Message::with(
                                message!("A charge code is written in capitals: `{0}`"),
                                [Arg::from(fixed.clone())],
                            ),
                        )
                        .with_fix(super::rename_fix(token.span, fixed)),
                    );
                }
                self.mark_at(self.pos - 1, TokenKind::Code);
                name
            }
            _ => {
                self.expected(Message::new(message!(
                    "Expected a charge code after `as`, like `FSC`"
                )));
                self.missing_name()
            }
        }
    }

    /// What an `if` or `require` tests: an expression, or `let name = optional`.
    fn condition(&mut self) -> Condition {
        if self.at_word("let") {
            self.bump();
            let name = self.value_name(Message::new(message!("Expected a name after `let`")));
            let value = if self.eat("=").is_some() {
                self.expr()
            } else {
                self.expected(Message::with(
                    message!("Expected `=` and the optional value to bind to `{0}`"),
                    [Arg::from(name.text.clone())],
                ));
                self.missing()
            };
            return Condition::Let { name, value };
        }
        Condition::Expr(self.expr())
    }

    pub(super) fn condition_expr(&mut self) -> Condition {
        self.condition()
    }

    /// A statement block, or an empty one with a diagnostic when the brace is absent.
    fn body(&mut self, what: Message) -> Block {
        if self.at("{") {
            return self.block(BlockKind::Nested);
        }
        self.expected(what);
        let at = self.previous_span().end;
        Block {
            stmts: Vec::new(),
            span: Span::new(at, at),
        }
    }

    fn if_statement(&mut self) -> IfStmt {
        self.bump();
        let mut branches = Vec::new();
        let mut otherwise = None;
        loop {
            let condition = self.condition();
            let block = self.body(Message::new(message!(
                "Expected the `if` body in braces after the condition"
            )));
            branches.push((condition, block));
            if !self.at_word("else") {
                break;
            }
            self.bump();
            if self.at_word("if") {
                self.bump();
                continue;
            }
            otherwise =
                Some(self.body(Message::new(message!("Expected the `else` body in braces"))));
            break;
        }
        IfStmt {
            branches,
            otherwise,
        }
    }

    fn match_statement(&mut self) -> MatchStmt {
        let start = self.bump().span.start;
        let scrutinee = self.expr();
        let arms = self.arms(true);
        let end = self.previous_span().end;
        MatchStmt {
            scrutinee,
            arms,
            span: Span::new(start, end),
        }
    }

    /// `{ pattern [if guard] => body … }`, for a statement (bodies may be blocks) or an
    /// expression.
    pub(super) fn arms(&mut self, statement: bool) -> Vec<Arm> {
        let mut arms = Vec::new();
        let Some(open) = self.eat("{") else {
            self.expected(Message::new(message!(
                "Expected the `match` arms in braces after the value"
            )));
            return arms;
        };
        while !self.at("}") && !self.at_eof() && !self.stopped && !self.at_column_zero_item() {
            let before = self.pos;
            arms.push(self.arm(statement));
            if !self.newline_before() && !self.at("}") && !self.at_eof() {
                self.end_statement();
            }
            if self.pos == before {
                self.bump();
            }
        }
        if self.eat("}").is_none() {
            self.error(
                open,
                Message::new(message!("This `match` is missing its closing brace")),
            );
        }
        arms
    }

    fn arm(&mut self, statement: bool) -> Arm {
        let start = self.peek().span.start;
        let (pattern, _) = self.pattern();
        let guard = if self.at_word("if") {
            self.bump();
            Some(self.expr())
        } else {
            None
        };
        let body = if self.eat("=>").is_some() {
            if statement && self.at("{") {
                ArmBody::Block(self.block(BlockKind::Nested))
            } else {
                ArmBody::Expr(self.arm_value())
            }
        } else {
            self.expected(Message::new(message!(
                "Expected `=>` and what this arm gives"
            )));
            ArmBody::Expr(self.missing())
        };
        let end = self.previous_span().end.max(start);
        Arm {
            pattern,
            guard,
            body,
            span: Span::new(start, end),
        }
    }

    fn for_statement(&mut self) -> ForStmt {
        self.bump();
        let binding = self.value_name(Message::new(message!(
            "Expected the loop variable's name after `for`"
        )));
        let iterable = if self.eat_word("in").is_some() {
            self.expr()
        } else {
            self.expected(Message::new(message!(
                "Expected `in` and the list to walk, like `for stop in shipment.stops`"
            )));
            self.missing()
        };
        let filter = if self.at_word("where") {
            self.bump();
            Some(self.expr())
        } else {
            None
        };
        let body = self.body(Message::new(message!("Expected the loop's body in braces")));
        ForStmt {
            binding,
            iterable,
            filter,
            body,
        }
    }

    fn require(&mut self) -> StmtKind {
        self.bump();
        let condition = self.condition();
        let message = if self.at_word("else") {
            self.bump();
            self.required_text(Message::new(message!(
                "Expected the reason in quotes after `else`"
            )))
        } else {
            self.expected(Message::new(message!(
                "Expected `else` and the reason to refuse, like `else \"This rate needs a weight\"`"
            )));
            let at = self.previous_span().end;
            Text {
                parts: Vec::new(),
                span: Span::new(at, at),
            }
        };
        StmtKind::Require { condition, message }
    }

    fn round(&mut self) -> TotalClause {
        self.bump();
        let mode = if self.at_word("half") && self.word_at(1, "even") {
            self.eat_word("half");
            self.eat_word("even");
            RoundMode::HalfEven
        } else if self.eat_word("up").is_some() {
            RoundMode::Up
        } else if self.eat_word("down").is_some() {
            RoundMode::Down
        } else {
            RoundMode::HalfUp
        };
        if self.eat_word("to").is_none() {
            self.expected(Message::new(message!(
                "Expected `to` and what to round to, like `round to cents` or `round up to $5`"
            )));
            return TotalClause::Round {
                mode,
                to: RoundTo::Cents,
            };
        }
        let to = if self.at_word("cents") && !self.peek_is(1, ".") {
            self.eat_word("cents");
            RoundTo::Cents
        } else {
            RoundTo::Amount(self.expr())
        };
        TotalClause::Round { mode, to }
    }

    fn given(&mut self, block: BlockKind) -> StmtKind {
        let keyword = self.bump().span;
        self.position_rule(block, keyword, "given");
        let mut target = Vec::new();
        if self.at_word("input") {
            self.mark(TokenKind::Keyword);
            let token = self.bump();
            target.push(self.name_of(&token));
        } else {
            target.push(self.value_name(Message::new(message!(
                "Expected what to set after `given`, like `given shipment.weight = 1_240 lb`"
            ))));
        }
        while self.at(".") {
            self.bump();
            if let Some(name) = self.any_name() {
                target.push(name);
            } else {
                self.expected(Message::new(message!("Expected a field's name after `.`")));
                target.push(self.missing_name());
                break;
            }
        }
        let value = if self.eat("=").is_some() {
            self.expr()
        } else {
            self.expected(Message::new(message!("Expected `=` and the value to give")));
            self.missing()
        };
        StmtKind::Given { target, value }
    }

    fn expect_statement(&mut self, block: BlockKind) -> StmtKind {
        let keyword = self.bump().span;
        self.position_rule(block, keyword, "expect");
        if self.at_word("fails") {
            self.bump();
            let message = (self.kind() == TokenKind::Text && !self.newline_before())
                .then(|| self.text_literal());
            return StmtKind::Expect(Expect::Fails(message));
        }
        StmtKind::Expect(Expect::Holds(self.expr()))
    }

    fn position_rule(&mut self, block: BlockKind, keyword: Span, word: &str) {
        if block != BlockKind::Test {
            let message = if word == "given" {
                message!("`given` can only be used inside a `test` block")
            } else {
                message!("`expect` can only be used inside a `test` block")
            };
            self.report(Diagnostic::error(
                "misplaced",
                keyword,
                Message::new(message),
            ));
        }
    }
}
