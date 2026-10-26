//! The layout of expressions, patterns and types.

use super::Formatter;
use super::doc::Doc;
use trenova_module_kit::span::Span;
use crate::tariff::syntax::ast::Arg;
use crate::tariff::syntax::ast::ArmBody;
use crate::tariff::syntax::ast::BinaryOp;
use crate::tariff::syntax::ast::Condition;
use crate::tariff::syntax::ast::Expr;
use crate::tariff::syntax::ast::ExprKind;
use crate::tariff::syntax::ast::Pattern;
use crate::tariff::syntax::ast::PatternKind;
use crate::tariff::syntax::ast::Text;
use crate::tariff::syntax::ast::TypeExpr;
use crate::tariff::syntax::ast::TypeExprKind;
use crate::tariff::syntax::ast::UnaryOp;
use crate::tariff::syntax::ast::UnitExpr;

/// The unit of `.in(lb/ft3)`, written as a unit is (`lb/ft3`, `$/cwt`), not as a division.
fn written_unit(callee: &Expr, args: &[Arg]) -> Option<String> {
    fn names(expr: &Expr) -> Option<String> {
        match &expr.kind {
            ExprKind::Name(name) if expr.parens == 0 => Some(name.text.clone()),
            ExprKind::Binary {
                op: BinaryOp::Div,
                left,
                right,
                ..
            } if expr.parens == 0 => Some(format!("{}/{}", names(left)?, names(right)?)),
            _ => None,
        }
    }
    let ExprKind::Field { name, .. } = &callee.kind else {
        return None;
    };
    match args {
        [arg] if name.text == "in" && arg.name.is_none() => names(&arg.value),
        _ => None,
    }
}

/// The operators whose chains break one operand per line, grouped by precedence.
fn chain_class(op: BinaryOp) -> Option<u8> {
    match op {
        BinaryOp::Coalesce => Some(0),
        BinaryOp::Or => Some(1),
        BinaryOp::And => Some(2),
        BinaryOp::Add | BinaryOp::Sub => Some(3),
        _ => None,
    }
}

/// How a bracketed list is laid out.
#[derive(Clone, Copy)]
pub(super) struct Brackets {
    pub(super) open: &'static str,
    pub(super) close: &'static str,
    /// A space inside the brackets when flat: `{ a: 1 }`.
    pub(super) spaced: bool,
    /// A trailing comma when broken.
    pub(super) comma: bool,
}

pub(super) const PARENS: Brackets = Brackets {
    open: "(",
    close: ")",
    spaced: false,
    comma: true,
};

const SQUARE: Brackets = Brackets {
    open: "[",
    close: "]",
    spaced: false,
    comma: true,
};

pub(super) const BRACES: Brackets = Brackets {
    open: "{",
    close: "}",
    spaced: true,
    comma: true,
};

