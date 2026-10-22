//! Expressions, typed bidirectionally: [`Checker::expr`] takes the type expected where the
//! expression is written (if any) and returns the type it has. The expectation flows into
//! record literals, `[]`, `none`, bare enum variants and the branches of `if` and `match`;
//! everything else is inferred bottom-up.

use super::checker::{Body, Checker, Global, Want};
use super::describe::{quote, quote_span, zero};
use super::fixes::replace;
use super::methods::{self, Rule};
use super::operators::{self, Problem};
use super::scope::{LocalKind, PathKey};
use super::{HintKind, SymbolKind};
use crate::message;
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::syntax::ast::{
    BinaryOp, Condition, Expr, ExprKind, FieldInit, Name, Text, TextPart, UnaryOp, UnitExpr,
};
use crate::tariff::types::{EnumType, FieldDef, RecordType, Type};
use crate::tariff::units::{SIMPLE_UNITS, Unit, simple};
use std::sync::Arc;

impl Checker<'_> {
    /// The type of an expression, checked with the type expected where it is written.
    pub fn expr(&mut self, expr: &Expr, expected: Option<&Type>) -> Type {
        let ty = self.expr_kind(expr, expected);
        self.record(expr, ty)
    }

    /// Checks an expression against a type it must fit, reporting a mismatch (or an optional
    /// used where its value is needed), and returns its type.
    pub fn expect(&mut self, expr: &Expr, expected: &Type, want: &Want) -> Type {
        let actual = self.expr(expr, Some(expected));
        self.fits(expr, &actual, expected, want);
        actual
    }

    /// Reports when a value of type `actual` does not fit `expected`.
    pub fn fits(&mut self, expr: &Expr, actual: &Type, expected: &Type, want: &Want) -> bool {
        if expected.accepts(actual) {
            return true;
        }
        if let Type::Optional(inner) = actual
            && expected.accepts(inner)
        {
            self.optional_error(expr, actual, false);
            return false;
        }
        self.mismatch(expr, actual, expected, want);
        false
    }

    fn expr_kind(&mut self, expr: &Expr, expected: Option<&Type>) -> Type {
        match &expr.kind {
            ExprKind::Number { integer, unit, .. } => self.number(*integer, unit.as_ref()),
            ExprKind::Money { per, .. } => self.money(per.as_ref()),
            ExprKind::Percent(_) => Type::Percent,
            ExprKind::Text(text) => {
                self.text_parts(text);
                Type::Text
            }
            ExprKind::Date(_) => Type::Date,
            ExprKind::DateTime(_) => Type::DateTime,
            ExprKind::Bool(_) => Type::Bool,
            ExprKind::None => match expected {
                Some(ty @ Type::Optional(_)) => ty.clone(),
                Some(Type::Error) => Type::Error,
                _ => Type::Optional(Arc::new(Type::Never)),
            },
            ExprKind::Name(name) => self.name(expr, name),
            ExprKind::Path { head, member } => self.path(expr, head, member.as_ref(), expected),
            ExprKind::Field {
                target,
                name,
                optional,
            } => self.field(expr, target, name, *optional),
            ExprKind::Call { callee, args } => self.call(expr, callee, args, expected),
            ExprKind::Index { target, keys } => self.index(expr, target, keys),
            ExprKind::Unary { op, operand, .. } => self.unary(*op, operand),
            ExprKind::Binary {
                op, left, right, ..
            } => self.binary(expr, *op, left, right, expected),
            ExprKind::Range { start, end, .. } => {
                for part in [start, end].into_iter().flatten() {
                    self.expr(part, None);
                }
                self.report(Diagnostic::error(
                    "range",
                    expr.span,
                    Message::new(message!(
                        "A range can only follow `in`, be walked by `for`, or be a `match` pattern"
                    )),
                ));
                Type::Error
            }
            ExprKind::Fail(text) => {
                self.text_parts(text);
                Type::Never
            }
            ExprKind::List(items) => self.list(items, expected),
            ExprKind::Record(fields) => self.record_literal(expr, fields, expected),
            ExprKind::If {
                branches,
                otherwise,
            } => self.if_expr(branches, otherwise, expected),
            ExprKind::Match { scrutinee, arms } => self.match_expr(expr, scrutinee, arms, expected),
            ExprKind::Lambda { body, .. } => {
                self.lambda_depth += 1;
                self.expr(body, None);
                self.lambda_depth -= 1;
                Type::Error
            }
            ExprKind::Missing => Type::Error,
        }
    }

    fn number(&mut self, integer: bool, unit: Option<&UnitExpr>) -> Type {
        let plain = if integer { Type::Int } else { Type::Number };
        let Some(unit) = unit else {
            return plain;
        };
        match self.unit(unit) {
            Some(parsed) if parsed.dim().is_none() => plain,
            Some(parsed) => Type::quantity_in(&parsed),
            None => Type::Error,
        }
    }

    fn money(&mut self, per: Option<&UnitExpr>) -> Type {
        let Some(per) = per else {
            return Type::quantity_in(&Unit::simple(simple("$").unwrap_or(&SIMPLE_UNITS[0])));
        };
        if self.unit(per).is_none() {
            return Type::Error;
        }
        Unit::parse(&format!("$/{}", per.text()))
            .map_or(Type::Error, |unit| Type::quantity_in(&unit))
    }

    /// The unit a unit expression names, reporting a part that names none.
    pub fn unit(&mut self, unit: &UnitExpr) -> Option<Unit> {
        for part in &unit.parts {
            if part.text.is_empty() {
                return None;
            }
            if simple(&part.text).is_none() {
                self.unknown_unit(part);
                return None;
            }
        }
        Unit::parse(&unit.text()).ok()
    }

    /// Reports a unit name the language does not have.
    pub fn unknown_unit(&mut self, part: &Name) {
        let candidates: Vec<String> = SIMPLE_UNITS
            .iter()
            .filter(|unit| unit.symbol != "$" && unit.symbol != "%")
            .map(|unit| unit.symbol.to_owned())
            .collect();
        let diagnostic = Self::did_you_mean(
            "unknown-unit",
            part.span,
            &part.text,
            &candidates,
            Message::with(message!("{0} isn't a unit"), [quote(&part.text)]),
            Message::with(
                message!(
                    "{0} isn't a unit; the units are lb, cwt, kg, t, ton, mi, km, in, ft, m, ft3, m3, gal, l, min, h, d, °F, °C, pallet, piece and stop"
                ),
                [quote(&part.text)],
            ),
        );
        self.report(diagnostic);
    }

    /// Checks the interpolated parts of a text.
    pub fn text_parts(&mut self, text: &Text) {
        for part in &text.parts {
            if let TextPart::Expr(expr) = part {
                self.interpolation(expr);
            }
        }
    }

    fn interpolation(&mut self, expr: &Expr) {
        let ty = self.expr(expr, None);
        let fix = match &ty {
            Type::Error
            | Type::Never
            | Type::Bool
            | Type::Int
            | Type::Number
            | Type::Percent
            | Type::Text
            | Type::Date
            | Type::DateTime
            | Type::Quantity(_)
            | Type::Enum(_) => return,
            Type::Optional(inner) => zero(inner).map(|fallback| {
                let code = format!("{} ?? {fallback}", self.text(expr.span));
                replace(
                    Message::with(message!("Fall back to {0}"), [quote(&code)]),
                    expr.span,
                    code,
                )
            }),
            Type::List(_) => {
                let code = format!("{}.len", self.text(expr.span));
                Some(replace(
                    Message::with(message!("Write its length, {0}"), [quote(&code)]),
                    expr.span,
                    code,
                ))
            }
            Type::Record(record) => record
                .fields
                .iter()
                .find(|field| field.ty == Type::Text)
                .map(|field| {
                    let code = format!("{}.{}", self.text(expr.span), field.name);
                    replace(
                        Message::with(message!("Write its field {0}"), [quote(&code)]),
                        expr.span,
                        code,
                    )
                }),
        };
        let mut diagnostic = Diagnostic::error(
            "interpolation",
            expr.span,
            Message::with(
                message!("{0} is {1}, which can't be written into text"),
                [
                    quote_span(self.source, expr.span),
                    Arg::from(super::describe::words(&ty)),
                ],
            ),
        );
        if let Some(fix) = fix {
            diagnostic = diagnostic.with_fix(fix);
        }
        self.report(diagnostic);
    }

    /// A camelCase or UPPERCASE name read as a value.
    fn name(&mut self, expr: &Expr, name: &Name) -> Type {
        if name.text.is_empty() {
            return Type::Error;
        }
        if let Some(id) = self.scopes.find(&name.text) {
            self.scopes.read(id);
            let Some(local) = self.scopes.get(id) else {
                return Type::Error;
            };
            let (ty, symbol) = (local.ty.clone(), local.symbol);
            self.reference(name.span, symbol);
            return self.narrowed(expr, ty);
        }
        if matches!(self.body, Body::Test) {
            if let Some(ty) = self.bill_name(&name.text) {
                return ty;
            }
            if matches!(name.text.as_str(), "total" | "lines")
                && !self.globals.values.contains_key(&name.text)
            {
                self.report(Diagnostic::error(
                    "test-only",
                    name.span,
                    Message::with(
                        message!("{0} reads the bill, and only a tariff makes one: a library's tests check its functions"),
                        [quote(&name.text)],
                    ),
                ));
                return Type::Error;
            }
        }
        if let Some(global) = self.globals.values.get(&name.text).cloned() {
            let shared = matches!(global, Global::Module { .. })
                && self.sees_record()
                && self.schema.root(&name.text).is_some();
            if !shared {
                return self.global_value(expr, name, &global);
            }
        }
        if self.schema.root(&name.text).is_some() || name.text == "tables" {
            return self.root(expr, name);
        }
        if self.file_lets.contains(&name.text) && !matches!(self.body, Body::Top) {
            self.closed(name, Message::new(message!("the tariff's own binding")));
            return Type::Error;
        }
        if let Some(later) = self.scopes.declared_later(&name.text) {
            self.early.insert(name.text.clone());
            self.report(
                Diagnostic::error(
                    "before-definition",
                    name.span,
                    Message::with(
                        message!("{0} is used before it is defined. Move this below the line that defines it"),
                        [quote(&name.text)],
                    ),
                )
                .with_related(later, Message::new(message!("It is defined here"))),
            );
            return Type::Error;
        }
        self.unknown_name(name.span, &name.text);
        Type::Error
    }

    /// `total` and `lines` in a tariff's tests.
    fn bill_name(&mut self, name: &str) -> Option<Type> {
        if self.kind != crate::tariff::syntax::ast::FileKind::Tariff {
            return None;
        }
        if !matches!(name, "total" | "lines") {
            return None;
        }
        match name {
            "total" => Some(Type::money()),
            "lines" => Some(Type::list(Type::Record(line_type()))),
            _ => None,
        }
    }

    /// Reports a name a function, test or constant cannot see.
    pub fn closed(&mut self, name: &Name, what: Message) {
        let (code, message) = match self.body {
            Body::Function { .. } => (
                "closed-function",
                message!(
                    "{0} is {1}, which a function can't read: a function sees only its parameters, constants, types and other functions. Pass it in as a parameter"
                ),
            ),
            Body::Test => (
                "test-scope",
                message!(
                    "{0} is {1}, which a test can't read: a test reads the bill (`total`, `lines`, `charge(…)`), the record and the file's functions and constants"
                ),
            ),
            _ => (
                "not-constant",
                message!(
                    "{0} is {1}, which isn't known before rating: a constant or default must be computed from literals and other constants"
                ),
            ),
        };
        self.report(Diagnostic::error(
            code,
            name.span,
            Message::with(message, [quote(&name.text), Arg::from(what)]),
        ));
    }

    fn global_value(&mut self, expr: &Expr, name: &Name, global: &Global) -> Type {
        match global {
            Global::Input { ty, symbol } => {
                if !matches!(self.body, Body::Top) {
                    self.closed(name, Message::new(message!("an input")));
                    return Type::Error;
                }
                self.used.insert(name.text.clone());
                self.reference(name.span, *symbol);
                self.narrowed(expr, ty.clone())
            }
            Global::Const { ty, symbol, .. } => {
                self.used.insert(name.text.clone());
                self.reference(name.span, *symbol);
                ty.clone()
            }
            Global::Fn(info) => {
                self.used.insert(name.text.clone());
                self.reference(name.span, info.symbol);
                let call = format!("{}()", name.text);
                let mut diagnostic = Diagnostic::error(
                    "not-a-value",
                    name.span,
                    Message::with(
                        message!("{0} is a function: call it with its arguments, as in {1}"),
                        [quote(&name.text), quote(&call)],
                    ),
                );
                if info.params.iter().all(|param| param.default) {
                    diagnostic = diagnostic.with_fix(replace(
                        Message::with(message!("Call it: {0}"), [quote(&call)]),
                        name.span,
                        call,
                    ));
                }
                self.report(diagnostic);
                Type::Error
            }
            Global::Module { symbol, .. } => {
                self.used.insert(name.text.clone());
                self.reference(name.span, *symbol);
                self.report(Diagnostic::error(
                    "not-a-value",
                    name.span,
                    Message::with(
                        message!("{0} is a module: read one of its declarations, as in {1}"),
                        [quote(&name.text), quote(&format!("{}.name(…)", name.text))],
                    ),
                ));
                Type::Error
            }
        }
    }

    fn root(&mut self, expr: &Expr, name: &Name) -> Type {
        if !self.sees_record() {
            let what = if name.text == "tables" {
                Message::new(message!("the organisation's rate tables"))
            } else {
                Message::new(message!("the record being rated"))
            };
            self.closed(name, what);
            return Type::Error;
        }
        if name.text == "tables" {
            self.report(Diagnostic::error(
                "table",
                name.span,
                Message::new(message!(
                    "Read a rate table by its code and keys, as in `tables.Zones[lane.origin.zip3, lane.destination.zip3]`"
                )),
            ));
            return Type::Error;
        }
        if let Some(symbol) = self.root_symbol(&name.text) {
            self.reference(name.span, symbol);
        }
        let ty = self.schema.root(&name.text).cloned().unwrap_or(Type::Error);
        self.narrowed(expr, ty)
    }

    /// The symbol of a schema root: the program's first symbols are the roots, in schema order,
    /// then `tables`.
    pub fn root_symbol(&self, name: &str) -> Option<super::SymbolId> {
        let count = self.schema.roots.len() + 1;
        self.symbols
            .iter()
            .take(count)
            .position(|symbol| symbol.kind == SymbolKind::Root && symbol.name == name)
            .and_then(|index| u32::try_from(index).ok())
            .map(super::SymbolId)
    }

    /// A value's type with flow narrowing applied: present when an enclosing `and`, `or` or
    /// `match` arm has proven it present.
    pub fn narrowed(&self, expr: &Expr, ty: Type) -> Type {
        if !ty.is_optional() {
            return ty;
        }
        match self.path_key(expr) {
            Some(key) if self.scopes.is_narrowed(&key) => ty.unwrapped().clone(),
            _ => ty,
        }
    }

    /// A `PascalCase` name: an enum variant (bare or qualified) or a type used as a value.
    fn path(
        &mut self,
        expr: &Expr,
        head: &Name,
        member: Option<&Name>,
        expected: Option<&Type>,
    ) -> Type {
        if head.text.is_empty() {
            return Type::Error;
        }
        if let Some(member) = member {
            let Some(enumeration) = self.enum_named(&head.text) else {
                if super::lower::builtin(&head.text).is_some()
                    || self.globals.types.contains_key(&head.text)
                {
                    self.report(Diagnostic::error(
                        "not-a-value",
                        expr.span,
                        Message::with(
                            message!(
                                "{0} isn't a value; only an enum's variants can be read this way"
                            ),
                            [quote_span(self.source, expr.span)],
                        ),
                    ));
                } else {
                    self.unknown_type_name(head);
                }
                return Type::Error;
            };
            self.type_reference(head);
            return self.variant_of(expr, &enumeration, member);
        }
        if let Some(Type::Enum(enumeration)) = expected.map(Type::unwrapped) {
            if enumeration.has(&head.text) {
                return self.resolved_variant(expr, enumeration);
            }
            let enumeration = Arc::clone(enumeration);
            return self.variant_of(expr, &enumeration, head);
        }
        if super::lower::builtin(&head.text).is_some()
            || self.globals.types.contains_key(&head.text)
        {
            self.report(Diagnostic::error(
                "not-a-value",
                head.span,
                Message::with(message!("{0} is a type, not a value"), [quote(&head.text)]),
            ));
            return Type::Error;
        }
        self.bare_variant(expr, head)
    }

    fn enum_named(&self, name: &str) -> Option<Arc<EnumType>> {
        match self.globals.types.get(name) {
            Some((Type::Enum(enumeration), _)) => Some(Arc::clone(enumeration)),
            _ => None,
        }
    }

    fn type_reference(&mut self, head: &Name) {
        if let Some((_, Some(symbol))) = self.globals.types.get(&head.text) {
            let symbol = *symbol;
            self.reference(head.span, symbol);
        }
    }

    fn unknown_type_name(&mut self, head: &Name) {
        let candidates: Vec<String> = self.globals.types.keys().cloned().collect();
        let diagnostic = Self::did_you_mean(
            "unknown-type",
            head.span,
            &head.text,
            &candidates,
            Message::with(message!("{0} isn't a type"), [quote(&head.text)]),
            Message::with(message!("{0} isn't a type"), [quote(&head.text)]),
        );
        self.report(diagnostic);
    }

    fn resolved_variant(&mut self, expr: &Expr, enumeration: &Arc<EnumType>) -> Type {
        self.out.variants.insert(expr.id, Arc::clone(enumeration));
        Type::Enum(Arc::clone(enumeration))
    }

    /// `Enum.Variant`, or a bare variant checked against the enum expected.
    fn variant_of(&mut self, expr: &Expr, enumeration: &Arc<EnumType>, variant: &Name) -> Type {
        if enumeration.has(&variant.text) {
            return self.resolved_variant(expr, enumeration);
        }
        let candidates: Vec<String> = enumeration
            .variants
            .iter()
            .map(|(name, _)| name.clone())
            .collect();
        let shown = Type::Enum(Arc::clone(enumeration)).to_string();
        let diagnostic = Self::did_you_mean(
            "unknown-variant",
            variant.span,
            &variant.text,
            &candidates,
            Message::with(
                message!("{0} isn't a variant of {1}"),
                [quote(&variant.text), quote(&shown)],
            ),
            Message::with(
                message!("{0} isn't a variant of {1}, which has {2}"),
                [
                    quote(&variant.text),
                    quote(&shown),
                    Arg::Text(candidates.join(", ")),
                ],
            ),
        );
        self.report(diagnostic);
        Type::Error
    }

    /// A bare variant with no enum expected: the one enum that has it, or an ambiguity.
    fn bare_variant(&mut self, expr: &Expr, head: &Name) -> Type {
        let owners: Vec<Arc<EnumType>> = self
            .globals
            .enums
            .iter()
            .filter(|enumeration| enumeration.has(&head.text))
            .cloned()
            .collect();
        match owners.as_slice() {
            [one] => {
                let one = Arc::clone(one);
                self.resolved_variant(expr, &one)
            }
            [] => {
                let candidates: Vec<String> = self
                    .globals
                    .enums
                    .iter()
                    .flat_map(|enumeration| {
                        enumeration.variants.iter().map(|(name, _)| name.clone())
                    })
                    .collect();
                let diagnostic = Self::did_you_mean(
                    "unknown-variant",
                    head.span,
                    &head.text,
                    &candidates,
                    Message::with(message!("No enum has a variant {0}"), [quote(&head.text)]),
                    Message::with(message!("No enum has a variant {0}"), [quote(&head.text)]),
                );
                self.report(diagnostic);
                Type::Error
            }
            many => {
                let names: Vec<String> = many
                    .iter()
                    .map(|enumeration| Type::Enum(Arc::clone(enumeration)).to_string())
                    .collect();
                let mut diagnostic = Diagnostic::error(
                    "ambiguous-variant",
                    head.span,
                    Message::with(
                        message!("{0} is a variant of {1}: say which, as in {2}"),
                        [
                            quote(&head.text),
                            Arg::Text(names.join(", ")),
                            quote(&format!(
                                "{}.{}",
                                names.first().cloned().unwrap_or_default(),
                                head.text
                            )),
                        ],
                    ),
                );
                for name in &names {
                    let qualified = format!("{name}.{}", head.text);
                    diagnostic = diagnostic.with_fix(replace(
                        Message::with(message!("Change to {0}"), [quote(&qualified)]),
                        head.span,
                        qualified,
                    ));
                }
                self.report(diagnostic);
                Type::Error
            }
        }
    }

    /// `target.name` or `target?.name`, not called.
    fn field(&mut self, expr: &Expr, target: &Expr, name: &Name, optional: bool) -> Type {
        if let Some((module, alias)) = self.module_alias(target)
            && !self.root_member(module, &alias, &name.text)
        {
            return self.module_member(expr, module, &alias, name, false).0;
        }
        if self.is_tables(target) {
            self.report(Diagnostic::error(
                "table",
                expr.span,
                Message::with(
                    message!("Read a rate table with its keys, as in {0}"),
                    [quote(&format!("tables.{}[…]", name.text))],
                ),
            ));
            return Type::Error;
        }
        let Some(receiver) = self.receiver(expr, target, optional) else {
            return Type::Error;
        };
        let member = self.member(expr, &receiver, name, false);
        let ty = if optional && self.recorded(target.id).is_some_and(|t| t.is_optional()) {
            Type::optional(member)
        } else {
            member
        };
        self.narrowed(expr, ty)
    }

    /// The type a member is read from: the target's, unwrapped for `?.`, with an optional
    /// read through `.` reported (and `?.` offered).
    pub fn receiver(&mut self, expr: &Expr, target: &Expr, optional: bool) -> Option<Type> {
        let ty = self.expr(target, None);
        match &ty {
            Type::Error | Type::Never => None,
            Type::Optional(inner) if optional => Some((**inner).clone()),
            Type::Optional(inner) => {
                let inner = (**inner).clone();
                self.optional_member(expr, target, &ty);
                Some(inner)
            }
            _ if optional => {
                if let Some(span) = self.chain_span(target, expr) {
                    self.report(
                        Diagnostic::warning(
                            "needless-optional-chain",
                            span,
                            Message::with(
                                message!("{0} is never empty, so `?.` isn't needed"),
                                [quote_span(self.source, target.span)],
                            ),
                        )
                        .with_fix(replace(
                            Message::new(message!("Use `.`")),
                            span,
                            ".",
                        )),
                    );
                }
                Some(ty)
            }
            _ => Some(ty),
        }
    }

    fn optional_member(&mut self, expr: &Expr, target: &Expr, ty: &Type) {
        let before = self.out.diagnostics.len();
        self.optional_error(target, ty, true);
        if let (Some(span), Some(diagnostic)) = (
            self.chain_span(target, expr),
            self.out.diagnostics.get_mut(before),
        ) {
            diagnostic.fixes.insert(
                0,
                replace(Message::new(message!("Read it with `?.`")), span, "?."),
            );
        }
    }

    /// The span of the `.` or `?.` between a target and the member read from it.
    fn chain_span(&self, target: &Expr, expr: &Expr) -> Option<Span> {
        let gap = Span::new(target.span.end, expr.span.end);
        let text = self.text(gap);
        let offset = text.find('.')?;
        let start = target.span.end + u32::try_from(offset).ok()?;
        let question = offset > 0 && text.get(offset - 1..offset) == Some("?");
        Some(if question {
            Span::new(start - 1, start + 1)
        } else {
            Span::new(start, start + 1)
        })
    }

    /// The type of a member (field or property) of a value of type `ty`; `called` says it is
    /// the callee of a call (handled by the caller, so only fields reach here then).
    pub fn member(&mut self, expr: &Expr, ty: &Type, name: &Name, called: bool) -> Type {
        if name.text.is_empty() {
            return Type::Error;
        }
        if let Type::Record(record) = ty
            && let Some(field) = record.field(&name.text)
        {
            return field.ty.clone();
        }
        if let Some(method) = methods::find(ty, &name.text) {
            if method.property {
                return self.property_type(method.rule, ty);
            }
            if !called {
                let call = format!("{}()", name.text);
                let mut diagnostic = Diagnostic::error(
                    "method-call",
                    name.span,
                    Message::with(
                        message!("{0} is a method: call it with parentheses ({1})"),
                        [quote(&name.text), quote(&method.signature())],
                    ),
                );
                if method.required() == 0 {
                    diagnostic = diagnostic.with_fix(replace(
                        Message::with(message!("Call it: {0}"), [quote(&call)]),
                        name.span,
                        call,
                    ));
                }
                self.report(diagnostic);
            }
            return Type::Error;
        }
        self.unknown_member(expr, ty, name);
        Type::Error
    }

    fn property_type(&self, rule: Rule, ty: &Type) -> Type {
        match rule {
            Rule::Int => Type::Int,
            Rule::Bool => Type::Bool,
            Rule::Number => Type::Number,
            Rule::Date => Type::Date,
            Rule::Text => Type::Text,
            Rule::Weekday => self
                .schema
                .enumeration("Weekday")
                .map_or(Type::Error, Type::Enum),
            _ => ty.clone(),
        }
    }

    /// Reports a member a value does not have, with the closest one as a fix.
    pub fn unknown_member(&mut self, expr: &Expr, ty: &Type, name: &Name) {
        let mut candidates: Vec<String> = Vec::new();
        if let Type::Record(record) = ty {
            candidates.extend(record.fields.iter().map(|field| field.name.clone()));
        }
        if let Some(receiver) = methods::Receiver::of(ty) {
            candidates.extend(methods::of(receiver).map(|method| method.name.to_owned()));
        }
        let owner = match &expr.kind {
            ExprKind::Field { target, .. } => quote_span(self.source, target.span),
            _ => quote(&ty.to_string()),
        };
        let found = if matches!(ty, Type::Record(_)) {
            Message::with(message!("{0} has no field {1}"), [owner, quote(&name.text)])
        } else {
            Message::with(
                message!("{0} is {1}, which has no property or method {2}"),
                [
                    owner,
                    Arg::from(super::describe::words(ty)),
                    quote(&name.text),
                ],
            )
        };
        let diagnostic = Self::did_you_mean(
            "unknown-field",
            name.span,
            &name.text,
            &candidates,
            found.clone(),
            found,
        );
        self.report(diagnostic);
    }

    /// Whether `alias.name` reads a field of the schema root the alias shares its name with
    /// (`fuel.price` after `use std.fuel`): the module does not declare `name`, and the root is
    /// visible.
    pub fn root_member(&self, module: super::ModuleId, alias: &str, name: &str) -> bool {
        let declared = self
            .exports
            .get(usize::try_from(module.0).unwrap_or(usize::MAX))
            .is_some_and(|exports| exports.values.contains_key(name));
        !declared && self.sees_record() && self.schema.root(alias).is_some()
    }

    /// When an expression names a module alias (`fuel` after `use std.fuel`) and no local
    /// shadows it, the module and the alias.
    pub fn module_alias(&self, target: &Expr) -> Option<(super::ModuleId, String)> {
        let ExprKind::Name(name) = &target.kind else {
            return None;
        };
        if self.scopes.find(&name.text).is_some() {
            return None;
        }
        match self.globals.values.get(&name.text) {
            Some(Global::Module { module, .. }) => Some((*module, name.text.clone())),
            _ => None,
        }
    }

    /// Whether an expression is the `tables` root, visible here.
    pub fn is_tables(&self, target: &Expr) -> bool {
        matches!(&target.kind, ExprKind::Name(name) if name.text == "tables")
            && self.scopes.find("tables").is_none()
            && self.sees_record()
    }

    /// `alias.member`: a constant (read) or a function (`called`). Returns the type (for a
    /// constant) and the function, when it is one.
    pub fn module_member(
        &mut self,
        expr: &Expr,
        module: super::ModuleId,
        alias: &str,
        name: &Name,
        called: bool,
    ) -> (Type, Option<Arc<super::checker::FnInfo>>) {
        self.used.insert(alias.to_owned());
        if let ExprKind::Field { target, .. } = &expr.kind
            && let (ExprKind::Name(alias_name), Some(Global::Module { symbol, .. })) =
                (&target.kind, self.globals.values.get(alias))
        {
            let symbol = *symbol;
            self.reference(alias_name.span, symbol);
        }
        let exports = self
            .exports
            .get(usize::try_from(module.0).unwrap_or(usize::MAX));
        let found = exports
            .and_then(|exports| exports.values.get(&name.text))
            .cloned();
        let private = match &found {
            Some(Global::Fn(info)) => info.private,
            Some(Global::Const { private, .. }) => *private,
            _ => false,
        };
        if private && module != self.module {
            self.report(Diagnostic::error(
                "private",
                name.span,
                Message::with(
                    message!("{0} is private to {1}"),
                    [quote(&name.text), quote(alias)],
                ),
            ));
            return (Type::Error, None);
        }
        match found {
            Some(Global::Const { ty, symbol, .. }) => {
                self.reference(name.span, symbol);
                if called {
                    self.report(Diagnostic::error(
                        "not-callable",
                        name.span,
                        Message::with(
                            message!("{0} is a constant, not a function"),
                            [quote(&name.text)],
                        ),
                    ));
                    return (Type::Error, None);
                }
                (ty, None)
            }
            Some(Global::Fn(info)) => {
                self.reference(name.span, info.symbol);
                if !called {
                    self.report(Diagnostic::error(
                        "not-a-value",
                        name.span,
                        Message::with(
                            message!("{0} is a function: call it with its arguments"),
                            [quote(&format!("{alias}.{}", name.text))],
                        ),
                    ));
                    return (Type::Error, None);
                }
                (info.ret.clone(), Some(info))
            }
            _ => {
                let candidates: Vec<String> = exports
                    .map(|exports| exports.values.keys().cloned().collect())
                    .unwrap_or_default();
                let found = Message::with(
                    message!("{0} has no declaration {1}"),
                    [quote(alias), quote(&name.text)],
                );
                let diagnostic = Self::did_you_mean(
                    "unknown-member",
                    name.span,
                    &name.text,
                    &candidates,
                    found.clone(),
                    found,
                );
                self.report(diagnostic);
                (Type::Error, None)
            }
        }
    }

    /// `list[i]` and `text[i]`; rate tables are read in [`Checker::table_index`].
    fn index(&mut self, expr: &Expr, target: &Expr, keys: &[Expr]) -> Type {
        if let ExprKind::Field {
            target: root,
            name,
            optional: false,
        } = &target.kind
            && self.is_tables(root)
        {
            return self.table_index(expr, name, keys);
        }
        let ty = self.operand(target);
        let item = match &ty {
            Type::List(item) => (**item).clone(),
            Type::Text => Type::Text,
            Type::Error | Type::Never => {
                for key in keys {
                    self.expr(key, None);
                }
                return Type::Error;
            }
            other => {
                for key in keys {
                    self.expr(key, None);
                }
                self.report(Diagnostic::error(
                    "not-indexable",
                    target.span,
                    Message::with(
                        message!("{0} is {1}, which can't be indexed with `[…]`"),
                        [
                            quote_span(self.source, target.span),
                            Arg::from(super::describe::words(other)),
                        ],
                    ),
                ));
                return Type::Error;
            }
        };
        if let [key] = keys {
            self.expect(key, &Type::Int, &Want::Value);
        } else {
            for key in keys {
                self.expr(key, None);
            }
            self.report(Diagnostic::error(
                "arity",
                expr.span,
                Message::new(message!(
                    "A list or text is indexed by one whole number, from 0"
                )),
            ));
        }
        Type::optional(item)
    }

    /// An operand of an operator: its type, with an optional reported (and unwrapped, so one
    /// mistake is reported once).
    pub fn operand(&mut self, expr: &Expr) -> Type {
        let ty = self.expr(expr, None);
        if let Type::Optional(inner) = &ty {
            let inner = (**inner).clone();
            if !matches!(inner, Type::Never) {
                self.optional_error(expr, &ty, true);
            }
            return inner;
        }
        ty
    }

    fn unary(&mut self, op: UnaryOp, operand: &Expr) -> Type {
        match op {
            UnaryOp::Not => {
                self.expect(operand, &Type::Bool, &Want::Condition);
                Type::Bool
            }
            UnaryOp::Neg => {
                let ty = self.operand(operand);
                match &ty {
                    Type::Int
                    | Type::Number
                    | Type::Percent
                    | Type::Quantity(_)
                    | Type::Error
                    | Type::Never => ty,
                    other => {
                        self.report(Diagnostic::error(
                            "type-mismatch",
                            operand.span,
                            Message::with(
                                message!("{0} is {1}, which can't be negated"),
                                [
                                    quote_span(self.source, operand.span),
                                    Arg::from(super::describe::words(other)),
                                ],
                            ),
                        ));
                        Type::Error
                    }
                }
            }
        }
    }

    fn binary(
        &mut self,
        expr: &Expr,
        op: BinaryOp,
        left: &Expr,
        right: &Expr,
        expected: Option<&Type>,
    ) -> Type {
        match op {
            BinaryOp::And | BinaryOp::Or => {
                self.expect(left, &Type::Bool, &Want::Condition);
                let facts = self.facts(left, op == BinaryOp::And);
                self.scopes.push(Vec::new());
                for fact in facts {
                    self.scopes.narrow(fact);
                }
                self.expect(right, &Type::Bool, &Want::Condition);
                self.scopes.pop();
                Type::Bool
            }
            BinaryOp::Coalesce => self.coalesce(left, right, expected),
            BinaryOp::Eq | BinaryOp::Ne => {
                self.equality(op, left, right, expr.span);
                Type::Bool
            }
            BinaryOp::Lt | BinaryOp::Le | BinaryOp::Gt | BinaryOp::Ge => {
                let left_ty = self.operand(left);
                let right_ty = self.operand(right);
                if let Err(problem) = operators::compare(&left_ty, &right_ty) {
                    self.operator_error(
                        problem,
                        op,
                        (left, &left_ty),
                        (right, &right_ty),
                        expr.span,
                    );
                }
                Type::Bool
            }
            BinaryOp::In | BinaryOp::NotIn => {
                self.membership(left, right);
                Type::Bool
            }
            BinaryOp::Add | BinaryOp::Sub | BinaryOp::Mul | BinaryOp::Div | BinaryOp::Rem => {
                let left_ty = self.operand(left);
                let right_ty = self.operand(right);
                match operators::arithmetic(op, &left_ty, &right_ty) {
                    Ok(ty) => ty,
                    Err(problem) => {
                        self.operator_error(
                            problem,
                            op,
                            (left, &left_ty),
                            (right, &right_ty),
                            expr.span,
                        );
                        Type::Error
                    }
                }
            }
        }
    }

    /// What holds when a condition is true (`positive`) or false: the paths it proves present.
    pub fn facts(&self, condition: &Expr, positive: bool) -> Vec<PathKey> {
        match &condition.kind {
            ExprKind::Binary {
                op, left, right, ..
            } => match (op, positive) {
                (BinaryOp::Ne, true) | (BinaryOp::Eq, false) => {
                    let subject = match (&left.kind, &right.kind) {
                        (_, ExprKind::None) => left,
                        (ExprKind::None, _) => right,
                        _ => return Vec::new(),
                    };
                    self.path_key(subject).into_iter().collect()
                }
                (BinaryOp::And, true) | (BinaryOp::Or, false) => {
                    let mut out = self.facts(left, positive);
                    out.extend(self.facts(right, positive));
                    out
                }
                _ => Vec::new(),
            },
            ExprKind::Unary {
                op: UnaryOp::Not,
                operand,
                ..
            } => self.facts(operand, !positive),
            _ => Vec::new(),
        }
    }

    fn coalesce(&mut self, left: &Expr, right: &Expr, expected: Option<&Type>) -> Type {
        let left_ty = self.expr(left, expected.map(|ty| Type::optional(ty.clone())).as_ref());
        let inner = match &left_ty {
            Type::Optional(inner) => (**inner).clone(),
            Type::Error | Type::Never => {
                self.expr(right, expected);
                return Type::Error;
            }
            other => {
                let span = Span::new(left.span.end, right.span.end);
                self.report(
                    Diagnostic::warning(
                        "needless-fallback",
                        Span::new(right.span.start, right.span.end),
                        Message::with(
                            message!("{0} is never empty, so this fallback is never used"),
                            [quote_span(self.source, left.span)],
                        ),
                    )
                    .with_fix(replace(
                        Message::new(message!("Remove the fallback")),
                        span,
                        "",
                    )),
                );
                other.clone()
            }
        };
        let base = if matches!(inner, Type::Never) {
            expected.cloned()
        } else {
            Some(inner.clone())
        };
        let right_ty = self.expr(
            right,
            base.as_ref().map(|ty| Type::optional(ty.clone())).as_ref(),
        );
        if matches!(inner, Type::Never) {
            return right_ty;
        }
        let Some(joined) = inner.join(right_ty.unwrapped()) else {
            self.mismatch(right, &right_ty, &inner, &Want::Value);
            return inner;
        };
        if right_ty.is_optional() {
            Type::optional(joined)
        } else {
            joined
        }
    }

    fn equality(&mut self, op: BinaryOp, left: &Expr, right: &Expr, whole: Span) {
        let bare = |expr: &Expr| {
            matches!(
                expr.kind,
                ExprKind::Path { member: None, .. } | ExprKind::None
            )
        };
        let (first, second) = if bare(left) && !bare(right) {
            (right, left)
        } else {
            (left, right)
        };
        let first_ty = self.expr(first, None);
        let hint = match first_ty.unwrapped() {
            Type::Never => None,
            other => Some(Type::optional(other.clone())),
        };
        let second_ty = self.expr(second, hint.as_ref());
        for (value, value_ty, other) in [(first, &first_ty, second), (second, &second_ty, first)] {
            if matches!(other.kind, ExprKind::None)
                && !value_ty.is_optional()
                && !matches!(value_ty, Type::Error | Type::Never)
            {
                self.report(Diagnostic::error(
                    "never-none",
                    whole,
                    Message::with(
                        message!("{0} is never empty, so comparing it with `none` is always {1}"),
                        [
                            quote_span(self.source, value.span),
                            Arg::Text(String::from(if op == BinaryOp::Eq {
                                "false"
                            } else {
                                "true"
                            })),
                        ],
                    ),
                ));
                return;
            }
        }
        if let Err(problem) = operators::equality(&first_ty, &second_ty) {
            let (l, r) = if std::ptr::eq(first, left) {
                ((left, &first_ty), (right, &second_ty))
            } else {
                ((left, &second_ty), (right, &first_ty))
            };
            if matches!(problem, Problem::Kinds)
                && is_quantity(l.1.unwrapped())
                && is_quantity(r.1.unwrapped())
            {
                self.operator_error(
                    problem,
                    op,
                    (l.0, l.1.unwrapped()),
                    (r.0, r.1.unwrapped()),
                    whole,
                );
            } else {
                self.report(Diagnostic::error(
                    "type-mismatch",
                    whole,
                    Message::with(
                        message!("{0} is {1} and {2} is {3}, so they can't be compared"),
                        [
                            quote_span(self.source, l.0.span),
                            Arg::from(super::describe::words(l.1)),
                            quote_span(self.source, r.0.span),
                            Arg::from(super::describe::words(r.1)),
                        ],
                    ),
                ));
            }
        }
    }

    /// `x in list`, `x in lo..hi`.
    fn membership(&mut self, left: &Expr, right: &Expr) {
        if let ExprKind::Range { start, end, .. } = &right.kind {
            let item = self.operand(left);
            if !operators::ordered(&item) {
                self.report(Diagnostic::error(
                    "type-mismatch",
                    left.span,
                    Message::with(
                        message!("{0} is {1}, which has no order, so it can't be in a range"),
                        [
                            quote_span(self.source, left.span),
                            Arg::from(super::describe::words(&item)),
                        ],
                    ),
                ));
            }
            let mut reported = false;
            for part in [start, end].into_iter().flatten() {
                let ty = self.operand(part);
                if !reported
                    && operators::compare(&item, &ty).is_err()
                    && !matches!(item, Type::Error)
                {
                    reported = true;
                    self.report(Diagnostic::error(
                        "type-mismatch",
                        right.span,
                        Message::with(
                            message!("{0} is {1}, but the range {2} is of {3}"),
                            [
                                quote_span(self.source, left.span),
                                Arg::from(super::describe::words(&item)),
                                quote_span(self.source, right.span),
                                Arg::from(super::describe::words(&ty)),
                            ],
                        ),
                    ));
                }
            }
            self.record(right, Type::Error);
            return;
        }
        let bare = matches!(left.kind, ExprKind::Path { member: None, .. });
        if bare {
            let list = self.operand(right);
            let item = match &list {
                Type::List(item) => Some((**item).clone()),
                _ => None,
            };
            let left_ty = self.expr(left, item.as_ref());
            self.member_of(left, &left_ty, right, &list);
        } else {
            let left_ty = self.operand(left);
            let list = self.expr(right, Some(&Type::list(left_ty.clone())));
            let list = if let Type::Optional(inner) = &list {
                self.optional_error(right, &list, true);
                (**inner).clone()
            } else {
                list
            };
            self.member_of(left, &left_ty, right, &list);
        }
    }

    fn member_of(&mut self, left: &Expr, item: &Type, right: &Expr, list: &Type) {
        match list {
            Type::List(of) => {
                if item.join(of).is_none() {
                    self.report(Diagnostic::error(
                        "type-mismatch",
                        left.span,
                        Message::with(
                            message!("{0} is {1}, but {2} holds {3}"),
                            [
                                quote_span(self.source, left.span),
                                Arg::from(super::describe::words(item)),
                                quote_span(self.source, right.span),
                                Arg::from(super::describe::words(of)),
                            ],
                        ),
                    ));
                }
            }
            Type::Error | Type::Never => {}
            other => self.report(Diagnostic::error(
                "type-mismatch",
                right.span,
                Message::with(
                    message!("{0} is {1}; `in` needs a list or a range"),
                    [
                        quote_span(self.source, right.span),
                        Arg::from(super::describe::words(other)),
                    ],
                ),
            )),
        }
    }

    fn list(&mut self, items: &[Expr], expected: Option<&Type>) -> Type {
        if let Some(Type::List(item)) = expected.map(Type::unwrapped) {
            let item = (**item).clone();
            for value in items {
                self.expect(value, &item, &Want::Item);
            }
            return Type::list(item);
        }
        let mut joined: Option<Type> = None;
        for value in items {
            let ty = self.expr(value, joined.as_ref());
            joined = match joined {
                None => Some(ty),
                Some(so_far) => {
                    if let Some(next) = so_far.join(&ty) {
                        Some(next)
                    } else {
                        self.mismatch(value, &ty, &so_far, &Want::Item);
                        Some(so_far)
                    }
                }
            };
        }
        Type::list(joined.unwrap_or(Type::Never))
    }

    fn record_literal(
        &mut self,
        expr: &Expr,
        fields: &[FieldInit],
        expected: Option<&Type>,
    ) -> Type {
        if let Some(Type::Record(record)) = expected.map(Type::unwrapped) {
            let record = Arc::clone(record);
            self.record_against(expr, fields, &record);
            return Type::Record(record);
        }
        let mut defs: Vec<FieldDef> = Vec::with_capacity(fields.len());
        for field in fields {
            let ty = self.expr(&field.value, None);
            if defs.iter().any(|def| def.name == field.name.text) {
                self.duplicate_field(&field.name);
                continue;
            }
            defs.push(FieldDef {
                name: field.name.text.clone(),
                ty,
                doc: None,
            });
        }
        Type::Record(Arc::new(RecordType {
            name: None,
            fields: defs,
        }))
    }

    fn duplicate_field(&mut self, name: &Name) {
        self.report(Diagnostic::error(
            "duplicate",
            name.span,
            Message::with(
                message!("The field {0} is given twice"),
                [quote(&name.text)],
            ),
        ));
    }

    /// A record literal checked against the record type expected: every field, and no other.
    fn record_against(&mut self, expr: &Expr, fields: &[FieldInit], record: &Arc<RecordType>) {
        let mut seen: Vec<&str> = Vec::new();
        let shown = Type::Record(Arc::clone(record)).to_string();
        for field in fields {
            if seen.contains(&field.name.text.as_str()) {
                self.duplicate_field(&field.name);
                self.expr(&field.value, None);
                continue;
            }
            seen.push(&field.name.text);
            if let Some(def) = record.field(&field.name.text) {
                let ty = def.ty.clone();
                self.expect(
                    &field.value,
                    &ty,
                    &Want::Field {
                        field: field.name.text.clone(),
                    },
                );
            } else {
                self.expr(&field.value, None);
                let candidates: Vec<String> =
                    record.fields.iter().map(|f| f.name.clone()).collect();
                let found = Message::with(
                    message!("{0} has no field {1}"),
                    [quote(&shown), quote(&field.name.text)],
                );
                let diagnostic = Self::did_you_mean(
                    "unknown-field",
                    field.name.span,
                    &field.name.text,
                    &candidates,
                    found.clone(),
                    found,
                );
                self.report(diagnostic);
            }
        }
        let missing: Vec<&FieldDef> = record
            .fields
            .iter()
            .filter(|def| !seen.contains(&def.name.as_str()))
            .collect();
        if missing.is_empty() {
            return;
        }
        let names: Vec<String> = missing
            .iter()
            .map(|def| format!("`{}`", def.name))
            .collect();
        let mut diagnostic = Diagnostic::error(
            "missing-field",
            expr.span,
            Message::with(
                message!("{0} needs {1, plural, one {the field} other {the fields}} {2}"),
                [
                    quote(&shown),
                    Arg::from(missing.len()),
                    Arg::Text(names.join(", ")),
                ],
            ),
        );
        let zeros: Option<Vec<String>> = missing
            .iter()
            .map(|def| {
                let value = if def.ty.is_optional() {
                    Some(String::from("none"))
                } else {
                    zero(&def.ty)
                };
                value.map(|value| format!("{}: {value}", def.name))
            })
            .collect();
        let close = self
            .text(expr.span)
            .rfind('}')
            .and_then(|at| u32::try_from(at).ok());
        if let (Some(zeros), Some(close)) = (zeros, close) {
            let at = expr.span.start + close;
            let before = self.text(Span::new(expr.span.start, at)).trim_end();
            let separator = if fields.is_empty() || before.ends_with(',') {
                " "
            } else {
                ", "
            };
            let insert = format!("{separator}{} ", zeros.join(", "));
            let trimmed_at = expr.span.start + u32::try_from(before.len()).unwrap_or(0);
            diagnostic = diagnostic.with_fix(replace(
                Message::new(message!("Add the missing fields")),
                Span::new(trimmed_at, at),
                insert,
            ));
        }
        self.report(diagnostic);
    }

    fn if_expr(
        &mut self,
        branches: &[(Condition, Expr)],
        otherwise: &Expr,
        expected: Option<&Type>,
    ) -> Type {
        let mut joined: Option<Type> = None;
        for (condition, value) in branches {
            self.scopes.push(Vec::new());
            self.condition(condition);
            let ty = self.expr(value, expected.or(joined.as_ref()));
            self.pop_scope();
            joined = Some(self.join_branch(joined, value, &ty));
        }
        let ty = self.expr(otherwise, expected.or(joined.as_ref()));
        self.join_branch(joined, otherwise, &ty)
    }

    /// Joins one branch's value into the branches so far, reporting a value of another kind.
    pub fn join_branch(&mut self, joined: Option<Type>, value: &Expr, ty: &Type) -> Type {
        match joined {
            None => ty.clone(),
            Some(so_far) => {
                if let Some(next) = so_far.join(ty) {
                    return next;
                }
                self.report(Diagnostic::error(
                    "branch-mismatch",
                    value.span,
                    Message::with(
                        message!("This branch gives {0}, but the branches before it give {1}"),
                        [
                            Arg::from(super::describe::words(ty)),
                            Arg::from(super::describe::words(&so_far)),
                        ],
                    ),
                ));
                so_far
            }
        }
    }

    /// An `if` or `require` condition: a `Bool`, or `let name = optional`, binding the name in
    /// the innermost frame.
    pub fn condition(&mut self, condition: &Condition) {
        match condition {
            Condition::Expr(expr) => {
                self.expect(expr, &Type::Bool, &Want::Condition);
                for fact in self.facts(expr, true) {
                    let _ = fact;
                }
            }
            Condition::Let { name, value } => {
                let ty = self.expr(value, None);
                let bound = match &ty {
                    Type::Optional(inner) => (**inner).clone(),
                    Type::Error | Type::Never => Type::Error,
                    other => {
                        self.report(Diagnostic::warning(
                            "needless-let",
                            value.span,
                            Message::with(
                                message!("{0} is never empty, so `let` always binds it: use `let` on its own line instead"),
                                [quote_span(self.source, value.span)],
                            ),
                        ));
                        other.clone()
                    }
                };
                self.hint(name.span, &bound, HintKind::Pattern);
                self.declare(name, LocalKind::Bound, bound);
            }
        }
    }

    /// Pops a scope, reporting its unused `let`s and `var`s.
    pub fn pop_scope(&mut self) {
        let popped = self.scopes.pop();
        for id in popped {
            let Some(local) = self.scopes.get(id) else {
                continue;
            };
            if local.reads == 0
                && matches!(local.kind, LocalKind::Let | LocalKind::Var)
                && !self.expects_refusal
            {
                let diagnostic = Diagnostic::warning(
                    "unused",
                    local.span,
                    Message::with(message!("{0} is never used"), [quote(&local.name)]),
                );
                self.report(diagnostic);
            }
        }
    }
}

/// Whether a type is money or a quantity.
const fn is_quantity(ty: &Type) -> bool {
    matches!(ty, Type::Quantity(_))
}

/// `type Line = { label: Text, code: Text?, amount: Money, kind: LineKind }`, the lines of the
/// bill a test reads.
#[must_use]
pub fn line_type() -> Arc<RecordType> {
    let kind = Arc::new(EnumType {
        name: String::from("LineKind"),
        variants: ["Charge", "Minimum", "Maximum", "Rounding"]
            .iter()
            .map(|name| ((*name).to_owned(), None))
            .collect(),
        doc: None,
    });
    let field = |name: &str, ty: Type| FieldDef {
        name: name.to_owned(),
        ty,
        doc: None,
    };
    Arc::new(RecordType {
        name: Some(String::from("Line")),
        fields: vec![
            field("label", Type::Text),
            field("code", Type::optional(Type::Text)),
            field("amount", Type::money()),
            field("kind", Type::Enum(kind)),
        ],
    })
}
