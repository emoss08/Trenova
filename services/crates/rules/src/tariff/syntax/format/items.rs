//! The layout of the file, its items, blocks and statements.

use super::Formatter;
use super::doc::Doc;
use super::exprs::BRACES;
use super::exprs::PARENS;
use crate::tariff::syntax::ast::AssignOp;
use crate::tariff::syntax::ast::Block;
use crate::tariff::syntax::ast::Charge;
use crate::tariff::syntax::ast::ChoiceOption;
use crate::tariff::syntax::ast::Choose;
use crate::tariff::syntax::ast::Expect;
use crate::tariff::syntax::ast::File;
use crate::tariff::syntax::ast::FnBody;
use crate::tariff::syntax::ast::FnDecl;
use crate::tariff::syntax::ast::Item;
use crate::tariff::syntax::ast::ItemKind;
use crate::tariff::syntax::ast::RoundMode;
use crate::tariff::syntax::ast::RoundTo;
use crate::tariff::syntax::ast::Stmt;
use crate::tariff::syntax::ast::StmtKind;
use crate::tariff::syntax::ast::TotalClause;
use crate::tariff::syntax::ast::Use;

impl Formatter<'_> {
    pub(super) fn file(&mut self, file: &File) -> Doc {
        let mut parts = Vec::new();
        if let Some(header) = &file.header {
            for doc in self.leading(header.span.start, false) {
                parts.push(doc);
            }
            parts.push(Doc::text(format!(
                "{} {}",
                header.kind.keyword(),
                self.slice(header.name.span)
            )));
            self.last_end = header.span.end;
            if let Some(comment) = self.trailing(header.span.end) {
                parts.push(comment);
            }
        }
        let headed = file.header.is_some();
        let mut items = self.vertical(&file.items, headed, |item| item.span, Self::item);
        if !headed && items.first() == Some(&Doc::HardLine) {
            items.remove(0);
        }
        parts.extend(items);
        let end = u32::try_from(self.source.len()).unwrap_or(u32::MAX);
        parts.extend(self.closing(end));
        Doc::Concat(parts)
    }

    fn item(&mut self, item: &Item) -> Doc {
        let mut parts = Vec::new();
        if item.private.is_some() {
            parts.push(Doc::text("private "));
        }
        parts.push(match &item.kind {
            ItemKind::Use(using) => self.use_item(using, item.span.end),
            ItemKind::Input(input) => {
                let ty = self.type_expr(&input.ty);
                let mut parts = vec![Doc::text(format!("input {}: ", input.name.text)), ty];
                if let Some(default) = &input.default {
                    parts.push(Doc::text(" = "));
                    parts.push(self.expr(default));
                }
                if let Some(constraint) = &input.constraint {
                    parts.push(Doc::text(" where "));
                    parts.push(self.expr(constraint));
                }
                Doc::Concat(parts)
            }
            ItemKind::Const(constant) => {
                let mut parts = vec![Doc::text(format!("const {}", constant.name.text))];
                if let Some(ty) = &constant.ty {
                    parts.push(Doc::text(": "));
                    parts.push(self.type_expr(ty));
                }
                parts.push(Doc::text(" = "));
                parts.push(self.expr(&constant.value));
                Doc::Concat(parts)
            }
            ItemKind::Type(decl) => {
                let fields = self.list(
                    BRACES,
                    &decl.fields,
                    item.span.end,
                    |field| field.name.span.cover(field.ty.span),
                    |this, field| {
                        let ty = this.type_expr(&field.ty);
                        Doc::Concat(vec![Doc::text(format!("{}: ", field.name.text)), ty])
                    },
                );
                Doc::Concat(vec![
                    Doc::text(format!("type {} = ", decl.name.text)),
                    fields,
                ])
            }
            ItemKind::Enum(decl) => {
                let variants = self.list(
                    BRACES,
                    &decl.variants,
                    item.span.end,
                    |variant| variant.name.span,
                    |_, variant| Doc::text(variant.name.text.clone()),
                );
                Doc::Concat(vec![
                    Doc::text(format!("enum {} ", decl.name.text)),
                    variants,
                ])
            }
            ItemKind::Fn(function) => self.function(function),
            ItemKind::Test(test) => {
                let body = self.block(&test.body);
                Doc::Concat(vec![
                    Doc::text(format!("test {} ", self.slice(test.name.span))),
                    body,
                ])
            }
            ItemKind::Property(property) => {
                let body = self.block(&property.body);
                Doc::Concat(vec![
                    Doc::text(format!("property {} ", self.slice(property.name.span))),
                    body,
                ])
            }
            ItemKind::Stmt(stmt) => self.stmt(stmt),
        });
        Doc::Concat(parts)
    }

    fn use_item(&mut self, using: &Use, end: u32) -> Doc {
        let mut parts = vec![Doc::text(format!("use {}", using.dotted()))];
        if let Some(names) = &using.names {
            let close = using.alias.as_ref().map_or(end, |alias| alias.span.start);
            parts.push(Doc::text("."));
            parts.push(self.list(
                BRACES,
                names,
                close,
                |name| name.span,
                |_, name| Doc::text(name.text.clone()),
            ));
        }
        if let Some(alias) = &using.alias {
            parts.push(Doc::text(format!(" as {}", alias.text)));
        }
        Doc::Concat(parts)
    }

    fn function(&mut self, function: &FnDecl) -> Doc {
        let body_start = match &function.body {
            FnBody::Block(block) => block.span.start,
            FnBody::Expr(expr) => expr.span.start,
        };
        let close = function
            .ret
            .as_ref()
            .map_or(body_start, |ret| ret.span.start);
        let params = self.list(
            PARENS,
            &function.params,
            close,
            |param| param.span,
            |this, param| {
                let ty = this.type_expr(&param.ty);
                let mut parts = vec![Doc::text(format!("{}: ", param.name.text)), ty];
                if let Some(default) = &param.default {
                    parts.push(Doc::text(" = "));
                    parts.push(this.expr(default));
                }
                Doc::Concat(parts)
            },
        );
        let mut parts = vec![Doc::text(format!("fn {}", function.name.text)), params];
        if let Some(ret) = &function.ret {
            parts.push(Doc::text(" -> "));
            parts.push(self.type_expr(ret));
        }
        match &function.body {
            FnBody::Block(block) => {
                parts.push(Doc::text(" "));
                parts.push(self.block(block));
            }
            FnBody::Expr(expr) => {
                parts.push(Doc::text(" = "));
                parts.push(self.expr(expr));
            }
        }
        Doc::Concat(parts)
    }

    /// `{ statements }`, or `{}` when it holds nothing.
    pub(super) fn block(&mut self, block: &Block) -> Doc {
        let close = block.span.end.saturating_sub(1);
        let mut body = self.vertical(&block.stmts, false, |stmt| stmt.span, Self::stmt);
        body.extend(self.closing(close));
        self.last_end = self.last_end.max(block.span.end);
        if body.is_empty() {
            return Doc::text("{}");
        }
        Doc::Concat(vec![
            Doc::text("{"),
            Doc::Nest(body),
            Doc::HardLine,
            Doc::text("}"),
        ])
    }

    pub(super) fn stmt(&mut self, stmt: &Stmt) -> Doc {
        match &stmt.kind {
            StmtKind::Let(binding) | StmtKind::Var(binding) => {
                let word = if matches!(stmt.kind, StmtKind::Let(_)) {
                    "let"
                } else {
                    "var"
                };
                let mut parts = vec![Doc::text(format!("{word} {}", binding.name.text))];
                if let Some(ty) = &binding.ty {
                    parts.push(Doc::text(": "));
                    parts.push(self.type_expr(ty));
                }
                parts.push(Doc::text(" = "));
                parts.push(self.expr(&binding.value));
                Doc::Concat(parts)
            }
            StmtKind::Assign { target, op, value } => {
                let op = match op {
                    AssignOp::Set => "=",
                    AssignOp::Add => "+=",
                    AssignOp::Sub => "-=",
                };
                let value = self.expr(value);
                Doc::Concat(vec![Doc::text(format!("{} {op} ", target.text)), value])
            }
            StmtKind::Charge(charge) => self.charge(charge, stmt.span.end),
            StmtKind::Note(text) => Doc::Concat(vec![Doc::text("note "), self.text_doc(text)]),
            StmtKind::Review(text) => Doc::Concat(vec![Doc::text("review "), self.text_doc(text)]),
            StmtKind::Fail(text) => Doc::Concat(vec![Doc::text("fail "), self.text_doc(text)]),
            StmtKind::If(statement) => {
                let mut parts = Vec::new();
                for (index, (condition, block)) in statement.branches.iter().enumerate() {
                    parts.push(Doc::text(if index == 0 { "if " } else { " else if " }));
                    parts.push(self.condition(condition));
                    parts.push(Doc::text(" "));
                    parts.push(self.block(block));
                }
                if let Some(block) = &statement.otherwise {
                    parts.push(Doc::text(" else "));
                    parts.push(self.block(block));
                }
                Doc::Concat(parts)
            }
            StmtKind::Match(statement) => {
                let scrutinee = self.expr(&statement.scrutinee);
                let mut body = self.vertical(&statement.arms, false, |arm| arm.span, Self::arm);
                body.extend(self.closing(stmt.span.end));
                Doc::Concat(vec![
                    Doc::text("match "),
                    scrutinee,
                    Doc::text(" {"),
                    Doc::Nest(body),
                    Doc::HardLine,
                    Doc::text("}"),
                ])
            }
            StmtKind::For(statement) => {
                let mut parts = vec![
                    Doc::text(format!("for {} in ", statement.binding.text)),
                    self.expr(&statement.iterable),
                ];
                if let Some(filter) = &statement.filter {
                    parts.push(Doc::text(" where "));
                    parts.push(self.expr(filter));
                }
                parts.push(Doc::text(" "));
                parts.push(self.block(&statement.body));
                Doc::Concat(parts)
            }
            StmtKind::Choose(choose) => self.choose(choose, stmt.span.end),
            StmtKind::Vary { .. }
            | StmtKind::Assume(_)
            | StmtKind::Given { .. }
            | StmtKind::Expect(_) => self.test_stmt(&stmt.kind),
            StmtKind::Require { condition, message } => Doc::Concat(vec![
                Doc::text("require "),
                self.condition(condition),
                Doc::text(" else "),
                self.text_doc(message),
            ]),
            StmtKind::Return(value) => Doc::Concat(vec![Doc::text("return "), self.expr(value)]),
            StmtKind::Break => Doc::text("break"),
            StmtKind::Continue => Doc::text("continue"),
            StmtKind::Total(clause) => self.total(clause),
            StmtKind::Expr(value) => self.expr(value),
            StmtKind::Item(item) => self.item(item),
        }
    }

    /// `best "Label" { option … }`, as a statement or a value.
    pub(super) fn choose(&mut self, choose: &Choose, end: u32) -> Doc {
        let mut body = self.vertical(&choose.options, false, |option| option.span, Self::choice);
        body.extend(self.closing(end));
        Doc::Concat(vec![
            Doc::text(format!(
                "{} {} {{",
                choose.kind.keyword(),
                self.slice(choose.label.span)
            )),
            Doc::Nest(body),
            Doc::HardLine,
            Doc::text("}"),
        ])
    }

    /// The statements only tests and properties hold.
    fn test_stmt(&mut self, kind: &StmtKind) -> Doc {
        match kind {
            StmtKind::Given { target, value } => {
                let path: Vec<&str> = target.iter().map(|name| name.text.as_str()).collect();
                let value = self.expr(value);
                Doc::Concat(vec![
                    Doc::text(format!("given {} = ", path.join("."))),
                    value,
                ])
            }
            StmtKind::Expect(Expect::Holds(value)) => {
                Doc::Concat(vec![Doc::text("expect "), self.expr(value)])
            }
            StmtKind::Expect(Expect::Monotone {
                value,
                direction,
                target,
                ..
            }) => {
                let path: Vec<&str> = target.iter().map(|name| name.text.as_str()).collect();
                Doc::Concat(vec![
                    Doc::text("expect "),
                    self.expr(value),
                    Doc::text(format!(" {} in {}", direction.word(), path.join("."))),
                ])
            }
            StmtKind::Expect(Expect::Fails(message)) => match message {
                Some(message) => {
                    Doc::Concat(vec![Doc::text("expect fails "), self.text_doc(message)])
                }
                None => Doc::text("expect fails"),
            },
            StmtKind::Vary { target, values } => {
                let path: Vec<&str> = target.iter().map(|name| name.text.as_str()).collect();
                let mut parts = vec![Doc::text(format!("vary {}", path.join(".")))];
                if let Some(values) = values {
                    parts.push(Doc::text(" in "));
                    parts.push(self.expr(values));
                }
                Doc::Concat(parts)
            }
            StmtKind::Assume(value) => Doc::Concat(vec![Doc::text("assume "), self.expr(value)]),
            _ => Doc::Concat(Vec::new()),
        }
    }

    fn choice(&mut self, option: &ChoiceOption) -> Doc {
        let body = self.block(&option.body);
        Doc::Concat(vec![
            Doc::text(format!("option {} ", self.slice(option.label.span))),
            body,
        ])
    }

    fn charge(&mut self, charge: &Charge, end: u32) -> Doc {
        let mut parts = vec![Doc::text("charge "), self.text_doc(&charge.label)];
        if let Some(code) = &charge.code {
            parts.push(Doc::text(format!(" as {}", code.text)));
        }
        if !charge.options.is_empty() {
            let close = charge
                .amount
                .as_ref()
                .map_or(end, |amount| amount.span.start);
            parts.push(Doc::text(" "));
            parts.push(self.arguments(&charge.options, close));
        }
        if let Some(amount) = &charge.amount {
            parts.push(Doc::text(" = "));
            parts.push(self.expr(amount));
        }
        Doc::Concat(parts)
    }

    fn total(&mut self, clause: &TotalClause) -> Doc {
        match clause {
            TotalClause::Minimum(bound) | TotalClause::Maximum(bound) => {
                let word = if matches!(clause, TotalClause::Minimum(_)) {
                    "minimum "
                } else {
                    "maximum "
                };
                let mut parts = vec![Doc::text(word), self.expr(&bound.amount)];
                if let Some(on) = &bound.on {
                    parts.push(Doc::text(format!(" on {}", on.text)));
                }
                Doc::Concat(parts)
            }
            TotalClause::Round { mode, to } => {
                let mode = match mode {
                    RoundMode::HalfUp => "round to ",
                    RoundMode::HalfEven => "round half even to ",
                    RoundMode::Up => "round up to ",
                    RoundMode::Down => "round down to ",
                };
                let to = match to {
                    RoundTo::Cents => Doc::text("cents"),
                    RoundTo::Amount(amount) => self.expr(amount),
                };
                Doc::Concat(vec![Doc::text(mode), to])
            }
        }
    }
}
