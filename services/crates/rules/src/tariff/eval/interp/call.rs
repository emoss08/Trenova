//! Calls: user and library functions, module-qualified functions, methods on values, rate
//! table methods, the conversions `Number(x)` and `Int.parse(text)`, and in tests the record
//! constructors, `charge(…)` and `chosen(…)` (in `choose`).

use super::expr::{internal, unit_argument};
use super::{Env, Flow, Interp};
use crate::message::Arg;
use crate::span::Span;
use crate::tariff::eval::bill::Wanted;
use crate::tariff::eval::error::Stop;
use crate::tariff::eval::lists::{self, Host};
use crate::tariff::eval::record;
use crate::tariff::eval::value::Value;
use crate::tariff::eval::{methods, text};
use crate::tariff::syntax::ast::{Arg as CallArg, Expr, ExprKind, FnBody, FnDecl};
use rust_decimal::Decimal;

/// The record constructors tests may call, with the stop kind each builds.
const STOP_CONSTRUCTORS: [(&str, &str); 4] = [
    ("pickup", "Pickup"),
    ("delivery", "Delivery"),
    ("splitPickup", "SplitPickup"),
    ("splitDelivery", "SplitDelivery"),
];

impl<'a> Interp<'_, 'a> {
    pub(super) fn call(
        &mut self,
        env: &mut Env,
        expr: &'a Expr,
        callee: &'a Expr,
        args: &'a [CallArg],
    ) -> Result<Value, Stop> {
        match &callee.kind {
            ExprKind::Name(name) if env.get(&name.text).is_none() => {
                if let Some((module, decl)) = self.linked.function(env.module, &name.text) {
                    return self.call_function(env, module, decl, args, expr.span);
                }
                if self.view.is_some() && name.text == "charge" {
                    return self.read_charge(env, args);
                }
                if self.view.is_some() && name.text == "chosen" {
                    return self.read_chosen(env, args);
                }
                if self.constructors {
                    if let Some((_, kind)) = STOP_CONSTRUCTORS
                        .iter()
                        .find(|(constructor, _)| *constructor == name.text)
                    {
                        return self.construct(env, record::stop(kind, None), args, true);
                    }
                    if name.text == "commodity" {
                        return self.construct(env, record::commodity(), args, false);
                    }
                }
                Err(internal(
                    crate::message!("Nothing named {0} can be called"),
                    [Arg::from(name.text.clone())],
                ))
            }
            ExprKind::Path { head, member } => self.conversion(
                env,
                &head.text,
                member.as_ref().map(|m| m.text.as_str()),
                args,
            ),
            ExprKind::Field {
                target,
                name,
                optional,
            } => {
                if let ExprKind::Name(alias) = &target.kind
                    && env.get(&alias.text).is_none()
                    && let Some(module) = self.linked.alias(env.module, &alias.text)
                    && let Some((at, decl)) = self.linked.function(module, &name.text)
                {
                    return self.call_function(env, at, decl, args, expr.span);
                }
                if name.text == "charge" && self.is_bill(env, target) {
                    return self.read_bill_charge(env, args);
                }
                let receiver = self.eval(env, target)?;
                if *optional && receiver.is_none() {
                    return Ok(Value::None);
                }
                self.method(env, expr, &receiver, &name.text, args)
            }
            _ => Err(internal(crate::message!("This cannot be called"), [])),
        }
    }

    /// Calls a user or library function: arguments by position, then by name, then defaults.
    /// A default is a compile-time value (literals and constants), so it is evaluated in a
    /// scope of its own, which sees no parameter.
    pub(super) fn call_function(
        &mut self,
        env: &mut Env,
        module: usize,
        decl: &'a FnDecl,
        args: &'a [CallArg],
        span: Span,
    ) -> Result<Value, Stop> {
        let mut given: Vec<Option<Value>> = vec![None; decl.params.len()];
        let mut position = 0;
        for arg in args {
            let value = self.eval(env, &arg.value)?;
            let slot = if let Some(name) = &arg.name {
                decl.params
                    .iter()
                    .position(|param| param.name.text == name.text)
            } else {
                position += 1;
                Some(position - 1)
            };
            match slot.and_then(|slot| given.get_mut(slot)) {
                Some(entry) => *entry = Some(value),
                None => {
                    return Err(internal(
                        crate::message!("{0} has no parameter for this argument"),
                        [Arg::from(decl.name.text.clone())],
                    ));
                }
            }
        }
        let foreign = module != env.module;
        let outermost = foreign && self.anchor.is_none() && env.module == 0;
        if outermost {
            self.anchor = Some(span);
        }
        let result = self.run_function(module, decl, given);
        if outermost {
            self.anchor = None;
        }
        result.map_err(|stop| {
            if foreign {
                stop.moved_to(span)
            } else {
                stop.at(span)
            }
        })
    }

    fn run_function(
        &mut self,
        module: usize,
        decl: &'a FnDecl,
        given: Vec<Option<Value>>,
    ) -> Result<Value, Stop> {
        self.meter.enter().map_err(Stop::Failed)?;
        let mut scope = Env::new(module, true);
        let mut outcome = Ok(());
        for (param, value) in decl.params.iter().zip(given) {
            let value = match (value, &param.default) {
                (Some(value), _) => value,
                (None, Some(default)) => match self.eval(&mut Env::new(module, true), default) {
                    Ok(value) => value,
                    Err(stop) => {
                        outcome = Err(stop);
                        break;
                    }
                },
                (None, None) => {
                    outcome = Err(internal(
                        crate::message!("{0} needs a value for {1}"),
                        [
                            Arg::from(decl.name.text.clone()),
                            Arg::from(param.name.text.clone()),
                        ],
                    ));
                    break;
                }
            };
            scope.bind(&param.name.text, value, false);
        }
        if module == 0 {
            self.cover(match &decl.body {
                FnBody::Expr(body) => body.span,
                FnBody::Block(block) => block.span,
            });
        }
        let result = outcome.and_then(|()| match &decl.body {
            FnBody::Expr(body) => self.eval(&mut scope, body),
            FnBody::Block(block) => match self.exec_block(&mut scope, block)? {
                Flow::Return(value) => Ok(value),
                _ => Err(internal(
                    crate::message!("{0} ended without returning a value"),
                    [Arg::from(decl.name.text.clone())],
                )),
            },
        });
        self.meter.leave();
        result
    }

    /// `Number(x)` and `Int.parse(text)`.
    fn conversion(
        &mut self,
        env: &mut Env,
        head: &str,
        member: Option<&str>,
        args: &'a [CallArg],
    ) -> Result<Value, Stop> {
        let value = match args.first() {
            Some(arg) => self.eval(env, &arg.value)?,
            None => Value::None,
        };
        match (head, member, &value) {
            ("Number", None, Value::Int(number) | Value::Number(number)) => {
                Ok(Value::Number(*number))
            }
            ("Int", Some("parse"), Value::Text(content)) => Ok(parse_int(content)),
            _ => Err(internal(
                crate::message!("{0} cannot be converted like this"),
                [Arg::from(value.kind())],
            )),
        }
    }

    /// A test constructor: the sample stop or commodity with the named fields given.
    fn construct(
        &mut self,
        env: &mut Env,
        base: Value,
        args: &'a [CallArg],
        stop: bool,
    ) -> Result<Value, Stop> {
        let Value::Record(record) = base else {
            return Ok(base);
        };
        let mut built = (*record).clone();
        let mut dwell_given = false;
        for arg in args {
            let Some(name) = &arg.name else {
                return Err(internal(
                    crate::message!("A test constructor takes named arguments only"),
                    [],
                ));
            };
            let value = self.eval(env, &arg.value)?;
            if built.get(&name.text).is_some() {
                dwell_given |= name.text == "dwell";
                built.set(&name.text, value);
                continue;
            }
            match built.get("location") {
                Some(Value::Record(location)) if stop && location.get(&name.text).is_some() => {
                    let mut location = (**location).clone();
                    location.set(&name.text, value);
                    built.set("location", Value::Record(std::rc::Rc::new(location)));
                }
                _ => {
                    return Err(internal(
                        crate::message!("{0} is not a field this constructor sets"),
                        [Arg::from(name.text.clone())],
                    ));
                }
            }
        }
        let value = Value::Record(std::rc::Rc::new(built));
        Ok(if stop {
            record::with_dwell(value, dwell_given)
        } else {
            value
        })
    }

    /// `shipment.requested(CODE)`: whether any of the shipment's accessorials has the code, a
    /// step per accessorial compared.
    fn requested(
        &mut self,
        shipment: &crate::tariff::eval::value::Record,
        args: &'a [CallArg],
    ) -> Result<Value, Stop> {
        let code = match args.first().map(|arg| &arg.value.kind) {
            Some(ExprKind::Name(code)) => code.text.as_str(),
            _ => {
                return Err(internal(
                    crate::message!("requested needs an accessorial charge code"),
                    [],
                ));
            }
        };
        let Some(Value::List(accessorials)) = shipment.get("accessorials") else {
            return Err(internal(
                crate::message!("The shipment has no accessorials to read"),
                [],
            ));
        };
        self.meter.steps(accessorials.len()).map_err(Stop::Failed)?;
        let found = accessorials.iter().any(|accessorial| match accessorial {
            Value::Record(accessorial) => {
                matches!(accessorial.get("code"), Some(Value::Text(text)) if &**text == code)
            }
            _ => false,
        });
        Ok(Value::Bool(found))
    }

    /// `charge(CODE)` or `charge("Label")` in a test: the matching lines' sum, or `none`.
    /// What `charge(…)` or `bill.charge(…)` names: a code written bare, or a label.
    fn wanted(&mut self, env: &mut Env, args: &'a [CallArg]) -> Result<Wanted, Stop> {
        let Some(arg) = args.first() else {
            return Err(internal(
                crate::message!("charge needs a code or a label"),
                [],
            ));
        };
        match &arg.value.kind {
            ExprKind::Name(name)
            | ExprKind::Path {
                head: name,
                member: None,
            } if env.get(&name.text).is_none() => Ok(Wanted::Code(name.text.clone())),
            _ => match self.eval(env, &arg.value)? {
                Value::Text(label) => Ok(Wanted::Label(label.to_string())),
                other => Err(internal(
                    crate::message!("charge needs a code or a label, not {0}"),
                    [Arg::from(other.kind())],
                )),
            },
        }
    }

    /// `charge(CODE)` or `charge("Label")` in a test: the finished bill's lines it names.
    fn read_charge(&mut self, env: &mut Env, args: &'a [CallArg]) -> Result<Value, Stop> {
        let wanted = self.wanted(env, args)?;
        let Some(view) = &self.view else {
            return Ok(Value::None);
        };
        let mut total: Option<Decimal> = None;
        for line in &view.lines {
            let matches = match &wanted {
                Wanted::Code(code) => line.code.as_deref() == Some(code.as_str()),
                Wanted::Label(label) => line.label.as_deref() == Some(label.as_str()),
            };
            if matches {
                total = Some(total.unwrap_or_default().saturating_add(line.amount));
            }
        }
        Ok(total.map_or(Value::None, Value::money))
    }

    /// `bill.charge(CODE)` or `bill.charge("Label")`: the lines billed so far it names, or
    /// `none`.
    fn read_bill_charge(&mut self, env: &mut Env, args: &'a [CallArg]) -> Result<Value, Stop> {
        let wanted = self.wanted(env, args)?;
        let sums = self.sums(Some(&wanted))?;
        Ok(sums.charge.map_or(Value::None, Value::money))
    }

    fn method(
        &mut self,
        env: &mut Env,
        expr: &'a Expr,
        receiver: &Value,
        name: &str,
        args: &'a [CallArg],
    ) -> Result<Value, Stop> {
        match receiver {
            Value::List(items) => {
                let items = std::rc::Rc::clone(items);
                self.list_method(env, expr, &items, name, args)
            }
            Value::Table(code) => {
                let code = code.to_string();
                self.table_method(env, expr, &code, name, args)
            }
            Value::Record(shipment) if name == "requested" => {
                let shipment = std::rc::Rc::clone(shipment);
                self.requested(&shipment, args)
            }
            _ => {
                if name == "in" {
                    let unit = args.first().and_then(|arg| unit_argument(&arg.value));
                    return methods::call(receiver, name, &[], unit.as_ref())
                        .unwrap_or_else(|| Err(no_method(receiver, name)))
                        .map_err(Stop::Failed);
                }
                let values = self.bind_method_args(env, receiver, name, args)?;
                if let Value::Text(content) = receiver {
                    let regex = match (name, values.first()) {
                        ("matches", Some(Value::Text(pattern))) => {
                            Some(self.compile(pattern).map_err(Stop::Failed)?)
                        }
                        _ => None,
                    };
                    return text::call(content, name, &values, &mut self.meter, regex.as_ref())
                        .unwrap_or_else(|| Err(no_method(receiver, name)))
                        .map_err(Stop::Failed);
                }
                methods::call(receiver, name, &values, None)
                    .unwrap_or_else(|| Err(no_method(receiver, name)))
                    .map_err(Stop::Failed)
            }
        }
    }

    /// Evaluates a method's arguments into parameter order.
    fn bind_method_args(
        &mut self,
        env: &mut Env,
        receiver: &Value,
        name: &str,
        args: &'a [CallArg],
    ) -> Result<Vec<Value>, Stop> {
        let params = methods::params(receiver, name);
        let mut values = vec![Value::None; params.len().max(args.len())];
        let mut position = 0;
        for arg in args {
            let value = self.eval(env, &arg.value)?;
            let slot = if let Some(named) = &arg.name {
                params.iter().position(|param| *param == named.text)
            } else {
                position += 1;
                Some(position - 1)
            };
            match slot.and_then(|slot| values.get_mut(slot)) {
                Some(entry) => *entry = value,
                None => {
                    return Err(internal(
                        crate::message!("{0} has no parameter for this argument"),
                        [Arg::from(name)],
                    ));
                }
            }
        }
        while values.last().is_some_and(Value::is_none) && values.len() > args.len() {
            values.pop();
        }
        Ok(values)
    }

    fn table_method(
        &mut self,
        env: &mut Env,
        expr: &'a Expr,
        code: &str,
        name: &str,
        args: &'a [CallArg],
    ) -> Result<Value, Stop> {
        let mut keys = Vec::with_capacity(args.len());
        for arg in args {
            keys.push(self.eval(env, &arg.value)?);
        }
        let key = keys.first().cloned().unwrap_or(Value::None);
        let read = match name {
            "cell" => self.read_table(|tables, meter| tables.cell(code, &keys, meter)),
            "band" => self.read_table(|tables, meter| tables.band(code, &key, meter)),
            "interpolate" => self.read_table(|tables, meter| tables.interpolate(code, &key, meter)),
            _ => {
                return Err(internal(
                    crate::message!("A rate table has no method named {0}"),
                    [Arg::from(name)],
                ));
            }
        }?;
        Ok(self.answer(env, read, expr.span))
    }

    fn list_method(
        &mut self,
        env: &mut Env,
        expr: &'a Expr,
        items: &[Value],
        name: &str,
        args: &'a [CallArg],
    ) -> Result<Value, Stop> {
        let mut lambda = None;
        let mut values = Vec::new();
        for arg in args {
            match &arg.value.kind {
                ExprKind::Lambda { params, body } => lambda = Some((params, &**body)),
                _ => values.push(self.eval(env, &arg.value)?),
            }
        }
        let empty_sum = self
            .static_type(env.module, expr.id)
            .as_ref()
            .and_then(lists::zero_of);
        let mut host = LambdaHost {
            interp: self,
            env,
            lambda,
        };
        lists::call(items, name, &values, empty_sum.as_ref(), &mut host).unwrap_or_else(|| {
            Err(internal(
                crate::message!("A list has no method named {0}"),
                [Arg::from(name)],
            ))
        })
    }
}

