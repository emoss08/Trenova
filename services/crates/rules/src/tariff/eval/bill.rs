//! The bill: charge lines in order, notes, review reasons, and the total clauses that finish
//! it.
//!
//! A `charge` adds a [`Line`] with its label, code, exact amount, and the quantity and rate when
//! they were given. A charge's `minimum:` and `maximum:` options limit its amount before it is
//! added ([`limit`]): the line bills the limited amount and records the [`Limited`] amount it
//! replaced. Uncoded lines are freight; coded lines are accessorials. A line whose amount is
//! exactly zero is dropped (the trace records it). A coded line with a negative amount is an
//! error naming the code; an uncoded line may be negative (a discount).
//!
//! [`Bill::finish`] applies the total clauses, in the order the language fixes, each comparing
//! exact amounts:
//!
//! 1. `minimum m on freight`: when the freight lines fall short of `m`, a positive
//!    [`LineKind::Minimum`] line on freight makes up the difference;
//! 2. `maximum m on freight`: when they exceed `m`, a negative [`LineKind::Maximum`] line on
//!    freight brings them down;
//! 3. `minimum m`: when the total (the freight adjustments included) falls short of `m`, a
//!    positive [`LineKind::Minimum`] line makes up the difference;
//! 4. `maximum m`: when the total exceeds `m`, a negative [`LineKind::Maximum`] line brings it
//!    down;
//! 5. `round`: every line is rounded to cents with the clause's mode (half up when no `round`
//!    is written), the total is rounded to the clause's increment (cents or `$X`) with the same
//!    mode, and any difference between the rounded total and the rounded lines' sum is a
//!    [`LineKind::Rounding`] line, so the lines always add up to the total. A line that rounds
//!    to zero is dropped like a zero charge.
//!
//! Adjustment lines carry their kind and no label: the hosts label them in the reader's
//! language. They are uncoded, so they count as freight. A bill whose freight lines sum below
//! zero, or whose total is below zero, is a rating error.
//!
//! A `best` or `worst` runs each option into a bill of its own, compares their
//! [`Bill::subtotal`]s and [`Bill::splice`]s the kept one into the bill; [`Choices`] records
//! every evaluation for the result's `choices` and for `chosen(…)` in tests. While an option
//! runs, `bill` reads the bills outside it and its own ([`Sums`]).

use super::error::{Code, EvalError, Refusal};
use super::value::{Value, money_digits};
use crate::tariff::syntax::ast::{ChooseKind, RoundMode};
use rust_decimal::{Decimal, RoundingStrategy};
use std::collections::BTreeMap;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

/// What a line is.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum LineKind {
    /// A `charge` the tariff billed.
    Charge,
    /// The `minimum` clause's adjustment.
    Minimum,
    /// The `maximum` clause's adjustment.
    Maximum,
    /// The rounding remainder.
    Rounding,
}

impl LineKind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Charge => "charge",
            Self::Minimum => "minimum",
            Self::Maximum => "maximum",
            Self::Rounding => "rounding",
        }
    }

    /// The `LineKind` enum variant tests read through `lines`.
    #[must_use]
    pub const fn variant(self) -> &'static str {
        match self {
            Self::Charge => "Charge",
            Self::Minimum => "Minimum",
            Self::Maximum => "Maximum",
            Self::Rounding => "Rounding",
        }
    }
}

/// Which of a charge's limits applied.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum LimitKind {
    /// `minimum:` raised the amount.
    Minimum,
    /// `maximum:` lowered the amount.
    Maximum,
}

impl LimitKind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Minimum => "minimum",
            Self::Maximum => "maximum",
        }
    }
}

/// A charge's limit that changed its amount: which, and the amount it replaced.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Limited {
    /// Which limit applied.
    pub kind: LimitKind,
    /// The amount worked out before the limit: exact while charges are added, rounded like the
    /// line by [`Bill::finish`].
    pub computed: Decimal,
}

