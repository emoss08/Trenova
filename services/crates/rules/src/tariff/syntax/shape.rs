//! A syntax tree written as text without positions or ids: one line per item and statement,
//! blocks indented, expressions as compact S-expressions.
//!
//! Two trees with the same shape are the same program as written, whatever the spacing, so the
//! formatter checks with it that formatting kept the tree, and the syntax corpus records the
//! trees it expects with it.

use super::ast::Arg;
use super::ast::Arm;
use super::ast::ArmBody;
use super::ast::Block;
use super::ast::Condition;
use super::ast::Expect;
use super::ast::Expr;
use super::ast::ExprKind;
use super::ast::File;
use super::ast::FnBody;
use super::ast::Item;
use super::ast::ItemKind;
use super::ast::Pattern;
use super::ast::PatternKind;
use super::ast::RoundMode;
use super::ast::RoundTo;
use super::ast::Stmt;
use super::ast::StmtKind;
use super::ast::Text;
use super::ast::TextPart;
use super::ast::TotalClause;
use super::ast::TypeExpr;
use super::ast::TypeExprKind;
use super::ast::UnaryOp;
use super::ast::UnitExpr;
use std::fmt::Write;

/// The shape of a whole file: its header, doc and items.
#[must_use]
pub fn file(file: &File) -> String {
    let mut out = Shape::default();
    if let Some(header) = &file.header {
        out.line(
            0,
            &format!("{} {:?}", header.kind.keyword(), header.name.text),
        );
    }
    if let Some(doc) = &file.docs {
        out.line(0, &format!("file doc {:?}", doc.text));
    }
    for item in &file.items {
        out.item(0, item);
    }
    out.text
}

/// The shape of one expression.
#[must_use]
pub fn expr(expr: &Expr) -> String {
    let mut out = String::new();
    write_expr(&mut out, expr);
    out
}

#[derive(Default)]
struct Shape {
    text: String,
}

impl Shape {
    fn line(&mut self, depth: usize, content: &str) {
        for _ in 0..depth {
            self.text.push_str("  ");
        }
        self.text.push_str(content);
        self.text.push('\n');
    }

    fn item(&mut self, depth: usize, item: &Item) {
        if let Some(doc) = &item.docs {
            self.line(depth, &format!("doc {:?}", doc.text));
        }
        let private = if item.private.is_some() {
            "private "
        } else {
            ""
        };
        match &item.kind {
            ItemKind::Use(using) => {
                let mut line = format!("{private}use {}", using.dotted());
                if let Some(names) = &using.names {
                    let names: Vec<&str> = names.iter().map(|n| n.text.as_str()).collect();
                    let _ = write!(line, ".{{{}}}", names.join(", "));
                }
                if let Some(alias) = &using.alias {
                    let _ = write!(line, " as {}", alias.text);
                }
                self.line(depth, &line);
            }
            ItemKind::Input(input) => {
                let mut line = format!("{private}input {}: {}", input.name.text, ty(&input.ty));
                if let Some(default) = &input.default {
                    line.push_str(" = ");
                    write_expr(&mut line, default);
                }
                self.line(depth, &line);
            }
            ItemKind::Const(constant) => {
                let mut line = format!("{private}const {}", constant.name.text);
                if let Some(t) = &constant.ty {
                    let _ = write!(line, ": {}", ty(t));
                }
                line.push_str(" = ");
                write_expr(&mut line, &constant.value);
                self.line(depth, &line);
            }
            ItemKind::Type(decl) => {
                self.line(depth, &format!("{private}type {} =", decl.name.text));
                for field in &decl.fields {
                    if let Some(doc) = &field.docs {
                        self.line(depth + 1, &format!("doc {:?}", doc.text));
                    }
                    self.line(
                        depth + 1,
                        &format!("{}: {}", field.name.text, ty(&field.ty)),
                    );
                }
            }
            ItemKind::Enum(decl) => {
                self.line(depth, &format!("{private}enum {}", decl.name.text));
                for variant in &decl.variants {
                    if let Some(doc) = &variant.docs {
                        self.line(depth + 1, &format!("doc {:?}", doc.text));
                    }
                    self.line(depth + 1, &variant.name.text);
                }
            }
            ItemKind::Fn(function) => {
                let params: Vec<String> = function
                    .params
                    .iter()
                    .map(|param| {
                        let mut text = format!("{}: {}", param.name.text, ty(&param.ty));
                        if let Some(default) = &param.default {
                            text.push_str(" = ");
                            write_expr(&mut text, default);
                        }
                        text
                    })
                    .collect();
                let mut line = format!("{private}fn {}({})", function.name.text, params.join(", "));
                if let Some(ret) = &function.ret {
                    let _ = write!(line, " -> {}", ty(ret));
                }
                match &function.body {
                    FnBody::Block(block) => {
                        self.line(depth, &line);
                        self.block(depth + 1, block);
                    }
                    FnBody::Expr(body) => {
                        line.push_str(" = ");
                        write_expr(&mut line, body);
                        self.line(depth, &line);
                    }
                }
            }
            ItemKind::Test(test) => {
                self.line(depth, &format!("{private}test {:?}", test.name.text));
                self.block(depth + 1, &test.body);
            }
            ItemKind::Stmt(stmt) => {
                if !private.is_empty() {
                    self.line(depth, "private");
                }
                self.stmt(depth, stmt);
            }
        }
    }

