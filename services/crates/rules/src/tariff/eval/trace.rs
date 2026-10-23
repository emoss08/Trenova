//! The rate trace: what a program did, statement by statement, for the Studio's preview, the
//! shipment's rating detail and Nova.
//!
//! Events are recorded in execution order for the root file's statements (a library's insides
//! are its own business; a call into one shows as the binding it produced): bindings and
//! assignments with their values, the branch an `if` took and the arm a `match` took, each loop
//! with its iteration count and its first iterations, charges, dropped zero lines, notes,
//! refusals, the option each `best` and `worst` kept, and table lookups. Values are wire
//! values plus the text interpolation would write.
//!
//! The trace is capped so a long loop cannot make the response large: at most
//! [`MAX_EVENTS`] events (then one `truncated` event), [`MAX_ITERATIONS`] recorded iterations per
//! loop, lists shown to [`MAX_ITEMS`] items and display text to [`MAX_DISPLAY`] characters.
//! Lookups are kept separately for the response's `lookups`, capped at [`MAX_LOOKUPS`].

use super::tables::Lookup;
use super::value::Value;
use crate::span::{Span, Utf16Index};
use crate::tariff::wire::encode;
use rust_decimal::Decimal;
use serde_json::{Map, Value as Json};

/// The most events a trace keeps.
pub const MAX_EVENTS: usize = 2_000;
/// The most iterations recorded per loop.
pub const MAX_ITERATIONS: usize = 50;
/// The most items a traced list shows.
pub const MAX_ITEMS: usize = 50;
/// The most characters a display text keeps.
pub const MAX_DISPLAY: usize = 500;
/// The most lookups kept.
pub const MAX_LOOKUPS: usize = 500;

/// One thing a program did.
#[derive(Debug, Clone)]
pub enum Event {
    /// `let`, `var`, `if let`, `require let`, a match binding.
    Bind {
        /// The name bound.
        name: String,
        /// Its value.
        value: Value,
        /// The statement.
        span: Span,
    },
    /// `name = …`, `+=`, `-=`.
    Assign {
        /// The `var`.
        name: String,
        /// Its new value.
        value: Value,
        /// The statement.
        span: Span,
    },
    /// An `if`: the branch taken (`0` for the `if`, `1…` for each `else if`, the count for the
    /// `else`), or `None` when nothing ran.
    Branch {
        /// The branch taken.
        taken: Option<usize>,
        /// The statement.
        span: Span,
    },
    /// A `match`: the arm taken.
    Arm {
        /// The arm's index.
        arm: usize,
        /// The arm.
        span: Span,
    },
    /// A loop finished: how many iterations ran its body.
    Loop {
        /// The loop variable.
        binding: String,
        /// Iterations that ran the body.
        iterations: usize,
        /// The statement.
        span: Span,
    },
    /// One iteration, while under [`MAX_ITERATIONS`].
    Iteration {
        /// The loop variable.
        binding: String,
        /// The 0-based iteration.
        index: usize,
        /// The loop variable's value.
        value: Value,
        /// The loop statement.
        span: Span,
    },
    /// A charge added to the bill.
    Charge {
        /// The label.
        label: String,
        /// The code.
        code: Option<String>,
        /// The exact amount.
        amount: Decimal,
        /// The statement.
        span: Span,
    },
    /// A charge whose amount was zero, dropped.
    Dropped {
        /// The label.
        label: String,
        /// The code.
        code: Option<String>,
        /// The statement.
        span: Option<Span>,
    },
    /// A note.
    Note {
        /// The text.
        text: String,
        /// The statement.
        span: Span,
    },
    /// The program refused to rate.
    Refusal {
        /// The author's text.
        message: String,
        /// Where it is written.
        span: Span,
    },
    /// A `best` or `worst` finished: the label of the option it kept, or `None` when every
    /// option refused.
    Choice {
        /// The choice's label.
        label: String,
        /// The kept option's label.
        chosen: Option<String>,
        /// The statement.
        span: Span,
    },
    /// A table lookup: the index into the response's `lookups`.
    Lookup {
        /// The index.
        index: usize,
        /// The expression.
        span: Span,
    },
}