/// Runs a list method's lambda in the scope it was written in.
struct LambdaHost<'h, 'p, 'a> {
    interp: &'h mut Interp<'p, 'a>,
    env: &'h mut Env,
    lambda: Option<(&'a Vec<crate::tariff::syntax::ast::Name>, &'a Expr)>,
}

impl Host for LambdaHost<'_, '_, '_> {
    fn has_lambda(&self) -> bool {
        self.lambda.is_some()
    }

    fn apply(&mut self, item: &Value, index: usize) -> Result<Value, Stop> {
        let Some((params, body)) = self.lambda else {
            return Ok(item.clone());
        };
        self.env.push();
        if let Some(first) = params.first() {
            self.env.bind(&first.text, item.clone(), false);
        }
        if let Some(second) = params.get(1) {
            self.env
                .bind(&second.text, Value::Int(Decimal::from(index)), false);
        }
        let result = self.interp.eval(self.env, body);
        self.env.pop();
        result
    }

    fn meter(&mut self) -> &mut crate::tariff::eval::budget::Meter {
        &mut self.interp.meter
    }
}

fn no_method(receiver: &Value, name: &str) -> crate::tariff::eval::error::EvalError {
    crate::tariff::eval::error::EvalError::internal(crate::message::Message::with(
        crate::message!("{0} has no method named {1}"),
        [Arg::from(receiver.kind()), Arg::from(name)],
    ))
}

/// `Int.parse(text)`: a whole number written in plain digits, with an optional sign, or `none`.
fn parse_int(content: &str) -> Value {
    let digits = content.strip_prefix('-').unwrap_or(content);
    if digits.is_empty() || !digits.bytes().all(|byte| byte.is_ascii_digit()) {
        return Value::None;
    }
    content
        .parse::<Decimal>()
        .map_or(Value::None, |number| Value::Int(number.normalize()))
}