/// An amount with a charge's `minimum:` and `maximum:` applied, in that order, and the limit
/// that changed it, if one did.
///
/// # Errors
///
/// When the minimum is above the maximum.
pub fn limit(
    amount: Decimal,
    minimum: Option<Decimal>,
    maximum: Option<Decimal>,
) -> Result<(Decimal, Option<Limited>), EvalError> {
    if let (Some(low), Some(high)) = (minimum, maximum)
        && low > high
    {
        return Err(EvalError::with(
            Code::Argument,
            trenova_module_kit::message!("The charge's minimum {0} is above its maximum {1}"),
            [
                Arg::from(Value::money(low).display()),
                Arg::from(Value::money(high).display()),
            ],
        ));
    }
    if let Some(low) = minimum
        && amount < low
    {
        return Ok((
            low,
            Some(Limited {
                kind: LimitKind::Minimum,
                computed: amount,
            }),
        ));
    }
    if let Some(high) = maximum
        && amount > high
    {
        return Ok((
            high,
            Some(Limited {
                kind: LimitKind::Maximum,
                computed: amount,
            }),
        ));
    }
    Ok((amount, None))
}

/// One line of the bill.
#[derive(Debug, Clone)]
pub struct Line {
    /// What it is.
    pub kind: LineKind,
    /// The interpolated label; `None` for adjustments.
    pub label: Option<String>,
    /// The accessorial code; `None` for freight and adjustments.
    pub code: Option<String>,
    /// The amount in the rating currency: exact while charges are added, rounded by
    /// [`Bill::finish`].
    pub amount: Decimal,
    /// The `quantity:` option, when given.
    pub quantity: Option<Value>,
    /// The `rate:` option, when given.
    pub rate: Option<Value>,
    /// The charge's limit, when one changed its amount.
    pub limit: Option<Limited>,
    /// Whether the line is the adjustment of a total clause `on freight`.
    pub on_freight: bool,
    /// The `charge` statement, in the root file; `None` for adjustments.
    pub span: Option<Span>,
}

impl Line {
    /// Whether the line is freight (uncoded).
    #[must_use]
    pub const fn freight(&self) -> bool {
        self.code.is_none()
    }
}

/// A `note`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Note {
    /// The interpolated text.
    pub text: String,
    /// The statement.
    pub span: Span,
}

/// The total clauses a tariff wrote.
#[derive(Debug, Clone, Default)]
pub struct Clauses {
    /// `minimum m on freight`, in the rating currency.
    pub freight_minimum: Option<Decimal>,
    /// `maximum m on freight`.
    pub freight_maximum: Option<Decimal>,
    /// `minimum m`, in the rating currency.
    pub minimum: Option<Decimal>,
    /// `maximum m`.
    pub maximum: Option<Decimal>,
    /// `round …`: the mode and the increment the total rounds to (`0.01` for cents).
    pub round: Option<(RoundMode, Decimal)>,
}

/// A bill being built.
#[derive(Debug, Clone, Default)]
pub struct Bill {
    /// The lines, in the order they were billed.
    pub lines: Vec<Line>,
    /// The notes, in order.
    pub notes: Vec<Note>,
    /// The `review` reasons, in order, shaped like notes.
    pub reviews: Vec<Note>,
}

/// A finished bill.
#[derive(Debug, Clone)]
pub struct Finished {
    /// The lines, rounded, adjustments last.
    pub lines: Vec<Line>,
    /// The notes.
    pub notes: Vec<Note>,
    /// The `review` reasons: a bill with any is held for approval.
    pub reviews: Vec<Note>,
    /// Lines that came out zero and were dropped, for the trace.
    pub dropped: Vec<Line>,
    /// The uncoded lines' sum.
    pub freight_total: Decimal,
    /// The coded lines' sum.
    pub accessorial_total: Decimal,
    /// Every line's sum.
    pub total: Decimal,
}