    fn block(&mut self, depth: usize, block: &Block) {
        for stmt in &block.stmts {
            self.stmt(depth, stmt);
        }
    }

    fn stmt(&mut self, depth: usize, stmt: &Stmt) {
        match &stmt.kind {
            StmtKind::If(statement) => {
                for (index, (condition, block)) in statement.branches.iter().enumerate() {
                    let mut head = String::from(if index == 0 { "if " } else { "else if " });
                    write_condition(&mut head, condition);
                    self.line(depth, &head);
                    self.block(depth + 1, block);
                }
                if let Some(block) = &statement.otherwise {
                    self.line(depth, "else");
                    self.block(depth + 1, block);
                }
            }
            StmtKind::Match(statement) => {
                let mut head = String::from("match ");
                write_expr(&mut head, &statement.scrutinee);
                self.line(depth, &head);
                for arm in &statement.arms {
                    self.arm(depth + 1, arm);
                }
            }
            StmtKind::For(statement) => {
                let mut head = format!("for {} in ", statement.binding.text);
                write_expr(&mut head, &statement.iterable);
                if let Some(filter) = &statement.filter {
                    head.push_str(" where ");
                    write_expr(&mut head, filter);
                }
                self.line(depth, &head);
                self.block(depth + 1, &statement.body);
            }
            StmtKind::Item(item) => {
                self.line(depth, "item");
                self.item(depth + 1, item);
            }
            kind => {
                let mut line = String::new();
                write_simple_stmt(&mut line, kind);
                self.line(depth, &line);
            }
        }
    }

    fn arm(&mut self, depth: usize, arm: &Arm) {
        let mut head = String::from("arm ");
        write_pattern(&mut head, &arm.pattern);
        if let Some(guard) = &arm.guard {
            head.push_str(" if ");
            write_expr(&mut head, guard);
        }
        match &arm.body {
            ArmBody::Block(block) => {
                head.push_str(" =>");
                self.line(depth, &head);
                self.block(depth + 1, block);
            }
            ArmBody::Expr(body) => {
                head.push_str(" => ");
                write_expr(&mut head, body);
                self.line(depth, &head);
            }
        }
    }
}

