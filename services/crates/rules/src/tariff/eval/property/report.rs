//! A failed property's counterexample: the varied values of the failing case, the monotone
//! pair or the refusal, and a `test` block that reproduces it.
//!
//! The `test` is real Tariff, formatted with the crate's formatter (`tariff.format`), so the
//! Studio's "Save as test" inserts it as-is: the property's `given`s, each varied value as a
//! `given` (or a `let` for a varied name), and the property's other statements (its `expect
//! cond` lines and the `let`s they read); a monotone failure also pins the `before` case's
//! value with `expect <expression> == <value>`. It is named `"<property>: <values>"`.

use super::axes::{Axis, Target};
use crate::tariff::eval::StaticTypes;
use crate::tariff::eval::error::Refusal;
use crate::tariff::eval::ops::refusal_json;
use crate::tariff::eval::scaffold::{source_literal, text_literal};
use crate::tariff::eval::value::Value;
use crate::tariff::schema::Schema;
use crate::tariff::syntax::ast::{Direction, Expect, Expr, FileKind, Property, StmtKind};
use crate::tariff::syntax::format::format;
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::types::Type;
use crate::tariff::wire::encode;
use serde_json::{Map, Value as Json};
use std::fmt::Write as _;
use trenova_module_kit::span::{Span, Utf16Index};

/// A failed property's case.
#[derive(Debug, Clone)]
pub struct Counterexample {
    /// Each varied target with its value in the case (for a monotone failure, the `before`
    /// case).
    pub values: Vec<Varied>,
    /// The out-of-order pair, for a monotone failure.
    pub monotone: Option<Monotone>,
    /// The case's refusal, when it refused to rate.
    pub refusal: Option<Refusal>,
    /// A `test` block that reproduces the case.
    pub test: String,
}

/// One varied target's value in a counterexample.
#[derive(Debug, Clone)]
pub struct Varied {
    /// The target as written: `shipment.weight`, `input.rate`, `w`.
    pub target: String,
    /// The value as the Studio shows it: a listed item as the author wrote it, else as
    /// interpolation writes it.
    pub display: String,
    /// The value.
    pub value: Value,
}

/// Two neighbouring values of a monotone failure whose results are out of order.
#[derive(Debug, Clone)]
pub struct Monotone {
    /// The varied target compared along.
    pub target: String,
    /// The expression compared, as written.
    pub expression: String,
    /// Which way it must go.
    pub direction: Direction,
    /// The lower value of the target and the expression's result there.
    pub before: (Value, Value),
    /// The higher value and its result.
    pub after: (Value, Value),
}

impl Counterexample {
    /// The counterexample on the wire.
    #[must_use]
    pub fn to_json(&self, index: &Utf16Index) -> Json {
        let values: Vec<Json> = self
            .values
            .iter()
            .map(|varied| {
                let mut out = Map::new();
                out.insert(String::from("target"), Json::String(varied.target.clone()));
                out.insert(
                    String::from("display"),
                    Json::String(varied.display.clone()),
                );
                out.insert(String::from("value"), encode(&varied.value));
                Json::Object(out)
            })
            .collect();
        let mut out = Map::new();
        out.insert(String::from("values"), Json::Array(values));
        out.insert(
            String::from("monotone"),
            self.monotone.as_ref().map_or(Json::Null, Monotone::to_json),
        );
        out.insert(
            String::from("refusal"),
            self.refusal
                .as_ref()
                .map_or(Json::Null, |refusal| refusal_json(refusal, index)),
        );
        out.insert(String::from("test"), Json::String(self.test.clone()));
        Json::Object(out)
    }
}

impl Monotone {
    fn to_json(&self) -> Json {
        let side = |(at, result): &(Value, Value)| {
            let mut out = Map::new();
            out.insert(String::from("at"), shown(at));
            out.insert(String::from("result"), shown(result));
            Json::Object(out)
        };
        let mut out = Map::new();
        out.insert(String::from("target"), Json::String(self.target.clone()));
        out.insert(
            String::from("expression"),
            Json::String(self.expression.clone()),
        );
        out.insert(
            String::from("direction"),
            Json::String(self.direction.word().to_owned()),
        );
        out.insert(String::from("before"), side(&self.before));
        out.insert(String::from("after"), side(&self.after));
        Json::Object(out)
    }
}