/// What became of a charge.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Billed {
    /// It was added.
    Added,
    /// Its amount was zero, so it was dropped.
    Dropped,
}

/// One cent.
#[must_use]
pub fn cents() -> Decimal {
    Decimal::new(1, 2)
}

/// `amount` rounded to a multiple of `increment` with `mode`.
///
/// # Errors
///
/// When the increment is not positive, or the result overflows.
pub fn round_to(
    amount: Decimal,
    increment: Decimal,
    mode: RoundMode,
) -> Result<Decimal, EvalError> {
    if increment <= Decimal::ZERO {
        return Err(EvalError::with(
            Code::Argument,
            trenova_module_kit::message!("A rounding increment must be more than zero, not {0}"),
            [Arg::from(format!("${}", money_digits(increment)))],
        ));
    }
    let strategy = match mode {
        RoundMode::HalfUp => RoundingStrategy::MidpointAwayFromZero,
        RoundMode::HalfEven => RoundingStrategy::MidpointNearestEven,
        RoundMode::Up => RoundingStrategy::AwayFromZero,
        RoundMode::Down => RoundingStrategy::ToZero,
    };
    let steps = amount
        .checked_div(increment)
        .ok_or_else(EvalError::overflow)?
        .round_dp_with_strategy(0, strategy);
    steps
        .checked_mul(increment)
        .map(|value| value.normalize())
        .ok_or_else(EvalError::overflow)
}

fn sum<'a>(mut lines: impl Iterator<Item = &'a Line>) -> Result<Decimal, EvalError> {
    lines.try_fold(Decimal::ZERO, |total, line| {
        total
            .checked_add(line.amount)
            .ok_or_else(EvalError::overflow)
    })
}

fn adjustment(kind: LineKind, amount: Decimal, on_freight: bool) -> Line {
    Line {
        kind,
        label: None,
        code: None,
        amount,
        quantity: None,
        rate: None,
        limit: None,
        on_freight,
        span: None,
    }
}

/// A minimum and a maximum applied to `current`, the sum they bound: the adjustment lines they
/// add and the sum after them.
fn bound(
    lines: &mut Vec<Line>,
    current: Decimal,
    minimum: Option<Decimal>,
    maximum: Option<Decimal>,
    on_freight: bool,
) -> Result<Decimal, EvalError> {
    let mut current = current;
    if let Some(minimum) = minimum
        && current < minimum
    {
        let difference = minimum
            .checked_sub(current)
            .ok_or_else(EvalError::overflow)?;
        lines.push(adjustment(LineKind::Minimum, difference, on_freight));
        current = minimum;
    }
    if let Some(maximum) = maximum
        && current > maximum
    {
        let difference = maximum
            .checked_sub(current)
            .ok_or_else(EvalError::overflow)?;
        lines.push(adjustment(LineKind::Maximum, difference, on_freight));
        current = maximum;
    }
    Ok(current)
}

impl Bill {
    /// Adds a charge line.
    ///
    /// # Errors
    ///
    /// When a coded charge is negative.
    pub fn charge(&mut self, line: Line) -> Result<Billed, EvalError> {
        if line.amount.is_zero() {
            return Ok(Billed::Dropped);
        }
        if let Some(code) = &line.code
            && line.amount.is_sign_negative()
        {
            return Err(EvalError::new(
                Code::NegativeCharge,
                Message::with(
                    trenova_module_kit::message!(
                        "The {0} charge came out at {1}: a coded charge must be more than zero"
                    ),
                    [
                        Arg::from(code.clone()),
                        Arg::from(Value::money(line.amount).display()),
                    ],
                ),
            ));
        }
        self.lines.push(line);
        Ok(Billed::Added)
    }

    /// Adds a note.
    pub fn note(&mut self, note: Note) {
        self.notes.push(note);
    }

    /// Adds a `review` reason.
    pub fn review(&mut self, review: Note) {
        self.reviews.push(review);
    }

