//! Types calls: user functions, the standard library as functions and as methods, and the
//! lambdas list functions take.

use trenova_module_kit::diagnostic::{Diagnostic, nearest};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg as MessageArg, Message};
use trenova_module_kit::span::Span;

use super::env::Env;
use super::expr::{UnitKind, mismatch, unit_kind};
use super::{Checker, Decl, FnSig};
use crate::lading::facts::Target;
use crate::lading::stdlib;
use crate::lading::syntax::ast::{Arg, Expr, ExprKind, Ident, MemberKind};
use crate::lading::types::{Dim, Type};

/// One argument of a standard call, the receiver first.
#[derive(Clone, Copy)]
enum Input<'e> {
    /// A value: its expression, when written (a receiver used as a property has none).
    Value(Option<&'e Expr>, Span),
    /// A lambda.
    Lambda(&'e Ident, &'e Expr, Span),
}

impl Input<'_> {
    fn span(&self) -> Span {
        match self {
            Self::Value(_, span) | Self::Lambda(_, _, span) => *span,
        }
    }
}

/// What a standard function was given, typed.
struct Call<'e> {
    name: &'e str,
    inputs: Vec<Input<'e>>,
    types: Vec<Option<Type>>,
    span: Span,
    optional: bool,
}

impl Checker<'_> {
    /// Types a call expression.
    pub(crate) fn call(
        &mut self,
        env: &mut Env,
        callee: &Expr,
        args: &[Arg],
        span: Span,
        expected: Option<&Type>,
    ) -> Type {
        let _ = expected;
        match &callee.kind {
            ExprKind::Name(name) => {
                let is_function = stdlib::exists(name)
                    || matches!(self.program.decl(env.unit, name), Some((_, Decl::Fn(_))));
                if env.lookup(name).is_some() && !is_function {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "not-callable",
                            callee.span,
                            Message::with(
                                message!("{0} is a value, not a function"),
                                [MessageArg::from(name.as_str())],
                            ),
                        ),
                    );
                    return Type::Any;
                }
                if let Some((unit, Decl::Fn(_))) = self.program.decl(env.unit, name) {
                    self.occurrence(
                        env.unit,
                        callee.span,
                        Target::Decl {
                            unit,
                            name: name.clone(),
                        },
                        false,
                        false,
                    );
                    return self.user_call(env, unit, name, None, args, span);
                }
                if stdlib::exists(name) {
                    if let Some(function) = stdlib::named(name).next() {
                        self.occurrence(
                            env.unit,
                            callee.span,
                            Target::Stdlib(function.name),
                            false,
                            false,
                        );
                    }
                    let inputs = args.iter().map(input).collect();
                    return self.standard_typed(env, name, inputs, None, span);
                }
                let mut candidates: Vec<String> = stdlib::FUNCTIONS
                    .iter()
                    .map(|function| function.name.to_owned())
                    .collect();
                candidates.extend(self.visible_functions(env.unit));
                let library =
                    self.unused_library_with(env.unit, name, |decl| matches!(decl, Decl::Fn(_)));
                let mut diagnostic = if let Some(library) = &library {
                    Diagnostic::error(
                        "unknown-function",
                        callee.span,
                        Message::with(
                            message!("{0} is in the library {1}, which this file does not use"),
                            [
                                MessageArg::from(name.as_str()),
                                MessageArg::from(library.as_str()),
                            ],
                        ),
                    )
                    .with_fix(self.use_fix(env.unit, library))
                } else {
                    Diagnostic::error(
                        "unknown-function",
                        callee.span,
                        Message::with(
                            message!("There is no function named {0}"),
                            [MessageArg::from(name.as_str())],
                        ),
                    )
                };
                if library.is_none()
                    && let Some(close) = nearest(name, candidates.iter().map(String::as_str))
                {
                    diagnostic = diagnostic.with_fix(super::replace_fix(callee.span, close));
                }
                self.report(env.unit, diagnostic);
                for arg in args {
                    if !matches!(arg.value.kind, ExprKind::Lambda { .. }) {
                        self.expr(env, &arg.value, None);
                    }
                }
                Type::Any
            }
            ExprKind::Member {
                target,
                member,
                kind: MemberKind::Name,
            } => {
                let receiver = self.expr(env, target, None);
                self.method(env, &receiver, target, member, args, span)
            }
            _ => {
                self.expr(env, callee, None);
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "not-callable",
                        callee.span,
                        Message::new(message!("Only functions can be called")),
                    ),
                );
                Type::Any
            }
        }
    }

    fn visible_functions(&self, unit: usize) -> Vec<String> {
        let mut names: Vec<String> = self.program.units[unit]
            .decls
            .iter()
            .filter(|(_, decl)| matches!(decl, Decl::Fn(_)))
            .map(|(name, _)| name.clone())
            .collect();
        for library in self.program.units[unit].header().uses() {
            if let Some(&library_unit) = self.program.libraries.get(&library.name) {
                names.extend(
                    self.program.units[library_unit]
                        .decls
                        .iter()
                        .filter(|(_, decl)| matches!(decl, Decl::Fn(_)))
                        .map(|(name, _)| name.clone()),
                );
            }
        }
        names
    }

    fn method(
        &mut self,
        env: &mut Env,
        receiver: &Type,
        target: &Expr,
        member: &Ident,
        args: &[Arg],
        span: Span,
    ) -> Type {
        let name = member.name.as_str();
        if let Some((unit, Decl::Fn(_))) = self.program.decl(env.unit, name) {
            self.occurrence(
                env.unit,
                member.span,
                Target::Decl {
                    unit,
                    name: name.to_owned(),
                },
                false,
                false,
            );
            return self.user_call(
                env,
                unit,
                name,
                Some((receiver.clone(), target.span)),
                args,
                span,
            );
        }
        if stdlib::exists(name) {
            if let Some(function) = stdlib::named(name).next() {
                self.occurrence(
                    env.unit,
                    member.span,
                    Target::Stdlib(function.name),
                    false,
                    false,
                );
            }
            let mut inputs = vec![Input::Value(Some(target), target.span)];
            inputs.extend(args.iter().map(input));
            return self.standard_typed(env, name, inputs, Some(receiver), span);
        }
        let mut candidates: Vec<String> = stdlib::FUNCTIONS
            .iter()
            .map(|function| function.name.to_owned())
            .collect();
        candidates.extend(self.visible_functions(env.unit));
        let mut diagnostic = Diagnostic::error(
            "unknown-function",
            member.span,
            Message::with(
                message!("{0} has no method {1}"),
                [
                    MessageArg::from(receiver.unwrapped().to_string()),
                    MessageArg::from(name),
                ],
            ),
        );
        if let Some(close) = nearest(name, candidates.iter().map(String::as_str)) {
            diagnostic = diagnostic.with_fix(super::replace_fix(member.span, close));
        }
        self.report(env.unit, diagnostic);
        Type::Any
    }

    /// A standard function used as a property or called with arguments, its receiver already
    /// typed. `None` when no standard function by that name takes these arguments.
    pub(crate) fn stdlib_call(
        &mut self,
        env: &mut Env,
        name: &str,
        receiver: Option<(Type, Span)>,
        args: &[Arg],
        span: Span,
        property: bool,
    ) -> Option<Type> {
        if !stdlib::exists(name) {
            return None;
        }
        let mut inputs = Vec::new();
        let mut receiver_ty = None;
        if let Some((ty, receiver_span)) = receiver {
            inputs.push(Input::Value(None, receiver_span));
            receiver_ty = Some(ty);
        }
        inputs.extend(args.iter().map(input));
        if property && !takes_no_arguments(name, receiver_ty.as_ref()) {
            return None;
        }
        Some(self.standard_typed(env, name, inputs, receiver_ty.as_ref(), span))
    }

    /// Types a standard call: the values first, then the lambdas against the list's items.
    fn standard_typed(
        &mut self,
        env: &mut Env,
        name: &str,
        inputs: Vec<Input<'_>>,
        receiver: Option<&Type>,
        span: Span,
    ) -> Type {
        let mut types = Vec::with_capacity(inputs.len());
        for (index, item) in inputs.iter().enumerate() {
            match item {
                Input::Value(_, _) if index == 0 && receiver.is_some() => {
                    types.push(receiver.cloned());
                }
                Input::Value(Some(expr), _) => {
                    if unit_argument(name, index, expr) {
                        types.push(Some(Type::Any));
                    } else {
                        types.push(Some(self.expr(env, expr, None)));
                    }
                }
                Input::Value(None, _) => types.push(Some(Type::Any)),
                Input::Lambda(..) => types.push(None),
            }
        }
        let mut call = Call {
            name,
            inputs,
            types,
            span,
            optional: false,
        };
        let result = self.type_standard(env, &mut call);
        match result {
            Ok(ty) if call.optional => ty.optional(),
            Ok(ty) => ty,
            Err(diagnostic) => {
                self.report(env.unit, diagnostic);
                Type::Any
            }
        }
    }

    /// The value of argument `index`, unwrapped; marks the call optional when it may be none.
    fn arg(call: &mut Call<'_>, index: usize) -> Option<Type> {
        let ty = call.types.get(index)?.clone()?;
        if ty.is_optional() && !matches!(ty, Type::None) {
            call.optional = true;
        }
        Some(ty.unwrapped().clone())
    }

    fn arity(call: &Call<'_>, min: usize, max: usize) -> Result<(), Diagnostic> {
        let count = call.inputs.len();
        if count < min || count > max {
            let expected = if min == max {
                min.to_string()
            } else {
                format!("{min} to {max}")
            };
            return Err(Diagnostic::error(
                "argument-count",
                call.span,
                Message::with(
                    message!("{0} takes {1} arguments, and it is given {2}"),
                    [
                        MessageArg::from(call.name),
                        MessageArg::from(expected),
                        MessageArg::from(count.to_string()),
                    ],
                ),
            ));
        }
        Ok(())
    }

    fn want(call: &mut Call<'_>, index: usize, expected: &Type) -> Result<(), Diagnostic> {
        let Some(found) = Self::arg(call, index) else {
            return Ok(());
        };
        if found.fits(expected) || (matches!(expected, Type::Zone) && found.is_text()) {
            return Ok(());
        }
        Err(mismatch(call.inputs[index].span(), expected, &found))
    }

    /// Types a lambda argument with its parameter bound to `param`.
    fn lambda(
        &mut self,
        env: &mut Env,
        call: &Call<'_>,
        index: usize,
        param: &Type,
    ) -> Result<Option<Type>, Diagnostic> {
        match call.inputs.get(index) {
            None => Ok(None),
            Some(Input::Lambda(binder, body, _)) => {
                env.push();
                env.bind(&binder.name, param.clone(), binder.span);
                self.occurrence(
                    env.unit,
                    binder.span,
                    Target::Local {
                        unit: env.unit,
                        span: binder.span,
                    },
                    true,
                    false,
                );
                let ty = self.expr(env, body, None);
                env.pop();
                Ok(Some(ty))
            }
            Some(other) => Err(Diagnostic::error(
                "lambda-expected",
                other.span(),
                Message::with(
                    message!("{0} takes a function here, such as s => s.type == Pickup"),
                    [MessageArg::from(call.name)],
                ),
            )),
        }
    }

    #[allow(clippy::too_many_lines)]
    fn type_standard(&mut self, env: &mut Env, call: &mut Call<'_>) -> Result<Type, Diagnostic> {
        let first = Self::arg(call, 0).unwrap_or(Type::Any);
        let name = call.name;
        if let Type::List(item) = first.clone()
            && let Some(ty) = self.list_function(env, call, &item)?
        {
            return Ok(ty);
        }
        if matches!(first.unwrapped(), Type::Json)
            && let Some(ty) = self.list_function(env, call, &Type::Json)?
        {
            return Ok(ty);
        }
        let text = Type::Text;
        let int = Type::Int;
        match name {
            "trim" | "upper" | "lower" | "titleCase" | "digits" | "letters" | "clean" => {
                Self::arity(call, 1, 1)?;
                Self::want(call, 0, &text)?;
                Ok(Type::Text)
            }
            "take" | "drop" | "truncate" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 0, &text)?;
                Self::want(call, 1, &int)?;
                Ok(Type::Text)
            }
            "pad" => {
                Self::arity(call, 2, 4)?;
                Self::want(call, 0, &text)?;
                Self::want(call, 1, &int)?;
                Self::want(call, 2, &text)?;
                Self::want(call, 3, &Type::Bool)?;
                Ok(Type::Text)
            }
            "replace" => {
                Self::arity(call, 3, 3)?;
                for index in 0..3 {
                    Self::want(call, index, &text)?;
                }
                Ok(Type::Text)
            }
            "split" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 0, &text)?;
                Self::want(call, 1, &text)?;
                Ok(Type::Text.list())
            }
            "startsWith" | "endsWith" | "contains" | "checkDigit" | "isDate" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 0, &text)?;
                Self::want(call, 1, &text)?;
                Ok(Type::Bool)
            }
            "matches" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 0, &text)?;
                Self::want(call, 1, &text)?;
                if let Some(Input::Value(Some(pattern), span)) = call.inputs.get(1)
                    && let Some(text) = literal(pattern)
                    && let Err(problem) = regex_lite::RegexBuilder::new(&text)
                        .size_limit(1 << 20)
                        .build()
                {
                    return Err(Diagnostic::error(
                        "bad-pattern",
                        *span,
                        Message::with(
                            message!("The pattern does not read: {0}"),
                            [MessageArg::from(problem.to_string())],
                        ),
                    ));
                }
                Ok(Type::Bool)
            }
            "length" if call.inputs.len() == 2 => Self::measure_builder(call, Dim::Length),
            "length" => {
                Self::arity(call, 1, 1)?;
                Self::want(call, 0, &text)?;
                Ok(Type::Int)
            }
            "isBlank" => {
                Self::arity(call, 1, 1)?;
                let ty = call.types.first().cloned().flatten().unwrap_or(Type::Any);
                if !ty.unwrapped().is_text() {
                    return Err(mismatch(call.inputs[0].span(), &text, &ty));
                }
                Ok(Type::Bool)
            }
            "zip5" | "zip9" | "stateCode" | "phone" => {
                Self::arity(call, 1, 1)?;
                Self::want(call, 0, &text)?;
                Ok(Type::Text.optional())
            }
            "countryCode" => {
                Self::arity(call, 1, 2)?;
                Self::want(call, 0, &text)?;
                Self::want(call, 1, &int)?;
                Ok(Type::Text.optional())
            }
            "isNumber" => {
                Self::arity(call, 1, 1)?;
                Self::want(call, 0, &text)?;
                Ok(Type::Bool)
            }
            "date" | "time" if matches!(first, Type::DateTime) => {
                Self::arity(call, 1, 1)?;
                Ok(if name == "date" {
                    Type::Date
                } else {
                    Type::Time
                })
            }
            "date" | "time" => {
                Self::arity(call, 2, 2)?;
                Self::want_text_or_json(call, 0)?;
                Self::want(call, 1, &text)?;
                Ok(if name == "date" {
                    Type::Date
                } else {
                    Type::Time
                }
                .optional())
            }
            "dateTime" => {
                Self::arity(call, 3, 3)?;
                Self::want_text_or_json(call, 0)?;
                Self::want(call, 1, &text)?;
                Self::want(call, 2, &Type::Zone)?;
                Ok(Type::DateTime.optional())
            }
            "int" | "decimal" => {
                Self::arity(call, 1, 1)?;
                if !matches!(
                    first,
                    Type::Text
                        | Type::Code(..)
                        | Type::Int
                        | Type::Decimal
                        | Type::Json
                        | Type::Any
                ) {
                    return Err(mismatch(call.inputs[0].span(), &Type::Text, &first));
                }
                Ok(if name == "int" {
                    Type::Int
                } else {
                    Type::Decimal
                }
                .optional())
            }
            "text" => {
                Self::arity(call, 1, 1)?;
                let ty = call.types.first().cloned().flatten().unwrap_or(Type::Any);
                Ok(if ty.is_optional() {
                    Type::Text.optional()
                } else {
                    Type::Text
                })
            }
            "round" | "floor" | "ceil" | "abs" => {
                Self::arity(call, 1, if name == "round" { 2 } else { 1 })?;
                Self::want(call, 1, &int)?;
                match first {
                    Type::Int
                    | Type::Decimal
                    | Type::Money
                    | Type::Measure(_)
                    | Type::Duration
                    | Type::Percent
                    | Type::Any => Ok(first),
                    other => Err(mismatch(call.inputs[0].span(), &Type::Decimal, &other)),
                }
            }
            "min" | "max" => {
                Self::arity(call, 2, 2)?;
                let second = Self::arg(call, 1).unwrap_or(Type::Any);
                first
                    .join(&second)
                    .filter(Type::is_ordered)
                    .ok_or_else(|| mismatch(call.inputs[1].span(), &first, &second))
            }
            "money" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 0, &Type::Decimal)?;
                Self::want(call, 1, &text)?;
                Ok(Type::Money)
            }
            "amount" | "currency" => {
                Self::arity(call, 1, 1)?;
                Self::want(call, 0, &Type::Money)?;
                Ok(if name == "amount" {
                    Type::Decimal
                } else {
                    Type::Text
                })
            }
            "weight" => Self::measure_builder(call, Dim::Weight),
            "volume" => Self::measure_builder(call, Dim::Volume),
            "temperature" => Self::measure_builder(call, Dim::Temperature),
            "in" | "value" => {
                Self::arity(call, 2, 2)?;
                let Type::Measure(dim) = first else {
                    if first.is_any() {
                        return Ok(Type::Any);
                    }
                    return Err(mismatch(
                        call.inputs[0].span(),
                        &Type::Measure(Dim::Weight),
                        &first,
                    ));
                };
                let unit = call.inputs.get(1).and_then(|input| match input {
                    Input::Value(
                        Some(Expr {
                            kind: ExprKind::Name(unit),
                            ..
                        }),
                        span,
                    ) => Some((unit.clone(), *span)),
                    _ => None,
                });
                match unit {
                    Some((unit, _)) if unit_kind(&unit) == Some(UnitKind::Measure(dim)) => {}
                    other => {
                        let span = other.map_or(call.span, |(_, span)| span);
                        return Err(Diagnostic::error(
                            "unit-mismatch",
                            span,
                            Message::with(
                                message!("{0} takes a unit of {1}, such as {2}"),
                                [
                                    MessageArg::from(name),
                                    MessageArg::from(dim.name()),
                                    MessageArg::from(example_unit(dim)),
                                ],
                            ),
                        ));
                    }
                }
                Ok(if name == "in" {
                    Type::Measure(dim)
                } else {
                    Type::Decimal
                })
            }
            "at" => {
                Self::arity(call, 3, 3)?;
                let date = call.types.first().cloned().flatten().unwrap_or(Type::Any);
                let time = call.types.get(1).cloned().flatten().unwrap_or(Type::Any);
                if !matches!(date.unwrapped(), Type::Date | Type::Any | Type::None) {
                    return Err(mismatch(
                        call.inputs[0].span(),
                        &Type::Date.optional(),
                        &date,
                    ));
                }
                if !matches!(time.unwrapped(), Type::Time | Type::Any | Type::None) {
                    return Err(mismatch(
                        call.inputs[1].span(),
                        &Type::Time.optional(),
                        &time,
                    ));
                }
                Self::want(call, 2, &Type::Zone)?;
                Ok(Type::DateTime.optional())
            }
            "zone" | "zoneCode" => {
                Self::arity(call, 1, 1)?;
                Self::want(call, 0, &Type::DateTime)?;
                Ok(if name == "zone" {
                    Type::Zone
                } else {
                    Type::Text
                })
            }
            "inZone" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 0, &Type::DateTime)?;
                Self::want(call, 1, &Type::Zone)?;
                Ok(Type::DateTime)
            }
            "zoneFromCode" => {
                Self::arity(call, 1, 1)?;
                Self::want(call, 0, &text)?;
                Ok(Type::Zone.optional())
            }
            "addDays" | "addBusinessDays" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 1, &int)?;
                match first {
                    Type::Date | Type::DateTime | Type::Any => Ok(first),
                    other => Err(mismatch(call.inputs[0].span(), &Type::Date, &other)),
                }
            }
            "now" => {
                Self::arity(call, 0, 0)?;
                Ok(Type::DateTime)
            }
            "join" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 0, &Type::Text.list())?;
                Self::want(call, 1, &text)?;
                Ok(Type::Text)
            }
            "refs" | "address" | "party" | "elements" => Self::x12_function(call, &first),
            "count" | "first" | "last" | "isEmpty" | "find" | "filter" | "map" | "any" | "all"
            | "sum" | "sortBy" | "distinct" | "flatten" | "get" => {
                if first.is_any() {
                    for index in 1..call.inputs.len() {
                        self.lambda(env, call, index, &Type::Any).ok();
                    }
                    return Ok(Type::Any);
                }
                Err(Diagnostic::error(
                    "not-a-list",
                    call.inputs.first().map_or(call.span, Input::span),
                    Message::with(
                        message!("{0} works on a list, and this is {1}"),
                        [MessageArg::from(name), MessageArg::from(first.to_string())],
                    ),
                ))
            }
            _ => Err(Diagnostic::error(
                "unknown-function",
                call.span,
                Message::with(
                    message!("There is no function named {0}"),
                    [MessageArg::from(name)],
                ),
            )),
        }
    }

    fn want_text_or_json(call: &mut Call<'_>, index: usize) -> Result<(), Diagnostic> {
        let found = Self::arg(call, index).unwrap_or(Type::Any);
        if found.is_text() || matches!(found, Type::Json) {
            Ok(())
        } else {
            Err(mismatch(call.inputs[index].span(), &Type::Text, &found))
        }
    }

    fn measure_builder(call: &mut Call<'_>, dim: Dim) -> Result<Type, Diagnostic> {
        Self::arity(call, 2, 2)?;
        Self::want(call, 0, &Type::Decimal)?;
        Self::want(call, 1, &Type::Text)?;
        Ok(Type::Measure(dim).optional())
    }

    fn x12_function(call: &mut Call<'_>, first: &Type) -> Result<Type, Diagnostic> {
        Self::arity(call, 1, 1)?;
        let name = call.name;
        let fits = match name {
            "refs" => {
                matches!(first, Type::List(item) if matches!(item.as_ref(), Type::Segment(_) | Type::SegmentId(_) | Type::Any))
            }
            "elements" => matches!(first, Type::Segment(_) | Type::SegmentId(_)),
            _ => matches!(first, Type::Loop(_) | Type::LoopId(_)),
        } || first.is_any();
        if !fits {
            let expected = match name {
                "refs" => Type::SegmentId("L11".into()).list(),
                "elements" => Type::SegmentId("N1".into()),
                _ => Type::LoopId("0310".into()),
            };
            return Err(mismatch(call.inputs[0].span(), &expected, first));
        }
        Ok(match name {
            "refs" => Type::Map(Box::new(Type::Text)),
            "address" => Type::Record("Address").optional(),
            "party" => Type::Record("Party").optional(),
            _ => Type::Text.list(),
        })
    }

    /// The list functions, or `None` when `call` is not one.
    #[allow(clippy::too_many_lines)]
    fn list_function(
        &mut self,
        env: &mut Env,
        call: &mut Call<'_>,
        item: &Type,
    ) -> Result<Option<Type>, Diagnostic> {
        let name = call.name;
        let item_ty = item.clone();
        Ok(Some(match name {
            "count" => {
                Self::arity(call, 1, 2)?;
                if let Some(result) = self.lambda(env, call, 1, &item_ty)? {
                    self.want_bool(env, call, 1, &result);
                }
                Type::Int
            }
            "first" | "last" => {
                Self::arity(call, 1, 1)?;
                item_ty.optional()
            }
            "isEmpty" => {
                Self::arity(call, 1, 1)?;
                Type::Bool
            }
            "find" | "filter" | "any" | "all" => {
                Self::arity(call, 2, 2)?;
                if let Some(result) = self.lambda(env, call, 1, &item_ty)? {
                    self.want_bool(env, call, 1, &result);
                }
                match name {
                    "find" => item_ty.optional(),
                    "filter" => item_ty.list(),
                    _ => Type::Bool,
                }
            }
            "map" => {
                Self::arity(call, 2, 2)?;
                self.lambda(env, call, 1, &item_ty)?
                    .unwrap_or(Type::Any)
                    .list()
            }
            "sum" => {
                Self::arity(call, 1, 2)?;
                let summed = match self.lambda(env, call, 1, &item_ty)? {
                    Some(result) => result,
                    None => item_ty,
                };
                match summed.unwrapped() {
                    Type::Int
                    | Type::Decimal
                    | Type::Money
                    | Type::Measure(_)
                    | Type::Duration
                    | Type::Any => summed.unwrapped().clone(),
                    other => {
                        return Err(Diagnostic::error(
                            "not-summable",
                            call.span,
                            Message::with(
                                message!("{0} values cannot be added up"),
                                [MessageArg::from(other.to_string())],
                            ),
                        ));
                    }
                }
            }
            "min" | "max" => {
                Self::arity(call, 1, 2)?;
                if let Some(result) = self.lambda(env, call, 1, &item_ty)?
                    && !result.is_ordered()
                {
                    return Err(Diagnostic::error(
                        "not-comparable",
                        call.inputs[1].span(),
                        Message::with(
                            message!("{0} values cannot be ordered"),
                            [MessageArg::from(result.to_string())],
                        ),
                    ));
                }
                item_ty.optional()
            }
            "sortBy" => {
                Self::arity(call, 2, 2)?;
                if let Some(result) = self.lambda(env, call, 1, &item_ty)?
                    && !result.is_ordered()
                {
                    return Err(Diagnostic::error(
                        "not-comparable",
                        call.inputs[1].span(),
                        Message::with(
                            message!("{0} values cannot be ordered"),
                            [MessageArg::from(result.to_string())],
                        ),
                    ));
                }
                item_ty.list()
            }
            "distinct" => {
                Self::arity(call, 1, 1)?;
                item_ty.list()
            }
            "flatten" => {
                Self::arity(call, 1, 1)?;
                match item_ty {
                    Type::List(inner) => inner.as_ref().clone().list(),
                    Type::Any => Type::Any,
                    other => {
                        return Err(Diagnostic::error(
                            "not-a-list",
                            call.span,
                            Message::with(
                                message!("flatten needs a list of lists, and this is [{0}]"),
                                [MessageArg::from(other.to_string())],
                            ),
                        ));
                    }
                }
            }
            "get" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 1, &Type::Int)?;
                item_ty.optional()
            }
            "take" | "drop" => {
                Self::arity(call, 2, 2)?;
                Self::want(call, 1, &Type::Int)?;
                item_ty.list()
            }
            "contains" => {
                Self::arity(call, 2, 2)?;
                let wanted = Self::arg(call, 1).unwrap_or(Type::Any);
                if item_ty.join(&wanted).is_none() {
                    return Err(mismatch(call.inputs[1].span(), &item_ty, &wanted));
                }
                Type::Bool
            }
            "join" => {
                Self::arity(call, 2, 2)?;
                if !item_ty.unwrapped().is_text() {
                    return Err(mismatch(
                        call.inputs[0].span(),
                        &Type::Text.list(),
                        &item_ty.list(),
                    ));
                }
                Self::want(call, 1, &Type::Text)?;
                Type::Text
            }
            "length" if call.inputs.len() == 1 => {
                return Err(Diagnostic::error(
                    "use-count",
                    call.span,
                    Message::new(message!("A list's size is its count: items.count")),
                ));
            }
            _ => return Ok(None),
        }))
    }

    fn want_bool(&mut self, env: &Env, call: &Call<'_>, index: usize, ty: &Type) {
        if !matches!(ty.unwrapped(), Type::Bool | Type::Any) {
            self.report(
                env.unit,
                mismatch(call.inputs[index].span(), &Type::Bool, ty),
            );
        }
    }

    /// Types a call to a user function, `receiver` being its first argument for a method call.
    #[allow(clippy::too_many_lines)]
    fn user_call(
        &mut self,
        env: &mut Env,
        unit: usize,
        name: &str,
        receiver: Option<(Type, Span)>,
        args: &[Arg],
        span: Span,
    ) -> Type {
        if let Some(from) = env.function.clone() {
            self.record_call(from, (unit, name.to_owned()));
        }
        let Some(signature) = self.program.fns.get(&(unit, name.to_owned())).cloned() else {
            return Type::Any;
        };
        let FnSig { params, ret } = signature;
        let mut optional = false;
        let mut filled = vec![false; params.len()];
        let mut position = 0usize;
        if let Some((ty, receiver_span)) = receiver {
            if let Some((_, param, _)) = params.first() {
                optional |= self.fit_argument(env, &ty, param, receiver_span);
                filled[0] = true;
            }
            position = 1;
        }
        for arg in args {
            let index = if let Some(named) = &arg.name {
                if let Some(index) = params.iter().position(|(param, _, _)| *param == named.name) {
                    index
                } else {
                    let mut diagnostic = Diagnostic::error(
                        "unknown-argument",
                        named.span,
                        Message::with(
                            message!("{0} has no parameter {1}"),
                            [
                                MessageArg::from(name),
                                MessageArg::from(named.name.as_str()),
                            ],
                        ),
                    );
                    if let Some(close) = nearest(
                        &named.name,
                        params.iter().map(|(param, _, _)| param.as_str()),
                    ) {
                        diagnostic = diagnostic.with_fix(super::replace_fix(named.span, close));
                    }
                    self.report(env.unit, diagnostic);
                    self.expr(env, &arg.value, None);
                    continue;
                }
            } else {
                let index = position;
                position += 1;
                index
            };
            let Some((_, param, _)) = params.get(index) else {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "argument-count",
                        arg.value.span,
                        Message::with(
                            message!("{0} takes {1} arguments"),
                            [
                                MessageArg::from(name),
                                MessageArg::from(params.len().to_string()),
                            ],
                        ),
                    ),
                );
                self.expr(env, &arg.value, None);
                continue;
            };
            if filled[index] {
                self.report(
                    env.unit,
                    Diagnostic::error(
                        "argument-twice",
                        arg.value.span,
                        Message::with(
                            message!("{0} is given twice"),
                            [MessageArg::from(params[index].0.as_str())],
                        ),
                    ),
                );
            }
            filled[index] = true;
            let ty = self.expr(env, &arg.value, Some(param));
            optional |= self.fit_argument(env, &ty, param, arg.value.span);
        }
        let missing: Vec<&str> = params
            .iter()
            .zip(&filled)
            .filter(|((_, _, has_default), filled)| !**filled && !*has_default)
            .map(|((param, _, _), _)| param.as_str())
            .collect();
        if !missing.is_empty() {
            self.report(
                env.unit,
                Diagnostic::error(
                    "argument-missing",
                    span,
                    Message::with(
                        message!("{0} needs {1}"),
                        [MessageArg::from(name), MessageArg::from(missing.join(", "))],
                    ),
                ),
            );
        }
        if optional { ret.optional() } else { ret }
    }

    /// Checks an argument against its parameter; true when an optional value goes where a value
    /// is expected, which makes the call none when it is.
    fn fit_argument(&mut self, env: &Env, ty: &Type, param: &Type, span: Span) -> bool {
        if ty.fits(param) {
            return false;
        }
        if ty.is_optional() && ty.unwrapped().fits(param) {
            return true;
        }
        self.report(env.unit, mismatch(span, param, ty));
        false
    }
}