/// A statement that holds no block, on one line.
fn write_simple_stmt(line: &mut String, kind: &StmtKind) {
    match kind {
        StmtKind::Let(binding) | StmtKind::Var(binding) => {
            let word = if matches!(kind, StmtKind::Let(_)) {
                "let"
            } else {
                "var"
            };
            let _ = write!(line, "{word} {}", binding.name.text);
            if let Some(t) = &binding.ty {
                let _ = write!(line, ": {}", ty(t));
            }
            line.push_str(" = ");
            write_expr(line, &binding.value);
        }
        StmtKind::Assign { target, op, value } => {
            let op = match op {
                super::ast::AssignOp::Set => "=",
                super::ast::AssignOp::Add => "+=",
                super::ast::AssignOp::Sub => "-=",
            };
            let _ = write!(line, "{} {op} ", target.text);
            write_expr(line, value);
        }
        StmtKind::Charge(charge) => {
            line.push_str("charge ");
            write_text(line, &charge.label);
            if let Some(code) = &charge.code {
                let _ = write!(line, " as {}", code.text);
            }
            if !charge.options.is_empty() {
                line.push_str(" (");
                write_args(line, &charge.options);
                line.push(')');
            }
            if let Some(amount) = &charge.amount {
                line.push_str(" = ");
                write_expr(line, amount);
            }
        }
        StmtKind::Note(text) => write_word_text(line, "note", text),
        StmtKind::Fail(text) => write_word_text(line, "fail", text),
        StmtKind::Require { condition, message } => {
            line.push_str("require ");
            write_condition(line, condition);
            write_word_text(line, " else", message);
        }
        StmtKind::Return(value) => write_word_expr(line, "return", value),
        StmtKind::Break => line.push_str("break"),
        StmtKind::Continue => line.push_str("continue"),
        StmtKind::Total(clause) => write_total(line, clause),
        StmtKind::Given { target, value } => {
            let path: Vec<&str> = target.iter().map(|n| n.text.as_str()).collect();
            let _ = write!(line, "given {} = ", path.join("."));
            write_expr(line, value);
        }
        StmtKind::Expect(Expect::Holds(value)) => write_word_expr(line, "expect", value),
        StmtKind::Expect(Expect::Fails(message)) => {
            line.push_str("expect fails");
            if let Some(message) = message {
                write_word_text(line, "", message);
            }
        }
        StmtKind::Expr(value) => write_word_expr(line, "expr", value),
        StmtKind::If(_) | StmtKind::Match(_) | StmtKind::For(_) | StmtKind::Item(_) => {}
    }
}

fn write_word_text(line: &mut String, word: &str, text: &Text) {
    line.push_str(word);
    line.push(' ');
    write_text(line, text);
}

fn write_word_expr(line: &mut String, word: &str, value: &Expr) {
    line.push_str(word);
    line.push(' ');
    write_expr(line, value);
}

fn write_total(out: &mut String, clause: &TotalClause) {
    match clause {
        TotalClause::Minimum(value) => {
            out.push_str("minimum ");
            write_expr(out, value);
        }
        TotalClause::Maximum(value) => {
            out.push_str("maximum ");
            write_expr(out, value);
        }
        TotalClause::Round { mode, to } => {
            out.push_str(match mode {
                RoundMode::HalfUp => "round half-up to ",
                RoundMode::HalfEven => "round half-even to ",
                RoundMode::Up => "round up to ",
                RoundMode::Down => "round down to ",
            });
            match to {
                RoundTo::Cents => out.push_str("cents"),
                RoundTo::Amount(amount) => write_expr(out, amount),
            }
        }
    }
}

fn ty(t: &TypeExpr) -> String {
    match &t.kind {
        TypeExprKind::Named { name, args } if args.is_empty() => name.text.clone(),
        TypeExprKind::Named { name, args } => {
            let args: Vec<String> = args.iter().map(ty).collect();
            format!("{}<{}>", name.text, args.join(", "))
        }
        TypeExprKind::Quotient {
            numerator,
            denominator,
        } => format!("{}/{}", numerator.text, denominator.text),
        TypeExprKind::Record(fields) => {
            let fields: Vec<String> = fields
                .iter()
                .map(|field| {
                    let doc = field
                        .docs
                        .as_ref()
                        .map(|doc| format!("doc {:?} ", doc.text))
                        .unwrap_or_default();
                    format!("{doc}{}: {}", field.name.text, ty(&field.ty))
                })
                .collect();
            format!("{{{}}}", fields.join(", "))
        }
        TypeExprKind::Optional(inner) => format!("{}?", ty(inner)),
        TypeExprKind::Missing => String::from("?"),
    }
}

