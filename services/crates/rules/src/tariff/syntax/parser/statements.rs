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
use crate::tariff::syntax::ast::ChoiceOption;
use crate::tariff::syntax::ast::Choose;
use crate::tariff::syntax::ast::ChooseKind;
use crate::tariff::syntax::ast::Condition;
use crate::tariff::syntax::ast::Direction;
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
    /// A property body, where `vary`, `assume`, `given` and `expect` are allowed.
    Property,
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
        let outer = std::mem::replace(&mut self.statement_failed, false);
        let kind = self.statement_kind(block);
        self.skip_rest_if_failed(outer);
        self.leave();
        let end = self.previous_span().end.max(start);
        Stmt {
            span: Span::new(start, end),
            kind,
        }
    }

    fn statement_kind(&mut self, block: BlockKind) -> StmtKind {
        self.statement_start = self.pos;
        if self.kind() == TokenKind::Keyword {
            let word = self.text();
            if DECLARATION_KEYWORDS.contains(&word) {
                return self.misplaced_item(block, word);
            }
            if let Some(kind) = self.keyword_statement(block, word) {
                return kind;
            }
        }
        if let Some(kind) = self.contextual_statement(block) {
            return kind;
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

    /// A statement that starts with a contextual word: `best`, `worst`, `option`, `vary` or
    /// `assume`. Each is read as one only where what follows can't continue a name: a quoted
    /// label after `best`, `worst` and `option`, a target after `vary`, a value after `assume`.
    fn contextual_statement(&mut self, block: BlockKind) -> Option<StmtKind> {
        if self.kind() != TokenKind::Name {
            return None;
        }
        if self.at_labelled("best") {
            return Some(StmtKind::Choose(self.choose(ChooseKind::Best)));
        }
        if self.at_labelled("worst") {
            return Some(StmtKind::Choose(self.choose(ChooseKind::Worst)));
        }
        if self.at_labelled("option") {
            let span = self.peek().span;
            self.report(Diagnostic::error(
                "misplaced",
                span,
                Message::new(message!(
                    "An `option` can only be written inside `best` or `worst`"
                )),
            ));
            let option = self.choice_option();
            return Some(StmtKind::Choose(Choose {
                kind: ChooseKind::Best,
                label: Name {
                    text: String::new(),
                    span: Span::new(option.span.start, option.span.start),
                },
                span: option.span,
                options: vec![option],
            }));
        }
        if self.at_word("vary") && self.starts_target(1) {
            return Some(self.vary(block));
        }
        if self.at_word("assume") && (self.starts_value(1) || self.spaced_paren(1)) {
            let keyword = self.bump().span;
            self.mark_at(self.pos - 1, TokenKind::Keyword);
            self.property_rule(block, keyword, "assume");
            return Some(StmtKind::Assume(self.expr()));
        }
        None
    }

    /// Whether the token `ahead` places on, on the same line, starts a `given` or `vary`
    /// target: a name, or `input`.
    fn starts_target(&self, ahead: usize) -> bool {
        let token = self.peek_nth(ahead);
        !self.newline_before_nth(ahead)
            && (token.kind == TokenKind::Name
                || (token.kind == TokenKind::Keyword && self.slice(token.span) == "input"))
            && !matches!(
                self.slice(self.peek_nth(ahead + 1).span),
                "=" | "+=" | "-=" | "("
            )
    }

    /// Whether the token `ahead` places on is a `(` on the same line with a space before it:
    /// `assume (x ?? 0 lb) > 10 lb` is a statement, while `assume(x)` stays a call.
    fn spaced_paren(&self, ahead: usize) -> bool {
        let token = self.peek_nth(ahead);
        !self.newline_before_nth(ahead)
            && self.slice(token.span) == "("
            && self.peek_nth(ahead - 1).span.end < token.span.start
    }

    /// Whether the token `ahead` places on, on the same line, can start a value rather than
    /// continue the word before it.
    fn starts_value(&self, ahead: usize) -> bool {
        let token = self.peek_nth(ahead);
        if self.newline_before_nth(ahead) || token.kind == TokenKind::Eof {
            return false;
        }
        let text = self.slice(token.span);
        if matches!(token.kind, TokenKind::Keyword) {
            return matches!(text, "not" | "true" | "false" | "none" | "if" | "match")
                || (text == "charge" && self.slice(self.peek_nth(ahead + 1).span) == "(");
        }
        !matches!(
            text,
            "=" | "+="
                | "-="
                | "."
                | "?."
                | "("
                | "["
                | "??"
                | "+"
                | "*"
                | "/"
                | "%"
                | "=="
                | "!="
                | "<"
                | "<="
                | ">"
                | ">="
                | ".."
                | "..="
                | ","
                | ")"
                | "]"
                | "}"
                | "=>"
        )
    }

    /// `best "Label" { option "A" { … } … }`.
    fn choose(&mut self, kind: ChooseKind) -> Choose {
        self.mark(TokenKind::Keyword);
        let start = self.bump().span.start;
        let label = self.literal_name(Message::with(
            message!("Expected the choice's label in quotes after `{0}`"),
            [Arg::from(kind.keyword())],
        ));
        let mut options = Vec::new();
        let Some(open) = self.eat("{") else {
            self.expected(Message::with(
                message!("Expected the options in braces after the label: `{0} \"Linehaul\"`, then an `option \"Per mile\"` block for each method"),
                [Arg::from(kind.keyword())],
            ));
            return Choose {
                kind,
                label,
                options,
                span: Span::new(start, self.previous_span().end),
            };
        };
        if self.enter() {
            while !self.at("}") && !self.at_eof() && !self.stopped && !self.at_column_zero_item() {
                let before = self.pos;
                if self.at_labelled("option") {
                    options.push(self.choice_option());
                } else {
                    let span = self.peek().span;
                    self.error(
                        span,
                        Message::new(message!(
                            "A choice holds only `option \"Label\"` blocks, one for each method"
                        )),
                    );
                    self.recover();
                }
                self.end_statement();
                if self.pos == before {
                    self.bump();
                }
            }
            self.leave();
        }
        let end = if let Some(close) = self.eat("}") {
            close.end
        } else {
            self.error(
                open,
                Message::with(
                    message!("This `{0}` is missing its closing brace"),
                    [Arg::from(kind.keyword())],
                ),
            );
            self.previous_span().end
        };
        Choose {
            kind,
            label,
            options,
            span: Span::new(start, end),
        }
    }

    /// `option "Label" { … }`.
    fn choice_option(&mut self) -> ChoiceOption {
        self.mark(TokenKind::Keyword);
        let start = self.bump().span.start;
        let label = self.literal_name(Message::new(message!(
            "Expected the option's label in quotes after `option`"
        )));
        let body = self.body(Message::new(message!(
            "Expected what the option bills, in braces"
        )));
        ChoiceOption {
            label,
            span: Span::new(start, body.span.end.max(start)),
            body,
        }
    }

    /// `vary target in values`.
    fn vary(&mut self, block: BlockKind) -> StmtKind {
        self.mark(TokenKind::Keyword);
        let keyword = self.bump().span;
        self.property_rule(block, keyword, "vary");
        let target = self.target_path(Message::new(message!(
            "Expected what to vary after `vary`, like `vary shipment.weight in 1 lb..40_000 lb`"
        )));
        let values = if self.at_word("in") && !self.newline_before() {
            self.bump();
            Some(self.expr())
        } else {
            None
        };
        StmtKind::Vary { target, values }
    }

    /// A `given` or `vary` target: a record path, `input.name`, or a name.
    fn target_path(&mut self, message: Message) -> Vec<Name> {
        let mut target = Vec::new();
        if self.at_word("input") && !self.newline_before() {
            self.mark(TokenKind::Keyword);
            let token = self.bump();
            target.push(self.name_of(&token));
        } else {
            target.push(self.value_name(message));
        }
        while self.at(".") && !self.newline_before() {
            self.bump();
            if let Some(name) = self.same_line_name() {
                target.push(name);
            } else {
                self.expected(Message::new(message!("Expected a field's name after `.`")));
                target.push(self.missing_name());
                break;
            }
        }
        target
    }

    /// Reports `vary`, `assume` and a monotone `expect` outside a property.
    fn property_rule(&mut self, block: BlockKind, keyword: Span, word: &str) {
        if block != BlockKind::Property {
            self.report(Diagnostic::error(
                "misplaced",
                keyword,
                Message::with(
                    message!("`{0}` can only be used inside a `property` block"),
                    [Arg::from(word.to_owned())],
                ),
            ));
        }
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
        if self.kind() == TokenKind::Text && !self.newline_before() {
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
        let kind = if self.newline_before() {
            TokenKind::Eof
        } else {
            self.kind()
        };
        match kind {
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
            let outer = std::mem::replace(&mut self.guard, true);
            let guard = self.expr();
            self.guard = outer;
            Some(guard)
        } else {
            None
        };
        let outer = std::mem::replace(&mut self.guard, false);
        let body = self.arm_body(statement);
        self.guard = outer;
        let end = self.previous_span().end.max(start);
        Arm {
            pattern,
            guard,
            body,
            span: Span::new(start, end),
        }
    }

    /// What an arm gives, after its pattern and guard: `=> value`, or `=> { … }` in a
    /// statement.
    fn arm_body(&mut self, statement: bool) -> ArmBody {
        if self.eat("=>").is_some() {
            if statement && self.at("{") {
                ArmBody::Block(self.block(BlockKind::Nested))
            } else if statement && self.at_statement_keyword() {
                let stmt = self.statement(BlockKind::Nested);
                ArmBody::Block(Block {
                    span: stmt.span,
                    stmts: vec![stmt],
                })
            } else {
                ArmBody::Expr(self.arm_value())
            }
        } else {
            self.expected(Message::new(message!(
                "Expected `=>` and what this arm gives"
            )));
            ArmBody::Expr(self.missing())
        }
    }

    /// Whether the cursor starts a statement that is not an expression: the body of a `match`
    /// statement's arm may be one, read as a block holding it.
    fn at_statement_keyword(&self) -> bool {
        if self.at_labelled("best") || self.at_labelled("worst") {
            return true;
        }
        self.kind() == TokenKind::Keyword
            && match self.text() {
                "charge" => !self.peek_is(1, "("),
                word => matches!(
                    word,
                    "let"
                        | "var"
                        | "note"
                        | "if"
                        | "match"
                        | "for"
                        | "require"
                        | "fail"
                        | "return"
                        | "break"
                        | "continue"
                        | "minimum"
                        | "maximum"
                        | "round"
                        | "given"
                        | "expect"
                ),
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
        let target = self.target_path(Message::new(message!(
            "Expected what to set after `given`, like `given shipment.weight = 1_240 lb`"
        )));
        let keys = (self.at("[") && !self.newline_before()).then(|| self.bracket_keys().0);
        let value = if self.eat("=").is_some() {
            self.expr()
        } else {
            self.expected(Message::new(message!("Expected `=` and the value to give")));
            self.missing()
        };
        StmtKind::Given {
            target,
            keys,
            value,
        }
    }

    fn expect_statement(&mut self, block: BlockKind) -> StmtKind {
        let keyword = self.bump().span;
        self.position_rule(block, keyword, "expect");
        if self.at_word("fails") {
            let fails = self.bump().span;
            if block == BlockKind::Property {
                self.report(Diagnostic::error(
                    "misplaced",
                    fails,
                    Message::new(message!(
                        "A property can't expect a refusal: it states what holds when the tariff rates. Narrow it with `assume` or its ranges instead"
                    )),
                ));
            }
            let message = (self.kind() == TokenKind::Text && !self.newline_before())
                .then(|| self.text_literal());
            return StmtKind::Expect(Expect::Fails(message));
        }
        let value = self.expr();
        let direction = if self.at_word("nondecreasing") && !self.newline_before() {
            Some(Direction::Nondecreasing)
        } else if self.at_word("nonincreasing") && !self.newline_before() {
            Some(Direction::Nonincreasing)
        } else {
            None
        };
        let Some(direction) = direction else {
            return StmtKind::Expect(Expect::Holds(value));
        };
        self.mark(TokenKind::Keyword);
        let direction_span = self.bump().span;
        self.property_rule(block, direction_span, direction.word());
        if self.eat_word("in").is_none() {
            self.expected(Message::with(
                message!(
                    "Expected `in` and the varied value after `{0}`, like `{0} in shipment.weight`"
                ),
                [Arg::from(direction.word())],
            ));
        }
        let target = self.target_path(Message::with(
            message!("Expected the varied value after `{0} in`"),
            [Arg::from(direction.word())],
        ));
        StmtKind::Expect(Expect::Monotone {
            value,
            direction,
            direction_span,
            target,
        })
    }

    fn position_rule(&mut self, block: BlockKind, keyword: Span, word: &str) {
        if !matches!(block, BlockKind::Test | BlockKind::Property) {
            let message = if word == "given" {
                message!("`given` can only be used inside a `test` or `property` block")
            } else {
                message!("`expect` can only be used inside a `test` or `property` block")
            };
            self.report(Diagnostic::error(
                "misplaced",
                keyword,
                Message::new(message),
            ));
        }
    }
}