fn input(arg: &Arg) -> Input<'_> {
    match &arg.value.kind {
        ExprKind::Lambda { param, body } => Input::Lambda(param, body, arg.value.span),
        _ => Input::Value(Some(&arg.value), arg.value.span),
    }
}

/// Whether argument `index` of `name` is a unit written as a name (`weight.in(kg)`).
fn unit_argument(name: &str, index: usize, expr: &Expr) -> bool {
    matches!(name, "in" | "value")
        && index == 1
        && matches!(&expr.kind, ExprKind::Name(unit) if unit_kind(unit).is_some())
}

/// The text of a literal without interpolation.
fn literal(expr: &Expr) -> Option<String> {
    match &expr.kind {
        ExprKind::Raw(text) => Some(text.clone()),
        ExprKind::Text(parts) => match parts.as_slice() {
            [crate::lading::syntax::ast::TextPart::Literal(text)] => Some(text.clone()),
            [] => Some(String::new()),
            _ => None,
        },
        _ => None,
    }
}

fn example_unit(dim: Dim) -> &'static str {
    match dim {
        Dim::Weight => "lb",
        Dim::Length => "ft",
        Dim::Volume => "ft3",
        Dim::Temperature => "°F",
    }
}

/// Whether a standard function can be used as a property of a value of this type.
fn takes_no_arguments(name: &str, receiver: Option<&Type>) -> bool {
    let list = receiver.is_some_and(|ty| matches!(ty.unwrapped(), Type::List(_)));
    match name {
        "count" | "first" | "last" | "isEmpty" | "distinct" | "flatten" | "sum" | "min" | "max" => {
            list
        }
        "trim" | "upper" | "lower" | "titleCase" | "digits" | "letters" | "clean" | "length"
        | "isBlank" | "zip5" | "zip9" | "stateCode" | "phone" | "isNumber" | "countryCode"
        | "floor" | "ceil" | "abs" | "round" | "amount" | "currency" | "zone" | "zoneCode"
        | "date" | "time" | "text" | "int" | "decimal" => !list,
        "refs" | "address" | "party" | "elements" => true,
        _ => false,
    }
}