/// A trace being recorded.
#[derive(Debug, Clone, Default)]
pub struct Trace {
    enabled: bool,
    events: Vec<Event>,
    truncated: bool,
    lookups: Vec<(Lookup, Span)>,
}

impl Trace {
    /// A trace that records events when `enabled`; lookups are always kept.
    #[must_use]
    pub fn new(enabled: bool) -> Self {
        Self {
            enabled,
            ..Self::default()
        }
    }

    /// Whether events are recorded.
    #[must_use]
    pub const fn enabled(&self) -> bool {
        self.enabled
    }

    /// Records an event.
    pub fn push(&mut self, event: Event) {
        if !self.enabled {
            return;
        }
        if self.events.len() >= MAX_EVENTS {
            self.truncated = true;
            return;
        }
        self.events.push(event);
    }

    /// Records a lookup.
    pub fn lookup(&mut self, lookup: Lookup, span: Span) {
        if self.lookups.len() >= MAX_LOOKUPS {
            return;
        }
        let index = self.lookups.len();
        self.lookups.push((lookup, span));
        self.push(Event::Lookup { index, span });
    }

    /// The trace on the wire, or `null` when it was not asked for.
    #[must_use]
    pub fn encode(&self, index: &Utf16Index) -> Json {
        if !self.enabled {
            return Json::Null;
        }
        let mut events: Vec<Json> = self
            .events
            .iter()
            .map(|event| event_json(event, index))
            .collect();
        if self.truncated {
            events.push(object(vec![(
                "kind",
                Json::String(String::from("truncated")),
            )]));
        }
        Json::Array(events)
    }

    /// The lookups kept, in the order they ran.
    pub fn lookups(&self) -> impl Iterator<Item = &Lookup> {
        self.lookups.iter().map(|(lookup, _)| lookup)
    }

    /// The lookups on the wire.
    #[must_use]
    pub fn encode_lookups(&self, index: &Utf16Index) -> Json {
        Json::Array(
            self.lookups
                .iter()
                .map(|(lookup, span)| {
                    let (value, display) = traced(&lookup.value);
                    object(vec![
                        ("table", Json::String(lookup.table.clone())),
                        ("op", Json::String(lookup.op.to_owned())),
                        (
                            "keys",
                            Json::Array(lookup.keys.iter().map(|key| traced(key).0).collect()),
                        ),
                        (
                            "match",
                            lookup.matched.clone().map_or(Json::Null, Json::Array),
                        ),
                        ("adjusted", Json::Bool(lookup.adjusted)),
                        ("value", value),
                        ("display", Json::String(display)),
                        ("range", range(*span, index)),
                    ])
                })
                .collect(),
        )
    }
}

fn object(fields: Vec<(&str, Json)>) -> Json {
    let mut map = Map::new();
    for (name, value) in fields {
        map.insert(name.to_owned(), value);
    }
    Json::Object(map)
}

/// A span as a wire range.
#[must_use]
pub fn range(span: Span, index: &Utf16Index) -> Json {
    let range = index.range(span);
    object(vec![
        ("from", Json::from(range.from)),
        ("to", Json::from(range.to)),
    ])
}

/// A value as the trace shows it: its wire form with lists cut to [`MAX_ITEMS`] items, and its
/// display text cut to [`MAX_DISPLAY`] characters.
#[must_use]
pub fn traced(value: &Value) -> (Json, String) {
    let wire = match value {
        Value::List(items) if items.len() > MAX_ITEMS => {
            Json::Array(items.iter().take(MAX_ITEMS).map(encode).collect())
        }
        other => encode(other),
    };
    let display = match value {
        Value::List(items) if items.len() > MAX_ITEMS => format!("[{} items]", items.len()),
        other => other.display(),
    };
    (wire, ellipsize(&display))
}

