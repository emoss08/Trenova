//! Calls: user and library functions (named arguments, defaults, arity), module-qualified
//! calls, built-in methods with their lambdas, conversions, the test record constructors and
//! `charge(…)` and `chosen(…)` (in [`super::choose`]) in tests.

use super::checker::{Body, Checker, FnInfo, Global, Want};
use super::describe::{quote, quote_span, words};
use super::fixes::replace;
use super::methods::{self, Method, ParamKind, Rule};
use super::operators;
use super::scope::LocalKind;
use super::{Callee, HintKind};
use crate::message;
use crate::message::{Arg, Message};
use crate::span::Span;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::syntax::ast::{Arg as CallArg, BinaryOp, Expr, ExprKind, FileKind, Name};
use crate::tariff::types::{FieldDef, RecordType, Type};
use crate::tariff::units::{Unit, simple};
use std::sync::Arc;

/// The test record constructors and the schema record each builds.
pub const CONSTRUCTORS: &[(&str, &str)] = &[
    ("pickup", "Stop"),
    ("delivery", "Stop"),
    ("splitPickup", "Stop"),
    ("splitDelivery", "Stop"),
    ("commodity", "Commodity"),
];

/// A parameter as argument matching sees it.
struct Slot<'p> {
    name: &'p str,
    required: bool,
}

