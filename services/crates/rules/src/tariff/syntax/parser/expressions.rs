//! Expressions, by precedence from loosest to tightest: `??`, `or`, `and`, `not`, comparisons
//! (non-associative), ranges, `+ -`, `* / %`, unary `-`, then postfix field reads, calls and
//! indexes on a primary. Text literals and their interpolations are read here too.

use super::Parser;
use crate::message;
use crate::message::Arg;
use crate::message::Message;
use crate::span::Span;
use crate::tariff::diagnostic::Change;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::diagnostic::Fix;
use crate::tariff::syntax::ast::Arg as CallArg;
use crate::tariff::syntax::ast::BinaryOp;
use crate::tariff::syntax::ast::Condition;
use crate::tariff::syntax::ast::Expr;
use crate::tariff::syntax::ast::ExprKind;
use crate::tariff::syntax::ast::FieldInit;
use crate::tariff::syntax::ast::Name;
use crate::tariff::syntax::ast::Text;
use crate::tariff::syntax::ast::TextPart;
use crate::tariff::syntax::ast::UnaryOp;
use crate::tariff::syntax::ast::UnitExpr;
use crate::tariff::syntax::lexer;
use crate::tariff::syntax::lexer::DateLiteral;
use crate::tariff::syntax::token::TokenKind;

fn comparison(text: &str) -> Option<BinaryOp> {
    Some(match text {
        "==" => BinaryOp::Eq,
        "!=" => BinaryOp::Ne,
        "<" => BinaryOp::Lt,
        "<=" => BinaryOp::Le,
        ">" => BinaryOp::Gt,
        ">=" => BinaryOp::Ge,
        _ => return None,
    })
}

