//! The checker's recurring findings, worded once: a value of the wrong type, an optional used
//! where a value is needed, an operator whose operands do not combine, and a name nothing
//! declares. Each explains the mistake in pricing words and offers the fixes that are obvious.

use super::checker::{Body, Checker, Global, Want};
use super::describe::{quote, quote_span, unit_text, words, zero};
use super::fixes::{changes, indentation, replace};
use super::operators::{Problem, arithmetic};
use trenova_module_kit::span::Span;
use trenova_module_kit::diagnostic::nearest;
use trenova_module_kit::diagnostic::{Change, Diagnostic, Fix};
use crate::tariff::syntax::ast::{BinaryOp, Expr, ExprKind};
use crate::tariff::types::Type;
use crate::tariff::units::Dim;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

impl Checker<'_> {
    /// Reports `expr`, of type `actual`, where `expected` is needed.
    pub fn mismatch(&mut self, expr: &Expr, actual: &Type, expected: &Type, want: &Want) {
        let what = quote_span(self.source, expr.span);
        let (is, needed) = if words(actual) == words(expected) {
            (
                Arg::from(super::describe::qualified_words(actual)),
                Arg::from(super::describe::qualified_words(expected)),
            )
        } else {
            (Arg::from(words(actual)), Arg::from(words(expected)))
        };
        let message = match want {
            Want::Condition => Message::with(
                message!("{0} is {1}, but a condition must be true or false"),
                [what, is],
            ),
            Want::Amount => Message::with(
                message!("A charge's amount must be money, but {0} is {1}"),
                [what, is],
            ),
            Want::Total => Message::with(
                message!("A total clause takes money, but {0} is {1}"),
                [what, is],
            ),
            Want::Limit { option } => Message::with(
                message!("A charge's {0} must be money, but {1} is {2}"),
                [quote(option), what, is],
            ),
            Want::Argument { param, callee } => Message::with(
                message!("{0} is {1}, but {2} of {3} takes {4}"),
                [what, is, quote(param), quote(callee), needed],
            ),
            Want::Return { name } => Message::with(
                message!("{0} is {1}, but {2} returns {3}"),
                [what, is, quote(name), needed],
            ),
            Want::Annotation { name } => Message::with(
                message!("{0} is {1}, but {2} is declared as {3}"),
                [what, is, quote(name), needed],
            ),
            Want::Field { field } => Message::with(
                message!("{0} is {1}, but the field {2} is {3}"),
                [what, is, quote(field), needed],
            ),
            Want::Item => Message::with(
                message!("{0} is {1}, but the list holds {2}"),
                [what, is, needed],
            ),
            Want::Given { path } => Message::with(
                message!("{0} is {1}, but {2} is {3}"),
                [what, is, quote(path), needed],
            ),
            Want::Value => Message::with(
                message!("{0} is {1}, but {2} is needed here"),
                [what, is, needed],
            ),
        };
        let mut diagnostic = Diagnostic::error("type-mismatch", expr.span, message);
        if let Some(fix) = self.unit_fix(expr, expected) {
            diagnostic = diagnostic.with_fix(fix);
        }
        if let Some(fix) = self.clock_fix(expr, actual, expected) {
            diagnostic = diagnostic.with_fix(fix);
        }
        self.report(diagnostic);
    }

    /// Where a `Time` is needed: a whole hour written as a number (`8`, from before times of
    /// day had a type) as `@08:00`, and a date-time as its `.time`.
    pub fn clock_fix(&self, expr: &Expr, actual: &Type, expected: &Type) -> Option<Fix> {
        if !matches!(expected.unwrapped(), Type::Time) {
            return None;
        }
        if matches!(actual, Type::DateTime) {
            return Some(self.time_of_day_fix(expr, false));
        }
        let ExprKind::Number {
            value,
            integer: true,
            unit: None,
        } = &expr.kind
        else {
            return None;
        };
        let hour = u8::try_from(value.mantissa())
            .ok()
            .filter(|hour| *hour < 24)?;
        if value.scale() != 0 {
            return None;
        }
        let written = format!("@{hour:02}:00");
        Some(replace(
            Message::with(message!("Write it as the time {0}"), [quote(&written)]),
            expr.span,
            written,
        ))
    }

    /// The fix that compares a date-time's time of day: `at` becomes `at.time` (`at?.time` when
    /// it may be empty), in parentheses when it is an operation.
    pub fn time_of_day_fix(&self, expr: &Expr, optional: bool) -> Fix {
        let code = self.text(expr.span);
        let postfix = expr.parens > 0
            || matches!(
                expr.kind,
                ExprKind::Name(_)
                    | ExprKind::Field { .. }
                    | ExprKind::Call { .. }
                    | ExprKind::Index { .. }
                    | ExprKind::DateTime(_)
            );
        let dot = if optional { "?." } else { "." };
        let written = if postfix {
            format!("{code}{dot}time")
        } else {
            format!("({code}){dot}time")
        };
        replace(
            Message::with(message!("Use its time of day: {0}"), [quote(&written)]),
            expr.span,
            written,
        )
    }

    /// For a plain number literal where money or a quantity is needed, the fix that writes its
    /// unit: `500` → `500 lb`, `450` → `$450`.
    pub fn unit_fix(&self, expr: &Expr, expected: &Type) -> Option<Fix> {
        let ExprKind::Number { unit: None, .. } = &expr.kind else {
            return None;
        };
        if expr.parens > 0 {
            return None;
        }
        let Type::Quantity(quantity) = expected.unwrapped() else {
            return None;
        };
        let digits = self.text(expr.span);
        let written = if quantity.dim == Dim::money() {
            format!("${digits}")
        } else {
            let unit = unit_text(quantity)?;
            match unit.strip_prefix("$/") {
                Some(per) => format!("${digits}/{per}"),
                None => format!("{digits} {unit}"),
            }
        };
        Some(replace(
            Message::with(message!("Write it as {0}"), [quote(&written)]),
            expr.span,
            written,
        ))
    }

    /// Reports an optional `expr` (of type `ty`) used where its value is needed, with the fixes:
    /// a fallback, `if let` around the statement, and `require let` before it. `operand` says
    /// the expression is an operand, so a fallback needs parentheses.
    pub fn optional_error(&mut self, expr: &Expr, ty: &Type, operand: bool) {
        let inner = ty.unwrapped().clone();
        let mut diagnostic = Diagnostic::error(
            "optional",
            expr.span,
            Message::with(
                message!(
                    "{0} may be empty, so it can't be used as {1} here. Give it a fallback, bind it with `if let`, or refuse to rate without it"
                ),
                [quote_span(self.source, expr.span), Arg::from(words(&inner))],
            ),
        );
        let code = self.text(expr.span).to_owned();
        if let Some(fallback) = zero(&inner) {
            let written = if operand {
                format!("({code} ?? {fallback})")
            } else {
                format!("{code} ?? {fallback}")
            };
            diagnostic = diagnostic.with_fix(replace(
                Message::with(
                    message!("Fall back to {0}"),
                    [quote(&format!("{code} ?? {fallback}"))],
                ),
                expr.span,
                written,
            ));
        }
        for fix in self.binding_fixes(expr) {
            diagnostic = diagnostic.with_fix(fix);
        }
        self.report(diagnostic);
    }

    /// The `if let` and `require let` fixes for an optional expression, when the statement it
    /// is in can take them: its names are all declared before that statement, and it is not
    /// inside a lambda.
    fn binding_fixes(&self, expr: &Expr) -> Vec<Fix> {
        let Some(statement) = self.statements.last().copied() else {
            return Vec::new();
        };
        if self.lambda_depth > 0 || matches!(self.body, Body::Constant) {
            return Vec::new();
        }
        if !self.declared_before(expr, statement.start) {
            return Vec::new();
        }
        let Some(name) = self.fresh_name(expr) else {
            return Vec::new();
        };
        let code = self.text(expr.span).to_owned();
        let indent = indentation(self.source, statement.start);
        let mut out = Vec::new();
        let statement_text = self.text(statement);
        let binds = ["let ", "var ", "require "]
            .iter()
            .any(|keyword| statement_text.starts_with(keyword));
        if !binds {
            out.push(changes(
                Message::with(
                    message!("Bind it with {0}"),
                    [quote(&format!("if let {name} = {code}"))],
                ),
                vec![
                    Change {
                        span: Span::new(statement.start, statement.start),
                        insert: format!("if let {name} = {code} {{\n{indent}  "),
                    },
                    Change {
                        span: expr.span,
                        insert: name.clone(),
                    },
                    Change {
                        span: Span::new(statement.end, statement.end),
                        insert: format!("\n{indent}}}"),
                    },
                ],
            ));
        }
        let message = format!("This rate needs {}", code.replace('"', "'"));
        out.push(changes(
            Message::with(
                message!("Require it with {0}"),
                [quote(&format!("require let {name} = {code}"))],
            ),
            vec![
                Change {
                    span: Span::new(statement.start, statement.start),
                    insert: format!("require let {name} = {code} else \"{message}\"\n{indent}"),
                },
                Change {
                    span: expr.span,
                    insert: name,
                },
            ],
        ));
        out
    }

    /// Whether every name an expression reads was declared before a byte offset (so hoisting
    /// it above the statement there keeps its meaning).
    fn declared_before(&self, expr: &Expr, offset: u32) -> bool {
        match &expr.kind {
            ExprKind::Name(name) => self
                .scopes
                .find(&name.text)
                .and_then(|id| self.scopes.get(id))
                .is_none_or(|local| local.span.start < offset),
            ExprKind::Field { target, .. } => self.declared_before(target, offset),
            ExprKind::Call { callee, args } => {
                self.declared_before(callee, offset)
                    && args
                        .iter()
                        .all(|arg| self.declared_before(&arg.value, offset))
            }
            ExprKind::Index { target, keys } => {
                self.declared_before(target, offset)
                    && keys.iter().all(|key| self.declared_before(key, offset))
            }
            ExprKind::Binary { left, right, .. } => {
                self.declared_before(left, offset) && self.declared_before(right, offset)
            }
            ExprKind::Unary { operand, .. } => self.declared_before(operand, offset),
            ExprKind::Lambda { .. } | ExprKind::Match { .. } | ExprKind::If { .. } => false,
            _ => true,
        }
    }

    /// A name for binding an optional: its last field (`weight` for `shipment.weight`), or
    /// that with `Value` added when the name is taken.
    fn fresh_name(&self, expr: &Expr) -> Option<String> {
        let base = match &expr.kind {
            ExprKind::Name(name) => format!("{}Value", name.text),
            ExprKind::Field { name, .. } => name.text.clone(),
            _ => String::from("value"),
        };
        [base.clone(), format!("{base}Value")]
            .into_iter()
            .find(|candidate| !self.taken(candidate))
    }

    /// Whether a name is already visible, so a binding may not take it.
    pub fn taken(&self, name: &str) -> bool {
        self.scopes.find(name).is_some()
            || self.globals.values.contains_key(name)
            || self.schema.root(name).is_some()
            || name == "tables"
            || super::lower::builtin(name).is_some()
    }

    /// Reports two operands an arithmetic or comparison operator does not combine, saying what
    /// each is and what would have worked.
    pub fn operator_error(
        &mut self,
        problem: Problem,
        op: BinaryOp,
        left: (&Expr, &Type),
        right: (&Expr, &Type),
        whole: Span,
    ) {
        let left_code = self.text(left.0.span).to_owned();
        let right_code = self.text(right.0.span).to_owned();
        let written = quote(&format!("{left_code} {} {right_code}", op.symbol()));
        let message = match problem {
            Problem::PercentTimesPercent => Message::with(
                message!(
                    "{0} multiplies a percent by a percent, which means nothing in a price. Apply each percent to an amount instead"
                ),
                [written],
            ),
            Problem::Temperature => Message::with(
                message!("Temperatures can only be compared and subtracted, so {0} has no meaning"),
                [written],
            ),
            Problem::Remainder => Message::with(
                message!(
                    "{0} needs two whole numbers, but {1} is {2} and {3} is {4}. Round first with {5}"
                ),
                [
                    written,
                    quote(&left_code),
                    Arg::from(words(left.1)),
                    quote(&right_code),
                    Arg::from(words(right.1)),
                    quote("floor()"),
                ],
            ),
            Problem::Unordered => Self::unordered_message(op, (&left_code, left.1), &right_code),
            Problem::Kinds | Problem::Unnamed => {
                Self::kinds_message(op, (&left_code, left.1), (&right_code, right.1))
            }
        };
        let mut diagnostic = Diagnostic::error("unit-mismatch", whole, message);
        for (side, other) in [(left, right), (right, left)] {
            if matches!(side.1, Type::DateTime) && matches!(other.1, Type::Time) {
                diagnostic = diagnostic.with_fix(self.time_of_day_fix(side.0, false));
            }
        }
        if problem == Problem::Unordered
            && let Some(fix) = self.text_range_fix(op, left.0, right.0, whole)
        {
            diagnostic = diagnostic.with_fix(fix);
        }
        for (literal, other) in [(left.0, right.1), (right.0, left.1)] {
            if matches!(
                op,
                BinaryOp::Add
                    | BinaryOp::Sub
                    | BinaryOp::Lt
                    | BinaryOp::Le
                    | BinaryOp::Gt
                    | BinaryOp::Ge
                    | BinaryOp::Eq
                    | BinaryOp::Ne
            ) && let Some(fix) = self.unit_fix(literal, other)
            {
                diagnostic = diagnostic.with_fix(fix);
            }
        }
        self.report(diagnostic);
    }

    /// Why `left op right` can't order its operands: text has no order outside a range, an
    /// enum other than the ordered ones has none, and the rest name what does.
    fn unordered_message(op: BinaryOp, left: (&str, &Type), right_code: &str) -> Message {
        let (left_code, left_ty) = left;
        match left_ty {
            Type::Text => {
                let subject = if left_code.starts_with('"') && !right_code.starts_with('"') {
                    right_code
                } else {
                    left_code
                };
                Message::with(
                    message!(
                        "{0} is text, which has no order, so it can't be compared with {1}. To test a band of codes such as zips, use a range, compared character by character: {2}"
                    ),
                    [
                        quote(subject),
                        quote(op.symbol()),
                        quote(&format!("{subject} in \"30000\"..=\"31999\"")),
                    ],
                )
            }
            Type::Enum(_) => Message::with(
                message!(
                    "{0} is {1}, which has no order, so it can't be compared with {2}: only {3} order their variants. Compare with {4}, or list the variants with {5}"
                ),
                [
                    quote(left_code),
                    Arg::from(words(left_ty)),
                    quote(op.symbol()),
                    Arg::from(super::operators::ordered_enums()),
                    quote(&format!("{left_code} == {right_code}")),
                    quote(&format!("{left_code} in [a, b]")),
                ],
            ),
            _ => Message::with(
                message!(
                    "{0} is {1}, which can't be ordered: only numbers, money, quantities, dates, times and the {2} enums compare with {3}"
                ),
                [
                    quote(left_code),
                    Arg::from(words(left_ty)),
                    Arg::from(super::operators::ordered_enums()),
                    quote(op.symbol()),
                ],
            ),
        }
    }

    /// `zip < "31000"` as the range it means, `zip in .."31000"`: for `<`, `<=` and `>=` with a
    /// text literal on one side (`>` has no range, since a range's start is always included).
    fn text_range_fix(
        &self,
        op: BinaryOp,
        left: &Expr,
        right: &Expr,
        whole: Span,
    ) -> Option<trenova_module_kit::diagnostic::Fix> {
        let literal =
            |expr: &Expr| matches!(&expr.kind, ExprKind::Text(text) if text.literal().is_some());
        let (value, bound, op) = if literal(right) {
            (left, right, op)
        } else if literal(left) {
            let flipped = match op {
                BinaryOp::Lt => BinaryOp::Gt,
                BinaryOp::Le => BinaryOp::Ge,
                BinaryOp::Gt => BinaryOp::Lt,
                BinaryOp::Ge => BinaryOp::Le,
                other => other,
            };
            (right, left, flipped)
        } else {
            return None;
        };
        let value_code = self.text(value.span);
        let bound_code = self.text(bound.span);
        let written = match op {
            BinaryOp::Lt => format!("{value_code} in ..{bound_code}"),
            BinaryOp::Le => format!("{value_code} in ..={bound_code}"),
            BinaryOp::Ge => format!("{value_code} in {bound_code}.."),
            _ => return None,
        };
        Some(super::fixes::replace(
            Message::with(message!("Write it as the range {0}"), [quote(&written)]),
            whole,
            written,
        ))
    }

    /// "`rate` is money per hundredweight and `shipment.weight` is a weight, so `rate *
    /// shipment.weight` is money. `rate + shipment.weight` adds money to a weight."
    fn kinds_message(op: BinaryOp, left: (&str, &Type), right: (&str, &Type)) -> Message {
        let (left_code, left_ty) = left;
        let (right_code, right_ty) = right;
        let doing = Self::doing(op, left, right);
        let alternatives: &[BinaryOp] = match op {
            BinaryOp::Add
            | BinaryOp::Sub
            | BinaryOp::Lt
            | BinaryOp::Le
            | BinaryOp::Gt
            | BinaryOp::Ge
            | BinaryOp::Eq
            | BinaryOp::Ne => &[BinaryOp::Mul, BinaryOp::Div],
            BinaryOp::Mul => &[BinaryOp::Div],
            BinaryOp::Div => &[BinaryOp::Mul],
            _ => &[],
        };
        for alternative in alternatives {
            let Ok(result) = arithmetic(*alternative, left_ty, right_ty) else {
                continue;
            };
            let rate = |ty: &Type| matches!(ty, Type::Quantity(q) if q.dim.exponent(crate::tariff::units::Base::Money) == 1 && q.dim != Dim::money());
            if *alternative != BinaryOp::Mul
                || !result.is_money()
                || !(rate(left_ty) || rate(right_ty))
            {
                continue;
            }
            return Message::with(
                message!("{0} is {1} and {2} is {3}, so {4} is {5}. {6}"),
                [
                    quote(left_code),
                    Arg::from(words(left_ty)),
                    quote(right_code),
                    Arg::from(words(right_ty)),
                    quote(&format!(
                        "{left_code} {} {right_code}",
                        alternative.symbol()
                    )),
                    Arg::from(words(&result)),
                    Arg::from(doing),
                ],
            );
        }
        let rule = Message::new(match op {
            BinaryOp::Mul => message!(
                "Multiply by a number or a percent, or multiply a rate by the quantity it is priced per."
            ),
            BinaryOp::Div => message!(
                "Divide by a number, by the same kind of value, or money by a quantity to get a rate."
            ),
            _ => message!("Both sides must be the same kind of value."),
        });
        Message::with(
            message!("{0} is {1} and {2} is {3}. {4} {5}"),
            [
                quote(left_code),
                Arg::from(words(left_ty)),
                quote(right_code),
                Arg::from(words(right_ty)),
                Arg::from(doing),
                Arg::from(rule),
            ],
        )
    }

    /// What an operator does to its operands, in words: "`a + b` adds money to a weight."
    fn doing(op: BinaryOp, left: (&str, &Type), right: (&str, &Type)) -> Message {
        let written = quote(&format!("{} {} {}", left.0, op.symbol(), right.0));
        let a = Arg::from(words(left.1));
        let b = Arg::from(words(right.1));
        match op {
            BinaryOp::Add => Message::with(message!("{0} adds {1} to {2}."), [written, a, b]),
            BinaryOp::Sub => {
                Message::with(message!("{0} subtracts {2} from {1}."), [written, a, b])
            }
            BinaryOp::Mul => Message::with(message!("{0} multiplies {1} by {2}."), [written, a, b]),
            BinaryOp::Div => Message::with(message!("{0} divides {1} by {2}."), [written, a, b]),
            _ => Message::with(message!("{0} compares {1} with {2}."), [written, a, b]),
        }
    }

    /// Reports a name nothing declares, with the closest visible name as a fix.
    pub fn unknown_name(&mut self, span: Span, name: &str) {
        let mut candidates = self.scopes.visible();
        for (global, value) in &self.globals.values {
            let visible = match value {
                Global::Input { .. } => self.sees_record(),
                _ => true,
            };
            if visible {
                candidates.push(global.clone());
            }
        }
        if self.sees_record() {
            candidates.extend(
                self.schema
                    .roots
                    .iter()
                    .map(|(root, _, _)| (*root).to_owned()),
            );
            candidates.push(String::from("tables"));
        }
        let suggestion = nearest(name, candidates.iter().map(String::as_str)).map(str::to_owned);
        let diagnostic = match suggestion {
            Some(suggestion) => Diagnostic::error(
                "unknown-name",
                span,
                Message::with(
                    message!("Nothing is named {0}. Did you mean {1}?"),
                    [quote(name), quote(&suggestion)],
                ),
            )
            .with_fix(replace(
                Message::with(message!("Change to {0}"), [quote(&suggestion)]),
                span,
                suggestion,
            )),
            None => Diagnostic::error(
                "unknown-name",
                span,
                Message::with(
                    message!("Nothing is named {0}: declare it with `let` or as an `input`"),
                    [quote(name)],
                ),
            ),
        };
        self.report(diagnostic);
    }

    /// A finding for a name with a did-you-mean fix among candidates.
    pub fn did_you_mean(
        code: &'static str,
        span: Span,
        written: &str,
        candidates: &[String],
        found: Message,
        missing: Message,
    ) -> Diagnostic {
        match nearest(written, candidates.iter().map(String::as_str)) {
            Some(suggestion) => Diagnostic::error(
                code,
                span,
                Message::with(
                    message!("{0}. Did you mean {1}?"),
                    [Arg::from(found), quote(suggestion)],
                ),
            )
            .with_fix(replace(
                Message::with(message!("Change to {0}"), [quote(suggestion)]),
                span,
                suggestion,
            )),
            None => Diagnostic::error(code, span, missing),
        }
    }
}
