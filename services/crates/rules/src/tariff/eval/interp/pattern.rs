//! `match`: patterns, guards and arms.
//!
//! Arms are tried in order; the first whose pattern matches and whose guard holds runs, with the
//! names its pattern binds in a scope of its own. A literal matches an equal value (across units
//! and scales, as `==`); a range matches as `in` does, its ends literals; a variant matches its
//! enum value; `none` matches `none`; a name matches anything and binds it; a record pattern
//! matches when each named field matches; `A | B` when either does. The checker proves every
//! `match` exhaustive, so a value no arm covers is an internal error.

use super::expr::{in_range, internal};
use super::{Env, Flow, Interp};
use crate::span::Span;
use crate::tariff::eval::error::Stop;
use crate::tariff::eval::trace::Event;
use crate::tariff::eval::value::Value;
use crate::tariff::syntax::ast::{Arm, ArmBody, Pattern, PatternKind};

impl<'a> Interp<'_, 'a> {
    /// Whether `pattern` matches `value`, collecting the names it binds.
    fn matches(
        &mut self,
        env: &mut Env,
        pattern: &'a Pattern,
        value: &Value,
        bound: &mut Vec<(&'a str, Value)>,
    ) -> Result<bool, Stop> {
        match &pattern.kind {
            PatternKind::Wildcard => Ok(true),
            PatternKind::None => Ok(value.is_none()),
            PatternKind::Literal(literal) => {
                let expected = self.eval(env, literal)?;
                if value.is_none() || expected.is_none() {
                    return Ok(value.is_none() && expected.is_none());
                }
                value
                    .equals(&expected)
                    .map_err(|error| Stop::Failed(error.at(pattern.span)))
            }
            PatternKind::Range {
                start,
                end,
                inclusive,
            } => {
                let start = match start {
                    Some(start) => Some(self.eval(env, start)?),
                    None => None,
                };
                let end = match end {
                    Some(end) => Some(self.eval(env, end)?),
                    None => None,
                };
                in_range(value, start.as_ref(), end.as_ref(), *inclusive)
                    .map_err(|error| Stop::Failed(error.at(pattern.span)))
            }
            PatternKind::Variant { variant, .. } => {
                Ok(matches!(value, Value::Enum(name) if **name == *variant.text))
            }
            PatternKind::Binding(name) => {
                bound.push((name.text.as_str(), value.clone()));
                Ok(true)
            }
            PatternKind::Record(fields) => {
                let Value::Record(record) = value else {
                    return Ok(false);
                };
                for (name, inner) in fields {
                    let field = record.get(&name.text).cloned().unwrap_or(Value::None);
                    if !self.matches(env, inner, &field, bound)? {
                        return Ok(false);
                    }
                }
                Ok(true)
            }
            PatternKind::Missing => Err(internal(
                crate::message!("The tariff is incomplete here"),
                [],
            )),
            PatternKind::Or(alternatives) => {
                for alternative in alternatives {
                    let mark = bound.len();
                    if self.matches(env, alternative, value, bound)? {
                        return Ok(true);
                    }
                    bound.truncate(mark);
                }
                Ok(false)
            }
        }
    }

    /// The first arm that matches `value`, with its scope opened and its names bound; the
    /// caller runs the body and closes the scope.
    fn select(
        &mut self,
        env: &mut Env,
        value: &Value,
        arms: &'a [Arm],
    ) -> Result<Option<(usize, &'a Arm)>, Stop> {
        for (index, arm) in arms.iter().enumerate() {
            let mut bound = Vec::new();
            if !self.matches(env, &arm.pattern, value, &mut bound)? {
                continue;
            }
            env.push();
            for (name, value) in bound {
                env.bind(name, value, false);
            }
            let guard = match &arm.guard {
                None => true,
                Some(guard) => match self.eval(env, guard) {
                    Ok(Value::Bool(flag)) => flag,
                    Ok(other) => {
                        env.pop();
                        return Err(internal(
                            crate::message!("A guard must be true or false, not {0}"),
                            [crate::message::Arg::from(other.kind())],
                        ));
                    }
                    Err(stop) => {
                        env.pop();
                        return Err(stop);
                    }
                },
            };
            if guard {
                if self.traced(env) {
                    self.trace.push(Event::Arm {
                        arm: index,
                        span: arm.span,
                    });
                }
                return Ok(Some((index, arm)));
            }
            env.pop();
        }
        Ok(None)
    }

    /// A `match` expression's value.
    pub(super) fn match_value(
        &mut self,
        env: &mut Env,
        value: &Value,
        arms: &'a [Arm],
        span: Span,
    ) -> Result<Value, Stop> {
        let Some((_, arm)) = self.select(env, value, arms)? else {
            return Err(no_arm(value).at(span));
        };
        let result = match &arm.body {
            ArmBody::Expr(body) => self.eval(env, body),
            ArmBody::Block(block) => self.exec_block(env, block).and_then(|flow| match flow {
                Flow::Return(value) => Ok(value),
                _ => Err(internal(
                    crate::message!("A match arm used as a value must give one"),
                    [],
                )),
            }),
        };
        env.pop();
        result
    }

    /// A `match` statement.
    pub(super) fn match_statement(
        &mut self,
        env: &mut Env,
        value: &Value,
        arms: &'a [Arm],
        span: Span,
    ) -> Result<Flow, Stop> {
        let Some((_, arm)) = self.select(env, value, arms)? else {
            return Err(no_arm(value).at(span));
        };
        let result = match &arm.body {
            ArmBody::Expr(body) => self.eval(env, body).map(|_| Flow::Normal),
            ArmBody::Block(block) => self.exec_block(env, block),
        };
        env.pop();
        result
    }
}

fn no_arm(value: &Value) -> Stop {
    internal(
        crate::message!("No arm of this match covers {0}"),
        [crate::message::Arg::from(value.display())],
    )
}