    /// Applies the total clauses and checks the result.
    ///
    /// # Errors
    ///
    /// When the freight lines or the total come out below zero, a rounding increment is not
    /// positive, or a sum overflows.
    pub fn finish(self, clauses: &Clauses) -> Result<Finished, EvalError> {
        let mut lines = self.lines;
        let freight = sum(lines.iter().filter(|line| line.freight()))?;
        let accessorials = sum(lines.iter().filter(|line| !line.freight()))?;
        let freight = bound(
            &mut lines,
            freight,
            clauses.freight_minimum,
            clauses.freight_maximum,
            true,
        )?;
        let total = freight
            .checked_add(accessorials)
            .ok_or_else(EvalError::overflow)?;
        let total = bound(&mut lines, total, clauses.minimum, clauses.maximum, false)?;
        let (mode, increment) = clauses.round.unwrap_or((RoundMode::HalfUp, cents()));
        let mut kept = Vec::with_capacity(lines.len() + 1);
        let mut dropped = Vec::new();
        for mut line in lines {
            line.amount = round_to(line.amount, cents(), mode)?;
            if let Some(limited) = &mut line.limit {
                limited.computed = round_to(limited.computed, cents(), mode)?;
            }
            if line.amount.is_zero() {
                dropped.push(line);
            } else {
                kept.push(line);
            }
        }
        let rounded_total = round_to(total, increment, mode)?;
        let remainder = rounded_total
            .checked_sub(sum(kept.iter())?)
            .ok_or_else(EvalError::overflow)?;
        if !remainder.is_zero() {
            kept.push(adjustment(LineKind::Rounding, remainder, false));
        }
        let freight_total = sum(kept.iter().filter(|line| line.freight()))?;
        let accessorial_total = sum(kept.iter().filter(|line| !line.freight()))?;
        if freight_total.is_sign_negative() && !freight_total.is_zero() {
            return Err(below_zero(
                trenova_module_kit::message!(
                    "The freight lines add up to {0}: freight cannot be below zero"
                ),
                freight_total,
            ));
        }
        if rounded_total.is_sign_negative() && !rounded_total.is_zero() {
            return Err(below_zero(
                trenova_module_kit::message!(
                    "The bill's total is {0}: a total cannot be below zero"
                ),
                rounded_total,
            ));
        }
        Ok(Finished {
            lines: kept,
            notes: self.notes,
            reviews: self.reviews,
            dropped,
            freight_total,
            accessorial_total,
            total: rounded_total,
        })
    }
}

impl Bill {
    /// The exact sum of the lines billed so far: a choice's option is compared by this.
    ///
    /// # Errors
    ///
    /// When the sum overflows.
    pub fn subtotal(&self) -> Result<Decimal, EvalError> {
        sum(self.lines.iter())
    }

    /// Appends another bill's lines, notes and reviews, in their order: a choice's kept option
    /// joining the bill where the choice is written.
    pub fn splice(&mut self, other: Self) {
        self.lines.extend(other.lines);
        self.notes.extend(other.notes);
        self.reviews.extend(other.reviews);
    }

    /// Adds this bill's lines to `sums`.
    ///
    /// # Errors
    ///
    /// When a sum overflows.
    pub fn add_to(&self, sums: &mut Sums, wanted: Option<&Wanted>) -> Result<(), EvalError> {
        for line in &self.lines {
            let side = if line.freight() {
                &mut sums.freight
            } else {
                &mut sums.accessorials
            };
            *side = side
                .checked_add(line.amount)
                .ok_or_else(EvalError::overflow)?;
            let matches = match wanted {
                Some(Wanted::Code(code)) => line.code.as_deref() == Some(code.as_str()),
                Some(Wanted::Label(label)) => line.label.as_deref() == Some(label.as_str()),
                None => false,
            };
            if matches {
                sums.charge = Some(
                    sums.charge
                        .unwrap_or_default()
                        .checked_add(line.amount)
                        .ok_or_else(EvalError::overflow)?,
                );
            }
        }
        Ok(())
    }
}