impl Checker<'_> {
    /// `callee(args)`.
    pub fn call(
        &mut self,
        expr: &Expr,
        callee: &Expr,
        args: &[CallArg],
        expected: Option<&Type>,
    ) -> Type {
        match &callee.kind {
            ExprKind::Field {
                target,
                name,
                optional,
            } => {
                if let Some((module, alias)) = self.module_alias(target) {
                    let (_, info) = self.module_member(callee, module, &alias, name, true);
                    return match info {
                        Some(info) => {
                            self.function_call(expr, &info, args, &format!("{alias}.{}", name.text))
                        }
                        None => self.skip_args(args),
                    };
                }
                if let ExprKind::Field {
                    target: root,
                    name: code,
                    optional: false,
                } = &target.kind
                    && self.is_tables(root)
                {
                    return self.table_method(expr, code, name, args);
                }
                self.method_call(expr, callee, target, name, *optional, args)
            }
            ExprKind::Name(name) => self.named_call(expr, name, args, expected),
            ExprKind::Path { head, member } => {
                match (head.text.as_str(), member.as_ref().map(|m| m.text.as_str())) {
                    ("Number", None) => {
                        self.conversion(expr, args, "Number", &Type::Int, Type::Number)
                    }
                    ("Int", Some("parse")) => self.conversion(
                        expr,
                        args,
                        "Int.parse",
                        &Type::Text,
                        Type::optional(Type::Int),
                    ),
                    _ => {
                        self.report(Diagnostic::error(
                        "not-callable",
                        callee.span,
                        Message::with(
                            message!("{0} can't be called; the conversions are `Number(i)` and `Int.parse(text)`"),
                            [quote_span(self.source, callee.span)],
                        ),
                    ));
                        self.skip_args(args)
                    }
                }
            }
            _ => {
                let ty = self.expr(callee, None);
                if !matches!(ty, Type::Error) {
                    self.report(Diagnostic::error(
                        "not-callable",
                        callee.span,
                        Message::with(
                            message!("{0} is {1}, not a function"),
                            [quote_span(self.source, callee.span), Arg::from(words(&ty))],
                        ),
                    ));
                }
                self.skip_args(args)
            }
        }
    }

    /// Types the arguments of a call that could not be resolved, so their own mistakes are
    /// still reported.
    pub fn skip_args(&mut self, args: &[CallArg]) -> Type {
        for arg in args {
            if !matches!(arg.value.kind, ExprKind::Lambda { .. }) {
                self.expr(&arg.value, None);
            }
        }
        Type::Error
    }

    fn named_call(
        &mut self,
        expr: &Expr,
        name: &Name,
        args: &[CallArg],
        expected: Option<&Type>,
    ) -> Type {
        if name.text == "charge" {
            return self.charge_lookup(expr, args);
        }
        if self.scopes.find(&name.text).is_some() {
            let ty = self.expr(
                &Expr {
                    kind: ExprKind::Name(name.clone()),
                    ..expr.clone()
                },
                None,
            );
            if !matches!(ty, Type::Error) {
                self.report(Diagnostic::error(
                    "not-callable",
                    name.span,
                    Message::with(
                        message!("{0} is {1}, not a function"),
                        [quote(&name.text), Arg::from(words(&ty))],
                    ),
                ));
            }
            return self.skip_args(args);
        }
        if let Some(Global::Fn(info)) = self.globals.values.get(&name.text).cloned() {
            self.used.insert(name.text.clone());
            self.reference(name.span, info.symbol);
            return self.function_call(expr, &info, args, &name.text);
        }
        if name.text == "chosen" {
            return self.chosen_lookup(expr, args);
        }
        if let Some((_, record)) = CONSTRUCTORS
            .iter()
            .find(|(constructor, _)| *constructor == name.text)
        {
            return self.constructor(expr, name, record, args);
        }
        let _ = expected;
        if self.globals.values.contains_key(&name.text) || self.schema.root(&name.text).is_some() {
            self.report(Diagnostic::error(
                "not-callable",
                name.span,
                Message::with(message!("{0} is not a function"), [quote(&name.text)]),
            ));
            return self.skip_args(args);
        }
        let candidates: Vec<String> = self
            .globals
            .values
            .iter()
            .filter(|(_, global)| matches!(global, Global::Fn(_)))
            .map(|(name, _)| name.clone())
            .collect();
        let diagnostic = Self::did_you_mean(
            "unknown-function",
            name.span,
            &name.text,
            &candidates,
            Message::with(message!("No function is named {0}"), [quote(&name.text)]),
            Message::with(
                message!("No function is named {0}: declare it with `fn`, or import it with `use`"),
                [quote(&name.text)],
            ),
        );
        self.report(diagnostic);
        self.skip_args(args)
    }

    /// A call of a user or library function.
    pub fn function_call(
        &mut self,
        expr: &Expr,
        info: &Arc<FnInfo>,
        args: &[CallArg],
        label: &str,
    ) -> Type {
        self.callee(
            expr,
            Callee::Function {
                module: info.module,
                name: info.name.clone(),
            },
        );
        self.calls.push((info.module, info.name.clone(), expr.span));
        let slots: Vec<Slot<'_>> = info
            .params
            .iter()
            .map(|param| Slot {
                name: &param.name,
                required: !param.default,
            })
            .collect();
        let matched = self.match_args(expr, &slots, args, label);
        for (arg, slot) in args.iter().zip(&matched) {
            let Some(index) = slot else {
                self.lambda_refused(arg);
                continue;
            };
            let Some(param) = info.params.get(*index) else {
                continue;
            };
            if self.lambda_refused(arg) {
                continue;
            }
            self.expect(
                &arg.value,
                &param.ty,
                &Want::Argument {
                    param: param.name.clone(),
                    callee: label.to_owned(),
                },
            );
        }
        info.ret.clone()
    }

    /// Reports a lambda passed to something other than a list method; true when it was one.
    fn lambda_refused(&mut self, arg: &CallArg) -> bool {
        if matches!(arg.value.kind, ExprKind::Lambda { .. }) {
            self.report(Diagnostic::error(
                "lambda",
                arg.value.span,
                Message::new(message!(
                    "A lambda can only be passed to a list method, such as `where`, `map` or `sum`"
                )),
            ));
            return true;
        }
        if !matches!(arg.value.kind, ExprKind::Lambda { .. }) {
            return false;
        }
        true
    }

    /// Matches arguments to parameters: positional first, then by name. Returns, per
    /// argument, the index of its parameter (`None` for one that matches none, reported).
    fn match_args(
        &mut self,
        call: &Expr,
        slots: &[Slot<'_>],
        args: &[CallArg],
        label: &str,
    ) -> Vec<Option<usize>> {
        let mut filled: Vec<Option<Span>> = vec![None; slots.len()];
        let mut out = Vec::with_capacity(args.len());
        let mut named_seen = false;
        let mut unknown = false;
        let mut position = 0usize;
        for arg in args {
            let index = match &arg.name {
                Some(name) => {
                    named_seen = true;
                    let found = slots.iter().position(|slot| slot.name == name.text);
                    if found.is_none() {
                        unknown = true;
                        let candidates: Vec<String> =
                            slots.iter().map(|slot| slot.name.to_owned()).collect();
                        let message = Message::with(
                            message!("{0} has no parameter {1}"),
                            [quote(label), quote(&name.text)],
                        );
                        let diagnostic = Self::did_you_mean(
                            "unknown-argument",
                            name.span,
                            &name.text,
                            &candidates,
                            message.clone(),
                            message,
                        );
                        self.report(diagnostic);
                    }
                    found
                }
                None if named_seen => {
                    unknown = true;
                    self.report(Diagnostic::error(
                        "argument-order",
                        arg.span,
                        Message::new(message!("Positional arguments come before named ones")),
                    ));
                    None
                }
                None => {
                    let index = position;
                    position += 1;
                    if index >= slots.len() {
                        self.report(Diagnostic::error(
                            "arity",
                            arg.span,
                            Message::with(
                                message!(
                                    "{0} takes {1, plural, one {# argument} other {# arguments}}"
                                ),
                                [quote(label), Arg::from(slots.len())],
                            ),
                        ));
                        None
                    } else {
                        Some(index)
                    }
                }
            };
            let index = index.and_then(|index| {
                let slot = filled.get_mut(index)?;
                if let Some(first) = *slot {
                    self.report(
                        Diagnostic::error(
                            "duplicate-argument",
                            arg.span,
                            Message::with(
                                message!("{0} is given twice"),
                                [quote(slots.get(index).map_or("", |slot| slot.name))],
                            ),
                        )
                        .with_related(first, Message::new(message!("It is first given here"))),
                    );
                    return None;
                }
                *slot = Some(arg.span);
                Some(index)
            });
            out.push(index);
        }
        if !unknown {
            self.missing_args(call, slots, &filled, label);
        }
        out
    }

    /// Reports the required parameters no argument filled.
    fn missing_args(
        &mut self,
        call: &Expr,
        slots: &[Slot<'_>],
        filled: &[Option<Span>],
        label: &str,
    ) {
        let missing: Vec<String> = slots
            .iter()
            .zip(filled)
            .filter(|(slot, given)| slot.required && given.is_none())
            .map(|(slot, _)| format!("`{}`", slot.name))
            .collect();
        if !missing.is_empty() {
            self.report(Diagnostic::error(
                "arity",
                self.parens_span(call),
                Message::with(
                    message!("{0} needs {1, plural, one {the argument} other {the arguments}} {2}"),
                    [
                        quote(label),
                        Arg::from(missing.len()),
                        Arg::Text(missing.join(", ")),
                    ],
                ),
            ));
        }
    }

    /// The span of a call's parentheses, where a missing argument is reported.
    fn parens_span(&self, call: &Expr) -> Span {
        let ExprKind::Call { callee, .. } = &call.kind else {
            return call.span;
        };
        let text = self.text(Span::new(callee.span.end, call.span.end));
        let open = text
            .find('(')
            .and_then(|at| u32::try_from(at).ok())
            .unwrap_or(0);
        let close = text
            .rfind(')')
            .and_then(|at| u32::try_from(at + 1).ok())
            .unwrap_or(0);
        Span::new(
            callee.span.end + open,
            (callee.span.end + close).max(callee.span.end + open),
        )
    }

    /// A built-in method call on a value.
    fn method_call(
        &mut self,
        expr: &Expr,
        callee: &Expr,
        target: &Expr,
        name: &Name,
        optional: bool,
        args: &[CallArg],
    ) -> Type {
        let Some(receiver) = self.receiver(callee, target, optional) else {
            return self.skip_args(args);
        };
        let Some(method) = methods::find(&receiver, &name.text) else {
            if matches!(&receiver, Type::Record(record) if record.field(&name.text).is_some()) {
                self.report(Diagnostic::error(
                    "not-callable",
                    name.span,
                    Message::with(
                        message!("{0} is a field, not a method"),
                        [quote(&name.text)],
                    ),
                ));
            } else {
                self.unknown_member(callee, &receiver, name);
            }
            return self.skip_args(args);
        };
        self.record(callee, Type::Error);
        if method.property {
            let span = self.parens_span(expr);
            self.report(
                Diagnostic::error(
                    "property-call",
                    span,
                    Message::with(
                        message!("{0} is a property: write it without parentheses"),
                        [quote(&name.text)],
                    ),
                )
                .with_fix(replace(
                    Message::new(message!("Remove the parentheses")),
                    span,
                    "",
                )),
            );
            self.skip_args(args);
            let member = self.member(callee, &receiver, name, true);
            return self.optional_result(target, optional, member);
        }
        self.callee(expr, Callee::Method(method));
        let result = self.method_args(expr, method, &receiver, args);
        self.optional_result(target, optional, result)
    }

    fn optional_result(&self, target: &Expr, optional: bool, result: Type) -> Type {
        if optional && self.recorded(target.id).is_some_and(|ty| ty.is_optional()) {
            Type::optional(result)
        } else {
            result
        }
    }

    /// Checks a method's arguments and works out its result.
    fn method_args(
        &mut self,
        expr: &Expr,
        method: &'static Method,
        receiver: &Type,
        args: &[CallArg],
    ) -> Type {
        let slots: Vec<Slot<'_>> = method
            .params
            .iter()
            .map(|param| Slot {
                name: param.name,
                required: param.default.is_none(),
            })
            .collect();
        let label = method.name;
        let matched = self.match_args(expr, &slots, args, label);
        let item = match receiver {
            Type::List(item) => (**item).clone(),
            _ => Type::Error,
        };
        let mut same = receiver.clone();
        let mut lambda: Option<Type> = None;
        let mut unit: Option<Unit> = None;
        let mut places = false;
        for (arg, slot) in args.iter().zip(&matched) {
            let Some(param) = slot.and_then(|index| method.params.get(index)) else {
                self.skip_args(std::slice::from_ref(arg));
                continue;
            };
            let want = Want::Argument {
                param: param.name.to_owned(),
                callee: label.to_owned(),
            };
            match param.kind {
                ParamKind::Int => {
                    places = true;
                    self.expect(&arg.value, &Type::Int, &want);
                }
                ParamKind::Text => {
                    self.expect(&arg.value, &Type::Text, &want);
                    if method.name == "matches" {
                        self.check_pattern(&arg.value);
                    }
                }
                ParamKind::Same => {
                    let takes = if matches!(receiver, Type::Int) {
                        Type::Number
                    } else {
                        receiver.clone()
                    };
                    let ty = self.expect(&arg.value, &takes, &want);
                    if let Some(joined) = same.join(&ty) {
                        same = joined;
                    }
                }
                ParamKind::Unit => unit = self.unit_argument(&arg.value, receiver),
                ParamKind::Predicate
                | ParamKind::Map
                | ParamKind::Summable
                | ParamKind::Ordered
                | ParamKind::Key => {
                    lambda = self.lambda_argument(arg, &item, param.kind, method);
                }
            }
        }
        self.method_result(
            method,
            receiver,
            &item,
            Results {
                same,
                lambda,
                unit,
                places,
            },
            expr,
        )
    }

    fn method_result(
        &mut self,
        method: &Method,
        receiver: &Type,
        item: &Type,
        results: Results,
        expr: &Expr,
    ) -> Type {
        let Results {
            same,
            lambda,
            unit,
            places,
        } = results;
        let value = lambda.clone().unwrap_or_else(|| item.clone());
        match method.rule {
            Rule::Same => same,
            Rule::Int => Type::Int,
            Rule::Number => Type::Number,
            Rule::Bool => Type::Bool,
            Rule::Text => Type::Text,
            Rule::TextList => Type::list(Type::Text),
            Rule::Date => Type::Date,
            Rule::Weekday => self
                .schema
                .enumeration("Weekday")
                .map_or(Type::Error, Type::Enum),
            Rule::Round => {
                if matches!(receiver, Type::Int) || !places {
                    if places { Type::Number } else { Type::Int }
                } else {
                    Type::Number
                }
            }
            Rule::In => match (receiver, unit) {
                (Type::Quantity(_), Some(unit)) => Type::quantity_in(&unit),
                _ => Type::Error,
            },
            Rule::List => receiver.clone(),
            Rule::Map => Type::list(value),
            Rule::Item => Type::optional(item.clone()),
            Rule::Sum => self.summed(expr, &value, method),
            Rule::Extreme => {
                if !operators::ordered(&value) {
                    self.not_ordered(expr, &value, method);
                    return Type::Error;
                }
                Type::optional(value)
            }
            Rule::Average => match self.summed(expr, &value, method) {
                Type::Int => Type::Number,
                other => other,
            },
            Rule::Group => {
                let key = lambda.unwrap_or(Type::Error);
                Type::list(Type::Record(Arc::new(RecordType {
                    name: None,
                    fields: vec![
                        FieldDef {
                            name: String::from("key"),
                            ty: key,
                            doc: None,
                        },
                        FieldDef {
                            name: String::from("items"),
                            ty: receiver.clone(),
                            doc: None,
                        },
                    ],
                })))
            }
        }
    }

    fn summed(&mut self, expr: &Expr, value: &Type, method: &Method) -> Type {
        if operators::summable(value) {
            return value.clone();
        }
        let mut diagnostic = Diagnostic::error(
            "not-summable",
            expr.span,
            Message::with(
                message!("{0} works on numbers, money and quantities, but the values are {1}"),
                [quote(method.name), Arg::from(words(value))],
            ),
        );
        if value.is_optional() && operators::summable(value.unwrapped()) {
            diagnostic.message = Message::with(
                message!(
                    "{0} needs a value for every item, but the values are {1}: give each a fallback, as in {2}"
                ),
                [
                    quote(method.name),
                    Arg::from(words(value)),
                    quote(&format!(
                        "x => … ?? {}",
                        super::describe::zero(value.unwrapped()).unwrap_or_default()
                    )),
                ],
            );
        }
        self.report(diagnostic);
        Type::Error
    }

    fn not_ordered(&mut self, expr: &Expr, value: &Type, method: &Method) {
        let message = if value.is_optional() {
            Message::with(
                message!("{0} compares values, but these are {1}: give each a fallback first"),
                [quote(method.name), Arg::from(words(value))],
            )
        } else {
            Message::with(
                message!("{0} compares values, and {1} can't be ordered"),
                [quote(method.name), Arg::from(words(value))],
            )
        };
        self.report(Diagnostic::error("unordered", expr.span, message));
    }

    /// A lambda argument of a list method: its parameter is an item (and an optional second
    /// parameter the item's index), its body typed per what the method needs.
    fn lambda_argument(
        &mut self,
        arg: &CallArg,
        item: &Type,
        kind: ParamKind,
        method: &Method,
    ) -> Option<Type> {
        let ExprKind::Lambda { params, body } = &arg.value.kind else {
            self.expr(&arg.value, None);
            self.report(Diagnostic::error(
                "lambda",
                arg.value.span,
                Message::with(
                    message!("{0} takes a lambda, as in {1}"),
                    [
                        quote(method.name),
                        quote(&format!("{}(s => …)", method.name)),
                    ],
                ),
            ));
            return None;
        };
        self.scopes.push(Vec::new());
        self.lambda_depth += 1;
        for (index, param) in params.iter().enumerate() {
            let ty = if index == 0 { item.clone() } else { Type::Int };
            if index > 1 {
                self.report(Diagnostic::error(
                    "lambda",
                    param.span,
                    Message::new(message!(
                        "A lambda takes an item, and optionally its index: `(s, i) => …`"
                    )),
                ));
                continue;
            }
            self.hint(param.span, &ty, HintKind::Lambda);
            self.declare(param, LocalKind::Lambda, ty);
        }
        let ty = match kind {
            ParamKind::Predicate => self.expect(body, &Type::Bool, &Want::Condition),
            _ => self.expr(body, None),
        };
        if kind == ParamKind::Ordered && method.rule != Rule::Extreme && !operators::ordered(&ty) {
            self.not_ordered(body, &ty, method);
        }
        self.lambda_depth -= 1;
        self.scopes.pop();
        self.record(&arg.value, ty.clone());
        Some(ty)
    }

    /// `.in(unit)`: the unit written as names, of the receiver's dimension or, on a rate, of
    /// its denominator's; gives the unit the value converts to.
    fn unit_argument(&mut self, value: &Expr, receiver: &Type) -> Option<Unit> {
        let text = self.unit_names(value)?;
        let unit = Unit::parse(&text).ok()?;
        let Type::Quantity(quantity) = receiver else {
            return None;
        };
        let target = Unit::conversion_target(quantity.dim, &unit);
        if target.is_none() {
            let unit_words = match unit.dim() {
                dim if dim.is_none() => Message::new(message!("a count")),
                dim => words(&Type::quantity(dim)),
            };
            self.report(Diagnostic::error(
                "unit-mismatch",
                value.span,
                Message::with(
                    message!(
                        "{0} measures {1}, but this value is {2}, so it can't be converted to it"
                    ),
                    [
                        quote(&text),
                        Arg::from(unit_words),
                        Arg::from(words(receiver)),
                    ],
                ),
            ));
            return None;
        }
        target
    }

    /// The unit text of `.in(…)`'s argument: `cwt` or `lb/ft3`, reporting what is not a unit.
    fn unit_names(&mut self, value: &Expr) -> Option<String> {
        match &value.kind {
            ExprKind::Name(name) if simple(&name.text).is_some() => Some(name.text.clone()),
            ExprKind::Name(name) => {
                self.unknown_unit(name);
                None
            }
            ExprKind::Binary {
                op: BinaryOp::Div,
                left,
                right,
                ..
            } => {
                let top = self.unit_names(left)?;
                let bottom = self.unit_names(right)?;
                Some(format!("{top}/{bottom}"))
            }
            _ => {
                self.expr(value, None);
                self.report(Diagnostic::error(
                    "unknown-unit",
                    value.span,
                    Message::new(message!(
                        "Write the unit by name, as in `.in(cwt)` or `.in(lb/ft3)`"
                    )),
                ));
                None
            }
        }
    }

    /// A `matches` pattern written as a literal must be a valid regular expression.
    fn check_pattern(&mut self, value: &Expr) {
        let ExprKind::Text(text) = &value.kind else {
            return;
        };
        let Some(pattern) = text.literal() else {
            return;
        };
        if regex_lite::RegexBuilder::new(&pattern)
            .size_limit(1 << 20)
            .build()
            .is_err()
        {
            self.report(Diagnostic::error(
                "pattern",
                value.span,
                Message::with(
                    message!("{0} isn't a valid regular expression"),
                    [quote(&pattern)],
                ),
            ));
        }
    }

    /// `Number(i)` and `Int.parse(text)`.
    fn conversion(
        &mut self,
        expr: &Expr,
        args: &[CallArg],
        name: &'static str,
        takes: &Type,
        gives: Type,
    ) -> Type {
        self.callee(expr, Callee::Conversion(name));
        let param = if name == "Number" { "i" } else { "text" };
        let slots = [Slot {
            name: param,
            required: true,
        }];
        let matched = self.match_args(expr, &slots, args, name);
        for (arg, slot) in args.iter().zip(matched) {
            if slot.is_some() {
                self.expect(
                    &arg.value,
                    takes,
                    &Want::Argument {
                        param: param.to_owned(),
                        callee: name.to_owned(),
                    },
                );
            } else {
                self.skip_args(std::slice::from_ref(arg));
            }
        }
        gives
    }

    /// `pickup(…)` and the other test record constructors.
    fn constructor(&mut self, expr: &Expr, name: &Name, record: &str, args: &[CallArg]) -> Type {
        let Some(record) = self.schema.record(record) else {
            return self.skip_args(args);
        };
        if !matches!(self.body, Body::Test) {
            self.report(Diagnostic::error(
                "test-only",
                name.span,
                Message::with(
                    message!(
                        "{0} builds a sample record, so it can only be used inside a `test` block"
                    ),
                    [quote(&name.text)],
                ),
            ));
            return self.skip_args(args);
        }
        if let Some((constructor, _)) = CONSTRUCTORS.iter().find(|(c, _)| *c == name.text) {
            self.callee(expr, Callee::Constructor(constructor));
        }
        let mut seen: Vec<&str> = Vec::new();
        for arg in args {
            let Some(field_name) = &arg.name else {
                self.report(Diagnostic::error(
                    "argument-order",
                    arg.span,
                    Message::with(
                        message!("Name each field {0} sets, as in {1}"),
                        [
                            quote(&name.text),
                            quote(&format!("{}(weight: 500 lb)", name.text)),
                        ],
                    ),
                ));
                self.skip_args(std::slice::from_ref(arg));
                continue;
            };
            if seen.contains(&field_name.text.as_str()) {
                self.report(Diagnostic::error(
                    "duplicate-argument",
                    arg.span,
                    Message::with(message!("{0} is given twice"), [quote(&field_name.text)]),
                ));
            }
            seen.push(&field_name.text);
            if let Some(field) = record.field(&field_name.text) {
                let ty = field.ty.clone();
                self.expect(
                    &arg.value,
                    &ty,
                    &Want::Field {
                        field: field_name.text.clone(),
                    },
                );
            } else {
                self.skip_args(std::slice::from_ref(arg));
                let candidates: Vec<String> =
                    record.fields.iter().map(|f| f.name.clone()).collect();
                let found = Message::with(
                    message!("{0} has no field {1}"),
                    [
                        quote(record.name.as_deref().unwrap_or_default()),
                        quote(&field_name.text),
                    ],
                );
                let diagnostic = Self::did_you_mean(
                    "unknown-field",
                    field_name.span,
                    &field_name.text,
                    &candidates,
                    found.clone(),
                    found,
                );
                self.report(diagnostic);
            }
        }
        Type::Record(record)
    }

    /// `charge(CODE)` or `charge("Label")` in a tariff's test: the amount billed, or `none`.
    fn charge_lookup(&mut self, expr: &Expr, args: &[CallArg]) -> Type {
        if !matches!(self.body, Body::Test) || self.kind != FileKind::Tariff {
            self.report(Diagnostic::error(
                "test-only",
                expr.span,
                Message::new(message!("`charge(…)` reads a line of the bill, so it can only be used inside a tariff's `test` block")),
            ));
            return self.skip_args(args);
        }
        self.callee(expr, Callee::ChargeLookup);
        let result = Type::optional(Type::money());
        let [arg] = args else {
            self.report(Diagnostic::error(
                "arity",
                self.parens_span(expr),
                Message::new(message!("`charge(…)` takes one code or label, as in `charge(FSC)` or `charge(\"Linehaul\")`")),
            ));
            self.skip_args(args);
            return result;
        };
        match &arg.value.kind {
            ExprKind::Name(code) if is_code(&code.text) => {
                self.record(&arg.value, Type::Text);
                if self.code_known(code) && !self.billed.codes.contains(&code.text) {
                    self.report(Diagnostic::warning(
                        "not-billed",
                        code.span,
                        Message::with(
                            message!("This tariff never bills {0}, so this is always `none`"),
                            [quote(&code.text)],
                        ),
                    ));
                }
            }
            ExprKind::Text(text) => {
                self.expr(&arg.value, None);
                if let Some(label) = text.literal()
                    && !self.billed.any_label
                    && !self.billed.labels.contains(&label)
                {
                    let mut diagnostic = Diagnostic::warning(
                        "not-billed",
                        arg.value.span,
                        Message::with(
                            message!(
                                "This tariff never bills a line labelled {0}, so this is always `none`"
                            ),
                            [quote(&label)],
                        ),
                    );
                    if let Some(close) = crate::tariff::diagnostic::nearest(
                        &label,
                        self.billed.labels.iter().map(String::as_str),
                    ) {
                        let written = format!("\"{close}\"");
                        diagnostic = diagnostic.with_fix(replace(
                            Message::with(message!("Change to {0}"), [quote(&written)]),
                            arg.value.span,
                            written,
                        ));
                    }
                    self.report(diagnostic);
                }
            }
            _ => {
                self.expr(&arg.value, None);
                self.report(Diagnostic::error(
                    "type-mismatch",
                    arg.value.span,
                    Message::new(message!(
                        "`charge(…)` takes a charge code such as `FSC` or a label in quotes"
                    )),
                ));
            }
        }
        result
    }

    /// Checks a charge code against the context's codes; true when it is known (or codes are
    /// not loaded). An inactive code is reported here; an unknown one once per code, with all
    /// its uses, by [`super::codes`].
    pub fn code_known(&mut self, code: &Name) -> bool {
        let Some(codes) = &self.context.codes else {
            return true;
        };
        match codes.iter().find(|known| known.code == code.text) {
            Some(known) if known.active => true,
            Some(_) => {
                self.report(Diagnostic::error(
                    "inactive-code",
                    code.span,
                    Message::with(
                        message!("{0} is inactive in the accessorial charge catalog, so it can't be billed"),
                        [quote(&code.text)],
                    ),
                ));
                false
            }
            None => false,
        }
    }
}

/// What a method's arguments contributed to its result.
struct Results {
    same: Type,
    lambda: Option<Type>,
    unit: Option<Unit>,
    places: bool,
}

/// Whether a name is written as a charge code: upper case letters, digits and `_`.
#[must_use]
pub fn is_code(text: &str) -> bool {
    text.chars()
        .next()
        .is_some_and(|first| first.is_ascii_uppercase())
        && text
            .chars()
            .all(|c| c.is_ascii_uppercase() || c.is_ascii_digit() || c == '_')
}