impl Parser<'_> {
    /// A whole expression.
    pub(super) fn expr(&mut self) -> Expr {
        if !self.enter() {
            return self.missing();
        }
        let expr = self.coalesce();
        self.leave();
        expr
    }

    fn binary(&mut self, op: BinaryOp, op_span: Span, left: Expr, right: Expr) -> Expr {
        let height = self.height(&left).max(self.height(&right));
        let span = left.span.cover(right.span);
        self.node_with(
            ExprKind::Binary {
                op,
                op_span,
                left: Box::new(left),
                right: Box::new(right),
            },
            span,
            height,
        )
    }

    fn coalesce(&mut self) -> Expr {
        let mut left = self.or_expr();
        while self.at("??") && !self.stopped {
            let op = self.bump().span;
            let right = if self.at_word("fail") {
                self.fail_expr()
            } else {
                self.or_expr()
            };
            left = self.binary(BinaryOp::Coalesce, op, left, right);
        }
        left
    }

    /// `fail "message"` where a value is expected.
    fn fail_expr(&mut self) -> Expr {
        let start = self.bump().span;
        let text = self.required_text(Message::new(message!(
            "Expected the reason in quotes after `fail`"
        )));
        let span = start.cover(text.span);
        self.leaf(ExprKind::Fail(text), span)
    }

    fn or_expr(&mut self) -> Expr {
        let mut left = self.and_expr();
        while self.at_word("or") && !self.stopped {
            let op = self.bump().span;
            let right = self.and_expr();
            left = self.binary(BinaryOp::Or, op, left, right);
        }
        left
    }

    fn and_expr(&mut self) -> Expr {
        let mut left = self.not_expr();
        while self.at_word("and") && !self.stopped {
            let op = self.bump().span;
            let right = self.not_expr();
            left = self.binary(BinaryOp::And, op, left, right);
        }
        left
    }

    fn not_expr(&mut self) -> Expr {
        if self.at_word("not") && !self.word_at(1, "in") {
            let op = self.bump().span;
            if !self.enter() {
                return self.missing();
            }
            let operand = self.not_expr();
            self.leave();
            return self.unary(UnaryOp::Not, op, operand);
        }
        self.compare()
    }

    fn unary(&mut self, op: UnaryOp, op_span: Span, operand: Expr) -> Expr {
        let height = self.height(&operand);
        let span = op_span.cover(operand.span);
        self.node_with(
            ExprKind::Unary {
                op,
                op_span,
                operand: Box::new(operand),
            },
            span,
            height,
        )
    }

    /// The comparison operator at the cursor, with how many tokens it takes.
    fn comparison_here(&self) -> Option<(BinaryOp, usize)> {
        let token = self.peek();
        if token.kind == TokenKind::Punct {
            let next = self.peek_nth(1);
            if self.slice(token.span) == ">"
                && self.peek_is(1, "=")
                && next.span.start == token.span.end
            {
                return Some((BinaryOp::Ge, 2));
            }
            return comparison(self.slice(token.span)).map(|op| (op, 1));
        }
        if self.at_word("in") {
            return Some((BinaryOp::In, 1));
        }
        if self.at_word("not") && self.word_at(1, "in") {
            return Some((BinaryOp::NotIn, 2));
        }
        None
    }

    fn compare(&mut self) -> Expr {
        let mut left = self.range_expr();
        let mut previous_right: Option<Span> = None;
        while let Some((op, length)) = self.comparison_here() {
            if self.stopped {
                break;
            }
            let first = self.bump().span;
            let op_span = if length == 2 {
                first.cover(self.bump().span)
            } else {
                first
            };
            let right = self.range_expr();
            if let Some(middle) = previous_right {
                self.chained_comparison(middle, op_span);
            }
            previous_right = Some(right.span);
            left = self.binary(op, op_span, left, right);
        }
        left
    }

    /// `a < b < c`: one diagnostic, with a fix rewriting it to `a < b and b < c`.
    fn chained_comparison(&mut self, middle: Span, op_span: Span) {
        let middle_text = self.slice(middle).to_owned();
        self.report(
            Diagnostic::error(
                "syntax",
                op_span,
                Message::new(message!(
                    "Comparisons can't be chained; write each one and join them with `and`"
                )),
            )
            .with_fix(Fix {
                label: Message::with(
                    message!("Compare `{0}` twice, joined with `and`"),
                    [Arg::from(middle_text.clone())],
                ),
                changes: vec![Change {
                    span: Span::new(middle.end, middle.end),
                    insert: format!(" and {middle_text}"),
                }],
            }),
        );
    }

    fn at_range(&self) -> bool {
        self.at("..") || self.at("..=")
    }

    /// Whether the current token, on the same line, can start a range's end.
    fn starts_range_end(&self) -> bool {
        !self.newline_before() && self.starts_operand() && !self.at("{") && !self.at_range()
    }

    /// Whether the current token can start a value.
    pub(super) fn starts_operand(&self) -> bool {
        let token = self.peek();
        match token.kind {
            TokenKind::Number
            | TokenKind::Money
            | TokenKind::Percent
            | TokenKind::Date
            | TokenKind::Text
            | TokenKind::Name
            | TokenKind::TypeName
            | TokenKind::Code
            | TokenKind::Unit => true,
            TokenKind::Keyword => {
                matches!(
                    self.slice(token.span),
                    "true" | "false" | "none" | "if" | "match" | "not" | "fail"
                ) || (self.at_word("charge") && self.peek_is(1, "("))
            }
            TokenKind::Punct => {
                matches!(self.slice(token.span), "(" | "[" | "{" | "-" | ".." | "..=")
            }
            _ => false,
        }
    }

    fn range_expr(&mut self) -> Expr {
        if self.at_range() {
            let op = self.bump();
            let inclusive = self.slice(op.span) == "..=";
            let end = self.starts_range_end().then(|| self.sum());
            return self.range(None, op.span, end, inclusive);
        }
        let start = self.sum();
        if self.at_range() && !self.newline_before() {
            let op = self.bump();
            let inclusive = self.slice(op.span) == "..=";
            let end = self.starts_range_end().then(|| self.sum());
            return self.range(Some(start), op.span, end, inclusive);
        }
        start
    }

    fn range(&mut self, start: Option<Expr>, op: Span, end: Option<Expr>, inclusive: bool) -> Expr {
        let mut span = op;
        let mut height = 0;
        for side in [&start, &end].into_iter().flatten() {
            span = span.cover(side.span);
            height = height.max(self.height(side));
        }
        self.node_with(
            ExprKind::Range {
                start: start.map(Box::new),
                end: end.map(Box::new),
                inclusive,
            },
            span,
            height,
        )
    }

    fn sum(&mut self) -> Expr {
        let mut left = self.product();
        loop {
            let op = if self.at("+") {
                BinaryOp::Add
            } else if self.at("-") {
                if self.newline_before() && self.arm_nest == Some(self.nest) {
                    break;
                }
                BinaryOp::Sub
            } else {
                break;
            };
            if self.stopped {
                break;
            }
            let op_span = self.bump().span;
            let right = self.product();
            left = self.binary(op, op_span, left, right);
        }
        left
    }

    fn product(&mut self) -> Expr {
        let mut left = self.unary_expr();
        loop {
            let op = if self.at("*") {
                BinaryOp::Mul
            } else if self.at("/") {
                BinaryOp::Div
            } else if self.at("%") {
                BinaryOp::Rem
            } else {
                break;
            };
            if self.stopped {
                break;
            }
            let op_span = self.bump().span;
            let right = self.unary_expr();
            left = self.binary(op, op_span, left, right);
        }
        left
    }

    fn unary_expr(&mut self) -> Expr {
        if self.at("-") {
            let op = self.bump().span;
            if !self.enter() {
                return self.missing();
            }
            let operand = self.unary_expr();
            self.leave();
            return self.unary(UnaryOp::Neg, op, operand);
        }
        self.postfix()
    }

    fn postfix(&mut self) -> Expr {
        let mut expr = self.primary();
        while !self.stopped {
            let optional = self.at("?.");
            if self.at(".") || optional {
                let dot = self.bump().span;
                let name = self.same_line_name().unwrap_or_else(|| {
                    let end = dot.end;
                    Name {
                        text: String::new(),
                        span: Span::new(end, end),
                    }
                });
                if name.text.is_empty() {
                    self.expected(Message::with(
                        message!("Expected a name after `{0}`"),
                        [Arg::from(self.slice(dot).to_owned())],
                    ));
                }
                let height = self.height(&expr);
                let span = expr.span.cover(dot).cover(name.span);
                expr = self.node_with(
                    ExprKind::Field {
                        target: Box::new(expr),
                        name,
                        optional,
                    },
                    span,
                    height,
                );
            } else if self.at("(") && !self.newline_before() {
                expr = self.call(expr);
            } else if self.at("[") && !self.newline_before() {
                expr = self.index(expr);
            } else {
                break;
            }
        }
        expr
    }

    fn call(&mut self, callee: Expr) -> Expr {
        let args = self.arguments();
        let close = self.previous_span();
        if let ExprKind::Field { name, .. } = &callee.kind
            && name.text == "in"
        {
            for arg in &args {
                self.mark_unit_names(&arg.value);
            }
        }
        let mut height = self.height(&callee);
        for arg in &args {
            height = height.max(self.height(&arg.value).saturating_add(1));
        }
        let span = callee.span.cover(close);
        self.node_with(
            ExprKind::Call {
                callee: Box::new(callee),
                args,
            },
            span,
            height,
        )
    }

    /// Marks the names of `.in(lb/ft3)`'s unit as units, for highlighting.
    fn mark_unit_names(&mut self, expr: &Expr) {
        match &expr.kind {
            ExprKind::Name(name) if lexer::is_unit(&name.text) => {
                self.mark_span(name.span, TokenKind::Unit);
            }
            ExprKind::Binary {
                op: BinaryOp::Div,
                left,
                right,
                ..
            } => {
                self.mark_unit_names(left);
                self.mark_unit_names(right);
            }
            _ => {}
        }
    }

    /// `( arg, … )`: positional or named arguments, a lambda allowed as any argument's value.
    pub(super) fn arguments(&mut self) -> Vec<CallArg> {
        let open = self.bump().span;
        let mut args = Vec::new();
        if !self.enter() {
            return args;
        }
        while !self.at(")") && !self.at_eof() && !self.at_statement_line() && !self.stopped {
            let start = self.peek().span.start;
            let name = if self.word_kind_at(0) && self.peek_is(1, ":") {
                let name = self.any_name();
                self.bump();
                name
            } else {
                None
            };
            let value = if self.starts_lambda() {
                self.lambda()
            } else if self.starts_operand() || name.is_some() {
                self.expr()
            } else {
                self.expected(Message::new(message!("Expected a value, or `)`")));
                self.recover_in_list(")");
                continue;
            };
            let span = Span::new(start, value.span.end.max(start));
            args.push(CallArg { name, value, span });
            if self.eat(",").is_none() && !self.at_list_end(")") {
                self.missing_comma(")");
            }
        }
        self.leave();
        self.close(
            open,
            ")",
            Message::new(message!("This call is missing its closing `)`")),
        );
        args
    }

    /// A list item not followed by `,` or the closing bracket.
    fn missing_comma(&mut self, close: &str) {
        if self.at_statement_line() || self.at_eof() {
            return;
        }
        let at = self.previous_span().end;
        let diagnostic = Diagnostic::error(
            "syntax",
            self.here(),
            Message::with(message!("Expected `,` or `{0}`"), [Arg::from(close)]),
        );
        if self.starts_operand() && self.kind() != TokenKind::Error {
            self.report(diagnostic.with_fix(Fix {
                label: Message::new(message!("Add a comma")),
                changes: vec![Change {
                    span: Span::new(at, at),
                    insert: String::from(","),
                }],
            }));
        } else {
            if self.kind() != TokenKind::Error {
                self.report(diagnostic);
            }
            self.recover_in_list(close);
        }
    }

    /// Consumes `close`, or reports `message` at `open` when the list was left open.
    fn close(&mut self, open: Span, close: &str, message: Message) -> Span {
        if let Some(span) = self.eat(close) {
            return span;
        }
        self.error(open, message);
        self.previous_span()
    }

    fn starts_lambda(&self) -> bool {
        if self.kind() == TokenKind::Name && self.peek_is(1, "=>") {
            return true;
        }
        if !self.at("(") {
            return false;
        }
        let mut ahead = 1;
        loop {
            if self.peek_nth(ahead).kind != TokenKind::Name {
                return false;
            }
            ahead += 1;
            if self.peek_is(ahead, ",") {
                ahead += 1;
                continue;
            }
            return self.peek_is(ahead, ")") && self.peek_is(ahead + 1, "=>");
        }
    }

    /// `x => body` or `(a, b) => body`.
    fn lambda(&mut self) -> Expr {
        let start = self.peek().span;
        let mut params = Vec::new();
        if self.eat("(").is_some() {
            while self.kind() == TokenKind::Name {
                let token = self.bump();
                params.push(self.name_of(&token));
                if self.eat(",").is_none() {
                    break;
                }
            }
            self.eat(")");
        } else {
            let token = self.bump();
            params.push(self.name_of(&token));
        }
        self.eat("=>");
        let body = self.expr();
        let height = self.height(&body).saturating_add(1);
        let span = start.cover(body.span);
        self.node_with(
            ExprKind::Lambda {
                params,
                body: Box::new(body),
            },
            span,
            height,
        )
    }

    fn index(&mut self, target: Expr) -> Expr {
        let open = self.bump().span;
        let mut keys = Vec::new();
        if self.enter() {
            while !self.at("]") && !self.at_eof() && !self.at_statement_line() && !self.stopped {
                if !self.starts_operand() {
                    self.expected(Message::new(message!("Expected a key, or `]`")));
                    self.recover_in_list("]");
                    continue;
                }
                keys.push(self.expr());
                if self.eat(",").is_none() && !self.at_list_end("]") {
                    self.missing_comma("]");
                }
            }
            self.leave();
        }
        if keys.is_empty() {
            self.error(
                open,
                Message::new(message!("Expected at least one key in the brackets")),
            );
        }
        let close = self.close(
            open,
            "]",
            Message::new(message!("These brackets are missing their closing `]`")),
        );
        let mut height = self.height(&target);
        for key in &keys {
            height = height.max(self.height(key));
        }
        let span = target.span.cover(close);
        self.node_with(
            ExprKind::Index {
                target: Box::new(target),
                keys,
            },
            span,
            height,
        )
    }

    pub(super) fn primary(&mut self) -> Expr {
        let token = self.peek();
        match token.kind {
            TokenKind::Number => self.number(),
            TokenKind::Money => self.money(),
            TokenKind::Percent => {
                self.bump();
                let text = self.slice(token.span);
                let value =
                    lexer::parse_number(text.strip_suffix('%').unwrap_or(text)).unwrap_or_default();
                self.leaf(ExprKind::Percent(value), token.span)
            }
            TokenKind::Date => {
                self.bump();
                let kind = match lexer::parse_date_literal(self.slice(token.span)) {
                    Some(DateLiteral::Date(date)) => ExprKind::Date(date),
                    Some(DateLiteral::DateTime(at)) => ExprKind::DateTime(at),
                    None => ExprKind::Missing,
                };
                self.leaf(kind, token.span)
            }
            TokenKind::Text => {
                let text = self.text_literal();
                let height = text
                    .parts
                    .iter()
                    .filter_map(|part| match part {
                        TextPart::Expr(expr) => Some(self.height(expr).saturating_add(1)),
                        TextPart::Literal(..) => None,
                    })
                    .max()
                    .unwrap_or(0);
                let span = text.span;
                self.node_with(ExprKind::Text(text), span, height)
            }
            TokenKind::Name if self.peek_is(1, "=>") => self.misplaced_lambda(),
            TokenKind::Name | TokenKind::Code | TokenKind::Unit => {
                self.bump();
                if token.kind == TokenKind::Unit {
                    self.mark_at(self.pos - 1, TokenKind::Name);
                }
                let name = self.name_of(&token);
                self.leaf(ExprKind::Name(name), token.span)
            }
            TokenKind::TypeName => self.path(),
            TokenKind::Keyword => self.keyword_primary(),
            TokenKind::Punct => self.punct_primary(),
            TokenKind::Error => {
                self.bump();
                self.leaf(ExprKind::Missing, token.span)
            }
            _ => self.expected_value(),
        }
    }

    fn expected_value(&mut self) -> Expr {
        if self.pos == self.statement_start && !self.at_eof() {
            let token = self.peek();
            if token.kind != TokenKind::Error && !self.statement_failed {
                let found = self.slice(token.span).to_owned();
                self.error(
                    token.span,
                    Message::with(
                        message!("A line can't start with `{0}`"),
                        [Arg::from(found)],
                    ),
                );
            }
            self.bump();
            return self.leaf(ExprKind::Missing, token.span);
        }
        let previous = self.previous_span();
        let before = self.slice(previous).to_owned();
        let message = if self.pos == 0 || before.is_empty() || before.len() > 12 {
            Message::new(message!("Expected a value"))
        } else {
            Message::with(
                message!("Expected a value after `{0}`"),
                [Arg::from(before)],
            )
        };
        let span = if self.newline_before() || self.at_eof() {
            previous
        } else {
            self.peek().span
        };
        if self.kind() != TokenKind::Error && !self.statement_failed {
            self.error(span, message);
        }
        self.missing()
    }

    fn misplaced_lambda(&mut self) -> Expr {
        let lambda = self.lambda();
        self.report(Diagnostic::error(
            "misplaced",
            lambda.span,
            Message::new(message!(
                "A lambda like `x => …` can only be passed to a list method, like `stops.count(s => s.appointment)`"
            )),
        ));
        lambda
    }

    fn keyword_primary(&mut self) -> Expr {
        let token = self.peek();
        match self.slice(token.span) {
            "true" | "false" => {
                self.bump();
                let value = self.slice(token.span) == "true";
                self.leaf(ExprKind::Bool(value), token.span)
            }
            "none" => {
                self.bump();
                self.leaf(ExprKind::None, token.span)
            }
            "if" => self.if_expr(),
            "match" => self.match_expr(),
            "fail" => {
                let expr = self.fail_expr();
                self.report(Diagnostic::error(
                    "misplaced",
                    expr.span,
                    Message::new(message!(
                        "`fail` can only be a statement, or follow `??` as in `fuel.price ?? fail \"No fuel price\"`"
                    )),
                ));
                expr
            }
            "charge" if self.peek_is(1, "(") => {
                self.mark(TokenKind::Name);
                self.bump();
                let name = self.name_of(&token);
                self.leaf(ExprKind::Name(name), token.span)
            }
            _ => self.expected_value(),
        }
    }

    fn punct_primary(&mut self) -> Expr {
        if self.at("(") {
            if self.starts_lambda() {
                return self.misplaced_lambda();
            }
            return self.parenthesized();
        }
        if self.at("[") {
            return self.list();
        }
        if self.at("{") {
            return self.record_or_block();
        }
        self.expected_value()
    }

    fn parenthesized(&mut self) -> Expr {
        let open = self.bump().span;
        let mut inner = self.expr();
        let close = self.close(
            open,
            ")",
            Message::new(message!("This `(` is missing its closing `)`")),
        );
        inner.parens = inner.parens.saturating_add(1);
        inner.span = open.cover(close).cover(inner.span);
        inner
    }

    fn list(&mut self) -> Expr {
        let open = self.bump().span;
        let mut items = Vec::new();
        if self.enter() {
            while !self.at("]") && !self.at_eof() && !self.at_statement_line() && !self.stopped {
                if !self.starts_operand() {
                    self.expected(Message::new(message!("Expected a value, or `]`")));
                    self.recover_in_list("]");
                    continue;
                }
                items.push(self.expr());
                if self.eat(",").is_none() && !self.at_list_end("]") {
                    self.missing_comma("]");
                }
            }
            self.leave();
        }
        let close = self.close(
            open,
            "]",
            Message::new(message!("This list is missing its closing `]`")),
        );
        let height = items
            .iter()
            .map(|item| self.height(item))
            .max()
            .unwrap_or(0);
        self.node_with(ExprKind::List(items), open.cover(close), height)
    }

    /// Whether the `{` at the cursor opens a record literal: followed by `name :` or `}`.
    pub(super) fn starts_record(&self) -> bool {
        self.at("{") && (self.peek_is(1, "}") || (self.word_kind_at(1) && self.peek_is(2, ":")))
    }

    fn record_or_block(&mut self) -> Expr {
        let open = self.peek().span;
        if self.peek_is(1, "}") {
            self.bump();
            let close = self.bump().span;
            let span = open.cover(close);
            self.report(
                Diagnostic::error(
                    "syntax",
                    span,
                    Message::new(message!(
                        "A record needs at least one field, like `'{' name: value '}'`; an empty list is `[]`"
                    )),
                )
                .with_fix(Fix {
                    label: Message::new(message!("Use an empty list")),
                    changes: vec![Change {
                        span,
                        insert: String::from("[]"),
                    }],
                }),
            );
            return self.leaf(ExprKind::Record(Vec::new()), span);
        }
        if self.starts_record() {
            return self.record();
        }
        self.block_as_value()
    }

    /// A block where a value is expected: reported, skipped, and left missing.
    fn block_as_value(&mut self) -> Expr {
        let open = self.peek().span;
        let mut depth = 0usize;
        while !self.at_eof() {
            if self.at("{") {
                depth += 1;
            } else if self.at("}") {
                depth -= 1;
                if depth == 0 {
                    break;
                }
            }
            self.bump();
        }
        let close = self.eat("}").unwrap_or_else(|| self.previous_span());
        let span = open.cover(close);
        self.error(
            span,
            Message::new(message!(
                "A block of statements can't be used as a value here; a record is written `'{' name: value '}'`"
            )),
        );
        self.leaf(ExprKind::Missing, span)
    }

    fn record(&mut self) -> Expr {
        let open = self.bump().span;
        let mut fields = Vec::new();
        if self.enter() {
            while !self.at("}") && !self.at_eof() && !self.at_statement_line() && !self.stopped {
                let Some(name) = (self.peek_is(1, ":")).then(|| self.any_name()).flatten() else {
                    self.expected(Message::new(message!(
                        "Expected a field, written `name: value`, or a closing brace"
                    )));
                    self.recover_in_list("}");
                    continue;
                };
                self.bump();
                let value = self.expr();
                fields.push(FieldInit { name, value });
                if self.eat(",").is_none() && !self.at_list_end("}") {
                    self.missing_comma("}");
                }
            }
            self.leave();
        }
        let close = self.close(
            open,
            "}",
            Message::new(message!("This record is missing its closing brace")),
        );
        let height = fields
            .iter()
            .map(|field| self.height(&field.value))
            .max()
            .unwrap_or(0)
            .saturating_add(1);
        self.node_with(ExprKind::Record(fields), open.cover(close), height)
    }

    fn condition_height(&self, condition: &Condition) -> u8 {
        match condition {
            Condition::Expr(expr) | Condition::Let { value: expr, .. } => {
                self.height(expr).saturating_add(1)
            }
        }
    }

    /// `{ expr }`: a branch of an `if` that gives a value.
    fn value_body(&mut self) -> Expr {
        let Some(open) = self.eat("{") else {
            self.expected(Message::new(message!(
                "Expected the branch's value in braces, like `'{' $10 '}'`"
            )));
            return self.missing();
        };
        let value = if self.at("}") {
            self.expected(Message::new(message!("Expected the branch's value")));
            self.missing()
        } else {
            self.expr()
        };
        if !self.at("}") && !self.at_eof() && !self.at_column_zero_item() {
            self.expected(Message::new(message!(
                "A branch that gives a value holds one value, then a closing brace"
            )));
            self.recover_in_list("}");
        }
        self.close(
            open,
            "}",
            Message::new(message!("This branch is missing its closing brace")),
        );
        value
    }

    fn if_expr(&mut self) -> Expr {
        let start = self.bump().span;
        let mut branches = Vec::new();
        let mut height = 0u8;
        let otherwise = loop {
            let condition = self.condition_expr();
            let value = self.value_body();
            height = height
                .max(self.condition_height(&condition))
                .max(self.height(&value).saturating_add(1));
            branches.push((condition, value));
            if !self.at_word("else") {
                let at = self.previous_span().end;
                self.report(Diagnostic::error(
                    "syntax",
                    Span::new(start.start, at),
                    Message::new(message!(
                        "An `if` that gives a value needs an `else` branch"
                    )),
                ));
                break self.missing();
            }
            self.bump();
            if self.at_word("if") {
                self.bump();
                continue;
            }
            break self.value_body();
        };
        height = height.max(self.height(&otherwise));
        let span = start.cover(self.previous_span());
        self.node_with(
            ExprKind::If {
                branches,
                otherwise: Box::new(otherwise),
            },
            span,
            height,
        )
    }

    fn match_expr(&mut self) -> Expr {
        let start = self.bump().span;
        let scrutinee = self.expr();
        let arms = self.arms(false);
        let mut height = self.height(&scrutinee);
        for arm in &arms {
            if let crate::tariff::syntax::ast::ArmBody::Expr(body) = &arm.body {
                height = height.max(self.height(body).saturating_add(2));
            }
            if let Some(guard) = &arm.guard {
                height = height.max(self.height(guard).saturating_add(2));
            }
        }
        height = height.saturating_add(4);
        let span = start.cover(self.previous_span());
        self.node_with(
            ExprKind::Match {
                scrutinee: Box::new(scrutinee),
                arms,
            },
            span,
            height,
        )
    }

    /// The body of an arm that gives a value.
    pub(super) fn arm_value(&mut self) -> Expr {
        if self.at("{") && !self.starts_record() {
            return self.block_as_value();
        }
        let outer = self.arm_nest.replace(self.nest + 1);
        let value = self.expr();
        self.arm_nest = outer;
        value
    }

    fn number(&mut self) -> Expr {
        let token = self.bump();
        let text = self.slice(token.span);
        let value = lexer::parse_number(text).unwrap_or_default();
        let integer = !text.contains('.');
        let unit = self.unit_after();
        let span = unit
            .as_ref()
            .map_or(token.span, |unit| token.span.cover(unit.span));
        self.leaf(
            ExprKind::Number {
                value,
                integer,
                unit,
            },
            span,
        )
    }

    /// The unit tokens the lexer read after a number: `lb`, `lb/ft3`.
    fn unit_after(&mut self) -> Option<UnitExpr> {
        if self.kind() != TokenKind::Unit || self.newline_before() {
            return None;
        }
        let first = self.bump();
        let mut parts = vec![self.name_of(&first)];
        while self.at("/") && self.peek_nth(1).kind == TokenKind::Unit {
            self.bump();
            let unit = self.bump();
            parts.push(self.name_of(&unit));
        }
        let span = parts
            .iter()
            .fold(first.span, |span, part| span.cover(part.span));
        Some(UnitExpr { parts, span })
    }

    fn money(&mut self) -> Expr {
        let token = self.bump();
        let text = self.slice(token.span);
        let value = lexer::parse_number(text.get(1..).unwrap_or_default()).unwrap_or_default();
        let per = if self.at("/") && self.peek_nth(1).kind == TokenKind::Unit {
            self.bump();
            let first = self.bump();
            let mut parts = vec![self.name_of(&first)];
            while self.at("/") && self.peek_nth(1).kind == TokenKind::Unit {
                self.bump();
                let unit = self.bump();
                parts.push(self.name_of(&unit));
            }
            let span = parts
                .iter()
                .fold(first.span, |span, part| span.cover(part.span));
            Some(UnitExpr { parts, span })
        } else {
            None
        };
        let span = per
            .as_ref()
            .map_or(token.span, |per| token.span.cover(per.span));
        self.leaf(ExprKind::Money { value, per }, span)
    }

    /// `PascalCase`, optionally `.member`: a variant, or a type for a conversion.
    fn path(&mut self) -> Expr {
        let token = self.bump();
        let head = self.name_of(&token);
        let member = if self.at(".") && self.word_kind_at(1) {
            self.bump();
            self.any_name()
        } else {
            None
        };
        let span = member
            .as_ref()
            .map_or(head.span, |member| head.span.cover(member.span));
        self.leaf(ExprKind::Path { head, member }, span)
    }

    /// A text literal at the cursor: its literal runs, escapes and interpolations.
    pub(super) fn text_literal(&mut self) -> Text {
        let first = self.bump();
        let raw = self.slice(first.span);
        let mut parts: Vec<TextPart> = Vec::new();
        let body_start = first.span.start + 1;
        let mut closed = raw.len() >= 2 && raw.ends_with('"');
        let body_end = if closed {
            first.span.end - 1
        } else {
            first.span.end
        };
        push_literal(
            &mut parts,
            raw.get(1..raw.len() - usize::from(closed))
                .unwrap_or_default(),
            Span::new(body_start, body_end),
        );
        let mut end = first.span.end;
        while !closed {
            let token = self.peek();
            if token.span.start != end {
                break;
            }
            let raw = self.slice(token.span);
            match token.kind {
                TokenKind::Text => {
                    self.bump();
                    closed = raw.ends_with('"');
                    let literal_end = token.span.end - u32::from(closed);
                    push_literal(
                        &mut parts,
                        raw.get(..raw.len() - usize::from(closed))
                            .unwrap_or_default(),
                        Span::new(token.span.start, literal_end),
                    );
                    end = token.span.end;
                }
                TokenKind::Escape => {
                    self.bump();
                    push_literal(&mut parts, &decode_escape(raw), token.span);
                    end = token.span.end;
                }
                TokenKind::Error => {
                    self.bump();
                    end = token.span.end;
                }
                TokenKind::Interpolation => {
                    self.bump();
                    let expr = if self.peek().kind == TokenKind::Interpolation {
                        self.error(
                            token.span.cover(self.peek().span),
                            Message::new(message!(
                                "Expected a value inside the braces; a literal brace is written `'{{'`"
                            )),
                        );
                        self.missing()
                    } else {
                        self.expr()
                    };
                    parts.push(TextPart::Expr(expr));
                    if self.kind() == TokenKind::Interpolation {
                        end = self.bump().span.end;
                    } else {
                        self.skip_to_interpolation_end();
                        if self.kind() != TokenKind::Interpolation {
                            break;
                        }
                        end = self.bump().span.end;
                    }
                }
                _ => break,
            }
        }
        Text {
            parts,
            span: Span::new(first.span.start, end),
        }
    }

    /// After an interpolated value that did not end at its closing brace: skips what is left,
    /// with one diagnostic.
    fn skip_to_interpolation_end(&mut self) {
        let mut reported = false;
        while !self.at_eof() && self.kind() != TokenKind::Interpolation && !self.newline_before() {
            if !reported && self.kind() != TokenKind::Error {
                self.expected(Message::new(message!(
                    "Expected the closing brace of the interpolation"
                )));
                reported = true;
            }
            self.bump();
        }
    }
}

fn push_literal(parts: &mut Vec<TextPart>, text: &str, span: Span) {
    if let Some(TextPart::Literal(existing, existing_span)) = parts.last_mut() {
        existing.push_str(text);
        *existing_span = existing_span.cover(span);
        return;
    }
    if text.is_empty() && span.start == span.end {
        return;
    }
    parts.push(TextPart::Literal(text.to_owned(), span));
}

fn decode_escape(raw: &str) -> String {
    match raw {
        "\\\"" => String::from("\""),
        "\\\\" => String::from("\\"),
        "\\n" => String::from("\n"),
        "\\t" => String::from("\t"),
        "{{" => String::from("{"),
        "}}" => String::from("}"),
        _ => raw
            .strip_prefix("\\u{")
            .and_then(|rest| rest.strip_suffix('}'))
            .and_then(|hex| u32::from_str_radix(hex, 16).ok())
            .and_then(char::from_u32)
            .map(String::from)
            .unwrap_or_default(),
    }
}