fn ellipsize(text: &str) -> String {
    if text.chars().count() <= MAX_DISPLAY {
        return text.to_owned();
    }
    let cut: String = text.chars().take(MAX_DISPLAY - 1).collect();
    format!("{cut}\u{2026}")
}

fn value_fields(value: &Value) -> [(&'static str, Json); 2] {
    let (wire, display) = traced(value);
    [("value", wire), ("display", Json::String(display))]
}

fn text(value: &str) -> Json {
    Json::String(value.to_owned())
}

fn code(value: Option<&String>) -> Json {
    value.map_or(Json::Null, |code| Json::String(code.clone()))
}

fn event_json(event: &Event, index: &Utf16Index) -> Json {
    match event {
        Event::Bind { name, value, span } | Event::Assign { name, value, span } => {
            let kind = if matches!(event, Event::Bind { .. }) {
                "bind"
            } else {
                "assign"
            };
            let [value, display] = value_fields(value);
            object(vec![
                ("kind", text(kind)),
                ("name", text(name)),
                value,
                display,
                ("range", range(*span, index)),
            ])
        }
        Event::Branch { taken, span } => object(vec![
            ("kind", text("branch")),
            ("taken", taken.map_or(Json::Null, Json::from)),
            ("range", range(*span, index)),
        ]),
        Event::Arm { arm, span } => object(vec![
            ("kind", text("arm")),
            ("arm", Json::from(*arm)),
            ("range", range(*span, index)),
        ]),
        Event::Loop {
            binding,
            iterations,
            span,
        } => object(vec![
            ("kind", text("loop")),
            ("name", text(binding)),
            ("iterations", Json::from(*iterations)),
            ("range", range(*span, index)),
        ]),
        Event::Iteration {
            binding,
            index: iteration,
            value,
            span,
        } => {
            let [value, display] = value_fields(value);
            object(vec![
                ("kind", text("iteration")),
                ("name", text(binding)),
                ("index", Json::from(*iteration)),
                value,
                display,
                ("range", range(*span, index)),
            ])
        }
        Event::Charge {
            label,
            code: charge_code,
            amount,
            span,
        } => charge_json(label, charge_code.as_ref(), *amount, *span, index),
        Event::Dropped {
            label,
            code: charge_code,
            span,
        } => dropped_json(label, charge_code.as_ref(), *span, index),
        Event::Note { text: note, span } => object(vec![
            ("kind", text("note")),
            ("text", text(note)),
            ("range", range(*span, index)),
        ]),
        Event::Refusal { message, span } => object(vec![
            ("kind", text("refusal")),
            ("message", text(message)),
            ("range", range(*span, index)),
        ]),
        Event::Choice {
            label,
            chosen,
            span,
        } => object(vec![
            ("kind", text("choice")),
            ("label", text(label)),
            ("chosen", code(chosen.as_ref())),
            ("range", range(*span, index)),
        ]),
        Event::Lookup { index: at, span } => object(vec![
            ("kind", text("lookup")),
            ("lookup", Json::from(*at)),
            ("range", range(*span, index)),
        ]),
    }
}

/// A charge added to the bill, on the wire.
fn charge_json(
    label: &str,
    charge_code: Option<&String>,
    amount: Decimal,
    span: Span,
    index: &Utf16Index,
) -> Json {
    let [value, display] = value_fields(&Value::money(amount));
    object(vec![
        ("kind", text("charge")),
        ("label", text(label)),
        ("code", code(charge_code)),
        value,
        display,
        ("range", range(span, index)),
    ])
}

/// A zero line dropped from the bill, on the wire.
fn dropped_json(
    label: &str,
    charge_code: Option<&String>,
    span: Option<Span>,
    index: &Utf16Index,
) -> Json {
    object(vec![
        ("kind", text("dropped")),
        ("label", text(label)),
        ("code", code(charge_code)),
        ("range", span.map_or(Json::Null, |span| range(span, index))),
    ])
}