fn write_condition(out: &mut String, condition: &Condition) {
    match condition {
        Condition::Expr(value) => write_expr(out, value),
        Condition::Let { name, value } => {
            let _ = write!(out, "let {} = ", name.text);
            write_expr(out, value);
        }
    }
}

fn write_text(out: &mut String, text: &Text) {
    out.push_str("(text");
    for part in &text.parts {
        out.push(' ');
        match part {
            TextPart::Literal(literal, _) => {
                let _ = write!(out, "{literal:?}");
            }
            TextPart::Expr(value) => write_expr(out, value),
        }
    }
    out.push(')');
}

fn write_unit(out: &mut String, unit: &UnitExpr) {
    out.push('[');
    out.push_str(&unit.text());
    out.push(']');
}

fn write_args(out: &mut String, args: &[Arg]) {
    for (index, arg) in args.iter().enumerate() {
        if index > 0 {
            out.push(' ');
        }
        if let Some(name) = &arg.name {
            let _ = write!(out, "{}:", name.text);
        }
        write_expr(out, &arg.value);
    }
}

fn write_list(out: &mut String, items: &[Expr]) {
    for item in items {
        out.push(' ');
        write_expr(out, item);
    }
}

fn write_expr(out: &mut String, value: &Expr) {
    for _ in 0..value.parens {
        out.push_str("(paren ");
    }
    write_kind(out, &value.kind);
    for _ in 0..value.parens {
        out.push(')');
    }
}

fn write_kind(out: &mut String, kind: &ExprKind) {
    match kind {
        ExprKind::Text(text) => write_text(out, text),
        ExprKind::Number { .. }
        | ExprKind::Money { .. }
        | ExprKind::Percent(_)
        | ExprKind::Date(_)
        | ExprKind::DateTime(_) => write_literal(out, kind),
        ExprKind::Bool(value) => {
            let _ = write!(out, "{value}");
        }
        ExprKind::None => out.push_str("none"),
        ExprKind::Name(name) => out.push_str(&name.text),
        ExprKind::Path { head, member } => {
            out.push_str(&head.text);
            if let Some(member) = member {
                let _ = write!(out, "::{}", member.text);
            }
        }
        ExprKind::Field {
            target,
            name,
            optional,
        } => {
            out.push_str(if *optional { "(?. " } else { "(. " });
            write_expr(out, target);
            let _ = write!(out, " {})", name.text);
        }
        ExprKind::Call { callee, args } => {
            out.push_str("(call ");
            write_expr(out, callee);
            if !args.is_empty() {
                out.push(' ');
                write_args(out, args);
            }
            out.push(')');
        }
        ExprKind::Index { target, keys } => {
            out.push_str("(index ");
            write_expr(out, target);
            write_list(out, keys);
            out.push(')');
        }
        ExprKind::Unary { op, operand, .. } => {
            out.push_str(match op {
                UnaryOp::Neg => "(neg ",
                UnaryOp::Not => "(not ",
            });
            write_expr(out, operand);
            out.push(')');
        }
        ExprKind::Binary {
            op, left, right, ..
        } => {
            let _ = write!(out, "({} ", op.symbol());
            write_expr(out, left);
            out.push(' ');
            write_expr(out, right);
            out.push(')');
        }
        ExprKind::Range {
            start,
            end,
            inclusive,
        } => {
            out.push_str(if *inclusive { "(..= " } else { "(.. " });
            write_side(out, start.as_deref());
            out.push(' ');
            write_side(out, end.as_deref());
            out.push(')');
        }
        ExprKind::Fail(text) => {
            out.push_str("(fail ");
            write_text(out, text);
            out.push(')');
        }
        ExprKind::List(_)
        | ExprKind::Record(_)
        | ExprKind::If { .. }
        | ExprKind::Match { .. }
        | ExprKind::Lambda { .. } => write_compound(out, kind),
        ExprKind::Missing => out.push('?'),
    }
}

