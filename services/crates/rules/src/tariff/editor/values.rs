//! The last preview's values, as `tariff.hover` and `tariff.inlays` read them back.
//!
//! `values` is the `trace` array `tariff.evaluate` returned for the root file: events in
//! execution order, of which the bindings (`{"kind": "bind", "name", "value", "display",
//! "range"}`) and assignments (`"kind": "assign"`) carry values. A binding's value is found by
//! its name and the statement whose range holds the name; a binding run several times (in a
//! loop) shows its last value and how many times it ran. Anything else in the array, and any
//! malformed entry, is ignored: the trace is a hint, never a reason to refuse a request.

use trenova_module_kit::span::Range;
use serde_json::{Value as Json, json};

/// One traced value.
#[derive(Debug, Clone, PartialEq)]
pub struct Traced {
    /// The wire value.
    pub value: Json,
    /// The text interpolation would write.
    pub display: String,
    /// Where the binding or assignment's statement is, in UTF-16.
    pub range: Range,
    /// The name bound or assigned.
    pub name: String,
    /// Whether it is an assignment (`=`, `+=`, `-=`) rather than a binding.
    pub assign: bool,
}

/// The values of a trace.
#[derive(Debug, Clone, Default)]
pub struct Values {
    events: Vec<Traced>,
}

impl Values {
    /// Reads the values out of a trace; `None` reads as no values.
    #[must_use]
    pub fn read(trace: Option<&Json>) -> Self {
        let Some(Json::Array(events)) = trace else {
            return Self::default();
        };
        let events = events
            .iter()
            .filter_map(|event| {
                let kind = event.get("kind")?.as_str()?;
                if kind != "bind" && kind != "assign" {
                    return None;
                }
                let range = event.get("range")?;
                Some(Traced {
                    value: event.get("value").cloned().unwrap_or(Json::Null),
                    display: event.get("display")?.as_str()?.to_owned(),
                    range: Range {
                        from: u32::try_from(range.get("from")?.as_u64()?).ok()?,
                        to: u32::try_from(range.get("to")?.as_u64()?).ok()?,
                    },
                    name: event.get("name")?.as_str()?.to_owned(),
                    assign: kind == "assign",
                })
            })
            .collect();
        Self { events }
    }

    /// Every binding and assignment, in execution order.
    #[must_use]
    pub fn events(&self) -> &[Traced] {
        &self.events
    }

    /// Whether there are no values.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.events.is_empty()
    }

    /// The bindings of `name` made by the statement holding UTF-16 position `at`, in order.
    pub fn bindings<'a>(&'a self, name: &'a str, at: u32) -> impl Iterator<Item = &'a Traced> + 'a {
        self.events.iter().filter(move |event| {
            !event.assign && event.name == name && event.range.from <= at && at <= event.range.to
        })
    }

    /// The last value `name` took inside UTF-16 range `scope`: its binding or a later
    /// assignment, with how many times it was set.
    #[must_use]
    pub fn last_in(&self, name: &str, declared: u32, scope: Range) -> Option<(&Traced, usize)> {
        let mut last = None;
        let mut count = 0;
        for event in &self.events {
            let bound_here =
                !event.assign && event.range.from <= declared && declared <= event.range.to;
            let assigned_here =
                event.assign && scope.from <= event.range.from && event.range.to <= scope.to;
            if event.name == name && (bound_here || assigned_here) {
                last = Some(event);
                count += 1;
            }
        }
        last.map(|event| (event, count))
    }
}

/// A traced value as hover shows it: `{"value", "display", "times"}`.
#[must_use]
pub fn hover_json(traced: &Traced, times: usize) -> Json {
    json!({ "value": traced.value, "display": traced.display, "times": times })
}
