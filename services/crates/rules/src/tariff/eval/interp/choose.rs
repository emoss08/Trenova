//! `best` and `worst`: running each method and billing the cheapest (or dearest), and
//! `chosen("Label")` in tests.
//!
//! Each option runs in order, in a scope of its own, into a bill of its own, under the run's
//! one meter (so the budgets apply across all options). An option that refuses is out of the
//! running; an evaluation error stops the rating. Of the options that billed, `best` keeps the
//! lowest exact subtotal and `worst` the highest, a tie going to the option written first; its
//! lines and notes join the bill where the choice is written and the others are dropped. When
//! every option refuses, the rating refuses with the first option's refusal.
//!
//! Every evaluation is recorded ([`crate::tariff::eval::bill::Choices`]) with each option's
//! subtotal or refusal, and traced as a `choice` event.

use super::expr::internal;
use super::{Env, Flow, Interp};
use crate::message::Arg;
use crate::span::Span;
use crate::tariff::eval::bill::{Bill, ChoiceRecord, OptionOutcome, OptionRecord};
use crate::tariff::eval::error::Stop;
use crate::tariff::eval::trace::Event;
use crate::tariff::eval::value::Value;
use crate::tariff::syntax::ast::{Arg as CallArg, Choose, ChooseKind};
use rust_decimal::Decimal;

impl<'a> Interp<'_, 'a> {
    /// Runs a `best` or `worst` and bills the option it keeps.
    pub(super) fn choose(
        &mut self,
        env: &mut Env,
        choose: &'a Choose,
        span: Span,
    ) -> Result<Flow, Stop> {
        let slot = self.choices.begin();
        let outer = std::mem::take(&mut self.bill);
        let ran = self.options(env, choose);
        self.bill = outer;
        let (records, mut bills) = ran?;
        let chosen = pick(choose.kind, &records);
        let record = ChoiceRecord {
            label: choose.label.text.clone(),
            kind: choose.kind,
            span,
            chosen,
            options: records,
        };
        if self.traced(env) {
            self.trace.push(Event::Choice {
                label: record.label.clone(),
                chosen: record.chosen_label().map(str::to_owned),
                span,
            });
        }
        let first_refusal = record
            .options
            .iter()
            .find_map(|option| match &option.outcome {
                OptionOutcome::Refused(refusal) => Some(refusal.clone()),
                OptionOutcome::Billed(_) => None,
            });
        self.choices.finish(slot, record);
        match chosen
            .and_then(|index| bills.get_mut(index))
            .and_then(Option::take)
        {
            Some(kept) => {
                self.bill.splice(kept);
                Ok(Flow::Normal)
            }
            None => match first_refusal {
                Some(refusal) => Err(Stop::Refused(refusal)),
                None => Err(internal(
                    crate::message!("A choice needs an option to bill"),
                    [],
                )),
            },
        }
    }

    /// Runs every option of a choice in order, each into a bill of its own: what became of
    /// each, and the bills of those that billed.
    fn options(
        &mut self,
        env: &mut Env,
        choose: &'a Choose,
    ) -> Result<(Vec<OptionRecord>, Vec<Option<Bill>>), Stop> {
        let mut records = Vec::with_capacity(choose.options.len());
        let mut bills = Vec::with_capacity(choose.options.len());
        for option in &choose.options {
            self.cover_in(env, option.span);
            let ran = self.exec_block(env, &option.body);
            let billed = std::mem::take(&mut self.bill);
            let outcome = match ran {
                Ok(Flow::Normal) => {
                    let subtotal = billed
                        .subtotal()
                        .map_err(|error| Stop::Failed(error.at(option.span)))?;
                    bills.push(Some(billed));
                    OptionOutcome::Billed(subtotal)
                }
                Ok(Flow::Break | Flow::Continue | Flow::Return(_)) => {
                    return Err(internal(
                        crate::message!("An option runs to its end: it can't leave its choice"),
                        [],
                    )
                    .at(option.span));
                }
                Err(Stop::Refused(refusal)) => {
                    bills.push(None);
                    OptionOutcome::Refused(refusal)
                }
                Err(failed @ Stop::Failed(_)) => return Err(failed),
            };
            records.push(OptionRecord {
                label: option.label.text.clone(),
                span: option.span,
                outcome,
            });
        }
        Ok((records, bills))
    }

    /// `chosen("Label")` in a test: the label of the option the last evaluation of that choice
    /// kept, or `none` when it was not reached or every option refused.
    pub(super) fn read_chosen(
        &mut self,
        env: &mut Env,
        args: &'a [CallArg],
    ) -> Result<Value, Stop> {
        let Some(arg) = args.first() else {
            return Err(internal(
                crate::message!("chosen needs a choice's label"),
                [],
            ));
        };
        let label = match self.eval(env, &arg.value)? {
            Value::Text(label) => label,
            other => {
                return Err(internal(
                    crate::message!("chosen needs a choice's label, not {0}"),
                    [Arg::from(other.kind())],
                ));
            }
        };
        Ok(self
            .view
            .as_ref()
            .and_then(|view| view.chosen.get(&*label).cloned().flatten())
            .map_or(Value::None, |chosen| Value::text(&chosen)))
    }
}

/// The option a choice keeps: of those that billed, the lowest subtotal for `best` and the
/// highest for `worst`, the first written on a tie.
fn pick(kind: ChooseKind, records: &[OptionRecord]) -> Option<usize> {
    let mut kept: Option<(usize, Decimal)> = None;
    for (index, record) in records.iter().enumerate() {
        let OptionOutcome::Billed(subtotal) = record.outcome else {
            continue;
        };
        let better = kept.is_none_or(|(_, held)| match kind {
            ChooseKind::Best => subtotal < held,
            ChooseKind::Worst => subtotal > held,
        });
        if better {
            kept = Some((index, subtotal));
        }
    }
    kept.map(|(index, _)| index)
}