impl Formatter<'_> {
    /// A bracketed, comma-separated list: flat when it fits, else one item per line.
    pub(super) fn list<T>(
        &mut self,
        brackets: Brackets,
        items: &[T],
        close_at: u32,
        span: impl Fn(&T) -> Span,
        build: impl Fn(&mut Self, &T) -> Doc,
    ) -> Doc {
        let edge = || {
            if brackets.spaced {
                Doc::Line
            } else {
                Doc::SoftLine
            }
        };
        let mut inner = Vec::new();
        let mut previous_end = None;
        for item in items {
            let item_span = span(item);
            if let Some(end) = previous_end {
                inner.push(Doc::text(","));
                if let Some(comment) = self.trailing(end) {
                    inner.push(comment);
                }
                inner.push(Doc::Line);
            } else {
                inner.push(edge());
            }
            inner.extend(self.leading(item_span.start, false));
            inner.push(build(self, item));
            self.last_end = self.last_end.max(item_span.end);
            previous_end = Some(item_span.end);
        }
        if let Some(end) = previous_end {
            if brackets.comma {
                inner.push(Doc::IfBreak(","));
            }
            if let Some(comment) = self.trailing(end) {
                inner.push(comment);
            }
        }
        let closing = self.closing(close_at);
        let empty = items.is_empty();
        if empty && closing.is_empty() {
            return Doc::text(format!("{}{}", brackets.open, brackets.close));
        }
        inner.extend(closing);
        let mut parts = vec![Doc::text(brackets.open), Doc::Nest(inner)];
        parts.push(if empty { Doc::HardLine } else { edge() });
        parts.push(Doc::text(brackets.close));
        Doc::Group(parts)
    }

    /// The text of the literal token that starts at or after `start` (past any parentheses).
    fn literal_text(&self, start: u32) -> &str {
        let rest = self.source.get(start as usize..).unwrap_or_default();
        let skipped = rest.len() - rest.trim_start_matches(['(', ' ', '\t', '\n', '\r']).len();
        let at = start as usize + skipped;
        let text = self.source.get(at..).unwrap_or_default();
        let length = literal_length(text);
        text.get(..length).unwrap_or_default()
    }

    pub(super) fn text_doc(&self, text: &Text) -> Doc {
        Doc::text(self.slice(text.span))
    }

    pub(super) fn expr(&mut self, expr: &Expr) -> Doc {
        let inner = self.expr_kind(expr);
        if expr.parens == 0 {
            return inner;
        }
        let mut parts = Vec::new();
        for _ in 0..expr.parens {
            parts.push(Doc::text("("));
        }
        parts.push(inner);
        for _ in 0..expr.parens {
            parts.push(Doc::text(")"));
        }
        Doc::Concat(parts)
    }

    fn unit(unit: &UnitExpr) -> String {
        unit.text()
    }

    fn expr_kind(&mut self, expr: &Expr) -> Doc {
        match &expr.kind {
            ExprKind::Number { unit, .. } => {
                let number = self.literal_text(expr.span.start).to_owned();
                match unit {
                    Some(unit) => Doc::text(format!("{number} {}", Self::unit(unit))),
                    None => Doc::text(number),
                }
            }
            ExprKind::Money { per, .. } => {
                let amount = self.literal_text(expr.span.start).to_owned();
                match per {
                    Some(per) => Doc::text(format!("{amount}/{}", Self::unit(per))),
                    None => Doc::text(amount),
                }
            }
            ExprKind::Percent(_)
            | ExprKind::Date(_)
            | ExprKind::DateTime(_)
            | ExprKind::Time(_) => Doc::text(self.literal_text(expr.span.start).to_owned()),
            ExprKind::Text(text) => self.text_doc(text),
            ExprKind::Bool(value) => Doc::text(if *value { "true" } else { "false" }),
            ExprKind::None => Doc::text("none"),
            ExprKind::Name(name) => Doc::text(name.text.clone()),
            ExprKind::Path { head, member } => match member {
                Some(member) => Doc::text(format!("{}.{}", head.text, member.text)),
                None => Doc::text(head.text.clone()),
            },
            ExprKind::Field {
                target,
                name,
                optional,
            } => {
                let target = self.expr(target);
                let dot = if *optional { "?." } else { "." };
                Doc::Concat(vec![target, Doc::text(format!("{dot}{}", name.text))])
            }
            ExprKind::Call { callee, args } => {
                let unit = written_unit(callee, args);
                let callee = self.expr(callee);
                let args = match unit {
                    Some(unit) => Doc::text(format!("({unit})")),
                    None => self.arguments(args, expr.span.end),
                };
                Doc::Concat(vec![callee, args])
            }
            ExprKind::Index { target, keys } => {
                let target = self.expr(target);
                let keys = self.list(SQUARE, keys, expr.span.end, |k| k.span, Self::expr);
                Doc::Concat(vec![target, keys])
            }
            ExprKind::Unary { op, operand, .. } => {
                let operand = self.expr(operand);
                let op = match op {
                    UnaryOp::Neg => "-",
                    UnaryOp::Not => "not ",
                };
                Doc::Concat(vec![Doc::text(op), operand])
            }
            ExprKind::Binary { op, .. } if chain_class(*op).is_some() => self.chain(expr),
            ExprKind::Binary {
                op, left, right, ..
            } => {
                let left = self.expr(left);
                let right = self.expr(right);
                Doc::Concat(vec![left, Doc::text(format!(" {} ", op.symbol())), right])
            }
            ExprKind::Range {
                start,
                end,
                inclusive,
            } => {
                let mut parts = Vec::new();
                if let Some(start) = start {
                    parts.push(self.expr(start));
                }
                parts.push(Doc::text(if *inclusive { "..=" } else { ".." }));
                if let Some(end) = end {
                    parts.push(self.expr(end));
                }
                Doc::Concat(parts)
            }
            ExprKind::Fail(text) => Doc::Concat(vec![Doc::text("fail "), self.text_doc(text)]),
            ExprKind::List(_)
            | ExprKind::Record(_)
            | ExprKind::If { .. }
            | ExprKind::Match { .. }
            | ExprKind::Lambda { .. } => self.compound(expr),
            ExprKind::Choose(choose) => self.choose(choose, expr.span.end),
            ExprKind::Missing => Doc::text(""),
        }
    }

    /// Lists, records, `if` and `match` values, and lambdas.
    fn compound(&mut self, expr: &Expr) -> Doc {
        match &expr.kind {
            ExprKind::List(items) => {
                self.list(SQUARE, items, expr.span.end, |i| i.span, Self::expr)
            }
            ExprKind::Record(fields) => self.list(
                BRACES,
                fields,
                expr.span.end,
                |field| field.name.span.cover(field.value.span),
                |this, field| {
                    let value = this.expr(&field.value);
                    Doc::Concat(vec![Doc::text(format!("{}: ", field.name.text)), value])
                },
            ),
            ExprKind::If {
                branches,
                otherwise,
            } => self.if_expr(branches, otherwise),
            ExprKind::Match { scrutinee, arms } => {
                let scrutinee = self.expr(scrutinee);
                let close = expr.span.end;
                let mut parts = vec![Doc::text("match "), scrutinee, Doc::text(" {")];
                let mut body = self.vertical(arms, false, |arm| arm.span, Self::arm);
                body.extend(self.closing(close));
                parts.push(Doc::Nest(body));
                parts.push(Doc::HardLine);
                parts.push(Doc::text("}"));
                Doc::Concat(parts)
            }
            ExprKind::Lambda { params, body } => {
                let names: Vec<&str> = params.iter().map(|p| p.text.as_str()).collect();
                let head = if names.len() == 1 {
                    names.join("")
                } else {
                    format!("({})", names.join(", "))
                };
                let body = self.expr(body);
                Doc::Concat(vec![Doc::text(format!("{head} => ")), body])
            }
            _ => Doc::text(""),
        }
    }

    pub(super) fn arguments(&mut self, args: &[Arg], close_at: u32) -> Doc {
        self.list(
            PARENS,
            args,
            close_at,
            |arg| arg.span,
            |this, arg| {
                let value = this.expr(&arg.value);
                match &arg.name {
                    Some(name) => Doc::Concat(vec![Doc::text(format!("{}: ", name.text)), value]),
                    None => value,
                }
            },
        )
    }

    /// A chain of `+`/`-`, `and`, `or` or `??`: one group that breaks before each operator.
    fn chain(&mut self, expr: &Expr) -> Doc {
        let mut operands: Vec<(Option<(BinaryOp, Span)>, &Expr)> = Vec::new();
        collect_chain(expr, &mut operands);
        let mut rest = Vec::new();
        let mut first = None;
        for (op, operand) in operands {
            match op {
                None => first = Some(self.expr(operand)),
                Some((op, op_span)) => {
                    rest.push(Doc::Line);
                    rest.extend(self.leading(op_span.start, false));
                    rest.push(Doc::text(format!("{} ", op.symbol())));
                    rest.push(self.expr(operand));
                }
            }
        }
        Doc::Group(vec![
            first.unwrap_or_else(|| Doc::text("")),
            Doc::Nest(rest),
        ])
    }

    pub(super) fn condition(&mut self, condition: &Condition) -> Doc {
        match condition {
            Condition::Expr(expr) => self.expr(expr),
            Condition::Let { name, value } => {
                let value = self.expr(value);
                Doc::Concat(vec![Doc::text(format!("let {} = ", name.text)), value])
            }
        }
    }

    fn if_expr(&mut self, branches: &[(Condition, Expr)], otherwise: &Expr) -> Doc {
        let mut parts = Vec::new();
        for (index, (condition, value)) in branches.iter().enumerate() {
            parts.push(Doc::text(if index == 0 { "if " } else { " else if " }));
            parts.push(self.condition(condition));
            parts.push(Doc::text(" {"));
            let value = self.expr(value);
            parts.push(Doc::Nest(vec![Doc::Line, value]));
            parts.push(Doc::Line);
            parts.push(Doc::text("}"));
        }
        parts.push(Doc::text(" else {"));
        let otherwise = self.expr(otherwise);
        parts.push(Doc::Nest(vec![Doc::Line, otherwise]));
        parts.push(Doc::Line);
        parts.push(Doc::text("}"));
        Doc::Group(parts)
    }

    pub(super) fn arm(&mut self, arm: &crate::tariff::syntax::ast::Arm) -> Doc {
        let mut parts = vec![self.pattern(&arm.pattern)];
        if let Some(guard) = &arm.guard {
            parts.push(Doc::text(" if "));
            parts.push(self.expr(guard));
        }
        parts.push(Doc::text(" => "));
        match &arm.body {
            ArmBody::Expr(body) => parts.push(self.expr(body)),
            ArmBody::Block(block) => parts.push(self.block(block)),
        }
        Doc::Concat(parts)
    }

    pub(super) fn pattern(&mut self, pattern: &Pattern) -> Doc {
        match &pattern.kind {
            PatternKind::Wildcard | PatternKind::Missing => Doc::text("_"),
            PatternKind::None => Doc::text("none"),
            PatternKind::Literal(expr) => self.expr(expr),
            PatternKind::Range {
                start,
                end,
                inclusive,
            } => {
                let mut parts = Vec::new();
                if let Some(start) = start {
                    parts.push(self.expr(start));
                }
                parts.push(Doc::text(if *inclusive { "..=" } else { ".." }));
                if let Some(end) = end {
                    parts.push(self.expr(end));
                }
                Doc::Concat(parts)
            }
            PatternKind::Variant {
                enumeration,
                variant,
            } => match enumeration {
                Some(enumeration) => Doc::text(format!("{}.{}", enumeration.text, variant.text)),
                None => Doc::text(variant.text.clone()),
            },
            PatternKind::Binding(name) => Doc::text(name.text.clone()),
            PatternKind::Record(fields) => self.list(
                BRACES,
                fields,
                pattern.span.end,
                |(name, field)| name.span.cover(field.span),
                |this, (name, field)| {
                    let field = this.pattern(field);
                    Doc::Concat(vec![Doc::text(format!("{}: ", name.text)), field])
                },
            ),
            PatternKind::Or(alternatives) => {
                let mut parts = Vec::new();
                for (index, alternative) in alternatives.iter().enumerate() {
                    if index > 0 {
                        parts.push(Doc::text(" | "));
                    }
                    parts.push(self.pattern(alternative));
                }
                Doc::Concat(parts)
            }
        }
    }

    pub(super) fn type_expr(&mut self, ty: &TypeExpr) -> Doc {
        match &ty.kind {
            TypeExprKind::Named { name, args } if args.is_empty() => Doc::text(name.text.clone()),
            TypeExprKind::Named { name, args } => {
                let mut parts = vec![Doc::text(format!("{}<", name.text))];
                for (index, arg) in args.iter().enumerate() {
                    if index > 0 {
                        parts.push(Doc::text(", "));
                    }
                    parts.push(self.type_expr(arg));
                }
                parts.push(Doc::text(">"));
                Doc::Concat(parts)
            }
            TypeExprKind::Quotient {
                numerator,
                denominator,
            } => Doc::text(format!("{}/{}", numerator.text, denominator.text)),
            TypeExprKind::Record(fields) => self.list(
                BRACES,
                fields,
                ty.span.end,
                |field| field.name.span.cover(field.ty.span),
                |this, field| {
                    let ty = this.type_expr(&field.ty);
                    Doc::Concat(vec![Doc::text(format!("{}: ", field.name.text)), ty])
                },
            ),
            TypeExprKind::Optional(inner) => {
                Doc::Concat(vec![self.type_expr(inner), Doc::text("?")])
            }
            TypeExprKind::Missing => Doc::text(""),
        }
    }
}