/// What `bill.charge(…)` names.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Wanted {
    /// A charge code.
    Code(String),
    /// A label, as interpolated.
    Label(String),
}

/// The bill so far, as `bill` reads it: the exact sums of the lines billed so far.
#[derive(Debug, Clone, Copy, Default)]
pub struct Sums {
    /// The uncoded lines.
    pub freight: Decimal,
    /// The coded lines.
    pub accessorials: Decimal,
    /// The lines `bill.charge(…)` named, or `None` when there are none.
    pub charge: Option<Decimal>,
}

impl Sums {
    /// Every line.
    ///
    /// # Errors
    ///
    /// When the sum overflows.
    pub fn total(&self) -> Result<Decimal, EvalError> {
        self.freight
            .checked_add(self.accessorials)
            .ok_or_else(EvalError::overflow)
    }
}

/// The most choice evaluations a result reports.
pub const MAX_CHOICES: usize = 200;

/// What became of one option of a choice.
#[derive(Debug, Clone)]
pub enum OptionOutcome {
    /// It billed lines adding up to this exact subtotal.
    Billed(Decimal),
    /// It refused, so it was out of the running.
    Refused(Refusal),
}

/// One option of an evaluated choice.
#[derive(Debug, Clone)]
pub struct OptionRecord {
    /// The option's label.
    pub label: String,
    /// The `option` block.
    pub span: Span,
    /// Its subtotal or its refusal.
    pub outcome: OptionOutcome,
}

/// One evaluation of a `best` or `worst`.
#[derive(Debug, Clone)]
pub struct ChoiceRecord {
    /// The choice's label.
    pub label: String,
    /// `best` or `worst`.
    pub kind: ChooseKind,
    /// The statement.
    pub span: Span,
    /// The index of the option kept, or `None` when every option refused.
    pub chosen: Option<usize>,
    /// Every option, in the order written.
    pub options: Vec<OptionRecord>,
}

impl ChoiceRecord {
    /// The kept option's label.
    #[must_use]
    pub fn chosen_label(&self) -> Option<&str> {
        self.chosen
            .and_then(|index| self.options.get(index))
            .map(|option| option.label.as_str())
    }
}

/// The choices a run evaluated: the first [`MAX_CHOICES`] in the order their evaluation began
/// (a choice inside a loop once per iteration, an outer choice before the ones nested in its
/// options), and the label each choice kept the last time it was evaluated, for `chosen(…)`.
#[derive(Debug, Clone, Default)]
pub struct Choices {
    records: Vec<Option<ChoiceRecord>>,
    last: BTreeMap<String, Option<String>>,
}

impl Choices {
    /// Holds a place for a choice whose evaluation begins, while there is room.
    pub fn begin(&mut self) -> Option<usize> {
        if self.records.len() >= MAX_CHOICES {
            return None;
        }
        self.records.push(None);
        Some(self.records.len() - 1)
    }

    /// Records a finished choice in the place [`Choices::begin`] held for it.
    pub fn finish(&mut self, slot: Option<usize>, record: ChoiceRecord) {
        self.last.insert(
            record.label.clone(),
            record.chosen_label().map(str::to_owned),
        );
        if let Some(place) = slot.and_then(|slot| self.records.get_mut(slot)) {
            *place = Some(record);
        }
    }

    /// The finished choices, in order; a choice an evaluation error interrupted is left out.
    pub fn records(&self) -> impl Iterator<Item = &ChoiceRecord> {
        self.records.iter().flatten()
    }

    /// The label each choice kept the last time it was evaluated (`None` when every option
    /// refused), by the choice's label.
    #[must_use]
    pub const fn last(&self) -> &BTreeMap<String, Option<String>> {
        &self.last
    }
}

