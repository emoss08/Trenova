//! Rate tables: `tables.Code[k, …]`, `.cell(…)`, `.band(x)` and `.interpolate(x)`, typed from
//! the shapes the context describes.
//!
//! Each key is checked against its axis's key type ([`Axis::key_type`]); an exact axis also
//! takes a list of keys (`lane.origin.zones`), which the evaluator tries in order. A lookup
//! gives the table's value type, optional, since a miss is `none`.

use super::Callee;
use super::checker::{Checker, Want};
use super::describe::{quote, words};
use super::fixes::replace;
use crate::message;
use crate::message::{Arg, Message};
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::schema::{Axis, TableShape};
use crate::tariff::syntax::ast::{Arg as CallArg, Expr, Name};
use crate::tariff::types::{FieldDef, RecordType, Type};
use std::sync::Arc;

/// The methods a table has besides indexing.
pub const TABLE_METHODS: &[(&str, &str)] = &[
    (
        "cell",
        message!(
            "The matching cell: its `value`, its `minimum` charge (`Money?`) and its `deficit` flag, or `none`."
        ),
    ),
    (
        "band",
        message!(
            "On a one-axis table of bands, the band a value falls in (its `min`, `max` and `value`), or `none`."
        ),
    ),
    (
        "interpolate",
        message!(
            "On a one-axis table of bands, the value interpolated between the bands around a value."
        ),
    ),
];