/// The operands of a chain whose operators share `expr`'s precedence, left to right, each but
/// the first with the operator before it.
fn collect_chain<'e>(expr: &'e Expr, out: &mut Vec<(Option<(BinaryOp, Span)>, &'e Expr)>) {
    if let ExprKind::Binary {
        op,
        op_span,
        left,
        right,
    } = &expr.kind
    {
        let class = chain_class(*op);
        let left_continues = left.parens == 0
            && matches!(&left.kind, ExprKind::Binary { op: inner, .. } if chain_class(*inner) == class);
        if left_continues {
            collect_chain(left, out);
        } else {
            out.push((None, left));
        }
        out.push((Some((*op, *op_span)), right));
    }
}

/// The length of the literal token at the start of `text`: a number, `$` and an amount, a
/// percent or a date.
fn literal_length(text: &str) -> usize {
    let bytes = text.as_bytes();
    let mut index = 0;
    match bytes.first() {
        Some(b'@') => {
            index = 1;
            while bytes.get(index).is_some_and(|b| {
                b.is_ascii_digit() || matches!(b, b'-' | b':' | b'+' | b'T' | b'Z')
            }) {
                index += 1;
            }
            return index;
        }
        Some(b'$') => index = 1,
        _ => {}
    }
    while bytes
        .get(index)
        .is_some_and(|b| b.is_ascii_digit() || *b == b'_')
    {
        index += 1;
    }
    if bytes.get(index) == Some(&b'.') && bytes.get(index + 1).is_some_and(u8::is_ascii_digit) {
        index += 1;
        while bytes
            .get(index)
            .is_some_and(|b| b.is_ascii_digit() || *b == b'_')
        {
            index += 1;
        }
    }
    if bytes.get(index) == Some(&b'%') {
        index += 1;
    }
    index
}