/// `{"display", "value"}`.
fn shown(value: &Value) -> Json {
    let mut out = Map::new();
    out.insert(String::from("display"), Json::String(value.display()));
    out.insert(String::from("value"), encode(value));
    Json::Object(out)
}

/// What writing a counterexample's `test` reads.
pub struct Writing<'w> {
    /// The root file's source.
    pub source: &'w str,
    /// The schema, for writing record values.
    pub schema: &'w Schema,
    /// The checker's types, for writing the pinned value.
    pub types: &'w dyn StaticTypes,
    /// The root file's kind, for the header the formatter reads the test under.
    pub kind: FileKind,
}

impl std::fmt::Debug for Writing<'_> {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("Writing")
            .field("kind", &self.kind)
            .finish_non_exhaustive()
    }
}

impl Writing<'_> {
    /// The `test` block reproducing a case; `pinned` is a monotone failure's expression and
    /// its `before` value.
    #[must_use]
    pub fn test(
        &self,
        property: &Property,
        axes: &[Axis<'_>],
        values: &[Value],
        pinned: Option<(&Expr, &Value)>,
    ) -> String {
        let mut givens: Vec<String> = Vec::new();
        let mut lets: Vec<String> = Vec::new();
        let mut rest: Vec<String> = Vec::new();
        let mut varied = axes.iter().zip(values);
        for stmt in &property.body.stmts {
            match &stmt.kind {
                StmtKind::Given { .. } => givens.push(self.text(stmt.span).to_owned()),
                StmtKind::Vary { .. } => {
                    let Some((entry, value)) = varied.next() else {
                        continue;
                    };
                    let literal = entry.written_at(value).map_or_else(
                        || source_literal(self.schema, value, &entry.ty),
                        |span| self.text(span).to_owned(),
                    );
                    match &entry.target {
                        Target::Name(name) => lets.push(format!("let {name} = {literal}")),
                        Target::Path(_) | Target::Input(_) => {
                            givens.push(format!("given {} = {literal}", entry.label));
                        }
                    }
                }
                StmtKind::Assume(_) | StmtKind::Expect(Expect::Monotone { .. }) => {}
                _ => rest.push(self.text(stmt.span).to_owned()),
            }
        }
        if let Some((expression, value)) = pinned {
            let ty = self
                .types
                .type_of(ModuleId::ROOT, expression.id)
                .unwrap_or(Type::Error);
            rest.push(format!(
                "expect {} == {}",
                self.text(expression.span),
                source_literal(self.schema, value, &ty)
            ));
        }
        let shown: Vec<String> = axes
            .iter()
            .zip(values)
            .map(|(entry, value)| self.display(entry, value))
            .collect();
        let name = format!("{}: {}", property.name.text, shown.join(", "));
        let mut test = format!("test {} {{\n", text_literal(&name));
        for line in givens.iter().chain(&lets).chain(&rest) {
            let _ = writeln!(test, "  {line}");
        }
        test.push_str("}\n");
        self.formatted(test)
    }

    /// The test in the crate's canonical style; as written when it does not format.
    fn formatted(&self, test: String) -> String {
        let file = format!("{} \"x\"\n\n{test}", self.kind.keyword());
        let Ok(formatted) = format(&file) else {
            return test;
        };
        let Some(start) = formatted.find("\ntest ").map(|at| at + 1) else {
            return test;
        };
        formatted.get(start..).map_or(test, ToOwned::to_owned)
    }

    /// A varied value as the Studio shows it: a listed item as written, else as interpolation
    /// writes it.
    #[must_use]
    pub fn display(&self, axis: &Axis<'_>, value: &Value) -> String {
        axis.written_at(value)
            .map_or_else(|| value.display(), |span| self.text(span).to_owned())
    }

    fn text(&self, span: Span) -> &str {
        let start = usize::try_from(span.start).unwrap_or(usize::MAX);
        let end = usize::try_from(span.end).unwrap_or(usize::MAX);
        self.source.get(start..end).unwrap_or_default()
    }
}