/// A finished bill in the billing currency: each line multiplied by the exchange rate exactly
/// and rounded to cents with the bill's round mode, the total multiplied and rounded with the
/// bill's mode and increment, and any difference between the two added to the rounding line's
/// converted amount, or appended as one more entry when the bill has no rounding line, so the
/// converted lines always add up to the converted total.
#[derive(Debug, Clone)]
pub struct Converted {
    /// The converted amount of each line, in the bill's order, plus a trailing rounding entry
    /// when the bill had no rounding line and conversion needed one.
    pub lines: Vec<Decimal>,
    /// The converted total.
    pub total: Decimal,
}

impl Finished {
    /// The bill in another currency at `rate` (units of that currency per unit of the rating
    /// currency).
    ///
    /// # Errors
    ///
    /// When the rate is not above zero or a product overflows.
    pub fn convert(&self, rate: Decimal, clauses: &Clauses) -> Result<Converted, EvalError> {
        if rate <= Decimal::ZERO {
            return Err(EvalError::with(
                Code::Argument,
                trenova_module_kit::message!("An exchange rate must be more than zero, not {0}"),
                [Arg::from(super::value::format_decimal(rate))],
            ));
        }
        let (mode, increment) = clauses.round.unwrap_or((RoundMode::HalfUp, cents()));
        let mut lines = Vec::with_capacity(self.lines.len() + 1);
        for line in &self.lines {
            let exact = line
                .amount
                .checked_mul(rate)
                .ok_or_else(EvalError::overflow)?;
            lines.push(round_to(exact, cents(), mode)?);
        }
        let exact_total = self
            .total
            .checked_mul(rate)
            .ok_or_else(EvalError::overflow)?;
        let total = round_to(exact_total, increment, mode)?;
        let sum = lines.iter().try_fold(Decimal::ZERO, |sum, amount| {
            sum.checked_add(*amount).ok_or_else(EvalError::overflow)
        })?;
        let difference = total.checked_sub(sum).ok_or_else(EvalError::overflow)?;
        if !difference.is_zero() {
            let rounding = self
                .lines
                .iter()
                .position(|line| line.kind == LineKind::Rounding)
                .and_then(|index| lines.get_mut(index));
            match rounding {
                Some(amount) => {
                    *amount = amount
                        .checked_add(difference)
                        .ok_or_else(EvalError::overflow)?;
                }
                None => lines.push(difference),
            }
        }
        Ok(Converted { lines, total })
    }
}

fn below_zero(template: &'static str, amount: Decimal) -> EvalError {
    EvalError::with(
        Code::NegativeTotal,
        template,
        [Arg::from(Value::money(amount).display())],
    )
}

#[cfg(test)]
mod tests {
    use super::{Bill, Billed, Clauses, LimitKind, Line, LineKind, cents, limit, round_to};
    use crate::tariff::syntax::ast::RoundMode;
    use rust_decimal::Decimal;
    use std::str::FromStr;

    fn d(text: &str) -> Decimal {
        Decimal::from_str(text).expect("a decimal")
    }

    fn line(amount: &str, code: Option<&str>) -> Line {
        Line {
            kind: LineKind::Charge,
            label: Some(String::from("x")),
            code: code.map(str::to_owned),
            amount: d(amount),
            quantity: None,
            rate: None,
            limit: None,
            on_freight: false,
            span: None,
        }
    }

    #[test]
    fn rounding_modes_round_to_increments() {
        assert_eq!(
            round_to(d("2.345"), cents(), RoundMode::HalfUp).ok(),
            Some(d("2.35"))
        );
        assert_eq!(
            round_to(d("2.345"), cents(), RoundMode::HalfEven).ok(),
            Some(d("2.34"))
        );
        assert_eq!(
            round_to(d("2.341"), cents(), RoundMode::Up).ok(),
            Some(d("2.35"))
        );
        assert_eq!(
            round_to(d("2.349"), cents(), RoundMode::Down).ok(),
            Some(d("2.34"))
        );
        assert_eq!(
            round_to(d("451"), d("5"), RoundMode::Up).ok(),
            Some(d("455"))
        );
        assert!(round_to(d("1"), d("0"), RoundMode::Up).is_err());
    }

