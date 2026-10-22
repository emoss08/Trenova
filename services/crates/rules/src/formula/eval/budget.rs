//! The evaluator's budgets: how much work, text and list a formula may make.
//!
//! The Go engine bounded a formula with a five-second timer and expr's memory budget; a runaway
//! formula held a CPU until the timer fired and then kept it, because a goroutine cannot be
//! stopped. Here every unit of work is counted as it happens, so a formula stops at its limit,
//! deterministically, on the browser and the server alike:
//!
//! - a **step** is one node evaluated, or one element a built-in visits (a `map` over a thousand
//!   stops is a thousand steps plus the predicate's own);
//! - **text** is every byte of text an operation builds (concatenation, `replace`, `join`,
//!   `string`, `upper`, slices and the rest); text read from the environment is free;
//! - **items** are list items and map entries a formula creates (literals, ranges, `map`,
//!   `split`, `keys` and the rest).
//!
//! The defaults are [`Limits::DEFAULT`]. A request may lower any of them, never raise them. The
//! limits apply to each program of a request separately.

use super::error::Code;
use super::error::EvalError;
use crate::message::Arg;
use crate::message::Message;

/// The most a program may do.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Limits {
    /// Nodes evaluated plus elements visited by built-ins.
    pub steps: u64,
    /// Bytes of text built.
    pub text_bytes: u64,
    /// List items and map entries created.
    pub items: u64,
}

impl Limits {
    /// The defaults: a million steps, one MiB of text (as the Go engine's `maxEvaluationStrBytes`)
    /// and a hundred thousand items (as expr's `MemoryBudget`). A real rating formula uses a few
    /// hundred steps; a million is still a few tens of milliseconds.
    pub const DEFAULT: Self = Self {
        steps: 1_000_000,
        text_bytes: 1 << 20,
        items: 100_000,
    };
}

impl Default for Limits {
    fn default() -> Self {
        Self::DEFAULT
    }
}

/// What a program has used so far.
#[derive(Debug, Clone)]
pub struct Meter {
    limits: Limits,
    steps: u64,
    text_bytes: u64,
    items: u64,
}

impl Meter {
    /// A fresh meter under `limits`.
    #[must_use]
    pub const fn new(limits: Limits) -> Self {
        Self {
            limits,
            steps: 0,
            text_bytes: 0,
            items: 0,
        }
    }

    /// Spends `count` steps.
    pub fn steps(&mut self, count: usize) -> Result<(), EvalError> {
        self.steps = self.steps.saturating_add(count as u64);
        if self.steps > self.limits.steps {
            return Err(EvalError::new(
                Code::Budget,
                Message::new(crate::message!(
                    "formula evaluation exceeded its time budget"
                )),
            ));
        }
        Ok(())
    }

    /// Spends one step.
    pub fn step(&mut self) -> Result<(), EvalError> {
        self.steps(1)
    }

    /// Records `bytes` of text built.
    pub fn text(&mut self, bytes: usize) -> Result<(), EvalError> {
        self.text_bytes = self.text_bytes.saturating_add(bytes as u64);
        if self.text_bytes > self.limits.text_bytes {
            return Err(EvalError::new(
                Code::Budget,
                Message::with(
                    crate::message!(
                        "expression built more text than a formula may ({0, plural, one {# byte} other {# bytes}}); formulas compare codes, they do not assemble documents"
                    ),
                    [Arg::Count(
                        i64::try_from(self.limits.text_bytes).unwrap_or(i64::MAX),
                    )],
                ),
            ));
        }
        Ok(())
    }

    /// Records `count` list items or map entries created.
    pub fn items(&mut self, count: usize) -> Result<(), EvalError> {
        self.items = self.items.saturating_add(count as u64);
        if self.items > self.limits.items {
            return Err(EvalError::new(
                Code::Budget,
                Message::new(crate::message!("memory budget exceeded")),
            ));
        }
        Ok(())
    }

    /// Fails before an operation that would build `bytes` of text, without spending them; for
    /// operations whose size is known before they run (`replace`, `join`).
    pub fn text_would_fit(&self, bytes: usize) -> Result<(), EvalError> {
        let mut probe = self.clone();
        probe.text(bytes)
    }

    /// Fails before an operation that would create `count` items, without spending them; for
    /// ranges, whose size is known before they are built.
    pub fn items_would_fit(&self, count: u64) -> Result<(), EvalError> {
        let mut probe = self.clone();
        probe.items(usize::try_from(count).unwrap_or(usize::MAX))
    }
}

#[cfg(test)]
mod tests {
    use super::{Limits, Meter};
    use crate::formula::eval::error::Code;

    #[test]
    fn each_budget_stops_at_its_limit() {
        let mut meter = Meter::new(Limits {
            steps: 2,
            text_bytes: 3,
            items: 1,
        });
        assert!(meter.step().is_ok());
        assert!(meter.step().is_ok());
        assert_eq!(meter.step().map_err(|error| error.code), Err(Code::Budget));
        assert!(meter.text(3).is_ok());
        assert!(meter.text(1).is_err());
        assert!(meter.items_would_fit(1).is_ok());
        assert!(meter.items_would_fit(2).is_err());
    }
}
