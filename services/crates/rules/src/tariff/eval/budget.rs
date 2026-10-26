//! The evaluator's budgets: how much work, text, list and call nesting one program may use.
//!
//! Every unit of work is counted as it happens, so a runaway program stops at its limit,
//! deterministically, in the browser and on the server alike:
//!
//! - a **step** is one node evaluated, or one element a list method or table lookup visits;
//! - **text** is every byte of text the program builds (interpolation, `upper`, `replace`,
//!   `split` and the rest); text read from the record is free;
//! - **items** are list items and record fields the program creates (literals, ranges walked,
//!   `map`, `where`, `split` and the rest);
//! - **call depth** is how many user functions are running at once. The checker refuses
//!   recursion, so this is a second line of defence that also bounds the native stack.
//!
//! The defaults are [`Limits::DEFAULT`]. A request may lower any of them, never raise them. The
//! limits apply to each program of a request separately.

use super::error::{Code, EvalError};
use serde::Deserialize;
use trenova_module_kit::message::{Arg, Message};

/// The most a program may use.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Limits {
    /// Nodes evaluated plus elements visited.
    pub steps: u64,
    /// Bytes of text built.
    pub text_bytes: u64,
    /// List items and record fields created.
    pub items: u64,
    /// User functions running at once.
    pub call_depth: u32,
}

impl Limits {
    /// The defaults: a million steps, one MiB of text, a hundred thousand items and calls 64
    /// deep. A real tariff uses a few thousand steps; a million is still a few tens of
    /// milliseconds.
    pub const DEFAULT: Self = Self {
        steps: 1_000_000,
        text_bytes: 1 << 20,
        items: 100_000,
        call_depth: 64,
    };

    /// These limits lowered by a request's overrides; an override above a default is ignored.
    #[must_use]
    pub fn lowered(self, overrides: &BudgetWire) -> Self {
        Self {
            steps: overrides
                .steps
                .map_or(self.steps, |value| value.min(self.steps)),
            text_bytes: overrides
                .text_bytes
                .map_or(self.text_bytes, |value| value.min(self.text_bytes)),
            items: overrides
                .items
                .map_or(self.items, |value| value.min(self.items)),
            call_depth: overrides
                .call_depth
                .map_or(self.call_depth, |value| value.min(self.call_depth)),
        }
    }
}

impl Default for Limits {
    fn default() -> Self {
        Self::DEFAULT
    }
}

/// A request's budget overrides: `{"steps": 5000, "textBytes": 1024, "items": 100,
/// "callDepth": 8}`, each optional.
#[derive(Debug, Clone, Copy, Default, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct BudgetWire {
    /// Steps.
    #[serde(default)]
    pub steps: Option<u64>,
    /// Text bytes.
    #[serde(default)]
    pub text_bytes: Option<u64>,
    /// Items.
    #[serde(default)]
    pub items: Option<u64>,
    /// Call depth.
    #[serde(default)]
    pub call_depth: Option<u32>,
}

/// What a program has used so far.
#[derive(Debug, Clone)]
pub struct Meter {
    limits: Limits,
    steps: u64,
    text_bytes: u64,
    items: u64,
    depth: u32,
}

fn exhausted(template: &'static str, limit: u64) -> EvalError {
    EvalError::new(
        Code::Budget,
        Message::with(
            template,
            [Arg::Count(i64::try_from(limit).unwrap_or(i64::MAX))],
        ),
    )
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
            depth: 0,
        }
    }

    /// The limits this meter enforces.
    #[must_use]
    pub const fn limits(&self) -> Limits {
        self.limits
    }

    /// The steps spent so far.
    #[must_use]
    pub const fn steps_used(&self) -> u64 {
        self.steps
    }

    /// Spends `count` steps.
    pub fn steps(&mut self, count: usize) -> Result<(), EvalError> {
        self.steps = self
            .steps
            .saturating_add(u64::try_from(count).unwrap_or(u64::MAX));
        if self.steps > self.limits.steps {
            return Err(exhausted(
                trenova_module_kit::message!(
                    "The tariff used up its step budget ({0, plural, one {# step} other {# steps}}) before it finished"
                ),
                self.limits.steps,
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
        self.text_bytes = self
            .text_bytes
            .saturating_add(u64::try_from(bytes).unwrap_or(u64::MAX));
        if self.text_bytes > self.limits.text_bytes {
            return Err(exhausted(
                trenova_module_kit::message!(
                    "The tariff built more text than its budget allows ({0, plural, one {# byte} other {# bytes}})"
                ),
                self.limits.text_bytes,
            ));
        }
        Ok(())
    }

    /// Records `count` list items or record fields created.
    pub fn items(&mut self, count: usize) -> Result<(), EvalError> {
        self.items = self
            .items
            .saturating_add(u64::try_from(count).unwrap_or(u64::MAX));
        if self.items > self.limits.items {
            return Err(exhausted(
                trenova_module_kit::message!(
                    "The tariff created more list items than its budget allows ({0, plural, one {# item} other {# items}})"
                ),
                self.limits.items,
            ));
        }
        Ok(())
    }

    /// Fails before an operation that would create `count` items, without spending them.
    pub fn items_would_fit(&self, count: u64) -> Result<(), EvalError> {
        if self.items.saturating_add(count) > self.limits.items {
            return Err(exhausted(
                trenova_module_kit::message!(
                    "The tariff created more list items than its budget allows ({0, plural, one {# item} other {# items}})"
                ),
                self.limits.items,
            ));
        }
        Ok(())
    }

    /// Enters a user function.
    pub fn enter(&mut self) -> Result<(), EvalError> {
        if self.depth >= self.limits.call_depth {
            return Err(exhausted(
                trenova_module_kit::message!(
                    "The tariff called functions nested deeper than its budget allows ({0, plural, one {# call} other {# calls}})"
                ),
                u64::from(self.limits.call_depth),
            ));
        }
        self.depth += 1;
        Ok(())
    }

    /// Leaves a user function.
    pub const fn leave(&mut self) {
        self.depth = self.depth.saturating_sub(1);
    }
}

#[cfg(test)]
mod tests {
    use super::{BudgetWire, Limits, Meter};
    use crate::tariff::eval::error::Code;

    #[test]
    fn each_budget_stops_at_its_limit() {
        let mut meter = Meter::new(Limits {
            steps: 2,
            text_bytes: 3,
            items: 1,
            call_depth: 1,
        });
        assert!(meter.step().is_ok());
        assert!(meter.step().is_ok());
        assert_eq!(meter.step().map_err(|error| error.code), Err(Code::Budget));
        assert!(meter.text(3).is_ok());
        assert!(meter.text(1).is_err());
        assert!(meter.items_would_fit(1).is_ok());
        assert!(meter.items_would_fit(2).is_err());
        assert!(meter.enter().is_ok());
        assert!(meter.enter().is_err());
        meter.leave();
        assert!(meter.enter().is_ok());
    }

    #[test]
    fn overrides_only_lower() {
        let lowered = Limits::DEFAULT.lowered(&BudgetWire {
            steps: Some(10),
            items: Some(u64::MAX),
            ..BudgetWire::default()
        });
        assert_eq!(lowered.steps, 10);
        assert_eq!(lowered.items, Limits::DEFAULT.items);
    }
}