    #[test]
    fn the_rounding_line_makes_the_lines_add_up() {
        let mut bill = Bill::default();
        for _ in 0..3 {
            assert_eq!(bill.charge(line("0.335", None)).ok(), Some(Billed::Added));
        }
        let finished = bill.finish(&Clauses::default()).expect("a bill");
        assert_eq!(finished.total, d("1.01"));
        let kinds: Vec<LineKind> = finished.lines.iter().map(|line| line.kind).collect();
        assert_eq!(kinds.last(), Some(&LineKind::Rounding));
        let sum: Decimal = finished.lines.iter().map(|line| line.amount).sum();
        assert_eq!(sum, finished.total);
    }

    #[test]
    fn converted_lines_add_up_to_the_converted_total() {
        let mut bill = Bill::default();
        for amount in ["10.01", "20.02", "30.03"] {
            assert!(bill.charge(line(amount, None)).is_ok());
        }
        let clauses = Clauses::default();
        let finished = bill.finish(&clauses).expect("a bill");
        let converted = finished.convert(d("1.3333"), &clauses).expect("converted");
        let sum: Decimal = converted.lines.iter().copied().sum();
        assert_eq!(sum, converted.total);
        assert_eq!(converted.total, d("80.08"));
    }

    #[test]
    fn freight_clauses_come_before_the_whole_bills() {
        let mut bill = Bill::default();
        assert!(bill.charge(line("300", None)).is_ok());
        assert!(bill.charge(line("20", Some("FSC"))).is_ok());
        let clauses = Clauses {
            freight_minimum: Some(d("400")),
            minimum: Some(d("450")),
            ..Clauses::default()
        };
        let finished = bill.finish(&clauses).expect("a bill");
        let adjustments: Vec<(LineKind, bool, Decimal)> = finished
            .lines
            .iter()
            .filter(|line| line.kind != LineKind::Charge)
            .map(|line| (line.kind, line.on_freight, line.amount))
            .collect();
        assert_eq!(
            adjustments,
            vec![
                (LineKind::Minimum, true, d("100")),
                (LineKind::Minimum, false, d("30")),
            ]
        );
        assert_eq!(finished.freight_total, d("430"));
        assert_eq!(finished.total, d("450"));
    }

    #[test]
    fn a_line_limit_raises_then_lowers() {
        let raised = limit(d("300"), Some(d("450")), None).expect("limited");
        assert_eq!(raised.0, d("450"));
        assert_eq!(
            raised.1.map(|limited| limited.kind),
            Some(LimitKind::Minimum)
        );
        assert_eq!(raised.1.map(|limited| limited.computed), Some(d("300")));
        let lowered = limit(d("900"), Some(d("450")), Some(d("800"))).expect("limited");
        assert_eq!(lowered.0, d("800"));
        assert_eq!(
            lowered.1.map(|limited| limited.kind),
            Some(LimitKind::Maximum)
        );
        assert_eq!(
            limit(d("500"), Some(d("450")), Some(d("800")))
                .expect("kept")
                .1,
            None
        );
        assert!(limit(d("500"), Some(d("900")), Some(d("800"))).is_err());
    }

    #[test]
    fn coded_charges_must_be_positive() {
        let mut bill = Bill::default();
        assert!(bill.charge(line("-1", Some("FSC"))).is_err());
        assert_eq!(
            bill.charge(line("0", Some("FSC"))).ok(),
            Some(Billed::Dropped)
        );
        assert_eq!(bill.charge(line("-1", None)).ok(), Some(Billed::Added));
    }
}