/// Lists, records, `if`, `match` and lambdas.
fn write_compound(out: &mut String, kind: &ExprKind) {
    match kind {
        ExprKind::List(items) => {
            out.push_str("(list");
            write_list(out, items);
            out.push(')');
        }
        ExprKind::Record(fields) => {
            out.push_str("(record");
            for field in fields {
                let _ = write!(out, " {}:", field.name.text);
                write_expr(out, &field.value);
            }
            out.push(')');
        }
        ExprKind::If {
            branches,
            otherwise,
        } => {
            out.push_str("(if");
            for (condition, value) in branches {
                out.push_str(" [");
                write_condition(out, condition);
                out.push_str(" => ");
                write_expr(out, value);
                out.push(']');
            }
            out.push_str(" else ");
            write_expr(out, otherwise);
            out.push(')');
        }
        ExprKind::Match { scrutinee, arms } => {
            out.push_str("(match ");
            write_expr(out, scrutinee);
            for arm in arms {
                out.push_str(" [");
                write_pattern(out, &arm.pattern);
                if let Some(guard) = &arm.guard {
                    out.push_str(" if ");
                    write_expr(out, guard);
                }
                out.push_str(" => ");
                match &arm.body {
                    ArmBody::Expr(body) => write_expr(out, body),
                    ArmBody::Block(_) => out.push_str("{block}"),
                }
                out.push(']');
            }
            out.push(')');
        }
        ExprKind::Lambda { params, body } => {
            let params: Vec<&str> = params.iter().map(|p| p.text.as_str()).collect();
            let _ = write!(out, "(lambda ({}) ", params.join(" "));
            write_expr(out, body);
            out.push(')');
        }
        _ => {}
    }
}

/// Numbers, money, percents and dates.
fn write_literal(out: &mut String, kind: &ExprKind) {
    match kind {
        ExprKind::Number {
            value,
            integer,
            unit,
        } => {
            let _ = write!(out, "{value}");
            if !integer && !value.to_string().contains('.') {
                out.push('.');
            }
            if let Some(unit) = unit {
                write_unit(out, unit);
            }
        }
        ExprKind::Money { value, per } => {
            let _ = write!(out, "${value}");
            if let Some(per) = per {
                out.push('/');
                write_unit(out, per);
            }
        }
        ExprKind::Percent(value) => {
            let _ = write!(out, "{value}%");
        }
        ExprKind::Date(date) => {
            let _ = write!(out, "@{:04}-{:02}-{:02}", date.year, date.month, date.day);
        }
        ExprKind::DateTime(at) => {
            let _ = write!(
                out,
                "@{:04}-{:02}-{:02}T{:02}:{:02}:{:02}{:+}m",
                at.date.year,
                at.date.month,
                at.date.day,
                at.hour,
                at.minute,
                at.second,
                at.offset_minutes
            );
        }
        _ => {}
    }
}

fn write_side(out: &mut String, side: Option<&Expr>) {
    match side {
        Some(value) => write_expr(out, value),
        None => out.push('_'),
    }
}

fn write_pattern(out: &mut String, pattern: &Pattern) {
    match &pattern.kind {
        PatternKind::Wildcard => out.push('_'),
        PatternKind::None => out.push_str("none"),
        PatternKind::Literal(value) => write_expr(out, value),
        PatternKind::Range {
            start,
            end,
            inclusive,
        } => {
            out.push_str(if *inclusive { "(..= " } else { "(.. " });
            write_side(out, start.as_ref());
            out.push(' ');
            write_side(out, end.as_ref());
            out.push(')');
        }
        PatternKind::Variant {
            enumeration,
            variant,
        } => {
            if let Some(enumeration) = enumeration {
                let _ = write!(out, "{}::", enumeration.text);
            }
            out.push_str(&variant.text);
        }
        PatternKind::Binding(name) => {
            let _ = write!(out, "(bind {})", name.text);
        }
        PatternKind::Record(fields) => {
            out.push_str("(record");
            for (name, field) in fields {
                let _ = write!(out, " {}:", name.text);
                write_pattern(out, field);
            }
            out.push(')');
        }
        PatternKind::Or(alternatives) => {
            out.push_str("(|");
            for alternative in alternatives {
                out.push(' ');
                write_pattern(out, alternative);
            }
            out.push(')');
        }
        PatternKind::Missing => out.push('?'),
    }
}
