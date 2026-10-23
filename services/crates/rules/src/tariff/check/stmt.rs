//! Statements and blocks: bindings and assignment, charges and notes, `if`, `match` and
//! `for`, refusals, `return`, `break` and `continue`, total clauses, and a test's `given` and
//! `expect`.

use super::HintKind;
use super::checker::{Body, Checker, Global, Want};
use super::describe::{quote, quote_span, words};
use super::fixes::replace;
use super::operators;
use super::scope::LocalKind;
use crate::message;
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::syntax::ast::{
    ArmBody, AssignOp, BinaryOp, Binding, Block, Charge, Expect, Expr, ExprKind, FileKind, ForStmt,
    IfStmt, Name, RoundTo, Stmt, StmtKind, TotalClause,
};
use crate::tariff::types::Type;

impl Checker<'_> {
    /// A block: a scope of its own.
    pub fn block(&mut self, block: &Block) {
        self.scopes.push(later_names(&block.stmts));
        self.stmts(&block.stmts);
        self.pop_scope();
    }

    /// A sequence of statements in the current scope, with code after a statement that always
    /// leaves reported.
    pub fn stmts(&mut self, stmts: &[Stmt]) {
        let mut left: Option<Span> = None;
        for stmt in stmts {
            if let Some(exit) = left.take() {
                self.report(
                    Diagnostic::warning(
                        "unreachable",
                        stmt.span,
                        Message::new(message!(
                            "This never runs: the statement before it always leaves"
                        )),
                    )
                    .with_related(exit, Message::new(message!("It leaves here"))),
                );
            }
            self.stmt(stmt);
            if self.exits(stmt) {
                left = Some(stmt.span);
            }
        }
    }

    /// One statement.
    pub fn stmt(&mut self, stmt: &Stmt) {
        self.statements.push(stmt.span);
        self.stmt_kind(stmt);
        self.statements.pop();
    }

    fn stmt_kind(&mut self, stmt: &Stmt) {
        match &stmt.kind {
            StmtKind::Let(binding) => self.binding(binding, LocalKind::Let),
            StmtKind::Var(binding) => self.binding(binding, LocalKind::Var),
            StmtKind::Assign { target, op, value } => self.assign(target, *op, value),
            StmtKind::Charge(charge) => self.charge(stmt, charge),
            StmtKind::Note(text) => {
                self.billing(stmt.span, Message::new(message!("A note")));
                self.text_parts(text);
            }
            StmtKind::If(branches) => self.if_stmt(branches),
            StmtKind::Match(statement) => {
                self.match_arms(
                    statement.span,
                    &statement.scrutinee,
                    &statement.arms,
                    None,
                    true,
                );
            }
            StmtKind::For(looped) => self.for_stmt(looped),
            StmtKind::Require { condition, message } => {
                self.text_parts(message);
                self.condition(condition);
            }
            StmtKind::Fail(text) => self.text_parts(text),
            StmtKind::Return(value) => {
                if let Body::Function { name, ret } = self.body.clone() {
                    self.expect(value, &ret, &Want::Return { name });
                } else {
                    self.expr(value, None);
                    self.report(Diagnostic::error(
                        "misplaced",
                        stmt.span,
                        Message::new(message!("`return` can only be used inside a function")),
                    ));
                }
            }
            StmtKind::Break | StmtKind::Continue => {
                if self.loops == 0 {
                    self.report(Diagnostic::error(
                        "misplaced",
                        stmt.span,
                        Message::with(
                            message!("{0} can only be used inside a `for` loop"),
                            [quote(self.text(stmt.span))],
                        ),
                    ));
                }
            }
            StmtKind::Total(clause) => self.total(stmt, clause),
            StmtKind::Given { target, value } => self.given(stmt, target, value),
            StmtKind::Expect(expect) => {
                self.expected_yet = true;
                match expect {
                    Expect::Holds(condition) => {
                        self.expect(condition, &Type::Bool, &Want::Condition);
                    }
                    Expect::Fails(Some(text)) => self.text_parts(text),
                    Expect::Fails(None) => {}
                }
            }
            StmtKind::Expr(value) => {
                let ty = self.expr(value, None);
                if !matches!(ty, Type::Error) && !refuses(value) {
                    self.report(Diagnostic::warning(
                        "unused-value",
                        value.span,
                        Message::with(
                            message!("{0} is worked out and never used: bind it with `let`, bill it with `charge`, or remove it"),
                            [quote_span(self.source, value.span)],
                        ),
                    ));
                }
            }
            StmtKind::Item(_) => {}
        }
    }

    fn binding(&mut self, binding: &Binding, kind: LocalKind) {
        let ty = if let Some(annotation) = &binding.ty {
            let declared = self.lower_type(annotation);
            self.expect(
                &binding.value,
                &declared,
                &Want::Annotation {
                    name: binding.name.text.clone(),
                },
            );
            declared
        } else {
            let ty = self.expr(&binding.value, None);
            let ty = self.settle(&binding.name, &binding.value, ty);
            self.hint(binding.name.span, &ty, HintKind::Binding);
            ty
        };
        let ty = if matches!(ty, Type::Never) {
            Type::Error
        } else {
            ty
        };
        self.declare(&binding.name, kind, ty);
    }

    /// An inferred binding type that an empty value leaves unknown (`none`, `[]`) is an error
    /// asking for an annotation.
    fn settle(&mut self, name: &Name, value: &Expr, ty: Type) -> Type {
        let unknown = match &ty {
            Type::Optional(inner) | Type::List(inner) => matches!(**inner, Type::Never),
            _ => false,
        };
        if !unknown {
            return ty;
        }
        let example = if ty.is_optional() {
            format!("let {}: Weight? = none", name.text)
        } else {
            format!("let {}: List<Text> = []", name.text)
        };
        self.report(Diagnostic::error(
            "annotation-needed",
            value.span,
            Message::with(
                message!("{0} doesn't say what {1} holds: write its type, as in {2}"),
                [
                    quote_span(self.source, value.span),
                    quote(&name.text),
                    quote(&example),
                ],
            ),
        ));
        Type::Error
    }

    fn assign(&mut self, target: &Name, op: AssignOp, value: &Expr) {
        let Some(id) = self.scopes.find(&target.text) else {
            self.expr(value, None);
            if self.globals.values.contains_key(&target.text)
                || self.schema.root(&target.text).is_some()
            {
                self.report(Diagnostic::error(
                    "assign",
                    target.span,
                    Message::with(
                        message!("Only a `var` can be changed, and {0} isn't one"),
                        [quote(&target.text)],
                    ),
                ));
            } else {
                self.unknown_name(target.span, &target.text);
            }
            return;
        };
        let Some(local) = self.scopes.get(id).cloned() else {
            return;
        };
        self.reference(target.span, local.symbol);
        if local.kind != LocalKind::Var {
            let mut diagnostic = Diagnostic::error(
                "assign",
                target.span,
                Message::with(
                    message!(
                        "{0} can't be changed: only a `var` can. Declare it with `var` to accumulate into it"
                    ),
                    [quote(&target.text)],
                ),
            );
            if local.kind == LocalKind::Let
                && let Some(keyword) = self.let_keyword(local.span)
            {
                diagnostic = diagnostic.with_fix(replace(
                    Message::new(message!("Declare it with `var`")),
                    keyword,
                    "var",
                ));
            }
            self.report(diagnostic);
            self.expr(value, None);
            return;
        }
        let want = Want::Annotation {
            name: target.text.clone(),
        };
        match op {
            AssignOp::Set => {
                self.expect(value, &local.ty, &want);
            }
            AssignOp::Add | AssignOp::Sub => {
                let right = self.operand(value);
                let binary = if op == AssignOp::Add {
                    BinaryOp::Add
                } else {
                    BinaryOp::Sub
                };
                match operators::arithmetic(binary, &local.ty, &right) {
                    Ok(result) if local.ty.accepts(&result) => {}
                    Ok(result) => self.mismatch(value, &result, &local.ty, &want),
                    Err(problem) => {
                        let target_expr = Expr {
                            id: crate::tariff::syntax::ast::NodeId(u32::MAX),
                            span: target.span,
                            parens: 0,
                            kind: ExprKind::Name(target.clone()),
                        };
                        self.operator_error(
                            problem,
                            binary,
                            (&target_expr, &local.ty),
                            (value, &right),
                            target.span.cover(value.span),
                        );
                    }
                }
            }
        }
    }

    /// The `let` keyword before a binding's name.
    fn let_keyword(&self, name: Span) -> Option<Span> {
        let head = self.source.get(..usize::try_from(name.start).ok()?)?;
        let at = head.trim_end().strip_suffix("let")?.len();
        let start = u32::try_from(at).ok()?;
        Some(Span::new(start, start + 3))
    }

    /// Refuses a billing statement (a charge or note) outside a tariff's top level and its
    /// blocks.
    fn billing(&mut self, span: Span, what: Message) -> bool {
        let reason = match (&self.body, self.kind) {
            (Body::Function { .. }, _) => Some(Message::new(message!(
                "a function computes values; only a tariff bills them"
            ))),
            (Body::Test, _) => Some(Message::new(message!(
                "a test checks the bill the tariff makes"
            ))),
            (_, FileKind::Library) => Some(Message::new(message!("a library has no bill"))),
            (_, FileKind::Condition) => Some(Message::new(message!(
                "a condition decides true or false and bills nothing"
            ))),
            _ => None,
        };
        let Some(reason) = reason else {
            return true;
        };
        self.report(Diagnostic::error(
            "misplaced",
            span,
            Message::with(
                message!("{0} can't be written here: {1}"),
                [Arg::from(what), Arg::from(reason)],
            ),
        ));
        false
    }

    fn charge(&mut self, stmt: &Stmt, charge: &Charge) {
        self.billing(stmt.span, Message::new(message!("A charge")));
        self.text_parts(&charge.label);
        if let Some(code) = &charge.code
            && (super::calls::is_code(&code.text) || !code.text.is_empty())
        {
            self.code_known(code);
        }
        let (quantity, rate, unknown_option) = self.charge_options(charge);
        if let (Some((q, qt)), Some((r, rt))) = (&quantity, &rate) {
            let product = operators::arithmetic(BinaryOp::Mul, rt, qt);
            if !matches!(&product, Ok(ty) if ty.is_money() || matches!(ty, Type::Error)) {
                let mut message = Message::with(
                    message!(
                        "The rate {0} is {1} and the quantity {2} is {3}, so they don't multiply to money: a rate is priced per the quantity's unit, as in `(quantity: miles, rate: $2.35/mi)`"
                    ),
                    [
                        quote_span(self.source, r.span),
                        Arg::from(words(rt)),
                        quote_span(self.source, q.span),
                        Arg::from(words(qt)),
                    ],
                );
                if let Ok(product) = &product {
                    message = Message::with(
                        message!(
                            "The rate {0} is {1} and the quantity {2} is {3}, so together they are {4}, not money"
                        ),
                        [
                            quote_span(self.source, r.span),
                            Arg::from(words(rt)),
                            quote_span(self.source, q.span),
                            Arg::from(words(qt)),
                            Arg::from(words(product)),
                        ],
                    );
                }
                let span = charge.options.first().map_or(stmt.span, |first| {
                    first
                        .span
                        .cover(charge.options.last().map_or(first.span, |last| last.span))
                });
                self.report(Diagnostic::error("charge-option", span, message));
            }
        }
        match &charge.amount {
            Some(amount) => {
                self.expect(amount, &Type::money(), &Want::Amount);
            }
            None if (quantity.is_some() && rate.is_some()) || unknown_option => {}
            None => {
                self.report(Diagnostic::error(
                    "charge-amount",
                    stmt.span,
                    Message::new(message!(
                        "A charge needs an amount: write `= amount`, or give both `quantity:` and `rate:` to bill their product"
                    )),
                ));
            }
        }
    }

    /// A charge's `quantity:` and `rate:` options with their types, and whether an unknown
    /// option was written.
    fn charge_options<'c>(&mut self, charge: &'c Charge) -> (Given<'c>, Given<'c>, bool) {
        let mut unknown_option = false;
        let mut quantity: Option<(&Expr, Type)> = None;
        let mut rate: Option<(&Expr, Type)> = None;
        for option in &charge.options {
            let Some(name) = &option.name else {
                self.expr(&option.value, None);
                self.report(Diagnostic::error(
                    "charge-option",
                    option.span,
                    Message::new(message!(
                        "Name each charge option: `(quantity: q, rate: r)`"
                    )),
                ));
                continue;
            };
            let slot = match name.text.as_str() {
                "quantity" => &mut quantity,
                "rate" => &mut rate,
                _ => {
                    unknown_option = true;
                    self.expr(&option.value, None);
                    let candidates = vec![String::from("quantity"), String::from("rate")];
                    let message =
                        Message::with(message!("A charge has no option {0}"), [quote(&name.text)]);
                    let missing = Message::with(
                        message!(
                            "A charge has no option {0}; its options are `quantity` and `rate`"
                        ),
                        [quote(&name.text)],
                    );
                    let diagnostic = Self::did_you_mean(
                        "charge-option",
                        name.span,
                        &name.text,
                        &candidates,
                        message,
                        missing,
                    );
                    self.report(diagnostic);
                    continue;
                }
            };
            if slot.is_some() {
                self.expr(&option.value, None);
                self.report(Diagnostic::error(
                    "duplicate-argument",
                    option.span,
                    Message::with(message!("{0} is given twice"), [quote(&name.text)]),
                ));
                continue;
            }
            let ty = self.operand(&option.value);
            *slot = Some((&option.value, ty));
        }
        (quantity, rate, unknown_option)
    }

    fn if_stmt(&mut self, statement: &IfStmt) {
        for (condition, block) in &statement.branches {
            self.scopes.push(Vec::new());
            self.condition(condition);
            self.block(block);
            self.pop_scope();
        }
        if let Some(otherwise) = &statement.otherwise {
            self.block(otherwise);
        }
    }

    fn for_stmt(&mut self, looped: &ForStmt) {
        let item = self.iterable(&looped.iterable);
        self.scopes.push(Vec::new());
        self.hint(looped.binding.span, &item, HintKind::Loop);
        self.declare(&looped.binding, LocalKind::Loop, item);
        if let Some(filter) = &looped.filter {
            self.expect(filter, &Type::Bool, &Want::Condition);
        }
        self.loops += 1;
        self.block(&looped.body);
        self.loops -= 1;
        self.pop_scope();
    }

    /// The item type of what a `for` walks: a list's items, or the `Int`s of a range.
    fn iterable(&mut self, iterable: &Expr) -> Type {
        if let ExprKind::Range { start, end, .. } = &iterable.kind {
            self.record(iterable, Type::Error);
            for part in [start, end].into_iter().flatten() {
                self.expect(part, &Type::Int, &Want::Value);
            }
            if start.is_none() || end.is_none() {
                self.report(Diagnostic::error(
                    "range",
                    iterable.span,
                    Message::new(message!(
                        "A `for` walks a range with both ends, as in `0..n`"
                    )),
                ));
            }
            return Type::Int;
        }
        let ty = self.operand(iterable);
        match ty {
            Type::List(item) => (*item).clone(),
            Type::Error | Type::Never => Type::Error,
            other => {
                self.report(Diagnostic::error(
                    "not-iterable",
                    iterable.span,
                    Message::with(
                        message!("{0} is {1}; `for` walks a list or a range of whole numbers"),
                        [
                            quote_span(self.source, iterable.span),
                            Arg::from(words(&other)),
                        ],
                    ),
                ));
                Type::Error
            }
        }
    }

    fn total(&mut self, stmt: &Stmt, clause: &TotalClause) {
        if !matches!(self.body, Body::Top)
            || self.kind != FileKind::Tariff
            || self.statements.len() > 1
        {
            self.report(Diagnostic::error(
                "misplaced",
                stmt.span,
                Message::new(message!("Total clauses (`minimum`, `maximum`, `round`) apply to the whole bill, so they go at a tariff's top level, after its charges")),
            ));
        }
        match clause {
            TotalClause::Minimum(amount) | TotalClause::Maximum(amount) => {
                self.expect(amount, &Type::money(), &Want::Total);
            }
            TotalClause::Round {
                to: RoundTo::Amount(amount),
                ..
            } => {
                self.expect(amount, &Type::money(), &Want::Total);
                if let ExprKind::Money { value, per: None } = &amount.kind
                    && (value.is_sign_negative() || value.is_zero())
                {
                    self.report(Diagnostic::error(
                            "round-step",
                            amount.span,
                            Message::new(message!("The bill rounds to a multiple of this amount, so it must be more than `$0`")),
                        ));
                }
            }
            TotalClause::Round {
                to: RoundTo::Cents, ..
            } => {}
        }
    }

    fn given(&mut self, stmt: &Stmt, target: &[Name], value: &Expr) {
        if !matches!(self.body, Body::Test) {
            self.expr(value, None);
            return;
        }
        if self.expected_yet {
            self.report(Diagnostic::error(
                "given-order",
                stmt.span,
                Message::new(message!("Every `given` comes before the first `expect`: the tariff runs once, after the givens")),
            ));
        }
        let path: String = target
            .iter()
            .map(|part| part.text.as_str())
            .collect::<Vec<_>>()
            .join(".");
        let Some(ty) = self.given_target(target) else {
            self.expr(value, None);
            return;
        };
        self.expect(value, &ty, &Want::Given { path });
    }

    /// The type of what a `given` sets: an input, or a field of a record root.
    fn given_target(&mut self, target: &[Name]) -> Option<Type> {
        let (head, rest) = target.split_first()?;
        if head.text == "input" {
            let [name] = rest else {
                self.report(Diagnostic::error(
                    "given-target",
                    head.span,
                    Message::new(message!("Set an input by its name: `given input.rate = …`")),
                ));
                return None;
            };
            return if let Some(Global::Input { ty, symbol }) = self.globals.values.get(&name.text) {
                let (ty, symbol) = (ty.clone(), *symbol);
                self.reference(name.span, symbol);
                self.used.insert(name.text.clone());
                Some(ty)
            } else {
                let candidates: Vec<String> = self
                    .globals
                    .values
                    .iter()
                    .filter(|(_, global)| matches!(global, Global::Input { .. }))
                    .map(|(name, _)| name.clone())
                    .collect();
                let message =
                    Message::with(message!("This file has no input {0}"), [quote(&name.text)]);
                let diagnostic = Self::did_you_mean(
                    "given-target",
                    name.span,
                    &name.text,
                    &candidates,
                    message.clone(),
                    message,
                );
                self.report(diagnostic);
                None
            };
        }
        let Some(root) = self.schema.root(&head.text).cloned() else {
            self.report(Diagnostic::error(
                "given-target",
                head.span,
                Message::with(
                    message!("A `given` sets a field of the record (`shipment`, `lane`, `customer`, `fuel`, `rating`) or an input (`input.rate`), not {0}"),
                    [quote(&head.text)],
                ),
            ));
            return None;
        };
        if let Some(symbol) = self.root_symbol(&head.text) {
            self.reference(head.span, symbol);
        }
        if rest.is_empty() {
            self.report(Diagnostic::error(
                "given-target",
                head.span,
                Message::with(
                    message!("Set one field of {0}, as in {1}"),
                    [quote(&head.text), quote(&format!("given {}.…", head.text))],
                ),
            ));
            return None;
        }
        let mut ty = root;
        for part in rest {
            let Type::Record(record) = ty.clone() else {
                self.report(Diagnostic::error(
                    "given-target",
                    part.span,
                    Message::with(
                        message!("{0} has no fields to set"),
                        [Arg::from(words(&ty))],
                    ),
                ));
                return None;
            };
            let Some(field) = record.field(&part.text) else {
                let candidates: Vec<String> =
                    record.fields.iter().map(|f| f.name.clone()).collect();
                let message = Message::with(
                    message!("There is no field {0} to set"),
                    [quote(&part.text)],
                );
                let diagnostic = Self::did_you_mean(
                    "unknown-field",
                    part.span,
                    &part.text,
                    &candidates,
                    message.clone(),
                    message,
                );
                self.report(diagnostic);
                return None;
            };
            ty = field.ty.clone();
        }
        Some(ty)
    }

    /// Whether a statement always leaves its block (`return`, `fail`, `break`, `continue`, or
    /// an `if` or exhaustive `match` all of whose branches do).
    pub fn exits(&self, stmt: &Stmt) -> bool {
        match &stmt.kind {
            StmtKind::Return(_) | StmtKind::Fail(_) | StmtKind::Break | StmtKind::Continue => true,
            StmtKind::If(statement) => {
                statement
                    .otherwise
                    .as_ref()
                    .is_some_and(|block| self.block_exits(block))
                    && statement
                        .branches
                        .iter()
                        .all(|(_, block)| self.block_exits(block))
            }
            StmtKind::Match(statement) => {
                self.exhaustive.contains(&statement.span.start)
                    && statement.arms.iter().all(|arm| match &arm.body {
                        ArmBody::Block(block) => self.block_exits(block),
                        ArmBody::Expr(_) => false,
                    })
            }
            _ => false,
        }
    }

    /// Whether a block always leaves.
    pub fn block_exits(&self, block: &Block) -> bool {
        block.stmts.iter().any(|stmt| self.exits(stmt))
    }
}

/// A charge option's value and type, when it is written.
type Given<'c> = Option<(&'c Expr, Type)>;

/// Whether an expression statement refuses to rate when its value is missing (`x ?? fail
/// "…"`), so it is kept for that.
fn refuses(expr: &Expr) -> bool {
    match &expr.kind {
        ExprKind::Binary {
            op: BinaryOp::Coalesce,
            right,
            ..
        } => matches!(right.kind, ExprKind::Fail(_)) || refuses(right),
        _ => false,
    }
}

/// The `let`s and `var`s a block declares, for "used before it is defined".
#[must_use]
pub fn later_names(stmts: &[Stmt]) -> Vec<(String, Span)> {
    stmts
        .iter()
        .filter_map(|stmt| match &stmt.kind {
            StmtKind::Let(binding) | StmtKind::Var(binding) => {
                Some((binding.name.text.clone(), binding.name.span))
            }
            _ => None,
        })
        .collect()
}
