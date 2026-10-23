//! Statements, and running a file's top level.

use super::expr::internal;
use super::{Env, Flow, Interp, refusal};
use crate::message::Arg;
use crate::span::Span;
use crate::tariff::eval::bill::{Billed, Line, LineKind, Note, cents};
use crate::tariff::eval::error::{EvalError, Stop};
use crate::tariff::eval::operators;
use crate::tariff::eval::trace::{Event, MAX_ITERATIONS};
use crate::tariff::eval::value::Value;
use crate::tariff::syntax::ast::{
    AssignOp, Block, Charge, Expr, ExprKind, ForStmt, IfStmt, ItemKind, RoundTo, Stmt, StmtKind,
    TotalClause,
};
use rust_decimal::Decimal;
use rust_decimal::prelude::ToPrimitive;

impl<'a> Interp<'_, 'a> {
    /// Runs every file-level statement of a module, top to bottom, in `env`.
    pub fn run_file(&mut self, env: &mut Env) -> Result<(), Stop> {
        let Some(module) = self.linked.module(env.module) else {
            return Ok(());
        };
        let file = &module.file;
        for item in &file.items {
            if let ItemKind::Stmt(stmt) = &item.kind {
                match self.exec(env, stmt)? {
                    Flow::Normal => {}
                    Flow::Break | Flow::Continue | Flow::Return(_) => {
                        return Err(internal(
                            crate::message!(
                                "break, continue and return only work inside loops and functions"
                            ),
                            [],
                        )
                        .at(stmt.span));
                    }
                }
            }
        }
        Ok(())
    }

    /// Runs a block in its own scope.
    pub(super) fn exec_block(&mut self, env: &mut Env, block: &'a Block) -> Result<Flow, Stop> {
        env.push();
        let result = self.exec_all(env, &block.stmts);
        env.pop();
        result
    }

    /// Runs statements in the current scope, stopping at the first that does not end normally.
    pub fn exec_all(&mut self, env: &mut Env, stmts: &'a [Stmt]) -> Result<Flow, Stop> {
        for stmt in stmts {
            let flow = self.exec(env, stmt)?;
            if !matches!(flow, Flow::Normal) {
                return Ok(flow);
            }
        }
        Ok(Flow::Normal)
    }

    /// Runs one statement.
    pub fn exec(&mut self, env: &mut Env, stmt: &'a Stmt) -> Result<Flow, Stop> {
        self.meter
            .step()
            .map_err(|error| Stop::Failed(error.at(stmt.span)))?;
        self.exec_kind(env, stmt).map_err(|stop| stop.at(stmt.span))
    }

    fn exec_kind(&mut self, env: &mut Env, stmt: &'a Stmt) -> Result<Flow, Stop> {
        match &stmt.kind {
            StmtKind::Choose(choose) => self.choose(env, choose, stmt.span),
            StmtKind::Vary { .. } | StmtKind::Assume(_) => Err(Stop::Failed(
                crate::tariff::eval::error::EvalError::internal(crate::message::Message::new(
                    crate::message!("Phase 0 placeholder: this construct is not implemented yet"),
                )),
            )),
            StmtKind::Let(binding) | StmtKind::Var(binding) => {
                let value = self.eval(env, &binding.value)?;
                if self.traced(env) {
                    self.trace.push(Event::Bind {
                        name: binding.name.text.clone(),
                        value: value.clone(),
                        span: stmt.span,
                    });
                }
                env.bind(
                    &binding.name.text,
                    value,
                    matches!(stmt.kind, StmtKind::Var(_)),
                );
                Ok(Flow::Normal)
            }
            StmtKind::Assign { target, op, value } => {
                self.assign(env, &target.text, *op, value, stmt.span)?;
                Ok(Flow::Normal)
            }
            StmtKind::Charge(charge) => {
                self.charge(env, charge, stmt.span)?;
                Ok(Flow::Normal)
            }
            StmtKind::Note(text) => {
                let text = self.interpolate(env, text)?;
                if self.traced(env) {
                    self.trace.push(Event::Note {
                        text: text.clone(),
                        span: stmt.span,
                    });
                }
                self.bill.note(Note {
                    text,
                    span: stmt.span,
                });
                Ok(Flow::Normal)
            }
            StmtKind::If(statement) => self.if_statement(env, statement, stmt.span),
            StmtKind::Match(statement) => {
                let value = self.eval(env, &statement.scrutinee)?;
                self.match_statement(env, &value, &statement.arms, statement.span)
            }
            StmtKind::For(statement) => self.for_statement(env, statement, stmt.span),
            StmtKind::Require { condition, message } => {
                let holds = self.condition(env, condition)?;
                if holds {
                    return Ok(Flow::Normal);
                }
                let text = self.interpolate(env, message)?;
                Err(self.refuse(env, text, stmt.span))
            }
            StmtKind::Fail(message) => {
                let text = self.interpolate(env, message)?;
                Err(self.refuse(env, text, stmt.span))
            }
            StmtKind::Return(value) => Ok(Flow::Return(self.eval(env, value)?)),
            StmtKind::Break => Ok(Flow::Break),
            StmtKind::Continue => Ok(Flow::Continue),
            StmtKind::Total(clause) => {
                self.total_clause(env, clause)?;
                Ok(Flow::Normal)
            }
            StmtKind::Given { .. } | StmtKind::Expect(_) => Err(internal(
                crate::message!("given and expect only work inside a test"),
                [],
            )),
            StmtKind::Item(_) => Err(internal(
                crate::message!("A declaration cannot be written inside a block"),
                [],
            )),
            StmtKind::Expr(expr) => {
                let value = self.eval(env, expr)?;
                if env.module == 0 && !env.function {
                    self.last_value = Some(value);
                }
                Ok(Flow::Normal)
            }
        }
    }

    fn refuse(&mut self, env: &Env, text: String, span: Span) -> Stop {
        let span = self.root_span(env, span);
        if self.trace.enabled() {
            self.trace.push(Event::Refusal {
                message: text.clone(),
                span,
            });
        }
        refusal(text, span)
    }

    fn assign(
        &mut self,
        env: &mut Env,
        name: &str,
        op: AssignOp,
        value: &'a Expr,
        span: Span,
    ) -> Result<(), Stop> {
        let right = self.eval(env, value)?;
        let traced = self.traced(env);
        let Some(slot) = env.slot_mut(name) else {
            return Err(internal(
                crate::message!("Nothing named {0} is in scope here"),
                [Arg::from(name)],
            ));
        };
        if !slot.mutable {
            return Err(internal(
                crate::message!("{0} is not a var, so it cannot change"),
                [Arg::from(name)],
            ));
        }
        let next = match op {
            AssignOp::Set => right,
            AssignOp::Add => operators::add(&slot.value, &right).map_err(Stop::Failed)?,
            AssignOp::Sub => operators::sub(&slot.value, &right).map_err(Stop::Failed)?,
        };
        slot.value = next.clone();
        if traced {
            self.trace.push(Event::Assign {
                name: name.to_owned(),
                value: next,
                span,
            });
        }
        Ok(())
    }

    fn money(value: &Value, what: &'static str) -> Result<Decimal, Stop> {
        value.as_money().ok_or_else(|| {
            internal(
                crate::message!("{0} must be money, not {1}"),
                [Arg::from(what), Arg::from(value.kind())],
            )
        })
    }

    fn charge(&mut self, env: &mut Env, charge: &'a Charge, span: Span) -> Result<(), Stop> {
        let label = self.interpolate(env, &charge.label)?;
        let mut quantity = None;
        let mut rate = None;
        for option in &charge.options {
            let value = self.eval(env, &option.value)?;
            match option.name.as_ref().map(|name| name.text.as_str()) {
                Some("quantity") => quantity = Some(value),
                Some("rate") => rate = Some(value),
                _ => {
                    return Err(internal(
                        crate::message!("A charge takes only quantity and rate options"),
                        [],
                    ));
                }
            }
        }
        let amount = match (&charge.amount, &quantity, &rate) {
            (Some(amount), _, _) => self.eval(env, amount)?,
            (None, Some(quantity), Some(rate)) => {
                operators::mul(rate, quantity, true).map_err(Stop::Failed)?
            }
            _ => {
                return Err(internal(
                    crate::message!("A charge needs an amount, or both a quantity and a rate"),
                    [],
                ));
            }
        };
        let amount = Self::money(&amount, "A charge's amount")?;
        let code = charge.code.as_ref().map(|code| code.text.clone());
        let line = Line {
            kind: LineKind::Charge,
            label: Some(label.clone()),
            code: code.clone(),
            amount,
            quantity,
            rate,
            span: Some(span),
        };
        let billed = self
            .bill
            .charge(line)
            .map_err(|error| Stop::Failed(error.at(span)))?;
        if self.trace.enabled() {
            self.trace.push(match billed {
                Billed::Added => Event::Charge {
                    label,
                    code,
                    amount,
                    span,
                },
                Billed::Dropped => Event::Dropped {
                    label,
                    code,
                    span: Some(span),
                },
            });
        }
        Ok(())
    }

    fn total_clause(&mut self, env: &mut Env, clause: &'a TotalClause) -> Result<(), Stop> {
        match clause {
            TotalClause::Minimum(amount) => {
                let value = self.eval(env, amount)?;
                self.clauses.minimum = Some(Self::money(&value, "minimum")?);
            }
            TotalClause::Maximum(amount) => {
                let value = self.eval(env, amount)?;
                self.clauses.maximum = Some(Self::money(&value, "maximum")?);
            }
            TotalClause::Round { mode, to } => {
                let increment = match to {
                    RoundTo::Cents => cents(),
                    RoundTo::Amount(amount) => {
                        let value = self.eval(env, amount)?;
                        Self::money(&value, "round")?
                    }
                };
                self.clauses.round = Some((*mode, increment));
            }
        }
        Ok(())
    }

    fn if_statement(
        &mut self,
        env: &mut Env,
        statement: &'a IfStmt,
        span: Span,
    ) -> Result<Flow, Stop> {
        for (index, (condition, block)) in statement.branches.iter().enumerate() {
            env.push();
            let holds = match self.condition(env, condition) {
                Ok(holds) => holds,
                Err(stop) => {
                    env.pop();
                    return Err(stop);
                }
            };
            if holds {
                self.branch(env, Some(index), span);
                let result = self.exec_block(env, block);
                env.pop();
                return result;
            }
            env.pop();
        }
        if let Some(block) = &statement.otherwise {
            self.branch(env, Some(statement.branches.len()), span);
            self.exec_block(env, block)
        } else {
            self.branch(env, None, span);
            Ok(Flow::Normal)
        }
    }

    fn branch(&mut self, env: &Env, taken: Option<usize>, span: Span) {
        if self.traced(env) {
            self.trace.push(Event::Branch { taken, span });
        }
    }

    /// The values a `for` walks: a list, or an `Int` range written in place.
    fn iterable(&mut self, env: &mut Env, iterable: &'a Expr) -> Result<Walk, Stop> {
        if let ExprKind::Range {
            start,
            end,
            inclusive,
        } = &iterable.kind
        {
            let (Some(start), Some(end)) = (start, end) else {
                return Err(internal(
                    crate::message!("A for loop needs a range with both ends"),
                    [],
                ));
            };
            let first = self.eval(env, start)?;
            let last = self.eval(env, end)?;
            let (Value::Int(first), Value::Int(last)) = (&first, &last) else {
                return Err(internal(
                    crate::message!("A for loop walks a range of whole numbers"),
                    [],
                ));
            };
            let stop = if *inclusive {
                last.checked_add(Decimal::ONE)
                    .ok_or_else(EvalError::overflow)?
            } else {
                *last
            };
            return Ok(Walk::Range(
                first.to_i128().unwrap_or(0),
                stop.to_i128().unwrap_or(0),
            ));
        }
        match self.eval(env, iterable)? {
            Value::List(items) => Ok(Walk::List(items)),
            other => Err(internal(
                crate::message!("A for loop walks a list or a range, not {0}"),
                [Arg::from(other.kind())],
            )),
        }
    }

    fn for_statement(
        &mut self,
        env: &mut Env,
        statement: &'a ForStmt,
        span: Span,
    ) -> Result<Flow, Stop> {
        let walk = self.iterable(env, &statement.iterable)?;
        let name = statement.binding.text.as_str();
        let traced = self.traced(env);
        let mut position = 0usize;
        let mut ran = 0usize;
        while let Some(item) = walk.get(position) {
            position += 1;
            self.meter
                .step()
                .map_err(|error| Stop::Failed(error.at(span)))?;
            env.push();
            env.bind(name, item.clone(), false);
            let keep = match &statement.filter {
                None => Ok(true),
                Some(filter) => self.truth(env, filter),
            };
            let flow = match keep {
                Ok(true) => {
                    if traced && ran < MAX_ITERATIONS {
                        self.trace.push(Event::Iteration {
                            binding: name.to_owned(),
                            index: ran,
                            value: item,
                            span,
                        });
                    }
                    ran += 1;
                    self.exec_block(env, &statement.body)
                }
                Ok(false) => Ok(Flow::Continue),
                Err(stop) => Err(stop),
            };
            env.pop();
            match flow? {
                Flow::Normal | Flow::Continue => {}
                Flow::Break => break,
                returned @ Flow::Return(_) => return Ok(returned),
            }
        }
        if traced {
            self.trace.push(Event::Loop {
                binding: name.to_owned(),
                iterations: ran,
                span,
            });
        }
        Ok(Flow::Normal)
    }
}

/// What a `for` walks.
enum Walk {
    List(std::rc::Rc<Vec<Value>>),
    Range(i128, i128),
}

impl Walk {
    fn get(&self, position: usize) -> Option<Value> {
        match self {
            Self::List(items) => items.get(position).cloned(),
            Self::Range(start, stop) => {
                let at = start.checked_add(i128::try_from(position).ok()?)?;
                (at < *stop).then(|| Value::Int(Decimal::from_i128_with_scale(at, 0)))
            }
        }
    }
}