impl Checker<'_> {
    /// The table a code names, reporting an unknown one (with a fix to a code that differs
    /// only in case, or a close one).
    fn table(&mut self, code: &Name) -> Option<TableShape> {
        if let Some(shape) = self
            .context
            .tables
            .iter()
            .find(|shape| shape.code == code.text)
        {
            return Some(shape.clone());
        }
        let lower = code.text.to_lowercase();
        if let Some(shape) = self
            .context
            .tables
            .iter()
            .find(|shape| shape.code.to_lowercase() == lower)
        {
            let stored = shape.code.clone();
            self.report(
                Diagnostic::error(
                    "unknown-table",
                    code.span,
                    Message::with(
                        message!("No rate table has the code {0}; its code is written {1}"),
                        [quote(&code.text), quote(&stored)],
                    ),
                )
                .with_fix(replace(
                    Message::with(message!("Change to {0}"), [quote(&stored)]),
                    code.span,
                    stored.clone(),
                )),
            );
            return None;
        }
        let candidates: Vec<String> = self
            .context
            .tables
            .iter()
            .map(|shape| shape.code.clone())
            .collect();
        let diagnostic = Self::did_you_mean(
            "unknown-table",
            code.span,
            &code.text,
            &candidates,
            Message::with(
                message!("No rate table has the code {0}"),
                [quote(&code.text)],
            ),
            Message::with(
                message!("No rate table has the code {0}; only active rate matrices can be read"),
                [quote(&code.text)],
            ),
        );
        self.report(diagnostic);
        None
    }

    /// The type a table's cells hold.
    fn table_value(&mut self, shape: &TableShape, code: &Name) -> Type {
        shape.value().unwrap_or_else(|| {
            self.report(Diagnostic::error(
                "table-value",
                code.span,
                Message::with(
                    message!(
                        "The rate table {0} holds {1}, which isn't a value type a table can have"
                    ),
                    [quote(&shape.code), quote(&shape.value_type)],
                ),
            ));
            Type::Error
        })
    }

    /// Checks a lookup's keys against the table's axes.
    fn table_keys(&mut self, at: &Expr, shape: &TableShape, keys: &[&Expr]) {
        if keys.len() != shape.axes.len() {
            let labels: Vec<String> = shape
                .axes
                .iter()
                .map(|axis| {
                    if axis.label.is_empty() {
                        axis.kind.clone()
                    } else {
                        axis.label.clone()
                    }
                })
                .collect();
            self.report(Diagnostic::error(
                "table-keys",
                at.span,
                Message::with(
                    message!("{0} takes {1, plural, one {# key} other {# keys}} ({2}), but {3, plural, one {# is} other {# are}} given"),
                    [quote(&shape.code), Arg::from(shape.axes.len()), Arg::Text(labels.join(", ")), Arg::from(keys.len())],
                ),
            ));
        }
        for (index, key) in keys.iter().enumerate() {
            let Some(axis) = shape.axes.get(index) else {
                self.expr(key, None);
                continue;
            };
            self.table_key(key, axis, shape);
        }
    }

    fn table_key(&mut self, key: &Expr, axis: &Axis, shape: &TableShape) {
        let expected = axis.key_type(self.schema);
        let actual = self.expr(key, Some(&expected));
        let exact = axis.match_mode.eq_ignore_ascii_case("exact");
        if exact
            && let Type::List(item) = &actual
            && expected.accepts(item)
        {
            return;
        }
        let label = if axis.label.is_empty() {
            axis.kind.clone()
        } else {
            axis.label.clone()
        };
        self.fits(
            key,
            &actual,
            &expected,
            &Want::Argument {
                param: label,
                callee: format!("tables.{}", shape.code),
            },
        );
    }

    /// `tables.Code[k, …]`.
    pub fn table_index(&mut self, expr: &Expr, code: &Name, keys: &[Expr]) -> Type {
        let Some(shape) = self.table(code) else {
            for key in keys {
                self.expr(key, None);
            }
            return Type::Error;
        };
        let keys: Vec<&Expr> = keys.iter().collect();
        self.table_keys(expr, &shape, &keys);
        let value = self.table_value(&shape, code);
        Type::optional(value)
    }

    /// `tables.Code.cell(…)`, `.band(x)`, `.interpolate(x)`.
    pub fn table_method(
        &mut self,
        expr: &Expr,
        code: &Name,
        method: &Name,
        args: &[CallArg],
    ) -> Type {
        let known = TABLE_METHODS
            .iter()
            .find(|(name, _)| *name == method.text)
            .map(|(name, _)| *name);
        let Some(known) = known else {
            let candidates: Vec<String> = TABLE_METHODS
                .iter()
                .map(|(name, _)| (*name).to_owned())
                .collect();
            let found = Message::with(
                message!("A rate table has no method {0}"),
                [quote(&method.text)],
            );
            let missing = Message::with(
                message!(
                    "A rate table has no method {0}: read it with `[…]`, `.cell(…)`, `.band(x)` or `.interpolate(x)`"
                ),
                [quote(&method.text)],
            );
            let diagnostic = Self::did_you_mean(
                "unknown-field",
                method.span,
                &method.text,
                &candidates,
                found,
                missing,
            );
            self.report(diagnostic);
            return self.skip_args(args);
        };
        let Some(shape) = self.table(code) else {
            return self.skip_args(args);
        };
        self.callee(
            expr,
            Callee::Table {
                code: shape.code.clone(),
                method: known,
            },
        );
        for arg in args {
            if let Some(name) = &arg.name {
                self.report(Diagnostic::error(
                    "argument-order",
                    name.span,
                    Message::new(message!(
                        "A table's keys are given in axis order, without names"
                    )),
                ));
            }
        }
        let keys: Vec<&Expr> = args.iter().map(|arg| &arg.value).collect();
        let value = self.table_value(&shape, code);
        if known == "cell" {
            self.table_keys(expr, &shape, &keys);
            return Type::optional(record(&[
                ("value", value),
                ("minimum", Type::optional(Type::money())),
                ("deficit", Type::Bool),
            ]));
        }
        self.banded(expr, &shape, method, known, &keys, value)
    }

    /// `.band(x)` and `.interpolate(x)` on a one-axis table of bands.
    fn banded(
        &mut self,
        expr: &Expr,
        shape: &TableShape,
        method: &Name,
        known: &'static str,
        keys: &[&Expr],
        value: Type,
    ) -> Type {
        let banded = shape.axes.len() == 1
            && shape
                .axes
                .first()
                .is_some_and(|axis| axis.match_mode.eq_ignore_ascii_case("range"));
        if !banded {
            self.report(Diagnostic::error(
                "table-shape",
                method.span,
                Message::with(
                    message!("{0} works only on a table with one axis of bands, and {1} isn't one"),
                    [quote(known), quote(&shape.code)],
                ),
            ));
            for key in keys {
                self.expr(key, None);
            }
            return Type::Error;
        }
        self.table_keys(expr, shape, keys);
        let key = shape
            .axes
            .first()
            .map_or(Type::Error, |axis| axis.key_type(self.schema));
        if known == "interpolate" {
            if !matches!(
                value,
                Type::Quantity(_) | Type::Number | Type::Int | Type::Percent | Type::Error
            ) {
                self.report(Diagnostic::error(
                    "table-shape",
                    method.span,
                    Message::with(
                        message!("{0} holds {1}, which can't be interpolated"),
                        [quote(&shape.code), Arg::from(words(&value))],
                    ),
                ));
            }
            return Type::optional(value);
        }
        Type::optional(record(&[
            ("min", key.clone()),
            ("max", Type::optional(key)),
            ("value", value),
        ]))
    }
}

fn record(fields: &[(&str, Type)]) -> Type {
    Type::Record(Arc::new(RecordType {
        name: None,
        fields: fields
            .iter()
            .map(|(name, ty)| FieldDef {
                name: (*name).to_owned(),
                ty: ty.clone(),
                doc: None,
            })
            .collect(),
    }))
}
